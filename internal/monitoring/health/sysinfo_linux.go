//go:build linux

package health

import (
	"math"
	"runtime"
	"time"

	"golang.org/x/sys/unix"
)

// collectPlatform fills the host-specific metrics from Linux syscalls.
func (m *Monitor) collectPlatform(h *SystemHealth) {
	var info unix.Sysinfo_t
	if err := unix.Sysinfo(&info); err == nil {
		totalMem := info.Totalram * uint64(info.Unit)
		freeMem := info.Freeram * uint64(info.Unit)
		if totalMem > 0 {
			h.MemoryPercent = 100.0 - (float64(freeMem)/float64(totalMem))*100.0
		}
		h.LoadAvg1 = float64(info.Loads[0]) / 65536.0
		h.LoadAvg5 = float64(info.Loads[1]) / 65536.0
		h.LoadAvg15 = float64(info.Loads[2]) / 65536.0
	}

	h.CPUPercent = m.cpuPercent()

	h.DiskPercent = diskUsagePercent("/")
}

// cpuPercent computes CPU utilization since the previous sample using
// RUSAGE_SELF cpu time deltas.
func (m *Monitor) cpuPercent() float64 {
	var usage unix.Rusage
	if err := unix.Getrusage(unix.RUSAGE_SELF, &usage); err != nil {
		return 0
	}

	cpuTime := usage.Utime.Nano() + usage.Stime.Nano()
	now := time.Now().UnixNano()

	dt := now - m.lastCPUTime
	dcpu := cpuTime - m.lastCPUSample

	m.lastCPUSample = cpuTime
	m.lastCPUTime = now

	if dt <= 0 || dcpu == 0 {
		return 0
	}

	percent := (float64(dcpu) / float64(dt)) * 100.0 * float64(runtime.NumCPU())
	return math.Min(percent, 100.0)
}

// diskUsagePercent returns the used percentage of the filesystem at path.
func diskUsagePercent(path string) float64 {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0
	}
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	if total == 0 {
		return 0
	}
	return (float64(total-free) / float64(total)) * 100.0
}
