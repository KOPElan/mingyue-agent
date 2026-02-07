package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/KOPElan/mingyue-agent/internal/api"
	"github.com/KOPElan/mingyue-agent/internal/audit"
	"github.com/KOPElan/mingyue-agent/internal/config"
	"github.com/KOPElan/mingyue-agent/internal/filemanager"
	"github.com/KOPElan/mingyue-agent/internal/monitor"
	"google.golang.org/grpc"
)

type Server struct {
	config      *config.Config
	audit       *audit.Logger
	httpServer  *http.Server
	grpcServer  *grpc.Server
	udsListener net.Listener
	wg          sync.WaitGroup
}

func New(cfg *config.Config, auditLogger *audit.Logger) (*Server, error) {
	s := &Server{
		config: cfg,
		audit:  auditLogger,
	}

	if cfg.API.EnableHTTP {
		mux := http.NewServeMux()
		api.RegisterHTTPHandlers(mux, auditLogger)

		mon := monitor.New()
		monitorAPI := api.NewMonitorAPI(mon, auditLogger)
		monitorAPI.Register(mux)

		fileMgr := filemanager.New(cfg.Security.AllowedPaths, auditLogger)
		fileAPI := api.NewFileAPI(fileMgr, auditLogger)
		fileAPI.Register(mux)

		s.httpServer = &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.Server.ListenAddr, cfg.Server.HTTPPort),
			Handler:      mux,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		}
	}

	if cfg.API.EnableGRPC {
		s.grpcServer = grpc.NewServer()
	}

	return s, nil
}

func (s *Server) Start(ctx context.Context) error {
	if s.config.API.EnableHTTP {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Printf("HTTP server error: %v\n", err)
			}
		}()
	}

	if s.config.API.EnableGRPC {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()

			lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.config.Server.ListenAddr, s.config.Server.GRPCPort))
			if err != nil {
				fmt.Printf("gRPC listen error: %v\n", err)
				return
			}

			if err := s.grpcServer.Serve(lis); err != nil {
				fmt.Printf("gRPC server error: %v\n", err)
			}
		}()
	}

	if s.config.API.EnableUDS {
		if err := os.MkdirAll(filepath.Dir(s.config.Server.UDSPath), 0755); err != nil {
			return fmt.Errorf("create UDS directory: %w", err)
		}

		os.Remove(s.config.Server.UDSPath)

		lis, err := net.Listen("unix", s.config.Server.UDSPath)
		if err != nil {
			return fmt.Errorf("listen on UDS: %w", err)
		}
		s.udsListener = lis

		if err := os.Chmod(s.config.Server.UDSPath, 0666); err != nil {
			return fmt.Errorf("chmod UDS socket: %w", err)
		}

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()

			mux := http.NewServeMux()
			api.RegisterHTTPHandlers(mux, s.audit)

			mon := monitor.New()
			monitorAPI := api.NewMonitorAPI(mon, s.audit)
			monitorAPI.Register(mux)

			fileMgr := filemanager.New(s.config.Security.AllowedPaths, s.audit)
			fileAPI := api.NewFileAPI(fileMgr, s.audit)
			fileAPI.Register(mux)

			srv := &http.Server{Handler: mux}
			if err := srv.Serve(lis); err != nil && err != http.ErrServerClosed {
				fmt.Printf("UDS server error: %v\n", err)
			}
		}()
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	var firstErr error

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}

	if s.udsListener != nil {
		if err := s.udsListener.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
		os.Remove(s.config.Server.UDSPath)
	}

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}

	return firstErr
}
