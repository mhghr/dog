package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"monitoring-platform/internal/domain"
	"monitoring-platform/internal/repository"
)

const (
	otpLength         = 6
	otpTTL            = 5 * time.Minute
	otpMaxAttempts    = 5
	otpMinResendGap   = time.Minute
	otpHourlyLimit    = 5
	refreshTokenBytes = 48
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrGoogleDisabled     = errors.New("google login is not configured")
	ErrInvalidRedirectURI = errors.New("redirect uri is not allowed")
	ErrEmailNotVerified   = errors.New("google email is not verified")
)

// RateLimitError carries the retry hint for OTP throttling.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("rate limited, retry after %s", e.RetryAfter)
}

type TokenPair struct {
	AccessToken      string
	AccessExpiresIn  int
	RefreshToken     string
	RefreshExpiresIn int
}

type Config struct {
	RefreshTTL     time.Duration
	GoogleRedirect string
	OTPDevMode     bool
}

type Service struct {
	users  repository.UserRepository
	tokens repository.RefreshTokenRepository
	otps   repository.OTPRepository
	orgs   repository.OrganizationRepository
	issuer *TokenIssuer
	google *GoogleVerifier
	sms    SMSSender
	cfg    Config
	otpKey []byte
	logger *slog.Logger
}

func NewService(
	users repository.UserRepository,
	tokens repository.RefreshTokenRepository,
	otps repository.OTPRepository,
	orgs repository.OrganizationRepository,
	issuer *TokenIssuer,
	google *GoogleVerifier,
	sms SMSSender,
	cfg Config,
	otpSecret string,
	logger *slog.Logger,
) *Service {
	if cfg.RefreshTTL <= 0 {
		cfg.RefreshTTL = 30 * 24 * time.Hour
	}

	return &Service{
		users:  users,
		tokens: tokens,
		otps:   otps,
		orgs:   orgs,
		issuer: issuer,
		google: google,
		sms:    sms,
		cfg:    cfg,
		otpKey: []byte(otpSecret),
		logger: logger,
	}
}

// LoginWithGoogleCode implements the server-side authorization code flow
// used by the web console.
func (s *Service) LoginWithGoogleCode(ctx context.Context, code, redirectURI string) (domain.User, TokenPair, error) {
	if !s.google.Enabled() {
		return domain.User{}, TokenPair{}, ErrGoogleDisabled
	}

	if redirectURI == "" || redirectURI != s.cfg.GoogleRedirect {
		return domain.User{}, TokenPair{}, ErrInvalidRedirectURI
	}

	idToken, accessToken, err := s.google.ExchangeCode(ctx, code, redirectURI)
	if err != nil {
		s.logger.Warn("google code exchange failed", "error", err)
		return domain.User{}, TokenPair{}, ErrInvalidCredentials
	}

	claims, err := s.google.VerifyIDToken(ctx, idToken)
	if err != nil {
		s.logger.Warn("google id token rejected", "error", err)
		return domain.User{}, TokenPair{}, ErrInvalidCredentials
	}

	// Some ID tokens omit profile claims; enrich from userinfo (best effort).
	if (claims.Picture == "" || claims.Name == "") && accessToken != "" {
		if info, infoErr := s.google.FetchUserInfo(ctx, accessToken); infoErr == nil {
			if claims.Picture == "" {
				claims.Picture = info.Picture
			}
			if claims.Name == "" {
				claims.Name = info.Name
			}
		} else {
			s.logger.Warn("google userinfo fetch failed", "error", infoErr)
		}
	}

	return s.loginWithGoogleClaims(ctx, claims)
}

// LoginWithGoogleIDToken implements the mobile/native flow: the app obtains
// an ID token via Google Sign-In and posts it directly.
func (s *Service) LoginWithGoogleIDToken(ctx context.Context, idToken string) (domain.User, TokenPair, error) {
	if !s.google.Enabled() {
		return domain.User{}, TokenPair{}, ErrGoogleDisabled
	}

	claims, err := s.google.VerifyIDToken(ctx, idToken)
	if err != nil {
		s.logger.Warn("google id token rejected", "error", err)
		return domain.User{}, TokenPair{}, ErrInvalidCredentials
	}

	return s.loginWithGoogleClaims(ctx, claims)
}

