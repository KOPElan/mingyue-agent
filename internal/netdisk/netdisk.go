package netdisk

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Protocol represents the network filesystem protocol
type Protocol string

const (
	ProtocolCIFS Protocol = "cifs"
	ProtocolNFS  Protocol = "nfs"
)

// Share represents a network share
type Share struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Protocol    Protocol          `json:"protocol"`
	Host        string            `json:"host"`
	Path        string            `json:"path"`
	MountPoint  string            `json:"mount_point"`
	Username    string            `json:"username,omitempty"`
	Password    string            `json:"-"` // Never expose in JSON
	Options     map[string]string `json:"options"`
	AutoMount   bool              `json:"auto_mount"`
	Mounted     bool              `json:"mounted"`
	LastChecked time.Time         `json:"last_checked"`
	Healthy     bool              `json:"healthy"`
}

// Manager handles network disk operations
type Manager struct {
	shares             map[string]*Share
	allowedHosts       []string
	allowedMountPoints []string
	encryptionKey      []byte
	stateFile          string
	mu                 sync.RWMutex
	monitorInterval    time.Duration
	stopMonitor        chan struct{}
}

// Config represents network disk manager configuration
type Config struct {
	AllowedHosts       []string
	AllowedMountPoints []string
	EncryptionKey      string
	StateFile          string
	MonitorInterval    time.Duration
}

// New creates a new network disk manager
func New(cfg *Config) (*Manager, error) {
	if cfg.EncryptionKey == "" {
		return nil, fmt.Errorf("encryption key is required")
	}

	key := []byte(cfg.EncryptionKey)
	if len(key) < 32 {
		// Pad key to 32 bytes for AES-256
		padded := make([]byte, 32)
		copy(padded, key)
		key = padded
	} else {
		key = key[:32]
	}

	monitorInterval := cfg.MonitorInterval
	if monitorInterval == 0 {
		monitorInterval = 1 * time.Minute
	}

	stateFile := cfg.StateFile
	if stateFile == "" {
		stateFile = "/var/lib/mingyue-agent/netdisk-state.json"
	}

	m := &Manager{
		shares:             make(map[string]*Share),
		allowedHosts:       cfg.AllowedHosts,
		allowedMountPoints: cfg.AllowedMountPoints,
		encryptionKey:      key,
		stateFile:          stateFile,
		monitorInterval:    monitorInterval,
		stopMonitor:        make(chan struct{}),
	}

	// Load persisted state
	if err := m.loadState(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load state: %w", err)
	}

	// Start health monitor
	go m.healthMonitor()

	return m, nil
}

// AddShare adds a new network share configuration
func (m *Manager) AddShare(share *Share) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if share.ID == "" {
		share.ID = fmt.Sprintf("%s-%s-%d", share.Protocol, share.Host, time.Now().Unix())
	}

	// Validate host whitelist
	if len(m.allowedHosts) > 0 {
		allowed := false
		for _, host := range m.allowedHosts {
			if host == share.Host || host == "*" {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("host %s is not in allowed list", share.Host)
		}
	}

	// Validate mount point
	if !m.isAllowedMountPoint(share.MountPoint) {
		return fmt.Errorf("mount point %s is not allowed", share.MountPoint)
	}

	// Encrypt password if provided
	if share.Password != "" {
		encrypted, err := m.encrypt(share.Password)
		if err != nil {
			return fmt.Errorf("encrypt password: %w", err)
		}
		share.Password = encrypted
	}

	m.shares[share.ID] = share
	return m.saveState()
}

// RemoveShare removes a network share
func (m *Manager) RemoveShare(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	share, exists := m.shares[id]
	if !exists {
		return fmt.Errorf("share %s not found", id)
	}

	// Unmount if mounted
	if share.Mounted {
		if err := m.unmountShare(share); err != nil {
			return fmt.Errorf("unmount share: %w", err)
		}
	}

	delete(m.shares, id)
	return m.saveState()
}

// ListShares returns all configured shares
func (m *Manager) ListShares() []*Share {
	m.mu.RLock()
	defer m.mu.RUnlock()

	shares := make([]*Share, 0, len(m.shares))
	for _, share := range m.shares {
		// Create a copy without password
		shareCopy := *share
		shareCopy.Password = "" // Never expose password
		shares = append(shares, &shareCopy)
	}
	return shares
}

// Mount mounts a network share
func (m *Manager) Mount(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	share, exists := m.shares[id]
	if !exists {
		return fmt.Errorf("share %s not found", id)
	}

	if share.Mounted {
		return fmt.Errorf("share %s is already mounted", id)
	}

	if err := m.mountShare(share); err != nil {
		return err
	}

	share.Mounted = true
	share.Healthy = true
	share.LastChecked = time.Now()
	return m.saveState()
}

// Unmount unmounts a network share
func (m *Manager) Unmount(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	share, exists := m.shares[id]
	if !exists {
		return fmt.Errorf("share %s not found", id)
	}

	if !share.Mounted {
		return fmt.Errorf("share %s is not mounted", id)
	}

	if err := m.unmountShare(share); err != nil {
		return err
	}

	share.Mounted = false
	share.Healthy = false
	return m.saveState()
}

