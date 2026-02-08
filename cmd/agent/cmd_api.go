package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KOPElan/mingyue-agent/internal/audit"
	"github.com/KOPElan/mingyue-agent/internal/config"
	"github.com/KOPElan/mingyue-agent/internal/server"
	"github.com/spf13/cobra"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *loggingResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *loggingResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}

	n, err := w.ResponseWriter.Write(data)
	w.bytes += n
	return n, err
}

func requestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		writer := &loggingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(writer, r)

		status := writer.status
		if status == 0 {
			status = http.StatusOK
		}

		log.Printf("%s %s %d %dB %s %s %q", r.Method, r.URL.RequestURI(), status, writer.bytes, time.Since(start), r.RemoteAddr, r.UserAgent())
	})
}

func apiCmd() *cobra.Command {
	var configFile string
	var logFile string
	var noAudit bool
	var requestLog bool

	cmd := &cobra.Command{
		Use:   "start-api",
		Short: "Start the HTTP API with verbose logging",
		Long: `Start the Mingyue Agent HTTP API server with detailed request logging.

This is useful for local debugging to observe raw handler behavior without the full daemon.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var cfg *config.Config
			if localMode {
				localCfg, _, err := loadLocalConfig()
				if err != nil {
					return err
				}
				cfg = localCfg
			} else {
				resolvedConfig := resolveConfigPath(configFile)
				loadedCfg, err := config.Load(resolvedConfig)
				if err != nil {
					return fmt.Errorf("failed to load config: %w", err)
				}
				cfg = loadedCfg
			}

			var auditLogger *audit.Logger
			if cfg.Audit.Enabled && !noAudit {
				auditLogger, err = audit.New(cfg.Audit.LogPath, cfg.Audit.RemotePush, cfg.Audit.RemoteURL, cfg.Audit.Enabled)
				if err != nil {
					return fmt.Errorf("create audit logger: %w", err)
				}
				defer func() {
					if err := auditLogger.Close(); err != nil {
						log.Printf("audit logger close error: %v", err)
					}
				}()
			}

			handler, err := server.NewHTTPMux(cfg, auditLogger)
			if err != nil {
				return fmt.Errorf("create HTTP handlers: %w", err)
			}

			var output io.Writer = os.Stdout
			if logFile != "" {
				file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
				if err != nil {
					return fmt.Errorf("open log file: %w", err)
				}
				defer file.Close()
				output = io.MultiWriter(os.Stdout, file)
			}
			log.SetOutput(output)

			finalHandler := http.Handler(handler)
			if requestLog {
				finalHandler = requestLogging(finalHandler)
			}

			srv := &http.Server{
				Addr:         fmt.Sprintf("%s:%d", cfg.Server.ListenAddr, cfg.Server.HTTPPort),
				Handler:      finalHandler,
				ReadTimeout:  15 * time.Second,
				WriteTimeout: 15 * time.Second,
				IdleTimeout:  60 * time.Second,
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			errCh := make(chan error, 1)
			go func() {
				if cfg.API.TLSCert != "" && cfg.API.TLSKey != "" {
					errCh <- srv.ListenAndServeTLS(cfg.API.TLSCert, cfg.API.TLSKey)
					return
				}
				errCh <- srv.ListenAndServe()
			}()

			log.Printf("API server listening on %s", srv.Addr)

			select {
			case sig := <-sigCh:
				log.Printf("Received signal %v, shutting down...", sig)
				shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
				defer shutdownCancel()
				return srv.Shutdown(shutdownCtx)
			case err := <-errCh:
				if err != nil && err != http.ErrServerClosed {
					return fmt.Errorf("API server error: %w", err)
				}
				return nil
			}
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", defaultConfigPath, "Path to config file")
	cmd.Flags().StringVar(&logFile, "log-file", "", "Log file path (optional, logs also go to stdout)")
	cmd.Flags().BoolVar(&noAudit, "no-audit", false, "Disable audit logging for this command")
	cmd.Flags().BoolVar(&requestLog, "request-log", true, "Log each HTTP request")

	return cmd
}
