package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func monitorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "System monitoring operations",
		Long:  "Monitor system resources including CPU, memory, disk, and process statistics",
	}

	cmd.AddCommand(monitorStatsCmd())
	cmd.AddCommand(monitorHealthCmd())

	return cmd
}

func monitorStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Get system resource statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()

			resp, err := client.Get("/api/v1/monitor/stats")
			if err != nil {
				return err
			}

			var stats struct {
				CPU struct {
					Cores      int     `json:"cores"`
					UsagePC    float64 `json:"usage_percent"`
					LoadAvg1   float64 `json:"load_avg_1"`
					LoadAvg5   float64 `json:"load_avg_5"`
					LoadAvg15  float64 `json:"load_avg_15"`
				} `json:"cpu"`
				Memory struct {
					Total      int64   `json:"total"`
					Available  int64   `json:"available"`
					Used       int64   `json:"used"`
					UsedPC     float64 `json:"used_percent"`
					SwapTotal  int64   `json:"swap_total"`
					SwapUsed   int64   `json:"swap_used"`
				} `json:"memory"`
				Disk struct {
					Total    int64   `json:"total"`
					Free     int64   `json:"free"`
					Used     int64   `json:"used"`
					UsedPC   float64 `json:"used_percent"`
				} `json:"disk"`
				Process struct {
					PID        int   `json:"pid"`
					Goroutines int   `json:"goroutines"`
					MemAlloc   int64 `json:"mem_alloc"`
					MemSys     int64 `json:"mem_sys"`
					NumGC      int   `json:"num_gc"`
					OpenFiles  int   `json:"open_files"`
				} `json:"process"`
				Uptime float64 `json:"uptime"`
			}

			if err := json.Unmarshal(resp.Data, &stats); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			fmt.Println("=== CPU ===")
			fmt.Printf("Cores:         %d\n", stats.CPU.Cores)
			fmt.Printf("Usage:         %.2f%%\n", stats.CPU.UsagePC)
			fmt.Printf("Load Avg (1m): %.2f\n", stats.CPU.LoadAvg1)
			fmt.Printf("Load Avg (5m): %.2f\n", stats.CPU.LoadAvg5)
			fmt.Printf("Load Avg (15m): %.2f\n", stats.CPU.LoadAvg15)

			fmt.Println("\n=== Memory ===")
			fmt.Printf("Total:     %s\n", formatBytes(stats.Memory.Total))
			fmt.Printf("Used:      %s (%.2f%%)\n", formatBytes(stats.Memory.Used), stats.Memory.UsedPC)
			fmt.Printf("Available: %s\n", formatBytes(stats.Memory.Available))
			fmt.Printf("Swap Total: %s\n", formatBytes(stats.Memory.SwapTotal))
			fmt.Printf("Swap Used:  %s\n", formatBytes(stats.Memory.SwapUsed))

			fmt.Println("\n=== Disk ===")
			fmt.Printf("Total: %s\n", formatBytes(stats.Disk.Total))
			fmt.Printf("Used:  %s (%.2f%%)\n", formatBytes(stats.Disk.Used), stats.Disk.UsedPC)
			fmt.Printf("Free:  %s\n", formatBytes(stats.Disk.Free))

			fmt.Println("\n=== Process ===")
			fmt.Printf("PID:        %d\n", stats.Process.PID)
			fmt.Printf("Goroutines: %d\n", stats.Process.Goroutines)
			fmt.Printf("Memory:     %s\n", formatBytes(stats.Process.MemAlloc))
			fmt.Printf("Sys Memory: %s\n", formatBytes(stats.Process.MemSys))
			fmt.Printf("GC Runs:    %d\n", stats.Process.NumGC)
			fmt.Printf("Open Files: %d\n", stats.Process.OpenFiles)

			fmt.Printf("\n=== Uptime ===\n%.2f seconds\n", stats.Uptime)

			return nil
		},
	}
}

func monitorHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Get system health status",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()

			resp, err := client.Get("/api/v1/monitor/health")
			if err != nil {
				return err
			}

			var health struct {
				Status    string `json:"status"`
				Healthy   bool   `json:"healthy"`
				Timestamp string `json:"timestamp"`
			}

			if err := json.Unmarshal(resp.Data, &health); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			fmt.Printf("Status:    %s\n", health.Status)
			fmt.Printf("Healthy:   %v\n", health.Healthy)
			fmt.Printf("Timestamp: %s\n", health.Timestamp)

			return nil
		},
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
