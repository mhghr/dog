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

	"monitoring-platform/packages/shared/domain"
)

// Engine evaluates probe results against alert policies.
type Engine struct {
	alerts domain.AlertRepository
	Logger *slog.Logger
}

func NewEngine(alerts domain.AlertRepository, logger *slog.Logger) *Engine {
	return &Engine{alerts: alerts, Logger: logger}
}

// Evaluate checks one probe result against all applicable alert policies
// for its monitor, updating alert state and returning firing events.
func (e *Engine) Evaluate(ctx context.Context, result domain.ProbeResult) []domain.EvaluateResult {
	policies, err := e.alerts.ListActivePolicies(ctx, result.MonitorID)
	if err != nil {
		e.Logger.Error("alert evaluate: list policies failed", "monitor_id", result.MonitorID, "error", err)
		return nil
	}

	var events []domain.EvaluateResult

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

func (e *Engine) evaluatePolicy(ctx context.Context, policy domain.AlertPolicy, result domain.ProbeResult) *domain.EvaluateResult {
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
		alert = domain.Alert{
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
		return &domain.EvaluateResult{
			AlertID:     alert.ID,
			OldState:    oldState,
			NewState:    alert.State,
			MonitorID:   result.MonitorID,
			MonitorName: result.MonitorName,
			Severity:    policy.Severity,
			Title:       policy.Name,
			ChannelIDs:  policy.ChannelIDs,
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Notification Dispatcher
// ---------------------------------------------------------------------------

type Notifier struct {
	channels domain.ChannelRepository
	Logger   *slog.Logger
}

func NewNotifier(channels domain.ChannelRepository, logger *slog.Logger) *Notifier {
	return &Notifier{channels: channels, Logger: logger}
}

func (n *Notifier) Dispatch(ctx context.Context, event domain.EvaluateResult, channelIDs []string) {
	channels, err := n.channels.ListByIDs(ctx, channelIDs)
	if err != nil {
		n.Logger.Error("notifier: list channels failed", "error", err)
		return
	}

	notif := domain.Notification{
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

func (n *Notifier) sendWebhook(channel domain.NotificationChannel, notif domain.Notification) {
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

func (n *Notifier) sendEmail(channel domain.NotificationChannel, notif domain.Notification) {
	// Stub — Phase 3 primary delivery: log and mark as pending SMTP
	// integration. The channel config carries "to" address.
	n.Logger.Info("email notification (stub) sent to",
		"to", channel.Config["to"],
		"title", notif.Title,
		"state", notif.State,
	)
}

func (n *Notifier) sendTelegram(channel domain.NotificationChannel, notif domain.Notification) {
	botToken := channel.Config["bot_token"]
	chatID := channel.Config["chat_id"]
	if botToken == "" || chatID == "" {
		n.Logger.Warn("telegram: missing bot_token or chat_id", "channel", channel.Name)
		return
	}

	text := fmt.Sprintf("[ALERT] %s\n%s - %s\nState: %s", notif.Title, notif.Monitor, notif.MonitorID, notif.State)
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
