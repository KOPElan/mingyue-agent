package netmanager

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Interface represents a network interface
type Interface struct {
	Name        string    `json:"name"`
	MAC         string    `json:"mac"`
	IPAddresses []string  `json:"ip_addresses"`
	State       string    `json:"state"`
	Speed       int64     `json:"speed"`
	MTU         int       `json:"mtu"`
	RxBytes     uint64    `json:"rx_bytes"`
	TxBytes     uint64    `json:"tx_bytes"`
	RxPackets   uint64    `json:"rx_packets"`
	TxPackets   uint64    `json:"tx_packets"`
	RxErrors    uint64    `json:"rx_errors"`
	TxErrors    uint64    `json:"tx_errors"`
	Flags       []string  `json:"flags"`
	LastUpdated time.Time `json:"last_updated"`
}

// IPConfig represents IP configuration
type IPConfig struct {
	Interface  string   `json:"interface"`
	Method     string   `json:"method"` // "static" or "dhcp"
	Address    string   `json:"address,omitempty"`
	Netmask    string   `json:"netmask,omitempty"`
	Gateway    string   `json:"gateway,omitempty"`
	DNSServers []string `json:"dns_servers,omitempty"`
}

// ConfigHistory represents a historical configuration
type ConfigHistory struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Interface string    `json:"interface"`
	Config    IPConfig  `json:"config"`
	User      string    `json:"user"`
	Reason    string    `json:"reason"`
}

// PortInfo represents listening port information
type PortInfo struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Process  string `json:"process"`
	State    string `json:"state"`
	Address  string `json:"address"`
}

// Manager handles network management operations
type Manager struct {
	managementInterface string
	historyFile         string
	history             []ConfigHistory
	mu                  sync.RWMutex
}

// Config represents network manager configuration
type Config struct {
	ManagementInterface string
	HistoryFile         string
}

// New creates a new network manager
func New(cfg *Config) (*Manager, error) {
	historyFile := cfg.HistoryFile
	if historyFile == "" {
		historyFile = "/var/lib/mingyue-agent/network-history.json"
	}

	m := &Manager{
		managementInterface: cfg.ManagementInterface,
		historyFile:         historyFile,
		history:             []ConfigHistory{},
	}

	// Load history
	if err := m.loadHistory(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load history: %w", err)
	}

	return m, nil
}

// ListInterfaces returns all network interfaces
func (m *Manager) ListInterfaces() ([]Interface, error) {
	interfaces := []Interface{}

	// Read interface names from /sys/class/net
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return nil, fmt.Errorf("read /sys/class/net: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		iface, err := m.getInterfaceInfo(entry.Name())
		if err != nil {
			continue
		}

		interfaces = append(interfaces, iface)
	}

	return interfaces, nil
}

// GetInterface returns information about a specific interface
func (m *Manager) GetInterface(name string) (*Interface, error) {
	iface, err := m.getInterfaceInfo(name)
	if err != nil {
		return nil, err
	}
	return &iface, nil
}

// SetIPConfig sets IP configuration for an interface
func (m *Manager) SetIPConfig(config *IPConfig, user, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Prevent configuration of non-management interface
	if m.managementInterface != "" && config.Interface != m.managementInterface {
		return fmt.Errorf("can only configure management interface %s", m.managementInterface)
	}

	// Save current config to history before changing
	currentConfig, _ := m.getCurrentIPConfig(config.Interface)
	if currentConfig != nil {
		m.addToHistory(config.Interface, *currentConfig, user, "backup before change")
	}

	// Apply configuration
	if err := m.applyIPConfig(config); err != nil {
		return fmt.Errorf("apply config: %w", err)
	}

	// Add new config to history
	m.addToHistory(config.Interface, *config, user, reason)

	return m.saveHistory()
}

// RollbackConfig rolls back to a previous configuration
func (m *Manager) RollbackConfig(historyID string, user string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var targetConfig *ConfigHistory
	for i := range m.history {
		if m.history[i].ID == historyID {
			targetConfig = &m.history[i]
			break
		}
	}

	if targetConfig == nil {
		return fmt.Errorf("configuration %s not found in history", historyID)
	}

	// Apply historical configuration
	if err := m.applyIPConfig(&targetConfig.Config); err != nil {
		return fmt.Errorf("apply rollback config: %w", err)
	}

	// Add rollback to history
	m.addToHistory(targetConfig.Interface, targetConfig.Config, user, fmt.Sprintf("rollback to %s", historyID))

	return m.saveHistory()
}

// ListConfigHistory returns configuration history
func (m *Manager) ListConfigHistory(iface string) []ConfigHistory {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if iface == "" {
		return m.history
	}

	filtered := []ConfigHistory{}
	for _, h := range m.history {
		if h.Interface == iface {
			filtered = append(filtered, h)
		}
	}
	return filtered
}

