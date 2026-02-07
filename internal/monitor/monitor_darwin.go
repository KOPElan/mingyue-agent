//go:build darwin

package monitor

import (
	"fmt"
	"syscall"
)

func (m *Monitor) getMemoryStats() (MemoryStats, error) {
	// For macOS, we can use syscall to get page size and vm_stat
	// For now, return basic stats with zero values
	// A production implementation would use sysctl or vm_stat
	return MemoryStats{
		Total:       0,
		Available:   0,
		Used:        0,
		UsedPercent: 0,
		SwapTotal:   0,
		SwapUsed:    0,
	}, nil
}

func (m *Monitor) getDiskStats(path string) (DiskStats, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return DiskStats{}, fmt.Errorf("statfs: %w", err)
	}

	// macOS Statfs_t has different field names
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	used := total - free

	stats := DiskStats{
		Total:       total,
		Free:        free,
		Used:        used,
		UsedPercent: float64(used) / float64(total) * 100,
	}

	return stats, nil
}

func getLoadAverage() ([3]float64, error) {
	// macOS doesn't have Sysinfo, but we can use getloadavg syscall
	// For now, return zeros
	return [3]float64{0, 0, 0}, nil
}

func countOpenFiles() int {
	// macOS doesn't have /proc, would need to use lsof or other methods
	// Return 0 for now
	return 0
}
