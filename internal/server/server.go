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
	"github.com/KOPElan/mingyue-agent/internal/diskmanager"
	"github.com/KOPElan/mingyue-agent/internal/filemanager"
	"github.com/KOPElan/mingyue-agent/internal/monitor"
	"github.com/KOPElan/mingyue-agent/internal/netdisk"
	"github.com/KOPElan/mingyue-agent/internal/netmanager"
	"github.com/KOPElan/mingyue-agent/internal/sharemanager"
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

		diskMgr := diskmanager.New(cfg.Security.AllowedPaths)
		diskAPI := api.NewDiskHandlers(diskMgr, auditLogger)
		mux.HandleFunc("/api/v1/disk/list", diskAPI.ListDisks)
		mux.HandleFunc("/api/v1/disk/partitions", diskAPI.ListPartitions)
		mux.HandleFunc("/api/v1/disk/mount", diskAPI.Mount)
		mux.HandleFunc("/api/v1/disk/unmount", diskAPI.Unmount)
		mux.HandleFunc("/api/v1/disk/smart", diskAPI.GetSMART)

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
		mux.HandleFunc("/api/v1/netdisk/shares", netDiskAPI.ListShares)
		mux.HandleFunc("/api/v1/netdisk/shares/add", netDiskAPI.AddShare)
		mux.HandleFunc("/api/v1/netdisk/shares/remove", netDiskAPI.RemoveShare)
		mux.HandleFunc("/api/v1/netdisk/mount", netDiskAPI.MountShare)
		mux.HandleFunc("/api/v1/netdisk/unmount", netDiskAPI.UnmountShare)
		mux.HandleFunc("/api/v1/netdisk/status", netDiskAPI.GetShareStatus)

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
		mux.HandleFunc("/api/v1/network/interfaces", netMgrAPI.ListInterfaces)
		mux.HandleFunc("/api/v1/network/interface", netMgrAPI.GetInterface)
		mux.HandleFunc("/api/v1/network/config", netMgrAPI.SetIPConfig)
		mux.HandleFunc("/api/v1/network/rollback", netMgrAPI.RollbackConfig)
		mux.HandleFunc("/api/v1/network/history", netMgrAPI.ListConfigHistory)
		mux.HandleFunc("/api/v1/network/enable", netMgrAPI.EnableInterface)
		mux.HandleFunc("/api/v1/network/disable", netMgrAPI.DisableInterface)
		mux.HandleFunc("/api/v1/network/ports", netMgrAPI.ListListeningPorts)
		mux.HandleFunc("/api/v1/network/traffic", netMgrAPI.GetTrafficStats)

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
		mux.HandleFunc("/api/v1/shares", shareAPI.ListShares)
		mux.HandleFunc("/api/v1/shares/get", shareAPI.GetShare)
		mux.HandleFunc("/api/v1/shares/add", shareAPI.AddShare)
		mux.HandleFunc("/api/v1/shares/update", shareAPI.UpdateShare)
		mux.HandleFunc("/api/v1/shares/remove", shareAPI.RemoveShare)
		mux.HandleFunc("/api/v1/shares/enable", shareAPI.EnableShare)
		mux.HandleFunc("/api/v1/shares/disable", shareAPI.DisableShare)
		mux.HandleFunc("/api/v1/shares/rollback", shareAPI.RollbackConfig)

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

			diskMgr := diskmanager.New(s.config.Security.AllowedPaths)
			diskAPI := api.NewDiskHandlers(diskMgr, s.audit)
			mux.HandleFunc("/api/v1/disk/list", diskAPI.ListDisks)
			mux.HandleFunc("/api/v1/disk/partitions", diskAPI.ListPartitions)
			mux.HandleFunc("/api/v1/disk/mount", diskAPI.Mount)
			mux.HandleFunc("/api/v1/disk/unmount", diskAPI.Unmount)
			mux.HandleFunc("/api/v1/disk/smart", diskAPI.GetSMART)

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
