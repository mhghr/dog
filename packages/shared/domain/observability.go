package domain

import (
	"encoding/json"
	"time"
)

// ── Workspaces (environments within org, replaces projects) ──

type WorkspacePlan string

const (
	PlanFree       WorkspacePlan = "free"
	PlanPro        WorkspacePlan = "pro"
	PlanEnterprise WorkspacePlan = "enterprise"
)

type WorkspaceRole string

const (
	RoleOwner  WorkspaceRole = "owner"
	RoleAdmin  WorkspaceRole = "admin"
	RoleEditor WorkspaceRole = "editor"
	RoleViewer WorkspaceRole = "viewer"
)

type Workspace struct {
	ID             string                  `json:"id"`
	OrganizationID string                  `json:"organization_id"`
	Name           string                  `json:"name"`
	Slug           string                  `json:"slug"`
	Description    string                  `json:"description"`
	Plan           WorkspacePlan           `json:"plan"`
	Settings       map[string]any          `json:"settings"`
	CreatedAt      time.Time               `json:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at"`
}

type WorkspaceInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Plan        string `json:"plan,omitempty"`
}

type WorkspaceMember struct {
	WorkspaceID string        `json:"workspace_id"`
	UserID      string        `json:"user_id"`
	Role        WorkspaceRole `json:"role"`
	JoinedAt    time.Time     `json:"joined_at"`
}

type WorkspaceMemberInput struct {
	UserID string        `json:"user_id"`
	Role   WorkspaceRole `json:"role"`
}

// ── Resource Types (plugin registry) ──────────────────────────

type ResourceType struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	Category            string          `json:"category"`
	Slug                string          `json:"slug"`
	Icon                string          `json:"icon"`
	Capabilities        []string        `json:"capabilities"`
	ConfigurationSchema json.RawMessage `json:"configuration_schema"`
	CreatedAt           time.Time       `json:"created_at"`
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
	HealthStatus   string         `json:"health_status,omitempty"`
	HealthScore    float64        `json:"health_score,omitempty"`
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

// ── Monitors (attached to resources) ──────────────────────────

type Monitor struct {
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

	// Type is the probe executor key (e.g. "http", "ping") resolved from
	// the monitor_types registry. Used by the ingestion pipeline and worker.
	Type           MonitorType `json:"type,omitempty"`
	ResourceTarget string      `json:"resource_target,omitempty"`
	ProbeType      MonitorType `json:"-"`
}

type MonitorInput struct {
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

// MonitorTypeCode returns the probe executor key for a monitor type name,
// or "" when the type name has no matching probe executor. Resource types
// seed names like "HTTP Check" while probe executors are keyed on enum
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

type ProbeAssignment struct {
	ID        string    `json:"id"`
	MonitorID string    `json:"monitor_id"`
	ProbeID   string    `json:"probe_id"`
	Priority  int       `json:"priority"`
	CreatedAt time.Time `json:"created_at"`
}

// ── Monitor Jobs ───────────────────────────────────────────────

type JobStatus string

const (
	JobPending  JobStatus = "pending"
	JobRunning  JobStatus = "running"
	JobSuccess  JobStatus = "success"
	JobFailed   JobStatus = "failed"
	JobTimeout  JobStatus = "timeout"
)

type MonitorJob struct {
	ID           string     `json:"id"`
	MonitorID    string     `json:"monitor_id"`
	ProbeID      *string    `json:"probe_id,omitempty"`
	ScheduledAt  time.Time  `json:"scheduled_at"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	Status       JobStatus  `json:"status"`
	Attempt      int        `json:"attempt"`
	ErrorMessage string     `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ── Resource Health ────────────────────────────────────────────

type ResourceHealth string

const (
	HealthHealthy   ResourceHealth = "healthy"
	HealthDegraded  ResourceHealth = "degraded"
	HealthWarning   ResourceHealth = "warning"
	HealthCritical  ResourceHealth = "critical"
	HealthDown      ResourceHealth = "down"
	HealthUnknown   ResourceHealth = "unknown"
)

type ResourceHealthState struct {
	ResourceID       string         `json:"resource_id"`
	State            ResourceHealth `json:"state"`
	Score            float64        `json:"score"`
	ActiveAlerts     int            `json:"active_alerts"`
	ActiveWarnings   int            `json:"active_warnings"`
	LastEvaluatedAt  *time.Time     `json:"last_evaluated_at,omitempty"`
	StateChangedAt   *time.Time     `json:"state_changed_at,omitempty"`
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

// ── Monitor Type Parameters ────────────────────────────────────

type MonitorTypeParameter struct {
	ID             string `json:"id"`
	Key            string `json:"key"`
	MonitorType    string `json:"monitor_type"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	DataType       string `json:"data_type"`
	Unit           string `json:"unit"`
	Direction      string `json:"direction"`
	DefaultProfile string `json:"default_profile"`
}
