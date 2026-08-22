package api

import (
	"testing"
	"time"
)

func TestResolveStepDownsampling(t *testing.T) {
	cases := []struct {
		name       string
		window     time.Duration
		wantMinSec int
		wantMaxSec int
	}{
		// The spec downsampling table (section 18): the returned step must
		// respect the range and stay bounded to ~1500 points/series.
		{name: "15m high resolution", window: 15 * time.Minute, wantMinSec: 5, wantMaxSec: 15},
		{name: "1h medium resolution", window: time.Hour, wantMinSec: 15, wantMaxSec: 60},
		{name: "6h", window: 6 * time.Hour, wantMinSec: 60, wantMaxSec: 300},
		{name: "24h lower resolution", window: 24 * time.Hour, wantMinSec: 300, wantMaxSec: 1800},
		{name: "7d", window: 7 * 24 * time.Hour, wantMinSec: 1800, wantMaxSec: 7200},
		{name: "30d aggregated", window: 30 * 24 * time.Hour, wantMinSec: 7200, wantMaxSec: 86400},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			step := resolveStep("auto", tc.window)
			if step < tc.wantMinSec || step > tc.wantMaxSec {
				t.Fatalf("resolveStep(auto, %s) = %ds, want in [%d, %d]",
					tc.window, step, tc.wantMinSec, tc.wantMaxSec)
			}
			if int(tc.window.Seconds())/step > maxChartPoints {
				t.Fatalf("step %ds yields >%d points for %s window", step, maxChartPoints, tc.window)
			}
		})
	}
}

func TestResolveStepExplicitDurationHonorsPointBudget(t *testing.T) {
	// An explicit step is honored only when it does not exceed the max-point
	// budget (section 18: <= 1500 points/series). A 30s step over 24h would
	// yield 2880 points, so the resolver raises it to keep the budget.
	window := 24 * time.Hour
	step := resolveStep("30s", window)
	if points := int(window.Seconds()) / step; points > maxChartPoints {
		t.Fatalf("resolveStep(30s, 24h) = %ds -> %d points, exceeds budget %d",
			step, points, maxChartPoints)
	}
	if step < 30 {
		t.Fatalf("resolveStep(30s, 24h) = %d, want >= 30", step)
	}
}

func TestResolveStepNeverBelowOneSecond(t *testing.T) {
	// A 5-minute window with a huge explicit step must still yield >= 1s.
	step := resolveStep("3ms", 5*time.Minute)
	if step < 1 {
		t.Fatalf("resolveStep(3ms, 5m) = %d, want >= 1", step)
	}
}
