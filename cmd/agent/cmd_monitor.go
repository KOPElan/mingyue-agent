package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/KOPElan/mingyue-agent/internal/monitor"
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
			var stats *monitor.SystemStats
			if localMode {
				mon := localMonitor()
				result, err := mon.GetStats()
				if err != nil {
					return err
				}
				stats = result
			} else {
				client := getAPIClient()
				resp, err := client.Get("/api/v1/monitor/stats")
				if err != nil {
					return err
				}

				var apiStats monitor.SystemStats
				if err := json.Unmarshal(resp.Data, &apiStats); err != nil {
					return fmt.Errorf("failed to parse response: %w", err)
				}
				stats = &apiStats
			}

			fmt.Println("=== CPU ===")
			fmt.Printf("Cores:         %d\n", stats.CPU.Cores)
			fmt.Printf("Usage:         %.2f%%\n", stats.CPU.UsagePercent)
			fmt.Printf("Load Avg (1m): %.2f\n", stats.CPU.LoadAvg1)
			fmt.Printf("Load Avg (5m): %.2f\n", stats.CPU.LoadAvg5)
			fmt.Printf("Load Avg (15m): %.2f\n", stats.CPU.LoadAvg15)

			fmt.Println("\n=== Memory ===")
			fmt.Printf("Total:     %s\n", formatBytes(int64(stats.Memory.Total)))
			fmt.Printf("Used:      %s (%.2f%%)\n", formatBytes(int64(stats.Memory.Used)), stats.Memory.UsedPercent)
			fmt.Printf("Available: %s\n", formatBytes(int64(stats.Memory.Available)))
			fmt.Printf("Swap Total: %s\n", formatBytes(int64(stats.Memory.SwapTotal)))
			fmt.Printf("Swap Used:  %s\n", formatBytes(int64(stats.Memory.SwapUsed)))

			fmt.Println("\n=== Disk ===")
			fmt.Printf("Total: %s\n", formatBytes(int64(stats.Disk.Total)))
			fmt.Printf("Used:  %s (%.2f%%)\n", formatBytes(int64(stats.Disk.Used)), stats.Disk.UsedPercent)
			fmt.Printf("Free:  %s\n", formatBytes(int64(stats.Disk.Free)))

			fmt.Println("\n=== Process ===")
			fmt.Printf("PID:        %d\n", stats.Process.PID)
			fmt.Printf("Goroutines: %d\n", stats.Process.Goroutines)
			fmt.Printf("Memory:     %s\n", formatBytes(int64(stats.Process.MemAlloc)))
			fmt.Printf("Sys Memory: %s\n", formatBytes(int64(stats.Process.MemSys)))
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
			if localMode {
				mon := localMonitor()
				healthy := mon.IsHealthy()
				status := "healthy"
				if !healthy {
					status = "unhealthy"
				}
				fmt.Printf("Status:    %s\n", status)
				fmt.Printf("Healthy:   %v\n", healthy)
				fmt.Printf("Timestamp: %s\n", time.Now().Format(time.RFC3339))
				return nil
			}

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
