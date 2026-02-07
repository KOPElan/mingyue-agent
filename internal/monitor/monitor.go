package monitor

import (
	"os"
	"runtime"
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