func (s *Service) loginWithGoogleClaims(ctx context.Context, claims GoogleClaims) (domain.User, TokenPair, error) {
	email := claims.Email
	if !claims.EmailVerified {
		email = "" // Never link accounts through an unverified email.
	}

	user, err := s.users.UpsertGoogleUser(ctx, claims.Sub, email, claims.Name, claims.Picture)
	if err != nil {
		return domain.User{}, TokenPair{}, err
	}

	if err := s.ensureOrganization(ctx, &user); err != nil {
		return domain.User{}, TokenPair{}, err
	}

	pair, err := s.issueTokens(ctx, user)
	if err != nil {
		return domain.User{}, TokenPair{}, err
	}

	s.logger.Info("user logged in", "user_id", user.ID, "method", "google")
	return user, pair, nil
}

// RequestOTP creates and delivers a one-time code. In dev mode the code is
// also returned so local flows work without an SMS provider.
func (s *Service) RequestOTP(ctx context.Context, rawPhone string) (devCode string, retryAfter int, err error) {
	phone, err := NormalizePhone(rawPhone)
	if err != nil {
		return "", 0, ErrInvalidCredentials
	}

	recentCount, err := s.otps.CountSince(ctx, phone, time.Now().Add(-otpMinResendGap))
	if err != nil {
		return "", 0, err
	}
	if recentCount > 0 {
		return "", int(otpMinResendGap.Seconds()), &RateLimitError{RetryAfter: otpMinResendGap}
	}

	hourlyCount, err := s.otps.CountSince(ctx, phone, time.Now().Add(-time.Hour))
	if err != nil {
		return "", 0, err
	}
	if hourlyCount >= otpHourlyLimit {
		return "", int(time.Hour.Seconds()), &RateLimitError{RetryAfter: time.Hour}
	}

	code, err := generateOTPCode()
	if err != nil {
		return "", 0, err
	}

	if err := s.otps.Create(ctx, phone, s.hashOTP(phone, code), time.Now().Add(otpTTL)); err != nil {
		return "", 0, err
	}

	message := fmt.Sprintf("Monitoring Platform verification code: %s", code)
	if err := s.sms.Send(ctx, phone, message); err != nil {
		s.logger.Error("send otp sms failed", "error", err)
		return "", 0, fmt.Errorf("send otp: %w", err)
	}

	if s.cfg.OTPDevMode {
		return code, int(otpMinResendGap.Seconds()), nil
	}

	return "", int(otpMinResendGap.Seconds()), nil
}

// VerifyOTP checks the code and logs the user in, provisioning the account
// on first login.
func (s *Service) VerifyOTP(ctx context.Context, rawPhone, code string) (domain.User, TokenPair, error) {
	phone, err := NormalizePhone(rawPhone)
	if err != nil {
		return domain.User{}, TokenPair{}, ErrInvalidCredentials
	}

	otp, err := s.otps.LatestActive(ctx, phone)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, TokenPair{}, ErrInvalidCredentials
	}
	if err != nil {
		return domain.User{}, TokenPair{}, err
	}

	if otp.Attempts >= otpMaxAttempts {
		return domain.User{}, TokenPair{}, ErrInvalidCredentials
	}

	expected := s.hashOTP(phone, code)
	if !hmac.Equal([]byte(expected), []byte(otp.CodeHash)) {
		if err := s.otps.RegisterFailure(ctx, otp.ID); err != nil {
			s.logger.Warn("register otp failure", "error", err)
		}
		return domain.User{}, TokenPair{}, ErrInvalidCredentials
	}

	if err := s.otps.Consume(ctx, otp.ID); err != nil {
		return domain.User{}, TokenPair{}, err
	}

	user, err := s.users.UpsertPhoneUser(ctx, phone)
	if err != nil {
		return domain.User{}, TokenPair{}, err
	}

	if err := s.ensureOrganization(ctx, &user); err != nil {
		return domain.User{}, TokenPair{}, err
	}

	pair, err := s.issueTokens(ctx, user)
	if err != nil {
		return domain.User{}, TokenPair{}, err
	}

	s.logger.Info("user logged in", "user_id", user.ID, "method", "otp")
	return user, pair, nil
}

