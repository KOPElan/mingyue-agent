package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/KOPElan/mingyue-agent/internal/audit"
	"github.com/KOPElan/mingyue-agent/internal/diskmanager"
)

// DiskHandlers provides HTTP handlers for disk management operations
type DiskHandlers struct {
	manager *diskmanager.Manager
	audit   *audit.Logger
}

// NewDiskHandlers creates a new disk handlers instance
func NewDiskHandlers(manager *diskmanager.Manager, auditLogger *audit.Logger) *DiskHandlers {
	return &DiskHandlers{
		manager: manager,
		audit:   auditLogger,
	}
}

func (h *DiskHandlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/disk/list", h.ListDisks)
	mux.HandleFunc("/api/v1/disk/partitions", h.ListPartitions)
	mux.HandleFunc("/api/v1/disk/mount", h.Mount)
	mux.HandleFunc("/api/v1/disk/unmount", h.Unmount)
	mux.HandleFunc("/api/v1/disk/smart", h.GetSMART)
}

// ListPartitions handles GET /api/v1/disk/partitions
func (h *DiskHandlers) ListPartitions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	partitions, err := h.manager.ListPartitions()
	if err != nil {
		if h.audit != nil {
			h.audit.Log(r.Context(), &audit.Entry{
				Timestamp: time.Now(),
				Action:    "disk.list_partitions",
				Result:    "error",
				SourceIP:  r.RemoteAddr,
				Details: map[string]interface{}{
					"error": err.Error(),
				},
			})
		}
		writeJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "failed to list partitions: " + err.Error(),
		})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			Timestamp: time.Now(),
			Action:    "disk.list_partitions",
			Result:    "success",
			SourceIP:  r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    partitions,
	})
}

// ListDisks handles GET /api/v1/disk/list
func (h *DiskHandlers) ListDisks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	disks, err := h.manager.ListDisks()
	if err != nil {
		if h.audit != nil {
			h.audit.Log(r.Context(), &audit.Entry{
				Timestamp: time.Now(),
				Action:    "disk.list",
				Result:    "error",
				SourceIP:  r.RemoteAddr,
				Details: map[string]interface{}{
					"error": err.Error(),
				},
			})
		}
		writeJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "failed to list disks: " + err.Error(),
		})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			Timestamp: time.Now(),
			Action:    "disk.list",
			Result:    "success",
			SourceIP:  r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    disks,
	})
}

// Mount handles POST /api/v1/disk/mount
func (h *DiskHandlers) Mount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	var opts diskmanager.MountOptions
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "invalid request body: " + err.Error(),
		})
		return
	}

	if opts.Device == "" || opts.MountPoint == "" {
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "device and mount_point are required",
		})
		return
	}

	err := h.manager.Mount(opts)
	if err != nil {
		if h.audit != nil {
			h.audit.Log(r.Context(), &audit.Entry{
				Timestamp: time.Now(),
				Action:    "disk.mount",
				Resource:  opts.Device,
				Result:    "error",
				SourceIP:  r.RemoteAddr,
				Details: map[string]interface{}{
					"error":       err.Error(),
					"mount_point": opts.MountPoint,
				},
			})
		}
		writeJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "failed to mount: " + err.Error(),
		})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			Timestamp: time.Now(),
			Action:    "disk.mount",
			Resource:  opts.Device,
			Result:    "success",
			SourceIP:  r.RemoteAddr,
			Details: map[string]interface{}{
				"mount_point": opts.MountPoint,
			},
		})
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data: map[string]string{
			"message": "device mounted successfully",
		},
	})
}

// Unmount handles POST /api/v1/disk/unmount
func (h *DiskHandlers) Unmount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	var req struct {
		Target string `json:"target"`
		Force  bool   `json:"force"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "invalid request body: " + err.Error(),
		})
		return
	}

	if req.Target == "" {
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "target is required",
		})
		return
	}

	err := h.manager.Unmount(req.Target, req.Force)
	if err != nil {
		if h.audit != nil {
			h.audit.Log(r.Context(), &audit.Entry{
				Timestamp: time.Now(),
				Action:    "disk.unmount",
				Resource:  req.Target,
				Result:    "error",
				SourceIP:  r.RemoteAddr,
				Details: map[string]interface{}{
					"error": err.Error(),
					"force": req.Force,
				},
			})
		}
		writeJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "failed to unmount: " + err.Error(),
		})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			Timestamp: time.Now(),
			Action:    "disk.unmount",
			Resource:  req.Target,
			Result:    "success",
			SourceIP:  r.RemoteAddr,
			Details: map[string]interface{}{
				"force": req.Force,
			},
		})
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data: map[string]string{
			"message": "device unmounted successfully",
		},
	})
}

// GetSMART handles GET /api/v1/disk/smart
func (h *DiskHandlers) GetSMART(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	device := r.URL.Query().Get("device")
	if device == "" {
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "device parameter is required",
		})
		return
	}

	smartInfo, err := h.manager.GetSMARTInfo(device)
	if err != nil {
		if h.audit != nil {
			h.audit.Log(r.Context(), &audit.Entry{
				Timestamp: time.Now(),
				Action:    "disk.smart",
				Resource:  device,
				Result:    "error",
				SourceIP:  r.RemoteAddr,
				Details: map[string]interface{}{
					"error": err.Error(),
				},
			})
		}
		writeJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "failed to get SMART info: " + err.Error(),
		})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			Timestamp: time.Now(),
			Action:    "disk.smart",
			Resource:  device,
			Result:    "success",
			SourceIP:  r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    smartInfo,
	})
}
