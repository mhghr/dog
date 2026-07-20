package domain

import (
	"regexp"
	"strings"
	"time"
)

type MonitorType string

const (
	MonitorHTTP             MonitorType = "http"
	MonitorTCP              MonitorType = "tcp"
	MonitorDNS              MonitorType = "dns"
	MonitorPing             MonitorType = "ping"
	MonitorTLS              MonitorType = "tls"
	MonitorDomainExpiration MonitorType = "domain_expiration"
	MonitorSMTP             MonitorType = "smtp"
	MonitorNTP              MonitorType = "ntp"
)

var AllMonitorTypes = []MonitorType{
	MonitorHTTP,
	MonitorTCP,
	MonitorDNS,
	MonitorPing,
	MonitorTLS,
	MonitorDomainExpiration,
	MonitorSMTP,
	MonitorNTP,
}

func ParseMonitorType(value string) (MonitorType, bool) {
	candidate := MonitorType(value)
	for _, monitorType := range AllMonitorTypes {
		if monitorType == candidate {
			return candidate, true
		}
	}

	return "", false
}

type MonitorStatus string

const (
	StatusUp      MonitorStatus = "up"
	StatusDown    MonitorStatus = "down"
	StatusUnknown MonitorStatus = "unknown"
	StatusPaused  MonitorStatus = "paused"
)

func ParseMonitorStatus(value string) (MonitorStatus, bool) {
	switch MonitorStatus(value) {
	case StatusUp, StatusDown, StatusUnknown, StatusPaused:
		return MonitorStatus(value), true
	default:
		return "", false
	}
}

// MinIntervalSeconds enforces the per-type scheduling floor from the spec.
var MinIntervalSeconds = map[MonitorType]int{
	MonitorHTTP:             10,
	MonitorTCP:              10,
	MonitorPing:             10,
	MonitorDNS:              30,
	MonitorTLS:              300,
	MonitorDomainExpiration: 3600,
	MonitorSMTP:             30,
	MonitorNTP:              60,
}

// DefaultIntervalSeconds is used when the client omits interval_seconds.
var DefaultIntervalSeconds = map[MonitorType]int{
	MonitorHTTP:             60,
	MonitorTCP:              60,
	MonitorPing:             60,
	MonitorDNS:              60,
	MonitorTLS:              3600,
	MonitorDomainExpiration: 43200,
	MonitorSMTP:             60,
	MonitorNTP:              300,
}

type Monitor struct {
	ID              string
	Name            string
	Type            MonitorType
	Target          string
	IntervalSeconds int
	TimeoutMillis   int
	Retries         int
	Enabled         bool
	Config          map[string]any
	LastStatus      MonitorStatus
	LastCheckedAt   *time.Time
	NextRunAt       time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type LastResultSummary struct {
	Success        bool    `json:"success"`
	DurationMillis int64   `json:"duration_millis"`
	ErrorCode      *string `json:"error_code"`
}

type MonitorWithLastResult struct {
	Monitor
	LastResult *LastResultSummary
}

type MonitorListFilter struct {
	Page     int
	PageSize int
	Type     *MonitorType
	Status   *MonitorStatus
	Search   string
	Sort     string
	Order    string
}

type ProbeLocation struct {
	ID        string
	Name      string
	Code      string
	Enabled   bool
	CreatedAt time.Time
}

type LocationInput struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

var locationCodePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func ValidateLocationInput(input LocationInput) (ProbeLocation, map[string][]string) {
	fieldErrors := map[string][]string{}

	name := strings.TrimSpace(input.Name)
	if len([]rune(name)) < 2 || len([]rune(name)) > 100 {
		fieldErrors["name"] = append(fieldErrors["name"], "name must be between 2 and 100 characters")
	}

	code := strings.ToLower(strings.TrimSpace(input.Code))
	if len(code) < 2 || len(code) > 50 || !locationCodePattern.MatchString(code) {
		fieldErrors["code"] = append(fieldErrors["code"], "code must be 2-50 lowercase letters, digits and dashes")
	}

	if len(fieldErrors) > 0 {
		return ProbeLocation{}, fieldErrors
	}

	return ProbeLocation{Name: name, Code: code, Enabled: true}, nil
}
