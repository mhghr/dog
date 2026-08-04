package ingest

import (
	"context"
	"sync"
	"time"
)

type DedupStore interface {
	Exists(ctx context.Context, eventID string) (bool, error)
	MarkProcessed(ctx context.Context, eventID string) error
}

type cacheEntry struct {
	expiresAt time.Time
}

type Deduplicator struct {
	store   DedupStore
	mu      sync.RWMutex
	cache   map[string]cacheEntry
	maxSize int
	ttl     time.Duration
}

func NewDeduplicator(store DedupStore, maxSize int, cacheTTLSeconds int) *Deduplicator {
	if maxSize <= 0 {
		maxSize = 10000
	}
	if cacheTTLSeconds <= 0 {
		cacheTTLSeconds = 300
	}
	return &Deduplicator{
		store:   store,
		cache:   make(map[string]cacheEntry, maxSize),
		maxSize: maxSize,
		ttl:     time.Duration(cacheTTLSeconds) * time.Second,
	}
}

func (d *Deduplicator) IsDuplicate(ctx context.Context, eventID string) (bool, error) {
	d.mu.RLock()
	entry, inCache := d.cache[eventID]
	d.mu.RUnlock()

	if inCache && time.Now().Before(entry.expiresAt) {
		return true, nil
	}

	exists, err := d.store.Exists(ctx, eventID)
	if err != nil {
		return false, err
	}
	if exists {
		d.mu.Lock()
		if len(d.cache) < d.maxSize {
			d.cache[eventID] = cacheEntry{expiresAt: time.Now().Add(d.ttl)}
		}
		d.mu.Unlock()
		return true, nil
	}

	return false, nil
}

func (d *Deduplicator) MarkProcessed(ctx context.Context, eventID string, store DedupStore) error {
	if err := store.MarkProcessed(ctx, eventID); err != nil {
		return err
	}
	d.mu.Lock()
	if len(d.cache) >= d.maxSize {
		d.evictExpired()
	}
	if len(d.cache) < d.maxSize {
		d.cache[eventID] = cacheEntry{expiresAt: time.Now().Add(d.ttl)}
	}
	d.mu.Unlock()
	return nil
}

func (d *Deduplicator) evictExpired() {
	now := time.Now()
	for id, entry := range d.cache {
		if now.After(entry.expiresAt) {
			delete(d.cache, id)
		}
	}
}
