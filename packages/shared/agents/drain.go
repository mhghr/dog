package agents

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const drainKeyPrefix = "drain:"

type DrainManager struct {
	client *redis.Client
	repo   *Repository
}

func NewDrainManager(client *redis.Client, repo *Repository) *DrainManager {
	return &DrainManager{client: client, repo: repo}
}

func (m *DrainManager) DrainAgent(ctx context.Context, agentID uuid.UUID, deadline time.Time) error {
	agent, err := m.repo.GetAgent(ctx, agentID)
	if err != nil {
		return fmt.Errorf("get agent: %w", err)
	}

	if agent.Status == AgentDraining {
		return fmt.Errorf("%w: agent is already draining", ErrInvalidTransition)
	}

	if !CanTransition(agent.Status, AgentDraining) {
		return fmt.Errorf("%w: cannot transition from %s to draining", ErrInvalidTransition, agent.Status)
	}

	key := drainKeyPrefix + agentID.String()
	ttl := time.Until(deadline)
	if ttl <= 0 {
		ttl = time.Second
	}

	if err := m.client.Set(ctx, key, deadline.Format(time.RFC3339), ttl).Err(); err != nil {
		return fmt.Errorf("set drain key: %w", err)
	}

	if err := m.repo.UpdateAgentStatus(ctx, agentID, AgentDraining, StatusUpdateOpts{}); err != nil {
		m.client.Del(ctx, key)
		return fmt.Errorf("update agent status: %w", err)
	}

	return nil
}

func (m *DrainManager) CancelDrain(ctx context.Context, agentID uuid.UUID) error {
	agent, err := m.repo.GetAgent(ctx, agentID)
	if err != nil {
		return fmt.Errorf("get agent: %w", err)
	}

	if agent.Status != AgentDraining {
		return fmt.Errorf("%w: agent is not draining", ErrInvalidTransition)
	}

	key := drainKeyPrefix + agentID.String()
	if err := m.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("remove drain key: %w", err)
	}

	if err := m.repo.UpdateAgentStatus(ctx, agentID, AgentActive, StatusUpdateOpts{}); err != nil {
		return fmt.Errorf("update agent status: %w", err)
	}

	return nil
}

func (m *DrainManager) IsDraining(ctx context.Context, agentID uuid.UUID) (bool, error) {
	key := drainKeyPrefix + agentID.String()
	exists, err := m.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("check drain key: %w", err)
	}
	return exists > 0, nil
}

func (m *DrainManager) ExpireDrains(ctx context.Context) (int64, error) {
	var total int64
	iter := m.client.Scan(ctx, 0, drainKeyPrefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		ttl, err := m.client.TTL(ctx, iter.Val()).Result()
		if err != nil {
			continue
		}
		if ttl <= 0 {
			agentIDStr := iter.Val()[len(drainKeyPrefix):]
			agentID, parseErr := uuid.Parse(agentIDStr)
			if parseErr != nil {
				continue
			}

			if statusErr := m.repo.UpdateAgentStatus(ctx, agentID, AgentOffline, StatusUpdateOpts{}); statusErr != nil {
				continue
			}

			m.client.Del(ctx, iter.Val())
			total++
		}
	}
	return total, iter.Err()
}
