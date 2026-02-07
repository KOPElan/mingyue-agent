package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/KOPElan/mingyue-agent/internal/config"
	"github.com/KOPElan/mingyue-agent/internal/daemon"
	"github.com/spf13/cobra"
)

// @title Mingyue Agent API
// @version 1.0
// @description Mingyue Agent is the core local management service for the Mingyue Portal home server ecosystem, providing both remote collaboration agent and local privileged operations capabilities.
// @description
// @description ## Features
// @description - Secure File Management with path validation
// @description - Disk Management with SMART monitoring
// @description - Network Disk Management (CIFS/NFS)
// @description - System Network Management
// @description - Share Management (Samba/NFS)
// @description - System Resource Monitoring
// @description - Comprehensive Audit Logging
//
// @contact.name Mingyue Agent Team
// @contact.url https://github.com/KOPElan/mingyue-agent
//
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
//
// @host localhost:8080
// @BasePath /api/v1
//
// @schemes http https
//
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
//
// @securityDefinitions.apikey UserAuth
// @in header
// @name X-User

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "mingyue-agent",
		Short: "Mingyue Agent - Local management service for home servers",
		Long: `Mingyue Agent is the core local management service for the Mingyue Portal
home server ecosystem, providing both remote collaboration agent and
local privileged operations capabilities.`,
		Version: fmt.Sprintf("%s (built at %s)", version, buildTime),
	}

	rootCmd.AddCommand(startCmd())
	rootCmd.AddCommand(versionCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func startCmd() *cobra.Command {
	var configFile string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the agent daemon",
		Long: `Start the Mingyue Agent daemon process.

The agent will start HTTP API server (default port 8080), gRPC server (default port 9090),
and Unix domain socket for local communication. All servers can be configured via the config file.

Examples:
  # Start with default config
  mingyue-agent start

  # Start with custom config
  mingyue-agent start --config /path/to/config.yaml
  mingyue-agent start -c ./my-config.yaml

  # Generate example config
  cp config.example.yaml my-config.yaml

The daemon will run in the foreground and can be stopped with Ctrl+C (SIGINT) or SIGTERM.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			d, err := daemon.New(cfg)
			if err != nil {
				return fmt.Errorf("failed to create daemon: %w", err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			errCh := make(chan error, 1)
			go func() {
				errCh <- d.Start(ctx)
			}()

			select {
			case sig := <-sigCh:
				fmt.Printf("\nReceived signal %v, shutting down...\n", sig)
				cancel()
				return d.Shutdown(context.Background())
			case err := <-errCh:
				if err != nil {
					return fmt.Errorf("daemon error: %w", err)
				}
				return nil
			}
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", "/etc/mingyue-agent/config.yaml", "Path to config file")

	return cmd
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long: `Print detailed version information including build time and Git commit.

Examples:
  mingyue-agent version`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Mingyue Agent %s\n", version)
			fmt.Printf("Build Time: %s\n", buildTime)
			fmt.Printf("\nFor more information, visit:\n")
			fmt.Printf("  Documentation: https://github.com/KOPElan/mingyue-agent\n")
			fmt.Printf("  API Reference: docs/API.md\n")
		},
	}
}
