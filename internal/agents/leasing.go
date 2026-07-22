package agents

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const leaseKeyPrefix = "lease:"

type LeaseManager struct {
	client *redis.Client
}

func NewLeaseManager(client *redis.Client) *LeaseManager {
	return &LeaseManager{client: client}
}

func (m *LeaseManager) CreateLease(ctx context.Context, agentID, jobID string, ttl time.Duration) (string, time.Time, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", time.Time{}, fmt.Errorf("generate lease id: %w", err)
	}
	leaseID := hex.EncodeToString(buf)

	expiresAt := time.Now().UTC().Add(ttl)
	key := leaseKeyPrefix + leaseID

	err := m.client.Set(ctx, key, fmt.Sprintf("%s:%s", agentID, jobID), ttl).Err()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("create lease: %w", err)
	}

	return leaseID, expiresAt, nil
}

func (m *LeaseManager) RenewLease(ctx context.Context, leaseID string, ttl time.Duration) error {
	key := leaseKeyPrefix + leaseID
	exists, err := m.client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("check lease existence: %w", err)
	}
	if exists == 0 {
		return ErrLeaseNotFound
	}

	return m.client.Expire(ctx, key, ttl).Err()
}

func (m *LeaseManager) ReleaseLease(ctx context.Context, leaseID string) error {
	key := leaseKeyPrefix + leaseID
	return m.client.Del(ctx, key).Err()
}

func (m *LeaseManager) IsLeaseValid(ctx context.Context, leaseID string) (bool, error) {
	key := leaseKeyPrefix + leaseID
	exists, err := m.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("check lease: %w", err)
	}
	return exists > 0, nil
}

func (m *LeaseManager) ExpireLeases(ctx context.Context) error {
	iter := m.client.Scan(ctx, 0, leaseKeyPrefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		ttl, err := m.client.TTL(ctx, iter.Val()).Result()
		if err != nil {
			continue
		}
		if ttl <= 0 {
			m.client.Del(ctx, iter.Val())
		}
	}
	return iter.Err()
}
