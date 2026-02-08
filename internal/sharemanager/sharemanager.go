package sharemanager

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"
)

// ShareType represents the share protocol type
type ShareType string

const (
	ShareTypeSamba ShareType = "samba"
	ShareTypeNFS   ShareType = "nfs"
)

// AccessMode represents share access mode
type AccessMode string

const (
	AccessModeReadOnly  AccessMode = "ro"
	AccessModeReadWrite AccessMode = "rw"
)

// Share represents a shared directory configuration
type Share struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        ShareType         `json:"type"`
	Path        string            `json:"path"`
	Description string            `json:"description"`
	Users       []string          `json:"users"`
	Groups      []string          `json:"groups"`
	AccessMode  AccessMode        `json:"access_mode"`
	Options     map[string]string `json:"options"`
	Enabled     bool              `json:"enabled"`
	Healthy     bool              `json:"healthy"`
	LastChecked time.Time         `json:"last_checked"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Manager handles share management operations
type Manager struct {
	shares          map[string]*Share
	allowedPaths    []string
	sambaConfig     string
	nfsConfig       string
	backupDir       string
	stateFile       string
	mu              sync.RWMutex
	monitorInterval time.Duration
	stopMonitor     chan struct{}
}

// Config represents share manager configuration
type Config struct {
	AllowedPaths    []string
	SambaConfig     string
	NFSConfig       string
	BackupDir       string
	StateFile       string
	MonitorInterval time.Duration
}

// New creates a new share manager
func New(cfg *Config) (*Manager, error) {
	sambaConfig := cfg.SambaConfig
	if sambaConfig == "" {
		sambaConfig = "/etc/samba/smb.conf"
	}

	nfsConfig := cfg.NFSConfig
	if nfsConfig == "" {
		nfsConfig = "/etc/exports"
	}

	backupDir := cfg.BackupDir
	if backupDir == "" {
		backupDir = "/var/lib/mingyue-agent/share-backups"
	}

	stateFile := cfg.StateFile
	if stateFile == "" {
		stateFile = "/var/lib/mingyue-agent/share-state.json"
	}

	monitorInterval := cfg.MonitorInterval
	if monitorInterval == 0 {
		monitorInterval = 1 * time.Minute
	}

	// Create backup directory with fallback to temp dir on read-only filesystem
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		backupDir = filepath.Join(os.TempDir(), "mingyue-agent", "share-backups")
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			return nil, fmt.Errorf("create backup directory: %w", err)
		}
		// Also update state file to use temp directory
		stateFile = filepath.Join(os.TempDir(), "mingyue-agent", "share-state.json")
	}

	m := &Manager{
		shares:          make(map[string]*Share),
		allowedPaths:    cfg.AllowedPaths,
		sambaConfig:     sambaConfig,
		nfsConfig:       nfsConfig,
		backupDir:       backupDir,
		stateFile:       stateFile,
		monitorInterval: monitorInterval,
		stopMonitor:     make(chan struct{}),
	}

	// Load persisted state
	if err := m.loadState(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load state: %w", err)
	}

	// Start health monitor
	go m.healthMonitor()

	return m, nil
}

// AddShare adds a new share
func (m *Manager) AddShare(share *Share) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if share.ID == "" {
		share.ID = fmt.Sprintf("%s-%d", share.Name, time.Now().Unix())
	}

	// Validate path is in allowed list
	if !m.isAllowedPath(share.Path) {
		return fmt.Errorf("path %s is not in allowed paths", share.Path)
	}

	// Ensure path exists
	if _, err := os.Stat(share.Path); err != nil {
		return fmt.Errorf("share path does not exist: %w", err)
	}

	now := time.Now()
	share.CreatedAt = now
	share.UpdatedAt = now
	share.Enabled = true

	m.shares[share.ID] = share

	// Apply configuration
	if err := m.applyConfiguration(); err != nil {
		delete(m.shares, share.ID)
		return fmt.Errorf("apply configuration: %w", err)
	}

	return m.saveState()
}

// UpdateShare updates an existing share
func (m *Manager) UpdateShare(id string, updates *Share) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	share, exists := m.shares[id]
	if !exists {
		return fmt.Errorf("share %s not found", id)
	}

	// Validate path if changed
	if updates.Path != "" && updates.Path != share.Path {
		if !m.isAllowedPath(updates.Path) {
			return fmt.Errorf("path %s is not in allowed paths", updates.Path)
		}
		share.Path = updates.Path
	}

	// Update fields
	if updates.Name != "" {
		share.Name = updates.Name
	}
	if updates.Description != "" {
		share.Description = updates.Description
	}
	if len(updates.Users) > 0 {
		share.Users = updates.Users
	}
	if len(updates.Groups) > 0 {
		share.Groups = updates.Groups
	}
	if updates.AccessMode != "" {
		share.AccessMode = updates.AccessMode
	}
	if len(updates.Options) > 0 {
		share.Options = updates.Options
	}

	share.UpdatedAt = time.Now()

	// Apply configuration
	if err := m.applyConfiguration(); err != nil {
		return fmt.Errorf("apply configuration: %w", err)
	}

	return m.saveState()
}

// RemoveShare removes a share
func (m *Manager) RemoveShare(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.shares[id]; !exists {
		return fmt.Errorf("share %s not found", id)
	}

	delete(m.shares, id)

	// Apply configuration
	if err := m.applyConfiguration(); err != nil {
		return fmt.Errorf("apply configuration: %w", err)
	}

	return m.saveState()
}

// ListShares returns all shares
func (m *Manager) ListShares() []*Share {
	m.mu.RLock()
	defer m.mu.RUnlock()

	shares := make([]*Share, 0, len(m.shares))
	for _, share := range m.shares {
		shareCopy := *share
		shares = append(shares, &shareCopy)
	}
	return shares
}

// GetShare returns a specific share
func (m *Manager) GetShare(id string) (*Share, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	share, exists := m.shares[id]
	if !exists {
		return nil, fmt.Errorf("share %s not found", id)
	}

	shareCopy := *share
	return &shareCopy, nil
}

// EnableShare enables a share
func (m *Manager) EnableShare(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	share, exists := m.shares[id]
	if !exists {
		return fmt.Errorf("share %s not found", id)
	}

	share.Enabled = true
	share.UpdatedAt = time.Now()

	if err := m.applyConfiguration(); err != nil {
		return fmt.Errorf("apply configuration: %w", err)
	}

	return m.saveState()
}

// DisableShare disables a share
func (m *Manager) DisableShare(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	share, exists := m.shares[id]
	if !exists {
		return fmt.Errorf("share %s not found", id)
	}

	share.Enabled = false
	share.UpdatedAt = time.Now()

	if err := m.applyConfiguration(); err != nil {
		return fmt.Errorf("apply configuration: %w", err)
	}

	return m.saveState()
}

// RollbackConfig rolls back to a previous configuration
func (m *Manager) RollbackConfig(timestamp time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	backupFile := filepath.Join(m.backupDir, fmt.Sprintf("smb.conf.%d", timestamp.Unix()))
	if _, err := os.Stat(backupFile); err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}

	// Restore samba config
	if err := m.restoreConfig(backupFile, m.sambaConfig); err != nil {
		return fmt.Errorf("restore samba config: %w", err)
	}

	// Reload samba
	if err := m.reloadSamba(); err != nil {
		return fmt.Errorf("reload samba: %w", err)
	}

	return nil
}

// Stop stops the share manager
func (m *Manager) Stop() {
	close(m.stopMonitor)
}

// Private methods

func (m *Manager) isAllowedPath(path string) bool {
	if len(m.allowedPaths) == 0 {
		return false
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	for _, allowed := range m.allowedPaths {
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

func (m *Manager) applyConfiguration() error {
	// Backup current configurations
	if err := m.backupConfigs(); err != nil {
		return fmt.Errorf("backup configs: %w", err)
	}

	// Generate and apply Samba configuration
	sambaShares := []*Share{}
	nfsShares := []*Share{}

	for _, share := range m.shares {
		if !share.Enabled {
			continue
		}
		if share.Type == ShareTypeSamba {
			sambaShares = append(sambaShares, share)
		} else if share.Type == ShareTypeNFS {
			nfsShares = append(nfsShares, share)
		}
	}

	// Generate Samba config
	if len(sambaShares) > 0 {
		if err := m.generateSambaConfig(sambaShares); err != nil {
			return fmt.Errorf("generate samba config: %w", err)
		}

		// Test configuration
		if err := m.testSambaConfig(); err != nil {
			// Rollback on error
			m.restoreLatestBackup()
			return fmt.Errorf("invalid samba config: %w", err)
		}

		// Reload Samba
		if err := m.reloadSamba(); err != nil {
			return fmt.Errorf("reload samba: %w", err)
		}
	}

	// Generate NFS config
	if len(nfsShares) > 0 {
		if err := m.generateNFSConfig(nfsShares); err != nil {
			return fmt.Errorf("generate nfs config: %w", err)
		}

		// Reload NFS exports
		if err := m.reloadNFS(); err != nil {
			return fmt.Errorf("reload nfs: %w", err)
		}
	}

	return nil
}

func (m *Manager) generateSambaConfig(shares []*Share) error {
	tmpl := `# Generated by mingyue-agent at {{ .Timestamp }}
