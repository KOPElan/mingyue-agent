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

// verifyDirectories checks if all required directories exist and have correct permissions
func verifyDirectories(cfg *config.Config) error {
	requiredDirs := []struct {
		path        string
		description string
		owner       string // "root" or "mingyue-agent"
	}{
		{filepath.Dir(cfg.NetDisk.StateFile), "network disk state", "mingyue-agent"},
		{filepath.Dir(cfg.Network.HistoryFile), "network history", "mingyue-agent"},
		{cfg.ShareMgr.BackupDir, "share backups", "mingyue-agent"},
		{filepath.Dir(cfg.ShareMgr.StateFile), "share state", "mingyue-agent"},
	}

	var missingDirs []string
	var permissionErrors []string

	for _, dir := range requiredDirs {
		// Check if directory exists
		info, err := os.Stat(dir.path)
		if err != nil {
			if os.IsNotExist(err) {
				missingDirs = append(missingDirs, dir.path)
			} else {
				permissionErrors = append(permissionErrors, fmt.Sprintf("cannot access %s: %v", dir.path, err))
			}
			continue
		}

		// Check if it's a directory
		if !info.IsDir() {
			permissionErrors = append(permissionErrors, fmt.Sprintf("%s exists but is not a directory", dir.path))
			continue
		}
	}

	if len(missingDirs) > 0 || len(permissionErrors) > 0 {
		var msg strings.Builder
		msg.WriteString("Required directories are not properly configured:\n")

		if len(missingDirs) > 0 {
			msg.WriteString("\nMissing directories:\n")
			for _, dir := range missingDirs {
				msg.WriteString(fmt.Sprintf("  - %s\n", dir))
			}
		}

		if len(permissionErrors) > 0 {
			msg.WriteString("\nPermission/Access errors:\n")
			for _, errMsg := range permissionErrors {
				msg.WriteString(fmt.Sprintf("  - %s\n", errMsg))
			}
		}

		msg.WriteString("\n")
		msg.WriteString("To fix these issues, run the following commands:\n\n")
		msg.WriteString("  # Create directories if they don't exist\n")
		msg.WriteString("  sudo mkdir -p /var/lib/mingyue-agent/share-backups\n")
		msg.WriteString("  sudo mkdir -p /var/log/mingyue-agent\n")
		msg.WriteString("  sudo mkdir -p /var/run/mingyue-agent\n\n")
		msg.WriteString("  # Set correct ownership\n")
		msg.WriteString("  sudo chown -R mingyue-agent:mingyue-agent /var/lib/mingyue-agent\n")
		msg.WriteString("  sudo chown -R mingyue-agent:mingyue-agent /var/log/mingyue-agent\n")
		msg.WriteString("  sudo chown -R mingyue-agent:mingyue-agent /var/run/mingyue-agent\n\n")
		msg.WriteString("  # Set correct permissions\n")
		msg.WriteString("  sudo chmod -R 755 /var/lib/mingyue-agent\n")
		msg.WriteString("  sudo chmod -R 755 /var/log/mingyue-agent\n")
		msg.WriteString("  sudo chmod -R 755 /var/run/mingyue-agent\n")

		return fmt.Errorf(msg.String())
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
