//go:build !windows
// +build !windows

package diskmanager

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// Partition represents a disk partition
type Partition struct {
	Name       string  `json:"name"`
	Device     string  `json:"device"`
	MountPoint string  `json:"mount_point"`
	FileSystem string  `json:"filesystem"`
	Size       uint64  `json:"size"`
	Used       uint64  `json:"used"`
	Available  uint64  `json:"available"`
	UsedPct    float64 `json:"used_percent"`
	UUID       string  `json:"uuid"`
	Label      string  `json:"label"`
	ReadOnly   bool    `json:"read_only"`
}

// DiskInfo represents physical disk information
type DiskInfo struct {
	Device     string      `json:"device"`
	Model      string      `json:"model"`
	Size       uint64      `json:"size"`
	Partitions []Partition `json:"partitions"`
	SMART      *SMARTInfo  `json:"smart,omitempty"`
}

// SMARTInfo represents SMART health information
type SMARTInfo struct {
	Healthy      bool   `json:"healthy"`
	Temperature  int    `json:"temperature"`
	PowerOnHours int    `json:"power_on_hours"`
	RawData      string `json:"raw_data,omitempty"`
}

// MountOptions represents mount operation options
type MountOptions struct {
	Device     string   `json:"device"`
	MountPoint string   `json:"mount_point"`
	FileSystem string   `json:"filesystem"`
	Options    []string `json:"options"`
	ReadOnly   bool     `json:"read_only"`
}

// Manager handles disk management operations
type Manager struct {
	allowedMountPoints []string
}

// New creates a new disk manager
func New(allowedMountPoints []string) *Manager {
	return &Manager{
		allowedMountPoints: allowedMountPoints,
	}
}

// ListPartitions lists all available partitions
func (m *Manager) ListPartitions() ([]Partition, error) {
	var partitions []Partition

	// Read /proc/mounts for mounted filesystems
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/mounts: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 6 {
			continue
		}

		device := fields[0]
		mountPoint := fields[1]
		fsType := fields[2]
		options := fields[3]

		// Skip virtual filesystems
		if strings.HasPrefix(device, "/dev/") {
			partition := Partition{
				Device:     device,
				MountPoint: mountPoint,
				FileSystem: fsType,
				ReadOnly:   strings.Contains(options, "ro"),
			}

			// Get disk usage stats
			var stat syscall.Statfs_t
			if err := syscall.Statfs(mountPoint, &stat); err == nil {
				partition.Size = stat.Blocks * uint64(stat.Bsize)
				partition.Available = stat.Bavail * uint64(stat.Bsize)
				partition.Used = partition.Size - (stat.Bfree * uint64(stat.Bsize))
				if partition.Size > 0 {
					partition.UsedPct = float64(partition.Used) / float64(partition.Size) * 100
				}
			}

			// Get UUID and label using blkid
			if uuid, label := m.getDeviceInfo(device); uuid != "" || label != "" {
				partition.UUID = uuid
				partition.Label = label
			}

			partition.Name = filepath.Base(device)
			partitions = append(partitions, partition)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading /proc/mounts: %w", err)
	}

	return partitions, nil
}

// getDeviceInfo gets UUID and label for a device using blkid
func (m *Manager) getDeviceInfo(device string) (uuid, label string) {
	cmd := exec.Command("blkid", "-o", "export", device)
	output, err := cmd.Output()
	if err != nil {
		return "", ""
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "UUID=") {
			uuid = strings.TrimPrefix(line, "UUID=")
		} else if strings.HasPrefix(line, "LABEL=") {
			label = strings.TrimPrefix(line, "LABEL=")
		}
	}

	return uuid, label
}

// ListDisks lists all physical disks
func (m *Manager) ListDisks() ([]DiskInfo, error) {
	// Use lsblk to get disk information
	cmd := exec.Command("lsblk", "-J", "-b", "-o", "NAME,SIZE,MODEL,TYPE")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute lsblk: %w", err)
	}

	var result struct {
		BlockDevices []struct {
			Name     string `json:"name"`
			Size     uint64 `json:"size,string"`
			Model    string `json:"model"`
			Type     string `json:"type"`
			Children []struct {
				Name string `json:"name"`
				Size uint64 `json:"size,string"`
				Type string `json:"type"`
			} `json:"children"`
		} `json:"blockdevices"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse lsblk output: %w", err)
	}

	var disks []DiskInfo
	partitions, _ := m.ListPartitions()

	for _, dev := range result.BlockDevices {
		if dev.Type == "disk" {
			disk := DiskInfo{
				Device: "/dev/" + dev.Name,
				Model:  dev.Model,
				Size:   dev.Size,
			}

			// Match partitions to this disk
			for _, part := range partitions {
				if strings.HasPrefix(part.Device, disk.Device) {
					disk.Partitions = append(disk.Partitions, part)
				}
			}

			disks = append(disks, disk)
		}
	}

	return disks, nil
}

// Mount mounts a device to a mount point
func (m *Manager) Mount(opts MountOptions) error {
	// Validate mount point
	if !m.isAllowedMountPoint(opts.MountPoint) {
		return fmt.Errorf("mount point %s is not in allowed list", opts.MountPoint)
	}

	// Create mount point if it doesn't exist
	if err := os.MkdirAll(opts.MountPoint, 0755); err != nil {
		return fmt.Errorf("failed to create mount point: %w", err)
	}

	// Build mount command
	args := []string{}
	if opts.FileSystem != "" {
		args = append(args, "-t", opts.FileSystem)
	}
	if len(opts.Options) > 0 {
		args = append(args, "-o", strings.Join(opts.Options, ","))
	}
	if opts.ReadOnly {
		args = append(args, "-r")
	}
	args = append(args, opts.Device, opts.MountPoint)

	cmd := exec.Command("mount", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mount failed: %s: %w", string(output), err)
	}

	return nil
}

// Unmount unmounts a device or mount point
func (m *Manager) Unmount(target string, force bool) error {
	args := []string{}
	if force {
		args = append(args, "-f")
	}
	args = append(args, target)

	cmd := exec.Command("umount", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("unmount failed: %s: %w", string(output), err)
	}

	return nil
}

// GetSMARTInfo retrieves SMART information for a device
func (m *Manager) GetSMARTInfo(device string) (*SMARTInfo, error) {
	// Try using smartctl
	cmd := exec.Command("smartctl", "-H", "-A", device)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// smartctl returns non-zero even on success sometimes
		if len(output) == 0 {
			return nil, fmt.Errorf("smartctl failed: %w", err)
		}
	}

	info := &SMARTInfo{
		RawData: string(output),
	}

	// Parse basic health status
	if strings.Contains(string(output), "PASSED") {
		info.Healthy = true
	}

	// Parse temperature (simplified)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Temperature") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "Temperature_Celsius" && i+9 < len(fields) {
					fmt.Sscanf(fields[i+9], "%d", &info.Temperature)
				}
			}
		}
		if strings.Contains(line, "Power_On_Hours") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "Power_On_Hours" && i+9 < len(fields) {
					fmt.Sscanf(fields[i+9], "%d", &info.PowerOnHours)
				}
			}
		}
	}

	return info, nil
}

// isAllowedMountPoint checks if a mount point is in the allowed list
func (m *Manager) isAllowedMountPoint(mountPoint string) bool {
	if len(m.allowedMountPoints) == 0 {
		return true // No restrictions
	}

	absPath, err := filepath.Abs(mountPoint)
	if err != nil {
		return false
	}

	for _, allowed := range m.allowedMountPoints {
		if strings.HasPrefix(absPath, allowed) {
			return true
		}
	}

	return false
}
