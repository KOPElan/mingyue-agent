package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/KOPElan/mingyue-agent/internal/config"
	"github.com/spf13/cobra"
)

func fixPermissionsCmd() *cobra.Command {
	var configFile string
	var userName string
	var groupName string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "fix-permissions",
		Short: "Fix required directory and file permissions",
		Long: `Fix required directory and file permissions for Mingyue Agent.

This command should be run as root. It will create required directories,
set ownership, and ensure log files are writable by the service user.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if os.Geteuid() != 0 {
				return fmt.Errorf("fix-permissions must be run as root")
			}

			resolvedConfig := resolveConfigPath(configFile)
			cfg, err := config.Load(resolvedConfig)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if userName == "" {
				userName = "mingyue-agent"
			}
			if groupName == "" {
				groupName = userName
			}

			uid, gid, err := resolveUserGroup(userName, groupName)
			if err != nil {
				return err
			}

			paths := requiredPaths(cfg)
			files := requiredFiles(cfg)

			for _, dir := range paths {
				if dryRun {
					fmt.Printf("[dry-run] mkdir -p %s\n", dir)
					continue
				}
				if err := os.MkdirAll(dir, 0755); err != nil {
					return fmt.Errorf("create directory %s: %w", dir, err)
				}
				if err := chownRecursive(dir, uid, gid); err != nil {
					return fmt.Errorf("chown directory %s: %w", dir, err)
				}
			}

			for _, file := range files {
				if dryRun {
					fmt.Printf("[dry-run] touch %s\n", file)
					continue
				}
				if err := ensureFile(file); err != nil {
					return fmt.Errorf("ensure file %s: %w", file, err)
				}
				if err := os.Chown(file, uid, gid); err != nil {
					return fmt.Errorf("chown file %s: %w", file, err)
				}
				if err := os.Chmod(file, 0644); err != nil {
					return fmt.Errorf("chmod file %s: %w", file, err)
				}
			}

			fmt.Println("Permissions fixed successfully.")
			return nil
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", defaultConfigPath, "Path to config file")
	cmd.Flags().StringVar(&userName, "user", "mingyue-agent", "Service user name")
	cmd.Flags().StringVar(&groupName, "group", "", "Service group name (defaults to user)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print actions without making changes")

	return cmd
}

func requiredPaths(cfg *config.Config) []string {
	logDir := agentLogDir(cfg)
	paths := []string{
		filepath.Dir(cfg.NetDisk.StateFile),
		filepath.Dir(cfg.Network.HistoryFile),
		cfg.ShareMgr.BackupDir,
		filepath.Dir(cfg.ShareMgr.StateFile),
		filepath.Dir(cfg.Server.UDSPath),
		logDir,
	}

	return uniquePaths(paths)
}

func requiredFiles(cfg *config.Config) []string {
	files := []string{
		filepath.Join(agentLogDir(cfg), "agent.log"),
	}

	if cfg.Audit.Enabled && cfg.Audit.LogPath != "" {
		files = append(files, cfg.Audit.LogPath)
	}

	return uniquePaths(files)
}

func uniquePaths(paths []string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, path := range paths {
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		result = append(result, path)
	}
	return result
}

func resolveUserGroup(userName, groupName string) (int, int, error) {
	usr, err := user.Lookup(userName)
	if err != nil {
		return 0, 0, fmt.Errorf("lookup user %s: %w", userName, err)
	}

	uid, err := strconv.Atoi(usr.Uid)
	if err != nil {
		return 0, 0, fmt.Errorf("parse user id %s: %w", usr.Uid, err)
	}

	gid := usr.Gid
	if groupName != "" && groupName != userName {
		grp, err := user.LookupGroup(groupName)
		if err != nil {
			return 0, 0, fmt.Errorf("lookup group %s: %w", groupName, err)
		}
		gid = grp.Gid
	}

	gidInt, err := strconv.Atoi(gid)
	if err != nil {
		return 0, 0, fmt.Errorf("parse group id %s: %w", gid, err)
	}

	return uid, gidInt, nil
}

func chownRecursive(path string, uid, gid int) error {
	return filepath.WalkDir(path, func(target string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Chown(target, uid, gid)
	})
}

func ensureFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	return file.Close()
}

func agentLogDir(cfg *config.Config) string {
	if cfg.Audit.Enabled && cfg.Audit.LogPath != "" {
		return filepath.Dir(cfg.Audit.LogPath)
	}
	return "/var/log/mingyue-agent"
}
