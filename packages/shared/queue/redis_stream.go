package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"monitoring-platform/packages/shared/domain"
)

type StreamConfig struct {
	Stream         string
	Group          string
	DeadLetter     string
	MaxLen         int64
	LocationPrefix string
}

type RedisQueue struct {
	client *redis.Client
	cfg    StreamConfig
	logger *slog.Logger
}

func NewRedisQueue(client *redis.Client, cfg StreamConfig, logger *slog.Logger) *RedisQueue {
	if cfg.MaxLen <= 0 {
		cfg.MaxLen = 100000
	}

	return &RedisQueue{
		client: client,
		cfg:    cfg,
		logger: logger,
	}
}

// EnsureGroup creates the consumer group if it does not exist yet.
func (q *RedisQueue) EnsureGroup(ctx context.Context) error {
	err := q.client.XGroupCreateMkStream(ctx, q.cfg.Stream, q.cfg.Group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("create consumer group: %w", err)
	}

	return nil
}

func (q *RedisQueue) Publish(ctx context.Context, job domain.ProbeJob) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal probe job: %w", err)
	}

	return q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.cfg.Stream,
		MaxLen: q.cfg.MaxLen,
		Approx: true,
		Values: map[string]any{
			"payload": string(payload),
		},
	}).Err()
}

func (q *RedisQueue) PublishToLocation(ctx context.Context, locationCode string, job []byte) error {
	stream := q.cfg.Stream
	if locationCode != "" {
		prefix := q.cfg.LocationPrefix
		if prefix == "" {
			prefix = "probe-jobs"
		}
		stream = fmt.Sprintf("%s:%s", prefix, locationCode)
	}

	return q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		MaxLen: q.cfg.MaxLen,
		Approx: true,
		Values: map[string]any{
			"payload": string(job),
		},
	}).Err()
}

func (q *RedisQueue) Consume(ctx context.Context, consumerName string, count int64, block time.Duration) ([]redis.XMessage, error) {
	streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    q.cfg.Group,
		Consumer: consumerName,
		Streams:  []string{q.cfg.Stream, ">"},
		Count:    count,
		Block:    block,
	}).Result()

	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	if len(streams) == 0 {
		return nil, nil
	}

	return streams[0].Messages, nil
}

func (q *RedisQueue) Ack(ctx context.Context, messageID string) error {
	return q.client.XAck(ctx, q.cfg.Stream, q.cfg.Group, messageID).Err()
}

// AutoClaim recovers messages abandoned by dead consumers.
func (q *RedisQueue) AutoClaim(ctx context.Context, consumerName string, minIdle time.Duration, count int64) ([]redis.XMessage, error) {
	messages, _, err := q.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   q.cfg.Stream,
		Group:    q.cfg.Group,
		Consumer: consumerName,
		MinIdle:  minIdle,
		Start:    "0-0",
		Count:    count,
	}).Result()

	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	return messages, nil
}

// DeliveryCount reports how many times a pending message has been delivered.
func (q *RedisQueue) DeliveryCount(ctx context.Context, messageID string) (int64, error) {
	pending, err := q.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: q.cfg.Stream,
		Group:  q.cfg.Group,
		Start:  messageID,
		End:    messageID,
		Count:  1,
	}).Result()

	if err != nil {
		return 0, err
	}

	if len(pending) == 0 {
		return 0, nil
	}

	return pending[0].RetryCount, nil
}

// DeadLetter moves a poison message to the dead letter stream and acks it.
func (q *RedisQueue) DeadLetter(ctx context.Context, message redis.XMessage, reason string) error {
	values := map[string]any{
		"original_id": message.ID,
		"reason":      reason,
		"failed_at":   time.Now().UTC().Format(time.RFC3339),
	}
	if payload, ok := message.Values["payload"]; ok {
		values["payload"] = payload
	}

	if err := q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.cfg.DeadLetter,
		MaxLen: q.cfg.MaxLen,
		Approx: true,
		Values: values,
	}).Err(); err != nil {
		return fmt.Errorf("publish dead letter: %w", err)
	}

	return q.Ack(ctx, message.ID)
}

type Stats struct {
	Lag     int64 `json:"lag"`
	Pending int64 `json:"pending"`
}

// Stats reports backlog (entries not yet delivered) and pending (delivered
// but unacked) counts for the worker consumer group.
func (q *RedisQueue) Stats(ctx context.Context) (Stats, error) {
	groups, err := q.client.XInfoGroups(ctx, q.cfg.Stream).Result()
	if err != nil {
		if strings.Contains(err.Error(), "no such key") {
			return Stats{}, nil
		}
		return Stats{}, err
	}

	for _, group := range groups {
		if group.Name == q.cfg.Group {
			return Stats{Lag: group.Lag, Pending: group.Pending}, nil
		}
	}

	return Stats{}, nil
}
