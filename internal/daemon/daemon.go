package daemon

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
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

func New(cfg *config.Config) (*Daemon, error) {
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
