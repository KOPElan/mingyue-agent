//go:build linux

package monitor

import (
	"fmt"
	"os"
	"syscall"
)

func (m *Monitor) getMemoryStats() (MemoryStats, error) {
	var si syscall.Sysinfo_t
	err := syscall.Sysinfo(&si)
	if err != nil {
		return MemoryStats{}, fmt.Errorf("sysinfo: %w", err)
	}

	total := si.Totalram * uint64(si.Unit)
	free := si.Freeram * uint64(si.Unit)
	used := total - free

	stats := MemoryStats{
		Total:       total,
		Available:   free,
		Used:        used,
		UsedPercent: float64(used) / float64(total) * 100,
		SwapTotal:   si.Totalswap * uint64(si.Unit),
		SwapUsed:    (si.Totalswap - si.Freeswap) * uint64(si.Unit),
	}

	return stats, nil
}

func (m *Monitor) getDiskStats(path string) (DiskStats, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(path, &stat)
	if err != nil {
		return DiskStats{}, fmt.Errorf("statfs: %w", err)
	}

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
	var info syscall.Sysinfo_t
	err := syscall.Sysinfo(&info)
	if err != nil {
		return [3]float64{}, err
	}

	scale := float64(1 << 16)
	return [3]float64{
		float64(info.Loads[0]) / scale,
		float64(info.Loads[1]) / scale,
		float64(info.Loads[2]) / scale,
	}, nil
}

func countOpenFiles() int {
	dir := fmt.Sprintf("/proc/%d/fd", os.Getpid())
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	return len(entries)
}
