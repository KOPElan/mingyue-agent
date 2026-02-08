package daemon

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/KOPElan/mingyue-agent/internal/audit"
	"github.com/KOPElan/mingyue-agent/internal/config"
	"github.com/KOPElan/mingyue-agent/internal/server"
)

type Daemon struct {
	config *config.Config
	audit  *audit.Logger
	server *server.Server
	logDir string
}

// verifyDirectories checks if all required directories exist and are writable
func verifyDirectories(cfg *config.Config) error {
	requiredDirs := []struct {
		path        string
		description string
	}{
		{filepath.Dir(cfg.NetDisk.StateFile), "network disk state"},
		{filepath.Dir(cfg.Network.HistoryFile), "network history"},
		{cfg.ShareMgr.BackupDir, "share backups"},
		{filepath.Dir(cfg.ShareMgr.StateFile), "share state"},
		{cfg.Network.ConfigDir, "network config"},
	}

	var errors []string
	for _, dir := range requiredDirs {
		// Check if directory exists
		info, err := os.Stat(dir.path)
		if err != nil {
			if os.IsNotExist(err) {
				errors = append(errors, fmt.Sprintf("  - %s: directory does not exist: %s", dir.description, dir.path))
			} else {
				errors = append(errors, fmt.Sprintf("  - %s: cannot access directory: %s (%v)", dir.description, dir.path, err))
			}
			continue
		}

		// Check if it's a directory
		if !info.IsDir() {
			errors = append(errors, fmt.Sprintf("  - %s: path exists but is not a directory: %s", dir.description, dir.path))
			continue
		}

		// Check if writable (try creating a temp file)
		testFile := filepath.Join(dir.path, ".mingyue-agent-write-test")
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			errors = append(errors, fmt.Sprintf("  - %s: directory is not writable: %s (%v)", dir.description, dir.path, err))
		} else {
			os.Remove(testFile)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("Required directories are not accessible:\n%s\n\nPlease run the following commands to create and configure directories:\n  sudo mkdir -p /var/lib/mingyue-agent/share-backups\n  sudo mkdir -p /etc/mingyue-agent/network\n  sudo chown -R mingyue-agent:mingyue-agent /var/lib/mingyue-agent\n  sudo chmod -R 755 /var/lib/mingyue-agent",
			strings.Join(errors, "\n"))
	}

	return nil
}

func New(cfg *config.Config) (*Daemon, error) {
	// Verify all required directories before proceeding
	if err := verifyDirectories(cfg); err != nil {
		return nil, err
	}

	logDir := "/var/log/mingyue-agent"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		logDir = filepath.Join(os.TempDir(), "mingyue-agent")
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("create log directory: %w", err)
		}
	}

	auditLogger, err := audit.New(
		cfg.Audit.LogPath,
		cfg.Audit.RemotePush,
		cfg.Audit.RemoteURL,
		cfg.Audit.Enabled,
	)
	if err != nil {
		return nil, fmt.Errorf("create audit logger: %w", err)
	}

	srv, err := server.New(cfg, auditLogger)
	if err != nil {
		return nil, fmt.Errorf("create server: %w", err)
	}

	return &Daemon{
		config: cfg,
		audit:  auditLogger,
		server: srv,
		logDir: logDir,
	}, nil
}

func (d *Daemon) Start(ctx context.Context) error {
	logFile := filepath.Join(d.logDir, "agent.log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Warning: failed to open log file: %v", err)
	} else {
		defer f.Close()
		log.SetOutput(f)
	}

	startEntry := &audit.Entry{
		Timestamp: time.Now(),
		User:      "system",
		Action:    "daemon_start",
		Resource:  "agent",
		Result:    "success",
		Details: map[string]interface{}{
			"pid": os.Getpid(),
		},
	}

	if err := d.audit.Log(ctx, startEntry); err != nil {
		log.Printf("Warning: failed to log audit entry: %v", err)
	}

	log.Printf("Mingyue Agent starting (PID: %d)", os.Getpid())
	log.Printf("HTTP server on %s:%d", d.config.Server.ListenAddr, d.config.Server.HTTPPort)
	log.Printf("gRPC server on %s:%d", d.config.Server.ListenAddr, d.config.Server.GRPCPort)

	if err := d.server.Start(ctx); err != nil {
		return fmt.Errorf("start server: %w", err)
	}

	return nil
}

func (d *Daemon) Shutdown(ctx context.Context) error {
	log.Println("Mingyue Agent shutting down...")

	shutdownEntry := &audit.Entry{
		Timestamp: time.Now(),
		User:      "system",
		Action:    "daemon_shutdown",
		Resource:  "agent",
		Result:    "success",
	}

	if err := d.audit.Log(ctx, shutdownEntry); err != nil {
		log.Printf("Warning: failed to log audit entry: %v", err)
	}

	if err := d.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	if err := d.audit.Close(); err != nil {
		return fmt.Errorf("close audit logger: %w", err)
	}

	log.Println("Mingyue Agent stopped")
	return nil
}
