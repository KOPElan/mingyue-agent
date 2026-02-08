package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/KOPElan/mingyue-agent/internal/auth"
	"github.com/KOPElan/mingyue-agent/internal/config"
	"github.com/KOPElan/mingyue-agent/internal/diskmanager"
	"github.com/KOPElan/mingyue-agent/internal/filemanager"
	"github.com/KOPElan/mingyue-agent/internal/indexer"
	"github.com/KOPElan/mingyue-agent/internal/monitor"
	"github.com/KOPElan/mingyue-agent/internal/scheduler"
)

func loadLocalConfig() (*config.Config, string, error) {
	dataDir, err := resolveLocalDataDir()
	if err != nil {
		return nil, "", err
	}

	cfgPath := resolveConfigPath(localConfigPath)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, "", err
	}

	applyLocalOverrides(cfg, dataDir)
	return cfg, dataDir, nil
}

func resolveLocalDataDir() (string, error) {
	if localDataDir != "" {
		return localDataDir, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		return filepath.Join(os.TempDir(), "mingyue-agent"), nil
	}

	return filepath.Join(homeDir, ".mingyue-agent"), nil
}

func applyLocalOverrides(cfg *config.Config, dataDir string) {
	if cfg == nil {
		return
	}

	cfg.NetDisk.StateFile = filepath.Join(dataDir, "netdisk-state.json")
	cfg.Network.HistoryFile = filepath.Join(dataDir, "network-history.json")
	cfg.ShareMgr.BackupDir = filepath.Join(dataDir, "share-backups")
	cfg.ShareMgr.StateFile = filepath.Join(dataDir, "share-state.json")
	cfg.Server.UDSPath = filepath.Join(dataDir, "agent.sock")
	cfg.Audit.LogPath = filepath.Join(dataDir, "audit.log")

	cwd, err := os.Getwd()
	if err == nil && cwd != "" {
		cfg.Security.AllowedPaths = append(cfg.Security.AllowedPaths, cwd)
	}
	cfg.Security.AllowedPaths = append(cfg.Security.AllowedPaths, dataDir)
	cfg.Security.AllowedPaths = uniqueStrings(cfg.Security.AllowedPaths)
}

func ensureLocalDataDir(dataDir string) error {
	if dataDir == "" {
		return fmt.Errorf("local data directory is empty")
	}
	return os.MkdirAll(dataDir, 0755)
}

func localUser() string {
	if apiUser != "" {
		return apiUser
	}
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	return "local"
}

func localFileManager(cfg *config.Config) *filemanager.Manager {
	return filemanager.New(cfg.Security.AllowedPaths, nil)
}

func localDiskManager(cfg *config.Config) *diskmanager.Manager {
	return diskmanager.New(cfg.Security.AllowedPaths)
}

func localMonitor() *monitor.Monitor {
	return monitor.New()
}

func localIndexer(dataDir string) (*indexer.Indexer, error) {
	if err := ensureLocalDataDir(dataDir); err != nil {
		return nil, err
	}
	return indexer.New(filepath.Join(dataDir, "indexer.db"))
}

func localScheduler(dataDir string) (*scheduler.Scheduler, error) {
	if err := ensureLocalDataDir(dataDir); err != nil {
		return nil, err
	}
	return scheduler.New(scheduler.Config{
		DBPath: filepath.Join(dataDir, "scheduler.db"),
	})
}

func localAuthManager(dataDir string) (*auth.AuthManager, error) {
	if err := ensureLocalDataDir(dataDir); err != nil {
		return nil, err
	}
	return auth.New(auth.Config{
		DBPath: filepath.Join(dataDir, "auth.db"),
	})
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
