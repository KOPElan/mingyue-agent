package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func indexerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "indexer",
		Short: "File indexing and search operations",
		Long:  "Manage file indexing and search indexed files",
	}

	cmd.AddCommand(indexerScanCmd())
	cmd.AddCommand(indexerSearchCmd())
	cmd.AddCommand(indexerStatsCmd())

	return cmd
}

func indexerScanCmd() *cobra.Command {
	var (
		recursive   bool
		incremental bool
	)

	cmd := &cobra.Command{
		Use:   "scan <paths...>",
		Short: "Scan paths for file indexing",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()

			body := map[string]interface{}{
				"paths":       args,
				"recursive":   recursive,
				"incremental": incremental,
			}

			resp, err := client.Post("/api/v1/indexer/scan", body)
			if err != nil {
				return err
			}

			var result struct {
				FilesAdded   int `json:"files_added"`
				FilesUpdated int `json:"files_updated"`
			}

			if err := json.Unmarshal(resp.Data, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			fmt.Printf("Scan completed:\n")
			fmt.Printf("  Files added:   %d\n", result.FilesAdded)
			fmt.Printf("  Files updated: %d\n", result.FilesUpdated)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&recursive, "recursive", "r", true, "Scan directories recursively")
	cmd.Flags().BoolVarP(&incremental, "incremental", "i", true, "Incremental scan (skip unchanged files)")

	return cmd
}

func indexerSearchCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search indexed files",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()
			query := args[0]

			resp, err := client.Get(fmt.Sprintf("/api/v1/indexer/search?q=%s&limit=%d", url.QueryEscape(query), limit))
			if err != nil {
				return err
			}

			var results []struct {
				Path      string `json:"path"`
				Size      int64  `json:"size"`
				MediaType string `json:"media_type"`
			}

			if err := json.Unmarshal(resp.Data, &results); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			if len(results) == 0 {
				fmt.Println("No results found")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "TYPE\tSIZE\tPATH")
			for _, r := range results {
				sizeStr := formatBytes(r.Size)
				fmt.Fprintf(w, "%s\t%s\t%s\n", r.MediaType, sizeStr, r.Path)
			}
			w.Flush()

			fmt.Printf("\nFound %d results\n", len(results))

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Maximum number of results")

	return cmd
}

func indexerStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Get indexer statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()

			resp, err := client.Get("/api/v1/indexer/stats")
			if err != nil {
				return err
			}

			var stats struct {
				TotalFiles int    `json:"total_files"`
				TotalSize  int64  `json:"total_size"`
				LastScan   string `json:"last_scan"`
			}

			if err := json.Unmarshal(resp.Data, &stats); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			fmt.Printf("Total Files: %d\n", stats.TotalFiles)
			fmt.Printf("Total Size:  %s\n", formatBytes(stats.TotalSize))
			fmt.Printf("Last Scan:   %s\n", stats.LastScan)

			return nil
		},
	}
}
