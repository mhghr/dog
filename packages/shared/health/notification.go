package health

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type NotificationEngine struct {
	repo   Repository
	logger *slog.Logger
}

func NewNotificationEngine(repo Repository, logger *slog.Logger) *NotificationEngine {
	return &NotificationEngine{repo: repo, logger: logger}
}

func (n *NotificationEngine) SendTest(ctx context.Context, channelID string) error {
	channel, err := n.repo.GetNotificationChannel(ctx, channelID)
	if err != nil {
		return fmt.Errorf("channel not found: %w", err)
	}

	if !channel.Enabled {
		return fmt.Errorf("channel is disabled")
	}

	notif := HealthNotification{
		MonitorID:    "test-monitor",
		ParameterKey: "test.parameter",
		OldState:     HealthOK,
		NewState:     HealthWarning,
		CurrentValue: 42.0,
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
	}

	n.dispatch(ctx, channel, notif)
	return nil
}

type HealthNotification struct {
	MonitorID    string      `json:"monitor_id"`
	ParameterKey string      `json:"parameter_key"`
	OldState     HealthState `json:"old_state"`
	NewState     HealthState `json:"new_state"`
	CurrentValue float64     `json:"current_value"`
	Timestamp    string      `json:"timestamp"`
}

func (n *NotificationEngine) Evaluate(ctx context.Context, outcome EvaluateOutcome) {
	policies, err := n.repo.ListNotificationPolicies(ctx, outcome.MonitorID)
	if err != nil {
		n.logger.Error("health notification: list policies failed", "error", err)
		return
	}

	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}

		if policy.ParameterKey != nil && *policy.ParameterKey != outcome.ParameterKey {
			continue
		}

		trigger := determineTrigger(outcome.OldState, outcome.NewState)
		if !hasTrigger(policy.Triggers, trigger) && !hasTrigger(policy.Triggers, "all") {
			continue
		}

		channel, err := n.repo.GetNotificationChannel(ctx, policy.ChannelID)
		if err != nil {
			n.logger.Error("health notification: channel not found", "channel_id", policy.ChannelID, "error", err)
			continue
		}

		if !channel.Enabled {
			continue
		}

		notif := HealthNotification{
			MonitorID:    outcome.MonitorID,
			ParameterKey: outcome.ParameterKey,
			OldState:     outcome.OldState,
			NewState:     outcome.NewState,
			CurrentValue: outcome.CurrentValue,
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		}

		n.dispatch(ctx, channel, notif)
	}
}

func determineTrigger(oldState, newState HealthState) string {
	switch {
	case oldState == HealthOK && newState == HealthWarning:
		return "STATUS_ENTERED_WARNING"
	case oldState == HealthOK && newState == HealthError:
		return "STATUS_ENTERED_ERROR"
	case oldState == HealthWarning && newState == HealthError:
		return "STATUS_ENTERED_ERROR"
	case newState == HealthOK && (oldState == HealthWarning || oldState == HealthError):
		return "RECOVERED_TO_OK"
	case newState == HealthUnknown && oldState != HealthUnknown:
		return "STATUS_UNKNOWN"
	default:
		return "STATUS_CHANGED"
	}
}

func hasTrigger(triggers []string, trigger string) bool {
	for _, t := range triggers {
		if t == trigger {
			return true
		}
	}
	return false
}

func (n *NotificationEngine) dispatch(ctx context.Context, channel HealthNotificationChannel, notif HealthNotification) {
	switch channel.Type {
	case "webhook":
		n.sendWebhook(channel, notif)
	case "email":
		n.sendEmail(channel, notif)
	case "telegram":
		n.sendTelegram(channel, notif)
	case "slack":
		n.sendSlack(channel, notif)
	case "discord":
		n.sendDiscord(channel, notif)
	case "teams":
		n.sendTeams(channel, notif)
	default:
		n.logger.Warn("health notification: unknown channel type", "type", channel.Type)
	}
}

func (n *NotificationEngine) sendWebhook(channel HealthNotificationChannel, notif HealthNotification) {
	payload, _ := json.Marshal(notif)

	var config map[string]string
	_ = json.Unmarshal([]byte(channel.Config), &config)

	signature := ""
	if secret, ok := config["secret"]; ok && secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		signature = hex.EncodeToString(mac.Sum(nil))
	}

	req, err := http.NewRequest(http.MethodPost, config["url"], strings.NewReader(string(payload)))
	if err != nil {
		n.logger.Error("health webhook: create request failed", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if signature != "" {
		req.Header.Set("X-MP-Signature", "sha256="+signature)
	}
	req.Header.Set("X-MP-Event", "health")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		n.logger.Error("health webhook: dispatch failed", "channel", channel.Name, "error", err)
		return
	}
	resp.Body.Close()
	n.logger.Info("health webhook dispatched", "channel", channel.Name, "status", resp.StatusCode)
}

