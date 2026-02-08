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

	_ "github.com/KOPElan/mingyue-agent/docs"
	"github.com/KOPElan/mingyue-agent/internal/api"
	"github.com/KOPElan/mingyue-agent/internal/audit"
	"github.com/KOPElan/mingyue-agent/internal/config"
	"github.com/KOPElan/mingyue-agent/internal/diskmanager"
	"github.com/KOPElan/mingyue-agent/internal/filemanager"
	"github.com/KOPElan/mingyue-agent/internal/monitor"
	"github.com/KOPElan/mingyue-agent/internal/netdisk"
	"github.com/KOPElan/mingyue-agent/internal/netmanager"
	"github.com/KOPElan/mingyue-agent/internal/sharemanager"
	httpSwagger "github.com/swaggo/http-swagger"
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
		api.RegisterHTTPHandlers(mux, auditLogger, cfg)

		// Swagger UI
		mux.Handle("/swagger/", httpSwagger.WrapHandler)

		mon := monitor.New()
		monitorAPI := api.NewMonitorAPI(mon, auditLogger)
		monitorAPI.Register(mux)

		fileMgr := filemanager.New(cfg.Security.AllowedPaths, auditLogger)
		fileAPI := api.NewFileAPI(fileMgr, auditLogger, cfg.Security.MaxUploadSize)
		fileAPI.Register(mux)

		diskMgr := diskmanager.New(cfg.Security.AllowedPaths)
		diskAPI := api.NewDiskHandlers(diskMgr, auditLogger)
		diskAPI.Register(mux)

		// Network disk management
		netDiskMgr, err := netdisk.New(&netdisk.Config{
			AllowedHosts:       cfg.NetDisk.AllowedHosts,
			AllowedMountPoints: cfg.NetDisk.AllowedMountPoints,
			EncryptionKey:      cfg.NetDisk.EncryptionKey,
			StateFile:          cfg.NetDisk.StateFile,
		})
		if err != nil {
			return nil, fmt.Errorf("create network disk manager: %w", err)
		}
		netDiskAPI := api.NewNetDiskHandlers(netDiskMgr, auditLogger)
		netDiskAPI.Register(mux)

		// Network management
		netMgr, err := netmanager.New(&netmanager.Config{
			ManagementInterface: cfg.Network.ManagementInterface,
			HistoryFile:         cfg.Network.HistoryFile,
			ConfigDir:           cfg.Network.ConfigDir,
		})
		if err != nil {
			return nil, fmt.Errorf("create network manager: %w", err)
		}
		netMgrAPI := api.NewNetManagerHandlers(netMgr, auditLogger)
		netMgrAPI.Register(mux)

		// Share management
		shareMgr, err := sharemanager.New(&sharemanager.Config{
			AllowedPaths: cfg.ShareMgr.AllowedPaths,
			SambaConfig:  cfg.ShareMgr.SambaConfig,
			NFSConfig:    cfg.ShareMgr.NFSConfig,
			BackupDir:    cfg.ShareMgr.BackupDir,
			StateFile:    cfg.ShareMgr.StateFile,
		})
		if err != nil {
			return nil, fmt.Errorf("create share manager: %w", err)
		}
		shareAPI := api.NewShareHandlers(shareMgr, auditLogger)
		shareAPI.Register(mux)

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
			var err error
			if s.config.API.TLSCert != "" && s.config.API.TLSKey != "" {
				err = s.httpServer.ListenAndServeTLS(s.config.API.TLSCert, s.config.API.TLSKey)
			} else {
				err = s.httpServer.ListenAndServe()
			}
			if err != nil && err != http.ErrServerClosed {
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
		// Try to create UDS directory, fallback to temp dir on read-only filesystem
		udsDir := filepath.Dir(s.config.Server.UDSPath)
		if err := os.MkdirAll(udsDir, 0755); err != nil {
			s.config.Server.UDSPath = filepath.Join(os.TempDir(), "mingyue-agent", filepath.Base(s.config.Server.UDSPath))
			if err := os.MkdirAll(filepath.Dir(s.config.Server.UDSPath), 0755); err != nil {
				return fmt.Errorf("create UDS directory: %w", err)
			}
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
			api.RegisterHTTPHandlers(mux, s.audit, s.config)

			mon := monitor.New()
			monitorAPI := api.NewMonitorAPI(mon, s.audit)
			monitorAPI.Register(mux)

			fileMgr := filemanager.New(s.config.Security.AllowedPaths, s.audit)
			fileAPI := api.NewFileAPI(fileMgr, s.audit, s.config.Security.MaxUploadSize)
			fileAPI.Register(mux)

			diskMgr := diskmanager.New(s.config.Security.AllowedPaths)
			diskAPI := api.NewDiskHandlers(diskMgr, s.audit)
			diskAPI.Register(mux)

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
