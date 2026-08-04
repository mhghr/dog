package domain

import (
	"encoding/json"
	"time"
)

// ── Workspaces (environments within org, replaces projects) ──

type Workspace struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type WorkspaceInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ── Resource Types (plugin registry) ──────────────────────────

type ResourceType struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Category     string    `json:"category"`
	Icon         string    `json:"icon"`
	Capabilities []string  `json:"capabilities"`
	CreatedAt    time.Time `json:"created_at"`
}

// ── Resources (primary entity, replaces "monitor") ────────────

type Resource struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organization_id"`
	WorkspaceID    *string        `json:"workspace_id,omitempty"`
	ResourceTypeID string         `json:"resource_type_id"`
	CreatedBy      *string        `json:"created_by,omitempty"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Target         string         `json:"target"`
	Status         string         `json:"status"`
	Metadata       map[string]any `json:"metadata"`
	TypeName       string         `json:"type_name,omitempty"`
	TypeCategory   string         `json:"type_category,omitempty"`
	TypeIcon       string         `json:"type_icon,omitempty"`
	MonitorsCount  int            `json:"monitors_count,omitempty"`
	Tags           []Tag          `json:"tags,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type ResourceInput struct {
	WorkspaceID    *string        `json:"workspace_id,omitempty"`
	ResourceTypeID string         `json:"resource_type_id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Target         string         `json:"target"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	Tags           []TagInput     `json:"tags,omitempty"`
}

type ResourceListFilter struct {
	OrganizationID string
	WorkspaceID    string
	ResourceTypeID string
	Search         string
	Tags           map[string]string
	Status         string
	Page           int
	PageSize       int
}

// ── Tags ──────────────────────────────────────────────────────

type Tag struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Key            string `json:"key"`
	Value          string `json:"value"`
}

type TagInput struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ── Monitor Types (plugin-based) ─────────────────────────────

type MonitorTypeDef struct {
	ID                     string          `json:"id"`
	Name                   string          `json:"name"`
	Slug                   string          `json:"slug"`
	Category               string          `json:"category"`
	ExecutionType          string          `json:"execution_type"`
	ExecutorKey            string          `json:"executor_key"`
	Description            string          `json:"description"`
	Icon                   string          `json:"icon"`
	Enabled                bool            `json:"enabled"`
	MetricKeys             []string        `json:"metric_keys"`
	ConfigSchema           json.RawMessage `json:"config_schema"`
	DefaultConfiguration   json.RawMessage `json:"default_configuration"`
	MetricSchema           json.RawMessage `json:"metric_schema"`
	HealthParameters       json.RawMessage `json:"health_parameters"`
	SupportedResourceTypes json.RawMessage `json:"supported_resource_types"`
	CreatedAt              time.Time       `json:"created_at"`
	UpdatedAt              time.Time       `json:"updated_at"`
}

// ── Monitors v2 (attached to resources) ───────────────────────

type MonitorV2 struct {
	ID              string         `json:"id"`
	ResourceID      string         `json:"resource_id"`
	MonitorTypeID   string         `json:"monitor_type_id"`
	HealthProfileID *string        `json:"health_profile_id,omitempty"`
	CreatedBy       *string        `json:"created_by,omitempty"`
	Name            string         `json:"name"`
	Enabled         bool           `json:"enabled"`
	Configuration   map[string]any `json:"configuration"`
	Severity        string         `json:"severity"`
	IntervalSeconds int            `json:"interval_seconds"`
	TimeoutMillis   int            `json:"timeout_millis"`
	Retries         int            `json:"retries"`
	LastStatus      MonitorStatus  `json:"last_status"`
	LastCheckedAt   *time.Time     `json:"last_checked_at"`
	NextRunAt       time.Time      `json:"next_run_at"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`

	ResourceTarget string      `json:"resource_target,omitempty"`
	ProbeType      MonitorType `json:"-"`
}

type MonitorV2Input struct {
	MonitorTypeID   string         `json:"monitor_type_id"`
	HealthProfileID *string        `json:"health_profile_id,omitempty"`
	Name            string         `json:"name"`
	Enabled         *bool          `json:"enabled"`
	Configuration   map[string]any `json:"configuration"`
	Severity        string         `json:"severity"`
	IntervalSeconds int            `json:"interval_seconds"`
	TimeoutMillis   int            `json:"timeout_millis"`
	Retries         int            `json:"retries"`
}

// MonitorTypeCode returns the v1 probe executor key for this monitor type,
// or "" when the type name has no matching probe executor. Resource types
// seed names like "HTTP Check" while probe executors are keyed on v1 enum
// strings ("http", "ping", ...).
func MonitorTypeCode(typeName string) MonitorType {
	switch typeName {
	case "HTTP Check":
		return MonitorHTTP
	case "Ping":
		return MonitorPing
	case "TCP Port":
		return MonitorTCP
	case "DNS Resolution":
		return MonitorDNS
	case "SSL Certificate":
		return MonitorTLS
	case "Domain Expiry":
		return MonitorDomainExpiration
	case "SMTP Service":
		return MonitorSMTP
	case "NTP Time":
		return MonitorNTP
	default:
		return ""
	}
}

// ── Monitor Assignments ───────────────────────────────────────

type MonitorAssignment struct {
	ID              string  `json:"id"`
	MonitorID       string  `json:"monitor_id"`
	ExecutionType   string  `json:"execution_type"`
	AgentID         *string `json:"agent_id,omitempty"`
	ProbeLocationID *string `json:"probe_location_id,omitempty"`
}

// ── Events ────────────────────────────────────────────────────

type Event struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organization_id"`
	ResourceID     *string        `json:"resource_id,omitempty"`
	MonitorID      *string        `json:"monitor_id,omitempty"`
	Type           string         `json:"type"`
	Severity       string         `json:"severity"`
	Title          string         `json:"title"`
	Message        string         `json:"message"`
	Metadata       map[string]any `json:"metadata"`
	EventTimestamp time.Time      `json:"event_timestamp"`
	CreatedAt      time.Time      `json:"created_at"`
}