[global]
   workgroup = WORKGROUP
   server string = Mingyue Agent Share
   security = user
   map to guest = Bad User
   log file = /var/log/samba/log.%m
   max log size = 50

{{ range .Shares }}
[{{ .Name }}]
   path = {{ .Path }}
   {{ if .Description }}comment = {{ .Description }}{{ end }}
   {{ if eq .AccessMode "ro" }}read only = yes{{ else }}read only = no{{ end }}
   browseable = yes
   {{ if .Users }}valid users = {{ join .Users " " }}{{ end }}
   create mask = 0664
   directory mask = 0775
{{ range $key, $value := .Options }}   {{ $key }} = {{ $value }}
{{ end }}
{{ end }}
`

	t, err := template.New("samba").Funcs(template.FuncMap{
		"join": strings.Join,
	}).Parse(tmpl)
	if err != nil {
		return fmt.Errorf("parse template: %w", err)
	}

	file, err := os.Create(m.sambaConfig + ".new")
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	defer file.Close()

	data := struct {
		Timestamp time.Time
		Shares    []*Share
	}{
		Timestamp: time.Now(),
		Shares:    shares,
	}

	if err := t.Execute(file, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	// Move new config to actual location
	if err := os.Rename(m.sambaConfig+".new", m.sambaConfig); err != nil {
		return fmt.Errorf("move config: %w", err)
	}

	return nil
}

func (m *Manager) generateNFSConfig(shares []*Share) error {
	content := "# Generated by mingyue-agent\n"
	for _, share := range shares {
		line := fmt.Sprintf("%s *(", share.Path)
		if share.AccessMode == AccessModeReadOnly {
			line += "ro"
		} else {
			line += "rw"
		}
		line += ",sync,no_subtree_check"
		for key, value := range share.Options {
			if value == "" {
				line += "," + key
			} else {
				line += "," + key + "=" + value
			}
		}
		line += ")\n"
		content += line
	}

	if err := os.WriteFile(m.nfsConfig, []byte(content), 0644); err != nil {
		return fmt.Errorf("write nfs config: %w", err)
	}

	return nil
}

func (m *Manager) testSambaConfig() error {
	cmd := exec.Command("testparm", "-s", m.sambaConfig)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("testparm failed: %w, output: %s", err, string(output))
	}
	return nil
}

func (m *Manager) reloadSamba() error {
	// Try systemctl reload first
	cmd := exec.Command("systemctl", "reload", "smbd")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback to service command
		cmd = exec.Command("service", "smbd", "reload")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("reload smbd: %w, output: %s", err, string(output))
		}
	}
	return nil
}

func (m *Manager) reloadNFS() error {
	cmd := exec.Command("exportfs", "-ra")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("exportfs: %w, output: %s", err, string(output))
	}
	return nil
}

func (m *Manager) backupConfigs() error {
	timestamp := time.Now().Unix()

	// Backup Samba config
	if _, err := os.Stat(m.sambaConfig); err == nil {
		backupFile := filepath.Join(m.backupDir, fmt.Sprintf("smb.conf.%d", timestamp))
		if err := m.copyFile(m.sambaConfig, backupFile); err != nil {
			return fmt.Errorf("backup samba config: %w", err)
		}
	}

	// Backup NFS config
	if _, err := os.Stat(m.nfsConfig); err == nil {
		backupFile := filepath.Join(m.backupDir, fmt.Sprintf("exports.%d", timestamp))
		if err := m.copyFile(m.nfsConfig, backupFile); err != nil {
			return fmt.Errorf("backup nfs config: %w", err)
		}
	}

	// Clean old backups (keep last 10)
	m.cleanOldBackups(10)

	return nil
}

func (m *Manager) restoreConfig(backupFile, targetFile string) error {
	return m.copyFile(backupFile, targetFile)
}

func (m *Manager) restoreLatestBackup() error {
	// Find latest backup
	files, err := os.ReadDir(m.backupDir)
	if err != nil {
		return err
	}

	var latestSamba string
	for _, file := range files {
		if strings.HasPrefix(file.Name(), "smb.conf.") && file.Name() > latestSamba {
			latestSamba = file.Name()
		}
	}

	if latestSamba != "" {
		backupFile := filepath.Join(m.backupDir, latestSamba)
		return m.restoreConfig(backupFile, m.sambaConfig)
	}

	return fmt.Errorf("no backup found")
}

func (m *Manager) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

func (m *Manager) cleanOldBackups(keep int) {
	files, err := os.ReadDir(m.backupDir)
	if err != nil {
		return
	}

	if len(files) <= keep {
		return
	}

	// Sort by name (timestamp-based) and remove oldest
	for i := 0; i < len(files)-keep; i++ {
		os.Remove(filepath.Join(m.backupDir, files[i].Name()))
	}
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
		if !share.Enabled {
			continue
		}

		// Check if path is still accessible
		_, err := os.Stat(share.Path)
		share.Healthy = err == nil
		share.LastChecked = time.Now()
	}

	m.saveState()
}

func (m *Manager) saveState() error {
	dir := filepath.Dir(m.stateFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create state directory: %w", err)
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
	return nil
}
