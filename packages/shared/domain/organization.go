package domain

import (
	"context"
	"strings"
	"time"
)

type Organization struct {
	ID        string
	Name      string
	Slug      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrganizationInput struct {
	Name string `json:"name"`
}

func ValidateOrganizationInput(input OrganizationInput) (Organization, map[string][]string) {
	fieldErrors := map[string][]string{}

	name := strings.TrimSpace(input.Name)
	if nameLen := len([]rune(name)); nameLen < 2 || nameLen > 200 {
		fieldErrors["name"] = append(fieldErrors["name"], "name must be between 2 and 200 characters")
	}

	if len(fieldErrors) > 0 {
		return Organization{}, fieldErrors
	}

	slug := generateSlug(name)
	return Organization{Name: name, Slug: slug}, nil
}

var slugBadChars = strings.NewReplacer(
	" ", "-", "?", "", "!", "", "@", "", "#", "", "$", "", "%", "", "^", "",
	"&", "", "*", "", "(", "", ")", "", "+", "", "=", "", ":", "", ";", "",
	",", "", ".", "", "/", "", "\\", "", "|", "", "~", "", "`", "",
	"[", "",  "]", "",  "{", "",  "}", "",  "<", "",  ">", "",  "'", "",  "\"", "",
)

func generateSlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	slug = slugBadChars.Replace(slug)
	parts := strings.Fields(slug)
	if len(parts) > 8 {
		parts = parts[:8]
	}
	return strings.Join(parts, "-")
}

type OrgContextKey string

const OrgIDContextKey OrgContextKey = "org.id"

func OrgIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(OrgIDContextKey).(string)
	return id, ok && id != ""
}

// ── Alerting domain types ──

type AlertPolicyScope struct {
	MonitorIDs []string          `json:"monitor_ids,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
}

type AlertConditions struct {
	ConsecutiveFailures int `json:"consecutive_failures,omitempty"`
	HighLatencyMs       int `json:"high_latency_ms,omitempty"`
	PacketLossPercent   int `json:"packet_loss_percent,omitempty"`
	SSLExpiringDays     int `json:"ssl_expiring_days,omitempty"`
	DomainExpiringDays  int `json:"domain_expiring_days,omitempty"`
	SMTPStartTLSFail    bool `json:"smtp_starttls_fail,omitempty"`
	NTPOffsetMs         int `json:"ntp_offset_ms,omitempty"`
	DNSMismatch         bool `json:"dns_mismatch,omitempty"`
}