// EnableInterface enables a network interface
func (m *Manager) EnableInterface(name string) error {
	cmd := exec.Command("ip", "link", "set", name, "up")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("enable interface: %w, output: %s", err, string(output))
	}
	return nil
}

// DisableInterface disables a network interface
func (m *Manager) DisableInterface(name string) error {
	// Prevent disabling management interface
	if m.managementInterface != "" && name == m.managementInterface {
		return fmt.Errorf("cannot disable management interface")
	}

	cmd := exec.Command("ip", "link", "set", name, "down")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("disable interface: %w, output: %s", err, string(output))
	}
	return nil
}

// ListListeningPorts returns all listening ports
func (m *Manager) ListListeningPorts() ([]PortInfo, error) {
	ports := []PortInfo{}

	// Parse netstat or ss output
	cmd := exec.Command("ss", "-tulpn")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to netstat if ss is not available
		cmd = exec.Command("netstat", "-tulpn")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to get port info: %w", err)
		}
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "LISTEN") || strings.Contains(line, "UNCONN") {
			port := m.parsePortLine(line)
			if port != nil {
				ports = append(ports, *port)
			}
		}
	}

	return ports, nil
}

// GetTrafficStats returns traffic statistics for all interfaces
func (m *Manager) GetTrafficStats() (map[string]Interface, error) {
	interfaces, err := m.ListInterfaces()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]Interface)
	for _, iface := range interfaces {
		stats[iface.Name] = iface
	}

	return stats, nil
}

// Private methods

func (m *Manager) getInterfaceInfo(name string) (Interface, error) {
	iface := Interface{
		Name:        name,
		LastUpdated: time.Now(),
	}

	basePath := filepath.Join("/sys/class/net", name)

	// Read MAC address
	if data, err := os.ReadFile(filepath.Join(basePath, "address")); err == nil {
		iface.MAC = strings.TrimSpace(string(data))
	}

	// Read state
	if data, err := os.ReadFile(filepath.Join(basePath, "operstate")); err == nil {
		iface.State = strings.TrimSpace(string(data))
	}

	// Read MTU
	if data, err := os.ReadFile(filepath.Join(basePath, "mtu")); err == nil {
		if mtu, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
			iface.MTU = mtu
		}
	}

	// Read speed (may not be available for all interfaces)
	if data, err := os.ReadFile(filepath.Join(basePath, "speed")); err == nil {
		if speed, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); err == nil {
			iface.Speed = speed
		}
	}

	// Read statistics
	statsPath := filepath.Join(basePath, "statistics")
	if data, err := os.ReadFile(filepath.Join(statsPath, "rx_bytes")); err == nil {
		if val, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); err == nil {
			iface.RxBytes = val
		}
	}
	if data, err := os.ReadFile(filepath.Join(statsPath, "tx_bytes")); err == nil {
		if val, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); err == nil {
			iface.TxBytes = val
		}
	}
	if data, err := os.ReadFile(filepath.Join(statsPath, "rx_packets")); err == nil {
		if val, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); err == nil {
			iface.RxPackets = val
		}
	}
	if data, err := os.ReadFile(filepath.Join(statsPath, "tx_packets")); err == nil {
		if val, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); err == nil {
			iface.TxPackets = val
		}
	}
	if data, err := os.ReadFile(filepath.Join(statsPath, "rx_errors")); err == nil {
		if val, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); err == nil {
			iface.RxErrors = val
		}
	}
	if data, err := os.ReadFile(filepath.Join(statsPath, "tx_errors")); err == nil {
		if val, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); err == nil {
			iface.TxErrors = val
		}
	}

	// Get IP addresses using 'ip' command
	cmd := exec.Command("ip", "-o", "addr", "show", name)
	output, err := cmd.CombinedOutput()
	if err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(output))
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			for i, field := range fields {
				if field == "inet" || field == "inet6" {
					if i+1 < len(fields) {
						addr := strings.Split(fields[i+1], "/")[0]
						iface.IPAddresses = append(iface.IPAddresses, addr)
					}
				}
			}
		}
	}

	// Read flags
	if data, err := os.ReadFile(filepath.Join(basePath, "flags")); err == nil {
		flagsStr := strings.TrimSpace(string(data))
		// Parse hex flags (simplified)
		iface.Flags = []string{}
		if strings.Contains(flagsStr, "0x1") {
			iface.Flags = append(iface.Flags, "UP")
		}
	}

	return iface, nil
}

