//go:build windows

package monitor

func (m *Monitor) getMemoryStats() (MemoryStats, error) {
	// Windows would use GlobalMemoryStatusEx
	// For now, return basic stats with zero values
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
	// Windows uses GetDiskFreeSpaceEx
	// For a basic implementation, return zeros
	return DiskStats{
		Total:       0,
		Free:        0,
		Used:        0,
		UsedPercent: 0,
	}, nil
}

func getLoadAverage() ([3]float64, error) {
	// Windows doesn't have load average concept
	return [3]float64{0, 0, 0}, nil
}

func countOpenFiles() int {
	// Windows doesn't track open files in the same way
	return 0
}
