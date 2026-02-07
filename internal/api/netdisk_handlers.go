package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/KOPElan/mingyue-agent/internal/audit"
	"github.com/KOPElan/mingyue-agent/internal/netdisk"
)

// NetDiskHandlers provides HTTP handlers for network disk operations
type NetDiskHandlers struct {
	manager *netdisk.Manager
	audit   *audit.Logger
}

// NewNetDiskHandlers creates a new network disk handlers instance
func NewNetDiskHandlers(manager *netdisk.Manager, auditLogger *audit.Logger) *NetDiskHandlers {
	return &NetDiskHandlers{
		manager: manager,
		audit:   auditLogger,
	}
}

// ListShares handles GET /api/v1/netdisk/shares
func (h *NetDiskHandlers) ListShares(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	shares := h.manager.ListShares()

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			Timestamp: time.Now(),
			User:      getUser(r),
			Action:    "netdisk.list_shares",
			Result:    "success",
			SourceIP:  r.RemoteAddr,
			Details: map[string]interface{}{
				"count": len(shares),
			},
		})
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    shares,
	})
}

// AddShare handles POST /api/v1/netdisk/shares
func (h *NetDiskHandlers) AddShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	var share netdisk.Share
	if err := json.NewDecoder(r.Body).Decode(&share); err != nil {
		if h.audit != nil {
			h.audit.Log(r.Context(), &audit.Entry{
				Timestamp: time.Now(),
				User:      getUser(r),
				Action:    "netdisk.add_share",
				Result:    "error",
				SourceIP:  r.RemoteAddr,
				Details: map[string]interface{}{
					"error": "invalid request body",
				},
			})
		}
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "invalid request body: " + err.Error(),
		})
		return
	}

	if err := h.manager.AddShare(&share); err != nil {
		if h.audit != nil {
			h.audit.Log(r.Context(), &audit.Entry{
				Timestamp: time.Now(),
				User:      getUser(r),
				Action:    "netdisk.add_share",
				Resource:  share.Host + share.Path,
				Result:    "error",
				SourceIP:  r.RemoteAddr,
				Details: map[string]interface{}{
					"error":    err.Error(),
					"protocol": share.Protocol,
					"host":     share.Host,
				},
			})
		}
		writeJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "failed to add share: " + err.Error(),
		})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			Timestamp: time.Now(),
			User:      getUser(r),
			Action:    "netdisk.add_share",
			Resource:  share.Host + share.Path,
			Result:    "success",
			SourceIP:  r.RemoteAddr,
			Details: map[string]interface{}{
				"share_id":    share.ID,
				"protocol":    share.Protocol,
				"host":        share.Host,
				"mount_point": share.MountPoint,
			},
		})
	}

	writeJSON(w, http.StatusCreated, Response{
		Success: true,
		Data:    map[string]interface{}{"share_id": share.ID},
	})
}

// RemoveShare handles DELETE /api/v1/netdisk/shares/{id}
func (h *NetDiskHandlers) RemoveShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "share id is required",
		})
		return
	}

	if err := h.manager.RemoveShare(id); err != nil {
		if h.audit != nil {
			h.audit.Log(r.Context(), &audit.Entry{
				Timestamp: time.Now(),
				User:      getUser(r),
				Action:    "netdisk.remove_share",
				Resource:  id,
				Result:    "error",
				SourceIP:  r.RemoteAddr,
				Details: map[string]interface{}{
					"error": err.Error(),
				},
			})
		}
		writeJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "failed to remove share: " + err.Error(),
		})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			Timestamp: time.Now(),
			User:      getUser(r),
			Action:    "netdisk.remove_share",
			Resource:  id,
			Result:    "success",
			SourceIP:  r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    map[string]interface{}{"message": "share removed"},
	})
}

// MountShare handles POST /api/v1/netdisk/mount
func (h *NetDiskHandlers) MountShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "invalid request body: " + err.Error(),
		})
		return
	}

	if err := h.manager.Mount(req.ID); err != nil {
		if h.audit != nil {
			h.audit.Log(r.Context(), &audit.Entry{
				Timestamp: time.Now(),
				User:      getUser(r),
				Action:    "netdisk.mount",
				Resource:  req.ID,
				Result:    "error",
				SourceIP:  r.RemoteAddr,
				Details: map[string]interface{}{
					"error": err.Error(),
				},
			})
		}
		writeJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "failed to mount share: " + err.Error(),
		})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			Timestamp: time.Now(),
			User:      getUser(r),
			Action:    "netdisk.mount",
			Resource:  req.ID,
			Result:    "success",
			SourceIP:  r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    map[string]interface{}{"message": "share mounted"},
	})
}

// UnmountShare handles POST /api/v1/netdisk/unmount
func (h *NetDiskHandlers) UnmountShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "invalid request body: " + err.Error(),
		})
		return
	}

	if err := h.manager.Unmount(req.ID); err != nil {
		if h.audit != nil {
			h.audit.Log(r.Context(), &audit.Entry{
				Timestamp: time.Now(),
				User:      getUser(r),
				Action:    "netdisk.unmount",
				Resource:  req.ID,
				Result:    "error",
				SourceIP:  r.RemoteAddr,
				Details: map[string]interface{}{
					"error": err.Error(),
				},
			})
		}
		writeJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "failed to unmount share: " + err.Error(),
		})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			Timestamp: time.Now(),
			User:      getUser(r),
			Action:    "netdisk.unmount",
			Resource:  req.ID,
			Result:    "success",
			SourceIP:  r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    map[string]interface{}{"message": "share unmounted"},
	})
}

// GetShareStatus handles GET /api/v1/netdisk/shares/{id}/status
func (h *NetDiskHandlers) GetShareStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "share id is required",
		})
		return
	}

	status, err := h.manager.GetShareStatus(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, Response{
			Success: false,
			Error:   "share not found: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    status,
	})
}
