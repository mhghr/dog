package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"monitoring-platform/packages/shared/config"
	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/security"
)

const testEncryptionKey = "test-encryption-key"

// fakeMonitoringAgentRepo is an in-memory repository.MonitoringAgentRepository.
type fakeMonitoringAgentRepo struct {
	agentsByAgentID map[string]*domain.MonitoringAgent
}

func newFakeMonitoringAgentRepo() *fakeMonitoringAgentRepo {
	return &fakeMonitoringAgentRepo{agentsByAgentID: map[string]*domain.MonitoringAgent{}}
}

func (f *fakeMonitoringAgentRepo) Create(ctx context.Context, agent *domain.MonitoringAgent) error {
	f.agentsByAgentID[agent.AgentID] = agent
	return nil
}

func (f *fakeMonitoringAgentRepo) GetByAgentID(ctx context.Context, agentID string) (domain.MonitoringAgent, error) {
	agent, ok := f.agentsByAgentID[agentID]
	if !ok {
		return domain.MonitoringAgent{}, domain.ErrNotFound
	}
	return *agent, nil
}

func (f *fakeMonitoringAgentRepo) GetByID(ctx context.Context, id string) (domain.MonitoringAgent, error) {
	for _, agent := range f.agentsByAgentID {
		if agent.ID == id {
			return *agent, nil
		}
	}
	return domain.MonitoringAgent{}, domain.ErrNotFound
}

func (f *fakeMonitoringAgentRepo) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]domain.MonitoringAgent, int, error) {
	return nil, 0, nil
}

func (f *fakeMonitoringAgentRepo) Update(ctx context.Context, agent *domain.MonitoringAgent) error {
	f.agentsByAgentID[agent.AgentID] = agent
	return nil
}

func (f *fakeMonitoringAgentRepo) UpdateStatus(ctx context.Context, agentID string, status domain.MonitoringAgentStatus) error {
	agent, ok := f.agentsByAgentID[agentID]
	if !ok {
		return domain.ErrNotFound
	}
	agent.Status = status
	return nil
}

func (f *fakeMonitoringAgentRepo) UpdateHeartbeat(ctx context.Context, agentID string, hb domain.AgentHeartbeat) error {
	return nil
}

func (f *fakeMonitoringAgentRepo) Delete(ctx context.Context, agentID string) error {
	delete(f.agentsByAgentID, agentID)
	return nil
}

func (f *fakeMonitoringAgentRepo) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	return 0, nil
}

// fakeBootstrapTokenRepo is an in-memory repository.BootstrapTokenRepository.
type fakeBootstrapTokenRepo struct {
	tokensByHash map[string]*domain.BootstrapToken
}

func newFakeBootstrapTokenRepo() *fakeBootstrapTokenRepo {
	return &fakeBootstrapTokenRepo{tokensByHash: map[string]*domain.BootstrapToken{}}
}

func (f *fakeBootstrapTokenRepo) Create(ctx context.Context, token *domain.BootstrapToken) error {
	f.tokensByHash[token.TokenHash] = token
	return nil
}

func (f *fakeBootstrapTokenRepo) GetByTokenHash(ctx context.Context, hash string) (domain.BootstrapToken, error) {
	token, ok := f.tokensByHash[hash]
	if !ok {
		return domain.BootstrapToken{}, domain.ErrNotFound
	}
	return *token, nil
}

func (f *fakeBootstrapTokenRepo) MarkUsedIfValid(ctx context.Context, tokenID string) error {
	for _, token := range f.tokensByHash {
		if token.ID != tokenID {
			continue
		}
		if !token.IsValid() {
			return domain.ErrNotFound
		}
		now := time.Now()
		token.UsedAt = &now
		return nil
	}
	return domain.ErrNotFound
}

func (f *fakeBootstrapTokenRepo) MarkRevoked(ctx context.Context, tokenID string) error {
	for _, token := range f.tokensByHash {
		if token.ID == tokenID {
			now := time.Now()
			token.RevokedAt = &now
			return nil
		}
	}
	return domain.ErrNotFound
}

