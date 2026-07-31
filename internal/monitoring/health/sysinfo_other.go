//go:build !linux

package health

// collectPlatform is a no-op on non-Linux platforms. Production health
// monitoring is Linux-only; this stub lets the package and its tests compile
// on developer machines (e.g. Windows).
func (m *Monitor) collectPlatform(h *SystemHealth) {}
