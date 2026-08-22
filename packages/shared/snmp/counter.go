package snmp

import (
	"math"
	"sync"
	"time"
)

// CounterState tracks previous counter readings per (key) so rates can be
// computed from deltas. It is keyed by a string that must include the monitor
// id, interface index and metric suffix, e.g. "monitorID:3:interface.in_bps".
// Wrap-around is handled per the counter width (32/64 bit).
type CounterState struct {
	mu   sync.Mutex
	prev map[string]counterReading
}

type counterReading struct {
	value float64
	at    time.Time
}

// NewCounterState returns an empty counter state store.
func NewCounterState() *CounterState {
	return &CounterState{prev: make(map[string]counterReading)}
}

// Rate computes the per-second rate of a counter, handling wrap-around. The
// returned bool is false when no previous reading exists (first observation),
// the elapsed time is non-positive, or the corrected delta is implausible
// (a device reboot/reset, indistinguishable from a wrap that could not have
// happened within one poll interval).
func (c *CounterState) Rate(key string, current float64, width Bits, now time.Time) (float64, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	prev, ok := c.prev[key]
	c.prev[key] = counterReading{value: current, at: now}

	if !ok {
		return 0, false
	}

	elapsed := now.Sub(prev.at).Seconds()
	if elapsed <= 0 {
		return 0, false
	}

	delta := current - prev.value
	if delta < 0 {
		// Counter wrapped: add the modulus of the counter width.
		delta += width.Modulus()
	}

	// A delta larger than half the counter range between two polls is treated
	// as a reset (reboot) rather than a rate — it would otherwise produce a
	// nonsensical alerting spike.
	if delta < 0 || delta > width.Modulus()/2 {
		return 0, false
	}

	return delta / elapsed, true
}

// Bits is the bit width of an SNMP counter.
type Bits int

const (
	Bits32 Bits = 32
	Bits64 Bits = 64
)

// Modulus returns the wrap modulus for the counter width.
func (b Bits) Modulus() float64 {
	if b == Bits64 {
		return math.MaxUint64
	}
	return math.MaxUint32 + 1
}
