package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/KOPElan/mingyue-agent/internal/scheduler"
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
			var tasks []*scheduler.Task
			if localMode {
				_, dataDir, err := loadLocalConfig()
				if err != nil {
					return err
				}
				sched, err := localScheduler(dataDir)
				if err != nil {
					return err
				}
				tasks = sched.ListTasks()
			} else {
				client := getAPIClient()
				resp, err := client.Get("/api/v1/scheduler/tasks")
				if err != nil {
					return err
				}

				var apiTasks []*scheduler.Task
				if err := json.Unmarshal(resp.Data, &apiTasks); err != nil {
					return fmt.Errorf("failed to parse response: %w", err)
				}
				tasks = apiTasks
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
				lastRun := "Never"
				if t.LastRun != nil {
					lastRun = t.LastRun.Format(time.RFC3339)
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
			name := args[0]
			if localMode {
				_, dataDir, err := loadLocalConfig()
				if err != nil {
					return err
				}
				sched, err := localScheduler(dataDir)
				if err != nil {
					return err
				}
				task := &scheduler.Task{
					Name:     name,
					Type:     taskType,
					Schedule: schedule,
					Enabled:  enabled,
					Params:   map[string]interface{}{},
				}
				if err := sched.AddTask(task); err != nil {
					return err
				}
				fmt.Printf("Task added with ID: %s\n", task.ID)
				return nil
			}

			client := getAPIClient()
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
			taskID := args[0]
			if localMode {
				_, dataDir, err := loadLocalConfig()
				if err != nil {
					return err
				}
				sched, err := localScheduler(dataDir)
				if err != nil {
					return err
				}
				if err := sched.DeleteTask(taskID); err != nil {
					return err
				}
			} else {
				client := getAPIClient()
				if _, err := client.Post("/api/v1/scheduler/tasks/remove", map[string]string{
					"id": taskID,
				}); err != nil {
					return err
				}
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
			taskID := args[0]
			if localMode {
				_, dataDir, err := loadLocalConfig()
				if err != nil {
					return err
				}
				sched, err := localScheduler(dataDir)
				if err != nil {
					return err
				}
				if _, err := sched.ExecuteTask(context.Background(), taskID); err != nil {
					return err
				}
			} else {
				client := getAPIClient()
				if _, err := client.Post("/api/v1/scheduler/tasks/execute", map[string]string{
					"id": taskID,
				}); err != nil {
					return err
				}
			}

			fmt.Printf("Task %s executed\n", taskID)

			return nil
		},
	}
}
