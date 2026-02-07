package monitor

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"time"
)

type SystemStats struct {
	CPU     CPUStats     `json:"cpu"`
	Memory  MemoryStats  `json:"memory"`
	Disk    DiskStats    `json:"disk"`
	Process ProcessStats `json:"process"`
	Uptime  float64      `json:"uptime"`
}

type CPUStats struct {
	Cores        int     `json:"cores"`
	UsagePercent float64 `json:"usage_percent"`
	LoadAvg1     float64 `json:"load_avg_1"`
	LoadAvg5     float64 `json:"load_avg_5"`
	LoadAvg15    float64 `json:"load_avg_15"`
}

type MemoryStats struct {
	Total       uint64  `json:"total"`
	Available   uint64  `json:"available"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
	SwapTotal   uint64  `json:"swap_total"`
	SwapUsed    uint64  `json:"swap_used"`
}

type DiskStats struct {
	Total       uint64  `json:"total"`
	Free        uint64  `json:"free"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
}

type ProcessStats struct {
	PID        int    `json:"pid"`
	Goroutines int    `json:"goroutines"`
	MemAlloc   uint64 `json:"mem_alloc"`
	MemSys     uint64 `json:"mem_sys"`
	NumGC      uint32 `json:"num_gc"`
	OpenFiles  int    `json:"open_files"`
}

type Monitor struct {
	startTime time.Time
}

func New() *Monitor {
	return &Monitor{
		startTime: time.Now(),
	}
}

func (m *Monitor) GetStats() (*SystemStats, error) {
	stats := &SystemStats{
		Uptime: time.Since(m.startTime).Seconds(),
	}

	cpuStats, err := m.getCPUStats()
	if err == nil {
		stats.CPU = cpuStats
	}

	memStats, err := m.getMemoryStats()
	if err == nil {
		stats.Memory = memStats
	}

	diskStats, err := m.getDiskStats("/")
	if err == nil {
		stats.Disk = diskStats
	}

	procStats := m.getProcessStats()
	stats.Process = procStats

	return stats, nil
}

func (m *Monitor) getCPUStats() (CPUStats, error) {
	stats := CPUStats{
		Cores: runtime.NumCPU(),
	}

	loadAvg, err := getLoadAverage()
	if err == nil {
		stats.LoadAvg1 = loadAvg[0]
		stats.LoadAvg5 = loadAvg[1]
		stats.LoadAvg15 = loadAvg[2]
	}

	return stats, nil
}

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

func (m *Monitor) getProcessStats() ProcessStats {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	openFiles := countOpenFiles()

	stats := ProcessStats{
		PID:        os.Getpid(),
		Goroutines: runtime.NumGoroutine(),
		MemAlloc:   ms.Alloc,
		MemSys:     ms.Sys,
		NumGC:      ms.NumGC,
		OpenFiles:  openFiles,
	}

	return stats
}

func (m *Monitor) IsHealthy() bool {
	stats, err := m.GetStats()
	if err != nil {
		return false
	}

	if stats.Memory.UsedPercent > 95 {
		return false
	}

	if stats.Disk.UsedPercent > 98 {
		return false
	}

	return true
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
