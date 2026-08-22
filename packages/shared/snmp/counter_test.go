package snmp

import (
	"testing"
	"time"
)

func TestCounterState_RateBaseline(t *testing.T) {
	c := NewCounterState()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// First observation: no previous reading → no rate.
	if rate, ok := c.Rate("m:1:in", 1000, Bits32, now); ok || rate != 0 {
		t.Fatalf("first observation should not produce a rate, got %v %v", rate, ok)
	}

	// 2 seconds later, +2000 bytes → 1000 B/s.
	now = now.Add(2 * time.Second)
	rate, ok := c.Rate("m:1:in", 3000, Bits32, now)
	if !ok {
		t.Fatal("expected a rate on the second observation")
	}
	if rate != 1000 {
		t.Fatalf("expected 1000 B/s, got %v", rate)
	}
}

func TestCounterState_RateWrap32(t *testing.T) {
	c := NewCounterState()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// prev = 2^32 - 1000, then the counter wraps past 2^32 to 5000.
	// Total counted = 1000 + 5000 = 6000 over 10s → 600 B/s.
	c.Rate("m:1:in", 4294966296, Bits32, now) // 2^32 - 1000
	now = now.Add(10 * time.Second)

	rate, ok := c.Rate("m:1:in", 5000, Bits32, now)
	if !ok {
		t.Fatal("expected a rate after wrap")
	}
	if rate != 600 {
		t.Fatalf("expected 600 B/s after 32-bit wrap, got %v", rate)
	}
}

func TestCounterState_RateLarge64NoWrap(t *testing.T) {
	c := NewCounterState()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Large 64-bit values without a wrap produce an accurate rate.
	c.Rate("m:2:out", 1_000_000_000_000, Bits64, now)
	now = now.Add(5 * time.Second)

	rate, ok := c.Rate("m:2:out", 1_000_000_100_000, Bits64, now)
	if !ok {
		t.Fatal("expected a rate")
	}
	if rate != 20_000 {
		t.Fatalf("expected 20000 B/s, got %v", rate)
	}
}

func TestCounterState_NoNegativeRateOnReset(t *testing.T) {
	c := NewCounterState()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	c.Rate("m:3:in", 1_000_000, Bits32, now)
	now = now.Add(time.Minute)
	// Device rebooted: counter is now tiny. The reset guard must suppress the
	// nonsensical near-4-billion B/s rate.
	if _, ok := c.Rate("m:3:in", 10, Bits32, now); ok {
		t.Fatal("reset should not produce a rate")
	}
}

func TestBitsModulus(t *testing.T) {
	if Bits32.Modulus() != 4294967296 {
		t.Fatalf("32-bit modulus wrong: %v", Bits32.Modulus())
	}
	if Bits64.Modulus() != 18446744073709551615 {
		t.Fatalf("64-bit modulus wrong: %v", Bits64.Modulus())
	}
}