func (f *fakeBootstrapTokenRepo) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]domain.BootstrapToken, int, error) {
	return nil, 0, nil
}

func (f *fakeBootstrapTokenRepo) DeleteExpired(ctx context.Context) (int64, error) {
	return 0, nil
}

// fakeAgentConfigRepo is an in-memory repository.AgentConfigRepository.
type fakeAgentConfigRepo struct {
	configsByAgentID map[string][]*domain.AgentConfig
}

func newFakeAgentConfigRepo() *fakeAgentConfigRepo {
	return &fakeAgentConfigRepo{configsByAgentID: map[string][]*domain.AgentConfig{}}
}

func (f *fakeAgentConfigRepo) Create(ctx context.Context, cfg *domain.AgentConfig) error {
	f.configsByAgentID[cfg.AgentID] = append(f.configsByAgentID[cfg.AgentID], cfg)
	return nil
}

func (f *fakeAgentConfigRepo) GetActive(ctx context.Context, agentID string) (*domain.AgentConfig, error) {
	configs := f.configsByAgentID[agentID]
	for i := len(configs) - 1; i >= 0; i-- {
		if configs[i].IsActive {
			return configs[i], nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeAgentConfigRepo) GetByVersion(ctx context.Context, agentID string, version int) (*domain.AgentConfig, error) {
	for _, cfg := range f.configsByAgentID[agentID] {
		if cfg.Version == version {
			return cfg, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeAgentConfigRepo) DeactivateOlder(ctx context.Context, agentID string, keepVersion int) error {
	return nil
}

func (f *fakeAgentConfigRepo) ListVersions(ctx context.Context, agentID string, limit int) ([]domain.AgentConfig, error) {
	return nil, nil
}

func newTestHandler() *Handler {
	deps := Deps{
		Config:           &config.Config{AgentSecretEncryptionKey: testEncryptionKey},
		Logger:           slog.New(slog.DiscardHandler),
		MonitoringAgents: newFakeMonitoringAgentRepo(),
		BootstrapTokens:  newFakeBootstrapTokenRepo(),
		AgentConfigs:     newFakeAgentConfigRepo(),
	}
	return &Handler{deps: deps}
}

func seedToken(repo *fakeBootstrapTokenRepo, rawToken string, expiresAt time.Time, usedAt *time.Time) string {
	now := time.Now()
	token := &domain.BootstrapToken{
		ID:        "tok_" + rawToken,
		TenantID:  "tenant-1",
		TokenHash: sha256Hex(rawToken),
		ExpiresAt: expiresAt,
		UsedAt:    usedAt,
		CreatedAt: now,
	}
	repo.Create(context.Background(), token)
	return token.ID
}

func doBootstrapRequest(h *Handler, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/api/monitoring/bootstrap", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.bootstrapAgent(rr, r)
	return rr
}

func doBootstrapRequestWithContext(h *Handler, ctx context.Context, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/api/monitoring/bootstrap", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(ctx)
	rr := httptest.NewRecorder()
	h.bootstrapAgent(rr, r)
	return rr
}

func doCompleteRegistrationRequest(h *Handler, agentID, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/api/monitoring/agents/"+agentID+"/complete", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("agentID", agentID)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

	rr := httptest.NewRecorder()
	h.completeAgentRegistration(rr, r)
	return rr
}

func decodeBody(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body %q: %v", rr.Body.String(), err)
	}
	return body
}

func TestBootstrapSuccess(t *testing.T) {
	h := newTestHandler()
	seedToken(h.deps.BootstrapTokens.(*fakeBootstrapTokenRepo), "bt_test", time.Now().Add(1*time.Hour), nil)

	body := `{"bootstrap_token":"bt_test","hostname":"web-01","os":"linux","architecture":"amd64","agent_version":"1.0.0","capabilities":["cpu"]}`
	rr := doBootstrapRequestWithContext(h, context.Background(), body)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	resp := decodeBody(t, rr)
	agentID, _ := resp["agent_id"].(string)
	agentSecret, _ := resp["agent_secret"].(string)
	if !strings.HasPrefix(agentID, "ag_") {
		t.Fatalf("expected agent_id to start with ag_, got %q", agentID)
	}
	if agentSecret == "" {
		t.Fatal("expected non-empty agent_secret")
	}
	if cfgURL, _ := resp["config_url"].(string); !strings.Contains(cfgURL, agentID) {
		t.Fatalf("expected config_url to reference agent %q, got %q", agentID, cfgURL)
	}
	if hbURL, _ := resp["heartbeat_url"].(string); !strings.Contains(hbURL, agentID) {
		t.Fatalf("expected heartbeat_url to reference agent %q, got %q", agentID, hbURL)
	}

	agent, err := h.deps.MonitoringAgents.GetByAgentID(context.Background(), agentID)
	if err != nil {
		t.Fatalf("expected stored agent %q, got error: %v", agentID, err)
	}
	if agent.SecretHash == "" {
		t.Error("expected stored agent to have non-empty SecretHash")
	}
	if agent.SecretEncrypted == "" {
		t.Error("expected stored agent to have non-empty SecretEncrypted")
	}
	if agent.TenantID != "tenant-1" {
		t.Errorf("expected tenant_id tenant-1, got %q", agent.TenantID)
	}

	tokenRepo := h.deps.BootstrapTokens.(*fakeBootstrapTokenRepo)
	token, err := tokenRepo.GetByTokenHash(context.Background(), sha256Hex("bt_test"))
	if err != nil {
		t.Fatalf("expected token to exist, got error: %v", err)
	}
	if !token.IsUsed() {
		t.Error("expected bootstrap token to be marked used")
	}
}

func TestBootstrapMissingFields(t *testing.T) {
	h := newTestHandler()

	tests := []struct {
		name string
		body string
	}{
		{"missing token", `{"hostname":"web-01"}`},
		{"missing hostname", `{"bootstrap_token":"bt_test"}`},
		{"empty both", `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := doBootstrapRequest(h, tt.body)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestBootstrapUnknownToken(t *testing.T) {
	h := newTestHandler()

	rr := doBootstrapRequest(h, `{"bootstrap_token":"does-not-exist","hostname":"web-01"}`)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestBootstrapExpiredToken(t *testing.T) {
	h := newTestHandler()
	seedToken(h.deps.BootstrapTokens.(*fakeBootstrapTokenRepo), "bt_expired", time.Now().Add(-1*time.Hour), nil)

	rr := doBootstrapRequest(h, `{"bootstrap_token":"bt_expired","hostname":"web-01"}`)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestBootstrapAlreadyUsedToken(t *testing.T) {
	h := newTestHandler()
	usedAt := time.Now()
	seedToken(h.deps.BootstrapTokens.(*fakeBootstrapTokenRepo), "bt_used", time.Now().Add(1*time.Hour), &usedAt)

	rr := doBootstrapRequest(h, `{"bootstrap_token":"bt_used","hostname":"web-01"}`)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestBootstrapSecretEncrypted(t *testing.T) {
	h := newTestHandler()
	seedToken(h.deps.BootstrapTokens.(*fakeBootstrapTokenRepo), "bt_secret", time.Now().Add(1*time.Hour), nil)

	rr := doBootstrapRequest(h, `{"bootstrap_token":"bt_secret","hostname":"web-01","os":"linux","architecture":"amd64","agent_version":"1.0.0"}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	resp := decodeBody(t, rr)
	agentID, _ := resp["agent_id"].(string)
	returnedSecret, _ := resp["agent_secret"].(string)

	agent, err := h.deps.MonitoringAgents.GetByAgentID(context.Background(), agentID)
	if err != nil {
		t.Fatalf("expected stored agent, got error: %v", err)
	}

	decrypted, err := security.DecryptSecret(testEncryptionKey, agent.SecretEncrypted)
	if err != nil {
		t.Fatalf("failed to decrypt stored secret: %v", err)
	}
	if decrypted != returnedSecret {
		t.Fatalf("expected decrypted secret to equal returned agent_secret, got %q vs %q", decrypted, returnedSecret)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(agent.SecretHash), []byte(returnedSecret)); err != nil {
		t.Fatalf("expected stored SecretHash to match returned agent_secret: %v", err)
	}
}

func TestCompleteRegistrationSuccess(t *testing.T) {
	h := newTestHandler()
	secret := "known-agent-secret"
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash secret: %v", err)
	}
	now := time.Now()
	agent := &domain.MonitoringAgent{
		ID:              "agent-uuid-1",
		TenantID:        "tenant-1",
		ExternalID:      "ext-1",
		Hostname:        "web-01",
		OS:              "linux",
		Arch:            "amd64",
		Version:         "1.0.0",
		AgentID:         "ag_complete",
		SecretHash:      string(hash),
		SecretEncrypted: "encrypted",
		Status:          domain.AgentStatusActive,
		RegisteredAt:    now,
		UpdatedAt:       now,
		Labels:          map[string]string{"hostname": "web-01"},
		PrivateIPs:      []string{},
	}
	if err := h.deps.MonitoringAgents.Create(context.Background(), agent); err != nil {
		t.Fatalf("failed to seed agent: %v", err)
	}

	body := `{"secret_proof":"known-agent-secret","private_ips":["10.0.0.5"],"labels":{"env":"prod","region":"us-east-1"}}`
	rr := doCompleteRegistrationRequest(h, "ag_complete", body)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	updated, err := h.deps.MonitoringAgents.GetByAgentID(context.Background(), "ag_complete")
	if err != nil {
		t.Fatalf("expected stored agent, got error: %v", err)
	}
	if updated.Labels["env"] != "prod" || updated.Labels["region"] != "us-east-1" {
		t.Fatalf("expected labels to be updated, got %v", updated.Labels)
	}
	if len(updated.PrivateIPs) != 1 || updated.PrivateIPs[0] != "10.0.0.5" {
		t.Fatalf("expected private_ips to be updated, got %v", updated.PrivateIPs)
	}

	cfg, err := h.deps.AgentConfigs.GetActive(context.Background(), "ag_complete")
	if err != nil {
		t.Fatalf("expected default config to be created, got error: %v", err)
	}
	if cfg.Version != 1 {
		t.Fatalf("expected config version 1, got %d", cfg.Version)
	}
	if !cfg.IsActive {
		t.Error("expected default config to be active")
	}
}

func TestCompleteRegistrationBadSecret(t *testing.T) {
	h := newTestHandler()
	secret := "known-agent-secret"
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash secret: %v", err)
	}
	now := time.Now()
	agent := &domain.MonitoringAgent{
		ID:              "agent-uuid-2",
		TenantID:        "tenant-1",
		AgentID:         "ag_badsecret",
		SecretHash:      string(hash),
		SecretEncrypted: "encrypted",
		Status:          domain.AgentStatusActive,
		RegisteredAt:    now,
		UpdatedAt:       now,
		Labels:          map[string]string{},
		PrivateIPs:      []string{},
	}
	if err := h.deps.MonitoringAgents.Create(context.Background(), agent); err != nil {
		t.Fatalf("failed to seed agent: %v", err)
	}

	rr := doCompleteRegistrationRequest(h, "ag_badsecret", `{"secret_proof":"wrong-secret"}`)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestCompleteRegistrationMissingSecret(t *testing.T) {
	h := newTestHandler()
	hash, err := bcrypt.GenerateFromPassword([]byte("known-agent-secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("failed to hash secret: %v", err)
	}
	now := time.Now()
	agent := &domain.MonitoringAgent{
		ID:              "agent-uuid-3",
		TenantID:        "tenant-1",
		AgentID:         "ag_missingsecret",
		SecretHash:      string(hash),
		SecretEncrypted: "encrypted",
		Status:          domain.AgentStatusActive,
		RegisteredAt:    now,
		UpdatedAt:       now,
		Labels:          map[string]string{},
		PrivateIPs:      []string{},
	}
	if err := h.deps.MonitoringAgents.Create(context.Background(), agent); err != nil {
		t.Fatalf("failed to seed agent: %v", err)
	}

	rr := doCompleteRegistrationRequest(h, "ag_missingsecret", `{}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", rr.Code, rr.Body.String())
	}
}
