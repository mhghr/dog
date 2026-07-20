// Package heartbeat tracks liveness of distributed components via Redis keys
// with TTL, so the control plane can report scheduler/worker health without a
// direct connection to them.
package heartbeat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const keyPrefix = "monitoring:heartbeat:"

type Info struct {
	Component string    `json:"component"`
	Name      string    `json:"name"`
	LastSeen  time.Time `json:"last_seen"`
}

func key(component, name string) string {
	return keyPrefix + component + ":" + name
}

// Beat refreshes the liveness key for a component instance.
func Beat(ctx context.Context, client *redis.Client, component, name string, ttl time.Duration) error {
	return client.Set(ctx, key(component, name), time.Now().UTC().Format(time.RFC3339Nano), ttl).Err()
}

// Run keeps beating until the context is cancelled.
func Run(ctx context.Context, client *redis.Client, component, name string, interval, ttl time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	_ = Beat(ctx, client, component, name, ttl)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = Beat(ctx, client, component, name, ttl)
		}
	}
}

// Alive checks whether a specific component instance is alive.
func Alive(ctx context.Context, client *redis.Client, component, name string) (bool, time.Time) {
	value, err := client.Get(ctx, key(component, name)).Result()
	if err != nil {
		return false, time.Time{}
	}

	lastSeen, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return true, time.Time{}
	}

	return true, lastSeen
}

// List returns all live instances of a component (e.g. all workers).
func List(ctx context.Context, client *redis.Client, component string) ([]Info, error) {
	pattern := keyPrefix + component + ":*"

	var (
		cursor uint64
		infos  []Info
	)

	for {
		keys, nextCursor, err := client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("scan heartbeats: %w", err)
		}

		for _, fullKey := range keys {
			name := strings.TrimPrefix(fullKey, keyPrefix+component+":")

			info := Info{Component: component, Name: name}
			if value, err := client.Get(ctx, fullKey).Result(); err == nil {
				if lastSeen, err := time.Parse(time.RFC3339Nano, value); err == nil {
					info.LastSeen = lastSeen
				}
			}

			infos = append(infos, info)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return infos, nil
}
