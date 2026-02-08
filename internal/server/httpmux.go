package server

import (
	"fmt"
	"net/http"

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
)

// NewHTTPMux builds the HTTP handlers for the API server.
func NewHTTPMux(cfg *config.Config, auditLogger *audit.Logger) (*http.ServeMux, error) {
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

	return mux, nil
}
