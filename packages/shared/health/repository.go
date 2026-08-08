package health

import "context"

type Repository interface {
	ListParameterCatalog(ctx context.Context, monitorType string) ([]ParameterDefinition, error)
	GetParameterDefinition(ctx context.Context, monitorType, key string) (ParameterDefinition, error)

	ListParameterRules(ctx context.Context, monitorID string) ([]ParameterRule, error)
	GetParameterRule(ctx context.Context, monitorID, parameterKey string) (ParameterRule, error)
	UpsertParameterRule(ctx context.Context, rule *ParameterRule) error
	DeleteParameterRule(ctx context.Context, monitorID, parameterKey string) error

	UpsertHealthState(ctx context.Context, state *ParameterHealthState) error
	GetHealthState(ctx context.Context, monitorID, parameterKey string) (ParameterHealthState, error)
	ListHealthStates(ctx context.Context, monitorID string) ([]ParameterHealthState, error)

	ListNotificationChannels(ctx context.Context) ([]HealthNotificationChannel, error)
	GetNotificationChannel(ctx context.Context, id string) (HealthNotificationChannel, error)
	CreateNotificationChannel(ctx context.Context, ch *HealthNotificationChannel) error
	UpdateNotificationChannel(ctx context.Context, ch *HealthNotificationChannel) error
	DeleteNotificationChannel(ctx context.Context, id string) error

	ListNotificationPolicies(ctx context.Context, monitorID string) ([]NotificationPolicy, error)
	GetNotificationPolicy(ctx context.Context, id string) (NotificationPolicy, error)
	CreateNotificationPolicy(ctx context.Context, policy *NotificationPolicy) error
	UpdateNotificationPolicy(ctx context.Context, policy *NotificationPolicy) error
	DeleteNotificationPolicy(ctx context.Context, id string) error
}
