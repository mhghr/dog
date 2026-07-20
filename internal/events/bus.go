// Package events provides an in-process pub/sub bus used by the SSE gateway.
// The Bus interface is intentionally small so it can later be replaced by a
// Redis-backed implementation for multi-instance deployments.
package events

import (
	"sync"
)

type Event struct {
	Name string
	ID   string
	Data []byte
}

type Bus struct {
	mu          sync.RWMutex
	subscribers map[chan Event]struct{}
}

func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[chan Event]struct{}),
	}
}

// Subscribe registers a buffered subscriber channel. The returned cancel
// function must be called to release resources.
func (b *Bus) Subscribe(buffer int) (<-chan Event, func()) {
	if buffer <= 0 {
		buffer = 16
	}

	channel := make(chan Event, buffer)

	b.mu.Lock()
	b.subscribers[channel] = struct{}{}
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		if _, ok := b.subscribers[channel]; ok {
			delete(b.subscribers, channel)
			close(channel)
		}
		b.mu.Unlock()
	}

	return channel, cancel
}

// Publish delivers the event to all subscribers without blocking; slow
// subscribers drop events instead of stalling ingestion.
func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for subscriber := range b.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func (b *Bus) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}
