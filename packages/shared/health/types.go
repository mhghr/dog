package health

import "time"

type HealthState string

const (
	HealthOK      HealthState = "OK"
	HealthWarning HealthState = "WARNING"
	HealthError   HealthState = "ERROR"
	HealthUnknown HealthState = "UNKNOWN"
)

type HealthRuleMode string

const (
	ModeInheritDefault HealthRuleMode = "INHERIT_DEFAULT"
	ModeUseProfile     HealthRuleMode = "USE_PROFILE"
	ModeCustom         HealthRuleMode = "CUSTOM"
	ModeDisabled       HealthRuleMode = "DISABLED"
)

type HealthRuleProfile string

const (
	ProfileSensitive   HealthRuleProfile = "Sensitive"
	ProfileRecommended HealthRuleProfile = "Recommended"
	ProfileRelaxed     HealthRuleProfile = "Relaxed"
)

type ParameterDataType string

const (
	DataTypeNUMBER     ParameterDataType = "NUMBER"
	DataTypeBOOLEAN    ParameterDataType = "BOOLEAN"
	DataTypeENUM       ParameterDataType = "ENUM"
	DataTypeSTRING     ParameterDataType = "STRING"
	DataTypeDURATION   ParameterDataType = "DURATION"
	DataTypePERCENTAGE ParameterDataType = "PERCENTAGE"
	DataTypeBYTES      ParameterDataType = "BYTES"
	DataTypeTIMESTAMP  ParameterDataType = "TIMESTAMP"
)

type ParameterDirection string

const (
	DirHigherIsWorse  ParameterDirection = "HIGHER_IS_WORSE"
	DirLowerIsWorse   ParameterDirection = "LOWER_IS_WORSE"
	DirBooleanFailure ParameterDirection = "BOOLEAN_FAILURE"
	DirEnumState      ParameterDirection = "ENUM_STATE"
	DirRangeDeviation ParameterDirection = "RANGE_DEVIATION"
	DirChangeEvent    ParameterDirection = "CHANGE_EVENT"
	DirRate           ParameterDirection = "RATE"
	DirCount          ParameterDirection = "COUNT"
)

// ParameterDefinition is a catalog entry for a predefined health parameter.
type ParameterDefinition struct {
	Key            string              `json:"key"`
	MonitorType    string              `json:"monitor_type"`
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	DataType       ParameterDataType   `json:"data_type"`
	Unit           string              `json:"unit"`
	Direction      ParameterDirection  `json:"direction"`
	DefaultProfile HealthRuleProfile   `json:"default_profile"`
	DefaultWarning *float64            `json:"default_warning,omitempty"`
	DefaultError   *float64            `json:"default_error,omitempty"`
	Recovery       *float64            `json:"recovery,omitempty"`
}

// ParameterRule is a per-monitor override for a health parameter.
type ParameterRule struct {
	ID                  string            `json:"id"`
	MonitorID           string            `json:"monitor_id"`
	ParameterKey        string            `json:"parameter_key"`
	Mode                HealthRuleMode    `json:"mode"`
	Profile             *HealthRuleProfile `json:"profile,omitempty"`
	Aggregation         string            `json:"aggregation"`
	WindowType          string            `json:"window_type"`
	WindowValue         int               `json:"window_value"`
	WarningOperator     string            `json:"warning_operator"`
	WarningValue        *float64          `json:"warning_value,omitempty"`
	ErrorOperator       string            `json:"error_operator"`
	ErrorValue          *float64          `json:"error_value,omitempty"`
	RecoveryOperator    *string           `json:"recovery_operator,omitempty"`
	RecoveryValue       *float64          `json:"recovery_value,omitempty"`
	MinimumSamples      int               `json:"minimum_samples"`
	ConsecutiveFailures *int              `json:"consecutive_failures,omitempty"`
	ConsecutiveSuccesses *int             `json:"consecutive_successes,omitempty"`
	MissingDataPolicy   string            `json:"missing_data_policy"`
	MissedChecks        int               `json:"missed_checks"`
	CooldownSeconds     int               `json:"cooldown_seconds"`
	Enabled             bool              `json:"enabled"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
}

type HealthNotificationChannel struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Config    string    `json:"config"` // JSON string
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type NotificationPolicy struct {
	ID                    string    `json:"id"`
	MonitorID             *string   `json:"monitor_id,omitempty"`
	ParameterKey          *string   `json:"parameter_key,omitempty"`
	ChannelID             string    `json:"channel_id"`
	Triggers              []string  `json:"triggers"`
	DelaySeconds          int       `json:"delay_seconds"`
	RepeatIntervalSeconds int       `json:"repeat_interval_seconds"`
	CooldownSeconds       int       `json:"cooldown_seconds"`
	Enabled               bool      `json:"enabled"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type ParameterHealthState struct {
	MonitorID      string      `json:"monitor_id"`
	ParameterKey   string      `json:"parameter_key"`
	CurrentState   HealthState `json:"current_state"`
	CurrentValue   *float64    `json:"current_value,omitempty"`
	EvaluatedAt    *time.Time  `json:"evaluated_at,omitempty"`
	PreviousState  *HealthState `json:"previous_state,omitempty"`
	StateChangedAt *time.Time  `json:"state_changed_at,omitempty"`
}

// EvaluateResult represents a health state change that may trigger notifications.
type EvaluateOutcome struct {
	MonitorID    string
	ParameterKey string
	OldState     HealthState
	NewState     HealthState
	CurrentValue float64
}