// Refresh rotates the refresh token and issues a fresh access token.
func (s *Service) Refresh(ctx context.Context, rawRefreshToken string) (domain.User, TokenPair, error) {
	if rawRefreshToken == "" {
		return domain.User{}, TokenPair{}, ErrInvalidCredentials
	}

	newToken, err := generateRefreshToken()
	if err != nil {
		return domain.User{}, TokenPair{}, err
	}

	userID, err := s.tokens.Rotate(
		ctx,
		hashToken(rawRefreshToken),
		hashToken(newToken),
		time.Now().Add(s.cfg.RefreshTTL),
	)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound),
			errors.Is(err, domain.ErrTokenExpired):
			return domain.User{}, TokenPair{}, ErrInvalidCredentials
		case errors.Is(err, domain.ErrTokenReused):
			s.logger.Warn("refresh token reuse detected; sessions revoked")
			return domain.User{}, TokenPair{}, ErrInvalidCredentials
		default:
			return domain.User{}, TokenPair{}, err
		}
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return domain.User{}, TokenPair{}, err
	}

	accessToken, accessExpiresIn, err := s.issuer.IssueAccessToken(user, user.OrganizationID)
	if err != nil {
		return domain.User{}, TokenPair{}, err
	}

	return user, TokenPair{
		AccessToken:      accessToken,
		AccessExpiresIn:  accessExpiresIn,
		RefreshToken:     newToken,
		RefreshExpiresIn: int(s.cfg.RefreshTTL.Seconds()),
	}, nil
}

func (s *Service) Logout(ctx context.Context, rawRefreshToken string) error {
	if rawRefreshToken == "" {
		return nil
	}

	return s.tokens.Revoke(ctx, hashToken(rawRefreshToken))
}

func (s *Service) GetUser(ctx context.Context, userID string) (domain.User, error) {
	return s.users.GetByID(ctx, userID)
}

func (s *Service) issueTokens(ctx context.Context, user domain.User) (TokenPair, error) {
	accessToken, accessExpiresIn, err := s.issuer.IssueAccessToken(user, user.OrganizationID)
	if err != nil {
		return TokenPair{}, err
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}

	if err := s.tokens.Create(
		ctx,
		user.ID,
		hashToken(refreshToken),
		time.Now().Add(s.cfg.RefreshTTL),
	); err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:      accessToken,
		AccessExpiresIn:  accessExpiresIn,
		RefreshToken:     refreshToken,
		RefreshExpiresIn: int(s.cfg.RefreshTTL.Seconds()),
	}, nil
}

func (s *Service) ensureOrganization(ctx context.Context, user *domain.User) error {
	if user.OrganizationID != "" {
		return nil
	}

	name := strings.TrimSpace(user.Name)
	if name == "" {
		if user.Email != "" {
			parts := strings.SplitN(user.Email, "@", 2)
			name = parts[0]
		} else if user.Phone != "" {
			name = "user-" + user.Phone
		} else {
			name = "User"
		}
	}

	orgName := name + "'s Workspace"
	org, errs := domain.ValidateOrganizationInput(domain.OrganizationInput{Name: orgName})
	if len(errs) > 0 {
		org = domain.Organization{Name: "My Workspace", Slug: "my-workspace"}
	}
	if err := s.orgs.Create(ctx, &org); err != nil {
		return fmt.Errorf("auto-create organization: %w", err)
	}

	if err := s.users.SetOrganizationID(ctx, user.ID, org.ID); err != nil {
		return fmt.Errorf("set user organization: %w", err)
	}

	user.OrganizationID = org.ID

	s.logger.Info("auto-created organization", "user_id", user.ID, "org_id", org.ID, "org_name", org.Name)
	return nil
}

func (s *Service) hashOTP(phone, code string) string {
	mac := hmac.New(sha256.New, s.otpKey)
	mac.Write([]byte(phone))
	mac.Write([]byte(":"))
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

func generateOTPCode() (string, error) {
	const digits = "0123456789"

	buffer := make([]byte, otpLength)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate otp: %w", err)
	}

	code := make([]byte, otpLength)
	for index, value := range buffer {
		code[index] = digits[int(value)%len(digits)]
	}

	return string(code), nil
}

func generateRefreshToken() (string, error) {
	buffer := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
