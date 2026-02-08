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
	type dirCheck struct {
		path        string
		description string
	}

	logDir := agentLogDir(cfg)
	requiredDirs := []dirCheck{
		{filepath.Dir(cfg.NetDisk.StateFile), "network disk state"},
		{filepath.Dir(cfg.Network.HistoryFile), "network history"},
		{cfg.ShareMgr.BackupDir, "share backups"},
		{filepath.Dir(cfg.ShareMgr.StateFile), "share state"},
		{filepath.Dir(cfg.Server.UDSPath), "unix socket"},
		{logDir, "agent log"},
	}

	if cfg.Audit.Enabled && cfg.Audit.LogPath != "" {
		requiredDirs = append(requiredDirs, dirCheck{
			path:        filepath.Dir(cfg.Audit.LogPath),
			description: "audit log",
		})
	}

	var errors []string
	for _, dir := range requiredDirs {
		if err := ensureWritableDir(dir.path); err != nil {
			errors = append(errors, fmt.Sprintf("  - %s: %v", dir.description, err))
		}
	}

	if cfg.Audit.Enabled && cfg.Audit.LogPath != "" {
		if err := ensureWritableFile(cfg.Audit.LogPath); err != nil {
			errors = append(errors, fmt.Sprintf("  - audit log file: %v", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("Required directories are not accessible:\n%s\n\nFix by running:\n  sudo mingyue-agent fix-permissions --config /etc/mingyue-agent/config.yaml", strings.Join(errors, "\n"))
	}

	return nil
}

func ensureWritableDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", path)
		}
		return fmt.Errorf("cannot access directory: %s (%v)", path, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path exists but is not a directory: %s", path)
	}

	testFile := filepath.Join(path, ".mingyue-agent-write-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("directory is not writable: %s (%v)", path, err)
	}

	if err := os.Remove(testFile); err != nil {
		return fmt.Errorf("remove write test file: %s (%v)", path, err)
	}

	return nil
}

func ensureWritableFile(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("cannot open log file: %s (%v)", path, err)
	}
	return file.Close()
}

func agentLogDir(cfg *config.Config) string {
	if cfg.Audit.Enabled && cfg.Audit.LogPath != "" {
		return filepath.Dir(cfg.Audit.LogPath)
	}
	return "/var/log/mingyue-agent"
}

func New(cfg *config.Config) (*Daemon, error) {
	// Verify all required directories before proceeding
	if err := verifyDirectories(cfg); err != nil {
		return nil, err
	}

	logDir := agentLogDir(cfg)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("create log directory %s: %w", logDir, err)
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
