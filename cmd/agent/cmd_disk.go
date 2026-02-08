package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"text/tabwriter"

	"github.com/KOPElan/mingyue-agent/internal/diskmanager"
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
			var disks []diskmanager.DiskInfo
			if localMode {
				cfg, _, err := loadLocalConfig()
				if err != nil {
					return err
				}
				mgr := localDiskManager(cfg)
				result, err := mgr.ListDisks()
				if err != nil {
					return err
				}
				disks = result
			} else {
				client := getAPIClient()
				resp, err := client.Get("/api/v1/disk/list")
				if err != nil {
					return err
				}

				var apiDisks []diskmanager.DiskInfo
				if err := json.Unmarshal(resp.Data, &apiDisks); err != nil {
					return fmt.Errorf("failed to parse response: %w", err)
				}
				disks = apiDisks
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
			var partitions []diskmanager.Partition
			if localMode {
				cfg, _, err := loadLocalConfig()
				if err != nil {
					return err
				}
				mgr := localDiskManager(cfg)
				result, err := mgr.ListPartitions()
				if err != nil {
					return err
				}
				partitions = result
			} else {
				client := getAPIClient()
				resp, err := client.Get("/api/v1/disk/partitions")
				if err != nil {
					return err
				}

				var apiPartitions []diskmanager.Partition
				if err := json.Unmarshal(resp.Data, &apiPartitions); err != nil {
					return fmt.Errorf("failed to parse response: %w", err)
				}
				partitions = apiPartitions
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
					p.Device, mount, p.FileSystem, sizeGB, usedGB, availGB, p.Label)
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
			device := args[0]
			if localMode {
				cfg, _, err := loadLocalConfig()
				if err != nil {
					return err
				}
				mgr := localDiskManager(cfg)
				smart, err := mgr.GetSMARTInfo(device)
				if err != nil {
					return err
				}

				fmt.Printf("Device:      %s\n", device)
				fmt.Printf("Health:      %s\n", func() string {
					if smart.Healthy {
						return "PASSED"
					}
					return "FAILED"
				}())
				fmt.Printf("Temperature: %d°C\n", smart.Temperature)
				fmt.Printf("Power On:    %d hours\n", smart.PowerOnHours)
				return nil
			}

			client := getAPIClient()
			resp, err := client.Get(fmt.Sprintf("/api/v1/disk/smart?device=%s", url.QueryEscape(device)))
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
			device := args[0]
			if localMode {
				cfg, _, err := loadLocalConfig()
				if err != nil {
					return err
				}
				mgr := localDiskManager(cfg)
				if err := mgr.Mount(diskmanager.MountOptions{
					Device:     device,
					MountPoint: mountPoint,
					FileSystem: filesystem,
				}); err != nil {
					return err
				}
			} else {
				client := getAPIClient()
				if _, err := client.Post("/api/v1/disk/mount", map[string]string{
					"device":      device,
					"mount_point": mountPoint,
					"filesystem":  filesystem,
				}); err != nil {
					return err
				}
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
			target := args[0]
			if localMode {
				cfg, _, err := loadLocalConfig()
				if err != nil {
					return err
				}
				mgr := localDiskManager(cfg)
				if err := mgr.Unmount(target, force); err != nil {
					return err
				}
			} else {
				client := getAPIClient()
				if _, err := client.Post("/api/v1/disk/unmount", map[string]interface{}{
					"target": target,
					"force":  force,
				}); err != nil {
					return err
				}
			}

			fmt.Printf("Unmounted %s\n", target)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force unmount")

	return cmd
}
