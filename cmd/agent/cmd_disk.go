package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func diskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disk",
		Short: "Disk management operations",
		Long:  "Manage disks, partitions, and perform SMART monitoring",
	}

	cmd.AddCommand(diskListCmd())
	cmd.AddCommand(diskPartitionsCmd())
	cmd.AddCommand(diskSmartCmd())
	cmd.AddCommand(diskMountCmd())
	cmd.AddCommand(diskUnmountCmd())

	return cmd
}

func diskListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available disks",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()

			resp, err := client.Get("/api/v1/disk/list")
			if err != nil {
				return err
			}

			var disks []struct {
				Device string `json:"device"`
				Model  string `json:"model"`
				Size   int64  `json:"size"`
			}

			if err := json.Unmarshal(resp.Data, &disks); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "DEVICE\tMODEL\tSIZE (GB)")
			for _, d := range disks {
				sizeGB := float64(d.Size) / (1024 * 1024 * 1024)
				fmt.Fprintf(w, "%s\t%s\t%.2f\n", d.Device, d.Model, sizeGB)
			}
			w.Flush()

			return nil
		},
	}
}

func diskPartitionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "partitions",
		Short: "List all disk partitions",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()

			resp, err := client.Get("/api/v1/disk/partitions")
			if err != nil {
				return err
			}

			var partitions []struct {
				Device     string `json:"device"`
				MountPoint string `json:"mount_point"`
				Filesystem string `json:"filesystem"`
				Size       int64  `json:"size"`
				Used       int64  `json:"used"`
				Available  int64  `json:"available"`
				UUID       string `json:"uuid"`
				Label      string `json:"label"`
			}

			if err := json.Unmarshal(resp.Data, &partitions); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "DEVICE\tMOUNT\tFS\tSIZE (GB)\tUSED (GB)\tAVAIL (GB)\tLABEL")
			for _, p := range partitions {
				sizeGB := float64(p.Size) / (1024 * 1024 * 1024)
				usedGB := float64(p.Used) / (1024 * 1024 * 1024)
				availGB := float64(p.Available) / (1024 * 1024 * 1024)
				mount := p.MountPoint
				if mount == "" {
					mount = "-"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%.2f\t%.2f\t%.2f\t%s\n",
					p.Device, mount, p.Filesystem, sizeGB, usedGB, availGB, p.Label)
			}
			w.Flush()

			return nil
		},
	}
}

func diskSmartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "smart <device>",
		Short: "Get SMART information for a disk",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()
			device := args[0]

			resp, err := client.Get(fmt.Sprintf("/api/v1/disk/smart?device=%s", device))
			if err != nil {
				return err
			}

			var smart struct {
				Device      string `json:"device"`
				Model       string `json:"model"`
				SerialNo    string `json:"serial_number"`
				Healthy     bool   `json:"healthy"`
				Temperature int    `json:"temperature"`
				PowerOnHrs  int    `json:"power_on_hours"`
			}

			if err := json.Unmarshal(resp.Data, &smart); err != nil {
				return fmt.Errorf("failed to parse response: %w", err)
			}

			fmt.Printf("Device:      %s\n", smart.Device)
			fmt.Printf("Model:       %s\n", smart.Model)
			fmt.Printf("Serial:      %s\n", smart.SerialNo)
			fmt.Printf("Health:      %s\n", func() string {
				if smart.Healthy {
					return "PASSED"
				}
				return "FAILED"
			}())
			fmt.Printf("Temperature: %d°C\n", smart.Temperature)
			fmt.Printf("Power On:    %d hours\n", smart.PowerOnHrs)

			return nil
		},
	}
}

func diskMountCmd() *cobra.Command {
	var (
		mountPoint string
		filesystem string
	)

	cmd := &cobra.Command{
		Use:   "mount <device>",
		Short: "Mount a disk partition",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()
			device := args[0]

			_, err := client.Post("/api/v1/disk/mount", map[string]string{
				"device":      device,
				"mount_point": mountPoint,
				"filesystem":  filesystem,
			})
			if err != nil {
				return err
			}

			fmt.Printf("Mounted %s at %s\n", device, mountPoint)
			return nil
		},
	}

	cmd.Flags().StringVarP(&mountPoint, "mount-point", "m", "", "Mount point path (required)")
	cmd.Flags().StringVarP(&filesystem, "filesystem", "f", "ext4", "Filesystem type")
	cmd.MarkFlagRequired("mount-point")

	return cmd
}

func diskUnmountCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "unmount <target>",
		Short: "Unmount a disk partition",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := getAPIClient()
			target := args[0]

			_, err := client.Post("/api/v1/disk/unmount", map[string]interface{}{
				"target": target,
				"force":  force,
			})
			if err != nil {
				return err
			}

			fmt.Printf("Unmounted %s\n", target)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force unmount")

	return cmd
}
