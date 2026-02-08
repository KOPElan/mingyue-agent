//go:build windows
// +build windows

package diskmanager

import "fmt"

// Partition represents a disk partition.
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

// DiskInfo represents physical disk information.
type DiskInfo struct {
	Device     string      `json:"device"`
	Model      string      `json:"model"`
	Size       uint64      `json:"size"`
	Partitions []Partition `json:"partitions"`
	SMART      *SMARTInfo  `json:"smart,omitempty"`
}

// SMARTInfo represents SMART health information.
type SMARTInfo struct {
	Healthy      bool   `json:"healthy"`
	Temperature  int    `json:"temperature"`
	PowerOnHours int    `json:"power_on_hours"`
	RawData      string `json:"raw_data,omitempty"`
}

// MountOptions represents mount operation options.
type MountOptions struct {
	Device     string   `json:"device"`
	MountPoint string   `json:"mount_point"`
	FileSystem string   `json:"filesystem"`
	Options    []string `json:"options"`
	ReadOnly   bool     `json:"read_only"`
}

// Manager handles disk management operations.
type Manager struct {
	allowedMountPoints []string
}

// New creates a new disk manager.
func New(allowedMountPoints []string) *Manager {
	return &Manager{allowedMountPoints: allowedMountPoints}
}

// ListPartitions lists all available partitions.
func (m *Manager) ListPartitions() ([]Partition, error) {
	return nil, fmt.Errorf("disk operations are not supported on windows")
}

// ListDisks lists all physical disks.
func (m *Manager) ListDisks() ([]DiskInfo, error) {
	return nil, fmt.Errorf("disk operations are not supported on windows")
}

// Mount mounts a device to a mount point.
func (m *Manager) Mount(opts MountOptions) error {
	return fmt.Errorf("disk operations are not supported on windows")
}

// Unmount unmounts a device or mount point.
func (m *Manager) Unmount(target string, force bool) error {
	return fmt.Errorf("disk operations are not supported on windows")
}

// GetSMARTInfo retrieves SMART information for a device.
func (m *Manager) GetSMARTInfo(device string) (*SMARTInfo, error) {
	return nil, fmt.Errorf("disk operations are not supported on windows")
}
