package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func schedulerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scheduler",
		Short: "Task scheduler operations",
		Long:  "Manage scheduled tasks",
	}

	cmd.AddCommand(schedulerListCmd())
	cmd.AddCommand(schedulerAddCmd())
	cmd.AddCommand(schedulerRemoveCmd())
	cmd.AddCommand(schedulerExecuteCmd())

	return cmd
}

func schedulerListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all scheduled tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()

			resp, err := client.Get("/api/v1/scheduler/tasks")
			if err != nil {
				return err
			}

			var tasks []struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				Type     string `json:"type"`
				Schedule string `json:"schedule"`
				Enabled  bool   `json:"enabled"`
				LastRun  string `json:"last_run"`
			}

			if err := json.Unmarshal(resp.Data, &tasks); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			if len(tasks) == 0 {
				fmt.Println("No scheduled tasks")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tTYPE\tSCHEDULE\tENABLED\tLAST RUN")
			for _, t := range tasks {
				enabled := "No"
				if t.Enabled {
					enabled = "Yes"
				}
				lastRun := t.LastRun
				if lastRun == "" {
					lastRun = "Never"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
					t.ID, t.Name, t.Type, t.Schedule, enabled, lastRun)
			}
			w.Flush()

			return nil
		},
	}
}

func schedulerAddCmd() *cobra.Command {
	var (
		taskType string
		schedule string
		enabled  bool
	)

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new scheduled task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()
			name := args[0]

			body := map[string]interface{}{
				"name":     name,
				"type":     taskType,
				"schedule": schedule,
				"enabled":  enabled,
			}

			resp, err := client.Post("/api/v1/scheduler/tasks/add", body)
			if err != nil {
				return err
			}

			var result struct {
				ID string `json:"id"`
			}

			if err := json.Unmarshal(resp.Data, &result); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			fmt.Printf("Task added with ID: %s\n", result.ID)

			return nil
		},
	}

	cmd.Flags().StringVarP(&taskType, "type", "t", "cleanup", "Task type (cleanup, backup, indexing)")
	cmd.Flags().StringVarP(&schedule, "schedule", "s", "daily", "Schedule (daily, weekly, monthly, or cron expression)")
	cmd.Flags().BoolVarP(&enabled, "enabled", "e", true, "Enable task immediately")

	return cmd
}

func schedulerRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <task-id>",
		Short: "Remove a scheduled task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()
			taskID := args[0]

			_, err := client.Post("/api/v1/scheduler/tasks/remove", map[string]string{
				"id": taskID,
			})
			if err != nil {
				return err
			}

			fmt.Printf("Task %s removed\n", taskID)

			return nil
		},
	}
}

func schedulerExecuteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "execute <task-id>",
		Short: "Execute a task immediately",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()
			taskID := args[0]

			_, err := client.Post("/api/v1/scheduler/tasks/execute", map[string]string{
				"id": taskID,
			})
			if err != nil {
				return err
			}

			fmt.Printf("Task %s executed\n", taskID)

			return nil
		},
	}
}