func (m *Manager) getCurrentIPConfig(iface string) (*IPConfig, error) {
	config := &IPConfig{
		Interface: iface,
	}

	// Try to determine if using DHCP or static
	// This is simplified - real implementation would check network manager config
	cmd := exec.Command("ip", "addr", "show", iface)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	// Parse current IP
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "inet ") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "inet" && i+1 < len(fields) {
					parts := strings.Split(fields[i+1], "/")
					if len(parts) > 0 {
						config.Address = parts[0]
					}
					if len(parts) > 1 {
						// Convert CIDR to netmask (simplified)
						config.Netmask = parts[1]
					}
					break
				}
			}
		}
	}

	// Get gateway
	cmd = exec.Command("ip", "route", "show", "dev", iface)
	output, err = cmd.CombinedOutput()
	if err == nil {
		lines = strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "default via") {
				fields := strings.Fields(line)
				for i, field := range fields {
					if field == "via" && i+1 < len(fields) {
						config.Gateway = fields[i+1]
						break
					}
				}
			}
		}
	}

	config.Method = "static" // Default assumption

	return config, nil
}

func (m *Manager) applyIPConfig(config *IPConfig) error {
	if config.Method == "dhcp" {
		// Request DHCP configuration
		cmd := exec.Command("dhclient", config.Interface)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("dhclient failed: %w, output: %s", err, string(output))
		}
	} else if config.Method == "static" {
		// Flush existing addresses
		cmd := exec.Command("ip", "addr", "flush", "dev", config.Interface)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("flush addresses: %w, output: %s", err, string(output))
		}

		// Add static IP
		if config.Address != "" && config.Netmask != "" {
			cmd = exec.Command("ip", "addr", "add", fmt.Sprintf("%s/%s", config.Address, config.Netmask), "dev", config.Interface)
			if output, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("add address: %w, output: %s", err, string(output))
			}
		}

		// Add gateway
		if config.Gateway != "" {
			cmd = exec.Command("ip", "route", "add", "default", "via", config.Gateway, "dev", config.Interface)
			output, err := cmd.CombinedOutput()
			if err != nil && !strings.Contains(string(output), "File exists") {
				return fmt.Errorf("add gateway: %w, output: %s", err, string(output))
			}
		}

		// Update DNS if provided
		if len(config.DNSServers) > 0 {
			if err := m.updateDNS(config.DNSServers); err != nil {
				return fmt.Errorf("update DNS: %w", err)
			}
		}
	}

	return nil
}

func (m *Manager) updateDNS(servers []string) error {
	content := "# Generated by mingyue-agent\n"
	for _, server := range servers {
		content += fmt.Sprintf("nameserver %s\n", server)
	}

	// Backup existing resolv.conf
	if _, err := os.Stat("/etc/resolv.conf"); err == nil {
		os.Rename("/etc/resolv.conf", "/etc/resolv.conf.bak")
	}

	if err := os.WriteFile("/etc/resolv.conf", []byte(content), 0644); err != nil {
		return fmt.Errorf("write resolv.conf: %w", err)
	}

	return nil
}

func (m *Manager) addToHistory(iface string, config IPConfig, user, reason string) {
	history := ConfigHistory{
		ID:        fmt.Sprintf("%s-%d", iface, time.Now().Unix()),
		Timestamp: time.Now(),
		Interface: iface,
		Config:    config,
		User:      user,
		Reason:    reason,
	}

	m.history = append(m.history, history)

	// Keep only last 100 entries
	if len(m.history) > 100 {
		m.history = m.history[len(m.history)-100:]
	}
}

func (m *Manager) parsePortLine(line string) *PortInfo {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return nil
	}

	var protocol, address, state, process string
	var port int

	// Parse based on ss or netstat format
	if strings.HasPrefix(fields[0], "tcp") || strings.HasPrefix(fields[0], "udp") {
		protocol = fields[0]
		if len(fields) > 4 {
			// Parse address:port
			parts := strings.Split(fields[4], ":")
			if len(parts) >= 2 {
				address = strings.Join(parts[:len(parts)-1], ":")
				portStr := parts[len(parts)-1]
				if p, err := strconv.Atoi(portStr); err == nil {
					port = p
				}
			}
		}
		if len(fields) > 1 {
			state = fields[1]
		}
		if len(fields) > 6 {
			process = fields[6]
		}
	}

	if port == 0 {
		return nil
	}

	return &PortInfo{
		Port:     port,
		Protocol: protocol,
		Address:  address,
		State:    state,
		Process:  process,
	}
}

func (m *Manager) saveHistory() error {
	dir := filepath.Dir(m.historyFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create history directory %s: %w\n\nPlease ensure the directory exists and has correct permissions:\n  sudo mkdir -p %s\n  sudo chown -R $(whoami):$(whoami) %s", dir, err, dir, dir)
	}

	data, err := json.MarshalIndent(m.history, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal history: %w", err)
	}

	if err := os.WriteFile(m.historyFile, data, 0600); err != nil {
		return fmt.Errorf("write history file: %w", err)
	}

	return nil
}

func (m *Manager) loadHistory() error {
	data, err := os.ReadFile(m.historyFile)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &m.history); err != nil {
		return fmt.Errorf("unmarshal history: %w", err)
	}

	return nil
}
