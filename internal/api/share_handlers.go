package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/KOPElan/mingyue-agent/internal/audit"
	"github.com/KOPElan/mingyue-agent/internal/sharemanager"
)

// ShareHandlers provides HTTP handlers for share management operations
type ShareHandlers struct {
	manager *sharemanager.Manager
	audit   *audit.Logger
}

// NewShareHandlers creates a new share handlers instance
func NewShareHandlers(manager *sharemanager.Manager, auditLogger *audit.Logger) *ShareHandlers {
	return &ShareHandlers{
		manager: manager,
		audit:   auditLogger,
	}
}

// ListShares handles GET /api/v1/shares
func (h *ShareHandlers) ListShares(w http.ResponseWriter, r *http.Request) {
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
			Action:    "share.list",
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

// GetShare handles GET /api/v1/shares/{id}
func (h *ShareHandlers) GetShare(w http.ResponseWriter, r *http.Request) {
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

	share, err := h.manager.GetShare(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, Response{
			Success: false,
			Error:   "share not found: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    share,
	})
}

// AddShare handles POST /api/v1/shares
func (h *ShareHandlers) AddShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	var share sharemanager.Share
	if err := json.NewDecoder(r.Body).Decode(&share); err != nil {
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
				Action:    "share.add",
				Resource:  share.Path,
				Result:    "error",
				SourceIP:  r.RemoteAddr,
				Details: map[string]interface{}{
					"error": err.Error(),
					"name":  share.Name,
					"type":  share.Type,
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
			Action:    "share.add",
			Resource:  share.Path,
			Result:    "success",
			SourceIP:  r.RemoteAddr,
			Details: map[string]interface{}{
				"share_id": share.ID,
				"name":     share.Name,
				"type":     share.Type,
				"path":     share.Path,
			},
		})
	}

	writeJSON(w, http.StatusCreated, Response{
		Success: true,
		Data:    map[string]interface{}{"share_id": share.ID},
	})
}

// UpdateShare handles PUT /api/v1/shares/{id}
func (h *ShareHandlers) UpdateShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
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

	var updates sharemanager.Share
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "invalid request body: " + err.Error(),
		})
		return
	}

	if err := h.manager.UpdateShare(id, &updates); err != nil {
		if h.audit != nil {
			h.audit.Log(r.Context(), &audit.Entry{
				Timestamp: time.Now(),
				User:      getUser(r),
				Action:    "share.update",
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
			Error:   "failed to update share: " + err.Error(),
		})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			Timestamp: time.Now(),
			User:      getUser(r),
			Action:    "share.update",
			Resource:  id,
			Result:    "success",
			SourceIP:  r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    map[string]interface{}{"message": "share updated"},
	})
}

// RemoveShare handles DELETE /api/v1/shares/{id}
func (h *ShareHandlers) RemoveShare(w http.ResponseWriter, r *http.Request) {
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
				Action:    "share.remove",
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
			Action:    "share.remove",
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

// EnableShare handles POST /api/v1/shares/{id}/enable
func (h *ShareHandlers) EnableShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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

	if err := h.manager.EnableShare(id); err != nil {
		if h.audit != nil {
			h.audit.Log(r.Context(), &audit.Entry{
				Timestamp: time.Now(),
				User:      getUser(r),
				Action:    "share.enable",
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
			Error:   "failed to enable share: " + err.Error(),
		})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			Timestamp: time.Now(),
			User:      getUser(r),
			Action:    "share.enable",
			Resource:  id,
			Result:    "success",
			SourceIP:  r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    map[string]interface{}{"message": "share enabled"},
	})
}

// DisableShare handles POST /api/v1/shares/{id}/disable
func (h *ShareHandlers) DisableShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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

	if err := h.manager.DisableShare(id); err != nil {
		if h.audit != nil {
			h.audit.Log(r.Context(), &audit.Entry{
				Timestamp: time.Now(),
				User:      getUser(r),
				Action:    "share.disable",
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
			Error:   "failed to disable share: " + err.Error(),
		})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			Timestamp: time.Now(),
			User:      getUser(r),
			Action:    "share.disable",
			Resource:  id,
			Result:    "success",
			SourceIP:  r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    map[string]interface{}{"message": "share disabled"},
	})
}

// RollbackConfig handles POST /api/v1/shares/rollback
func (h *ShareHandlers) RollbackConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	var req struct {
		Timestamp int64 `json:"timestamp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "invalid request body: " + err.Error(),
		})
		return
	}

	timestamp := time.Unix(req.Timestamp, 0)
	if err := h.manager.RollbackConfig(timestamp); err != nil {
		if h.audit != nil {
			h.audit.Log(r.Context(), &audit.Entry{
				Timestamp: time.Now(),
				User:      getUser(r),
				Action:    "share.rollback",
				Result:    "error",
				SourceIP:  r.RemoteAddr,
				Details: map[string]interface{}{
					"error":            err.Error(),
					"target_timestamp": timestamp,
				},
			})
		}
		writeJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "failed to rollback config: " + err.Error(),
		})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			Timestamp: time.Now(),
			User:      getUser(r),
			Action:    "share.rollback",
			Result:    "success",
			SourceIP:  r.RemoteAddr,
			Details: map[string]interface{}{
				"target_timestamp": timestamp,
			},
		})
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    map[string]interface{}{"message": "config rolled back"},
	})
}