func (n *NotificationEngine) sendEmail(channel HealthNotificationChannel, notif HealthNotification) {
	var config map[string]string
	_ = json.Unmarshal([]byte(channel.Config), &config)

	n.logger.Info("health email notification sent",
		"to", config["to"],
		"monitor_id", notif.MonitorID,
		"parameter", notif.ParameterKey,
		"state", notif.NewState,
	)
}

func (n *NotificationEngine) sendTelegram(channel HealthNotificationChannel, notif HealthNotification) {
	var config map[string]string
	_ = json.Unmarshal([]byte(channel.Config), &config)

	botToken := config["bot_token"]
	chatID := config["chat_id"]
	if botToken == "" || chatID == "" {
		n.logger.Warn("health telegram: missing bot_token or chat_id", "channel", channel.Name)
		return
	}

	text := fmt.Sprintf("🩺 %s\nMonitor: %s\nParameter: %s\nState: %s → %s\nValue: %.2f",
		notif.ParameterKey, notif.MonitorID, notif.ParameterKey,
		notif.OldState, notif.NewState, notif.CurrentValue,
	)
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	form := url.Values{
		"chat_id": {chatID},
		"text":    {text},
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(apiURL, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		n.logger.Error("health telegram: dispatch failed", "error", err)
		return
	}
	resp.Body.Close()
	n.logger.Info("health telegram dispatched", "channel", channel.Name, "status", resp.StatusCode)
}

func (n *NotificationEngine) sendSlack(channel HealthNotificationChannel, notif HealthNotification) {
	var config map[string]string
	_ = json.Unmarshal([]byte(channel.Config), &config)

	webhookURL := config["webhook_url"]
	if webhookURL == "" {
		n.logger.Warn("health slack: missing webhook_url", "channel", channel.Name)
		return
	}

	text := fmt.Sprintf("*Health Alert*: %s\nMonitor: `%s`\nState: %s → %s\nValue: %.2f",
		notif.ParameterKey, notif.MonitorID,
		notif.OldState, notif.NewState, notif.CurrentValue,
	)

	payload, _ := json.Marshal(map[string]string{"text": text})
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		n.logger.Error("health slack: dispatch failed", "error", err)
		return
	}
	resp.Body.Close()
	n.logger.Info("health slack dispatched", "channel", channel.Name, "status", resp.StatusCode)
}

func (n *NotificationEngine) sendDiscord(channel HealthNotificationChannel, notif HealthNotification) {
	var config map[string]string
	_ = json.Unmarshal([]byte(channel.Config), &config)

	webhookURL := config["webhook_url"]
	if webhookURL == "" {
		n.logger.Warn("health discord: missing webhook_url", "channel", channel.Name)
		return
	}

	content := fmt.Sprintf("**Health Alert: %s**\nMonitor: `%s`\nState: %s → %s\nValue: %.2f",
		notif.ParameterKey, notif.MonitorID,
		notif.OldState, notif.NewState, notif.CurrentValue,
	)

	payload, _ := json.Marshal(map[string]string{"content": content})
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		n.logger.Error("health discord: dispatch failed", "error", err)
		return
	}
	resp.Body.Close()
	n.logger.Info("health discord dispatched", "channel", channel.Name, "status", resp.StatusCode)
}

func (n *NotificationEngine) sendTeams(channel HealthNotificationChannel, notif HealthNotification) {
	var config map[string]string
	_ = json.Unmarshal([]byte(channel.Config), &config)

	webhookURL := config["webhook_url"]
	if webhookURL == "" {
		n.logger.Warn("health teams: missing webhook_url", "channel", channel.Name)
		return
	}

	text := fmt.Sprintf("Health Alert: %s - Monitor: %s - State: %s → %s - Value: %.2f",
		notif.ParameterKey, notif.MonitorID,
		notif.OldState, notif.NewState, notif.CurrentValue,
	)

	payload, _ := json.Marshal(map[string]string{"text": text})
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		n.logger.Error("health teams: dispatch failed", "error", err)
		return
	}
	resp.Body.Close()
	n.logger.Info("health teams dispatched", "channel", channel.Name, "status", resp.StatusCode)
}