// GetShareStatus returns the status of a specific share
func (m *Manager) GetShareStatus(id string) (*Share, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	share, exists := m.shares[id]
	if !exists {
		return nil, fmt.Errorf("share %s not found", id)
	}

	// Create a copy without password
	shareCopy := *share
	shareCopy.Password = ""
	return &shareCopy, nil
}

// Stop stops the network disk manager
func (m *Manager) Stop() {
	close(m.stopMonitor)
}

// Private methods

func (m *Manager) isAllowedMountPoint(mountPoint string) bool {
	if len(m.allowedMountPoints) == 0 {
		return false
	}

	absPath, err := filepath.Abs(mountPoint)
	if err != nil {
		return false
	}

	for _, allowed := range m.allowedMountPoints {
		absAllowed, err := filepath.Abs(allowed)
		if err != nil {
			continue
		}

		rel, err := filepath.Rel(absAllowed, absPath)
		if err != nil {
			continue
		}

		if !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel) {
			return true
		}
	}
	return false
}

func (m *Manager) mountShare(share *Share) error {
	// Create mount point if it doesn't exist
	if err := os.MkdirAll(share.MountPoint, 0755); err != nil {
		return fmt.Errorf("create mount point: %w", err)
	}

	var cmd *exec.Cmd
	switch share.Protocol {
	case ProtocolCIFS:
		cmd = m.buildCIFSMountCommand(share)
	case ProtocolNFS:
		cmd = m.buildNFSMountCommand(share)
	default:
		return fmt.Errorf("unsupported protocol: %s", share.Protocol)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mount failed: %w, output: %s", err, string(output))
	}

	return nil
}

func (m *Manager) unmountShare(share *Share) error {
	cmd := exec.Command("umount", share.MountPoint)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try force unmount if normal unmount fails
		cmd = exec.Command("umount", "-f", share.MountPoint)
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("unmount failed: %w, output: %s", err, string(output))
		}
	}
	return nil
}

func (m *Manager) buildCIFSMountCommand(share *Share) *exec.Cmd {
	source := fmt.Sprintf("//%s%s", share.Host, share.Path)

	opts := []string{}
	if share.Username != "" {
		opts = append(opts, fmt.Sprintf("username=%s", share.Username))
	}

	if share.Password != "" {
		// Decrypt password
		password, err := m.decrypt(share.Password)
		if err == nil {
			opts = append(opts, fmt.Sprintf("password=%s", password))
		}
	}

	// Add custom options
	for key, value := range share.Options {
		opts = append(opts, fmt.Sprintf("%s=%s", key, value))
	}

	args := []string{"-t", "cifs"}
	if len(opts) > 0 {
		args = append(args, "-o", strings.Join(opts, ","))
	}
	args = append(args, source, share.MountPoint)

	return exec.Command("mount", args...)
}

func (m *Manager) buildNFSMountCommand(share *Share) *exec.Cmd {
	source := fmt.Sprintf("%s:%s", share.Host, share.Path)

	opts := []string{}
	// Add custom options
	for key, value := range share.Options {
		if value == "" {
			opts = append(opts, key)
		} else {
			opts = append(opts, fmt.Sprintf("%s=%s", key, value))
		}
	}

	args := []string{"-t", "nfs"}
	if len(opts) > 0 {
		args = append(args, "-o", strings.Join(opts, ","))
	}
	args = append(args, source, share.MountPoint)

	return exec.Command("mount", args...)
}

func (m *Manager) healthMonitor() {
	ticker := time.NewTicker(m.monitorInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.checkAllShares()
		case <-m.stopMonitor:
			return
		}
	}
}

func (m *Manager) checkAllShares() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, share := range m.shares {
		if !share.Mounted {
			continue
		}

		// Check if mount point is still accessible
		_, err := os.Stat(share.MountPoint)
		healthy := err == nil

		// Try to remount if unhealthy and auto-mount is enabled
		if !healthy && share.AutoMount {
			if err := m.unmountShare(share); err == nil {
				time.Sleep(1 * time.Second)
				if err := m.mountShare(share); err == nil {
					healthy = true
				}
			}
		}

		share.Healthy = healthy
		share.LastChecked = time.Now()
		if !healthy {
			share.Mounted = false
		}
	}

	m.saveState()
}

func (m *Manager) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(m.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (m *Manager) decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(m.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func (m *Manager) saveState() error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(m.stateFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create state directory %s: %w\n\nPlease ensure the directory exists and has correct permissions:\n  sudo mkdir -p %s\n  sudo chown -R $(whoami):$(whoami) %s", dir, err, dir, dir)
	}

	data, err := json.MarshalIndent(m.shares, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	if err := os.WriteFile(m.stateFile, data, 0600); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}

	return nil
}

func (m *Manager) loadState() error {
	data, err := os.ReadFile(m.stateFile)
	if err != nil {
		return err
	}

	var shares map[string]*Share
	if err := json.Unmarshal(data, &shares); err != nil {
		return fmt.Errorf("unmarshal state: %w", err)
	}

	m.shares = shares

	// Mark all shares as unmounted on startup
	for _, share := range m.shares {
		share.Mounted = false
		share.Healthy = false
	}

	return nil
}
