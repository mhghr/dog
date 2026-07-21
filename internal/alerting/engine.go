package alerting

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"monitoring-platform/internal/domain"
)

// ---------------------------------------------------------------------------
// Domain types
// ---------------------------------------------------------------------------

type NotificationChannel struct {
	ID             string
	OrganizationID string
	Name           string
	Type           string // email | webhook | telegram
	Config         map[string]string
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type AlertPolicy struct {
	ID                 string
	OrganizationID     string
	ProjectID          string
	Name               string
	Scope              domain.AlertPolicyScope
	Conditions         domain.AlertConditions
	Severity           string
	OpeningFailures    int
	ResolvingSuccesses int
	CooldownSeconds    int
	RenotifySeconds    int
	Enabled            bool
	ChannelIDs         []string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Alert struct {
	ID                  string
	OrganizationID      string
	PolicyID            string
	MonitorID           string
	MonitorName         string
	State               string
	Severity            string
	Title               string
	Description         string
	DedupKey            string
	ConsecutiveFailures int
	ConsecutiveSuccesses int
	OpenedAt            *time.Time
	ResolvedAt          *time.Time
	CreatedAt           time.Time
}

// EvaluateResult decides whether a single probe result should change any
// alert state. It returns the list of changed alerts. The caller (result
// ingestion pipeline) stores them and publishes notification events.
type EvaluateResult struct {
	AlertID    string
	OldState   string
	NewState   string
	MonitorID  string
	MonitorName string
	Severity   string
	Title      string
	ChannelIDs []string
}

// Engine evaluates probe results against alert policies.
type Engine struct {
	alerts  AlertRepository
	Logger  *slog.Logger
}

type AlertRepository interface {
	ListActivePolicies(ctx context.Context, monitorID string) ([]AlertPolicy, error)
	FindByDedup(ctx context.Context, dedupKey string) (Alert, error)
	UpsertAlert(ctx context.Context, alert *Alert) error
	ListFiring(ctx context.Context) ([]Alert, error)
	RecordNotification(ctx context.Context, alertID string) error
}

func NewEngine(alerts AlertRepository, logger *slog.Logger) *Engine {
	return &Engine{alerts: alerts, Logger: logger}
}

// Evaluate checks one probe result against all applicable alert policies
// for its monitor, updating alert state and returning firing events.
func (e *Engine) Evaluate(ctx context.Context, result domain.ProbeResult) []EvaluateResult {
	policies, err := e.alerts.ListActivePolicies(ctx, result.MonitorID)
	if err != nil {
		e.Logger.Error("alert evaluate: list policies failed", "monitor_id", result.MonitorID, "error", err)
		return nil
	}

	var events []EvaluateResult

	for _, policy := range policies {
		if !matchesScope(policy.Scope, result) {
			continue
		}

		outcome := e.evaluatePolicy(ctx, policy, result)
		if outcome != nil {
			events = append(events, *outcome)
		}
	}

	return events
}

func matchesScope(scope domain.AlertPolicyScope, result domain.ProbeResult) bool {
	if len(scope.MonitorIDs) > 0 && !slices.Contains(scope.MonitorIDs, result.MonitorID) {
		return false
	}
	return true
}

func (e *Engine) evaluatePolicy(ctx context.Context, policy AlertPolicy, result domain.ProbeResult) *EvaluateResult {
	dedupKey := fmt.Sprintf("%s:%s", policy.ID, result.MonitorID)

	existing, err := e.alerts.FindByDedup(ctx, dedupKey)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		e.Logger.Error("alert: find existing failed", "dedup", dedupKey, "error", err)
		return nil
	}

	isFailure := result.Status == domain.StatusDown
	isSuccess := result.Status == domain.StatusUp

	alert := existing
	if errors.Is(err, domain.ErrNotFound) {
		alert = Alert{
			OrganizationID: policy.OrganizationID,
			PolicyID:       policy.ID,
			MonitorID:      result.MonitorID,
			MonitorName:    result.MonitorName,
			State:          "pending",
			Severity:       policy.Severity,
			Title:          policy.Name,
			DedupKey:       dedupKey,
		}
		alert.ID = uuid.Must(uuid.NewV7()).String()
	}

	oldState := alert.State

	if isFailure {
		alert.ConsecutiveFailures++
		alert.ConsecutiveSuccesses = 0

		if alert.State == "pending" && alert.ConsecutiveFailures >= policy.OpeningFailures {
			alert.State = "firing"
		}
		if alert.State == "recovering" && alert.ConsecutiveFailures >= policy.OpeningFailures {
			alert.State = "firing"
		}
	} else if isSuccess && alert.State != "pending" && alert.State != "resolved" {
		alert.ConsecutiveSuccesses++

		if alert.State == "firing" && alert.ConsecutiveSuccesses >= policy.ResolvingSuccesses {
			alert.State = "recovering"
		}
	}

	if alert.State == "recovering" && alert.ConsecutiveSuccesses >= policy.ResolvingSuccesses+2 {
		alert.State = "resolved"
		now := time.Now().UTC()
		alert.ResolvedAt = &now
	}

	if alert.State != "resolved" && alert.State != "pending" {
		alert.Description = fmt.Sprintf("%d consecutive failures (opening threshold: %d)", alert.ConsecutiveFailures, policy.OpeningFailures)
	} else if alert.State == "resolved" {
		alert.Description = fmt.Sprintf("Resolved after %d consecutive successes", alert.ConsecutiveSuccesses)
	}

	alert.DedupKey = dedupKey
	if err := e.alerts.UpsertAlert(ctx, &alert); err != nil {
		e.Logger.Error("alert: upsert failed", "alert_id", alert.ID, "error", err)
		return nil
	}

	if alert.State != oldState && (alert.State == "firing" || alert.State == "recovering" || alert.State == "resolved") {
		return &EvaluateResult{
			AlertID:    alert.ID,
			OldState:   oldState,
			NewState:   alert.State,
			MonitorID:  result.MonitorID,
			MonitorName: result.MonitorName,
			Severity:   policy.Severity,
			Title:      policy.Name,
			ChannelIDs: policy.ChannelIDs,
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Notification Dispatcher
// ---------------------------------------------------------------------------

type Notifier struct {
	channels ChannelRepository
	Logger   *slog.Logger
}

type ChannelRepository interface {
	ListByIDs(ctx context.Context, ids []string) ([]NotificationChannel, error)
}

func NewNotifier(channels ChannelRepository, logger *slog.Logger) *Notifier {
	return &Notifier{channels: channels, Logger: logger}
}

type Notification struct {
	AlertID    string
	Title      string
	State      string
	Severity   string
	Monitor    string
	MonitorID  string
	FiredAt    time.Time
}

func (n *Notifier) Dispatch(ctx context.Context, event EvaluateResult, channelIDs []string) {
	channels, err := n.channels.ListByIDs(ctx, channelIDs)
	if err != nil {
		n.Logger.Error("notifier: list channels failed", "error", err)
		return
	}

	notif := Notification{
		AlertID:   event.AlertID,
		Title:     event.Title,
		State:     event.NewState,
		Severity:  event.Severity,
		Monitor:   event.MonitorName,
		MonitorID: event.MonitorID,
		FiredAt:   time.Now().UTC(),
	}

	for _, channel := range channels {
		if !channel.Enabled {
			continue
		}
		switch channel.Type {
		case "webhook":
			go n.sendWebhook(channel, notif)
		case "email":
			go n.sendEmail(channel, notif)
		case "telegram":
			go n.sendTelegram(channel, notif)
		}
	}
}

func (n *Notifier) sendWebhook(channel NotificationChannel, notif Notification) {
	payload, _ := json.Marshal(notif)

	signature := ""
	if secret, ok := channel.Config["secret"]; ok && secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		signature = hex.EncodeToString(mac.Sum(nil))
	}

	req, err := http.NewRequest(http.MethodPost, channel.Config["url"], strings.NewReader(string(payload)))
	if err != nil {
		n.Logger.Error("webhook: create request failed", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if signature != "" {
		req.Header.Set("X-MP-Signature", "sha256="+signature)
	}
	req.Header.Set("X-MP-Event", "alert")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		n.Logger.Error("webhook: dispatch failed", "channel", channel.Name, "error", err)
		return
	}
	resp.Body.Close()
	n.Logger.Info("webhook dispatched", "channel", channel.Name, "status", resp.StatusCode)
}

func (n *Notifier) sendEmail(channel NotificationChannel, notif Notification) {
	// Stub — Phase 3 primary delivery: log and mark as pending SMTP
	// integration. The channel config carries "to" address.
	n.Logger.Info("email notification (stub) sent to",
		"to", channel.Config["to"],
		"title", notif.Title,
		"state", notif.State,
	)
}

func (n *Notifier) sendTelegram(channel NotificationChannel, notif Notification) {
	botToken := channel.Config["bot_token"]
	chatID := channel.Config["chat_id"]
	if botToken == "" || chatID == "" {
		n.Logger.Warn("telegram: missing bot_token or chat_id", "channel", channel.Name)
		return
	}

	text := fmt.Sprintf("🚨 %s\n%s — %s\nState: %s", notif.Title, notif.Monitor, notif.MonitorID, notif.State)
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	form := url.Values{
		"chat_id": {chatID},
		"text":    {text},
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(apiURL, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		n.Logger.Error("telegram: dispatch failed", "error", err)
		return
	}
	resp.Body.Close()
	n.Logger.Info("telegram dispatched", "channel", channel.Name, "status", resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------
