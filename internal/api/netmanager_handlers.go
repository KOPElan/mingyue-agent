package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/KOPElan/mingyue-agent/internal/audit"
	"github.com/KOPElan/mingyue-agent/internal/netmanager"
)

// NetManagerHandlers provides HTTP handlers for network management operations
type NetManagerHandlers struct {
	manager *netmanager.Manager
	audit   *audit.Logger
}

// NewNetManagerHandlers creates a new network manager handlers instance
func NewNetManagerHandlers(manager *netmanager.Manager, auditLogger *audit.Logger) *NetManagerHandlers {
	return &NetManagerHandlers{
		manager: manager,
		audit:   auditLogger,
	}
}

func (h *NetManagerHandlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/network/interfaces", h.ListInterfaces)
	mux.HandleFunc("/api/v1/network/interface", h.GetInterface)
	mux.HandleFunc("/api/v1/network/config", h.SetIPConfig)
	mux.HandleFunc("/api/v1/network/rollback", h.RollbackConfig)
	mux.HandleFunc("/api/v1/network/history", h.ListConfigHistory)
	mux.HandleFunc("/api/v1/network/enable", h.EnableInterface)
	mux.HandleFunc("/api/v1/network/disable", h.DisableInterface)
	mux.HandleFunc("/api/v1/network/ports", h.ListListeningPorts)
	mux.HandleFunc("/api/v1/network/traffic", h.GetTrafficStats)
}

// ListInterfaces handles GET /api/v1/network/interfaces
func (h *NetManagerHandlers) ListInterfaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	interfaces, err := h.manager.ListInterfaces()
	if err != nil {
		if h.audit != nil {
			h.audit.Log(r.Context(), &audit.Entry{
				Timestamp: time.Now(),
				User:      getUser(r),
				Action:    "network.list_interfaces",
				Result:    "error",
				SourceIP:  r.RemoteAddr,
				Details: map[string]interface{}{
					"error": err.Error(),
				},
			})
		}
		writeJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "failed to list interfaces: " + err.Error(),
		})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			Timestamp: time.Now(),
			User:      getUser(r),
			Action:    "network.list_interfaces",
			Result:    "success",
			SourceIP:  r.RemoteAddr,
			Details: map[string]interface{}{
				"count": len(interfaces),
			},
		})
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    interfaces,
	})
}

// GetInterface handles GET /api/v1/network/interfaces/{name}
func (h *NetManagerHandlers) GetInterface(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "interface name is required",
		})
		return
	}

	iface, err := h.manager.GetInterface(name)
	if err != nil {
		writeJSON(w, http.StatusNotFound, Response{
			Success: false,
			Error:   "interface not found: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    iface,
	})
}

// SetIPConfig handles POST /api/v1/network/config
func (h *NetManagerHandlers) SetIPConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	var req struct {
		Config netmanager.IPConfig `json:"config"`
		Reason string              `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "invalid request body: " + err.Error(),
		})
		return
	}

	user := getUser(r)
	if err := h.manager.SetIPConfig(&req.Config, user, req.Reason); err != nil {
		if h.audit != nil {
			h.audit.Log(r.Context(), &audit.Entry{
				Timestamp: time.Now(),
				User:      user,
				Action:    "network.set_ip_config",
				Resource:  req.Config.Interface,
				Result:    "error",
				SourceIP:  r.RemoteAddr,
				Details: map[string]interface{}{
					"error":  err.Error(),
					"method": req.Config.Method,
				},
			})
		}
		writeJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "failed to set IP config: " + err.Error(),
		})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			Timestamp: time.Now(),
			User:      user,
			Action:    "network.set_ip_config",
			Resource:  req.Config.Interface,
			Result:    "success",
			SourceIP:  r.RemoteAddr,
			Details: map[string]interface{}{
				"method":  req.Config.Method,
				"address": req.Config.Address,
				"reason":  req.Reason,
			},
		})
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    map[string]interface{}{"message": "IP config updated"},
	})
}

// RollbackConfig handles POST /api/v1/network/rollback
func (h *NetManagerHandlers) RollbackConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	var req struct {
		HistoryID string `json:"history_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "invalid request body: " + err.Error(),
		})
		return
	}

	user := getUser(r)
	if err := h.manager.RollbackConfig(req.HistoryID, user); err != nil {
		if h.audit != nil {
			h.audit.Log(r.Context(), &audit.Entry{
				Timestamp: time.Now(),
				User:      user,
				Action:    "network.rollback_config",
				Resource:  req.HistoryID,
				Result:    "error",
				SourceIP:  r.RemoteAddr,
				Details: map[string]interface{}{
					"error": err.Error(),
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
			User:      user,
			Action:    "network.rollback_config",
			Resource:  req.HistoryID,
			Result:    "success",
			SourceIP:  r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    map[string]interface{}{"message": "config rolled back"},
	})
}

// ListConfigHistory handles GET /api/v1/network/history
func (h *NetManagerHandlers) ListConfigHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	iface := r.URL.Query().Get("interface")
	history := h.manager.ListConfigHistory(iface)

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    history,
	})
}

// EnableInterface handles POST /api/v1/network/enable
func (h *NetManagerHandlers) EnableInterface(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	var req struct {
		Interface string `json:"interface"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "invalid request body: " + err.Error(),
		})
		return
	}

	if err := h.manager.EnableInterface(req.Interface); err != nil {
		if h.audit != nil {
			h.audit.Log(r.Context(), &audit.Entry{
				Timestamp: time.Now(),
				User:      getUser(r),
				Action:    "network.enable_interface",
				Resource:  req.Interface,
				Result:    "error",
				SourceIP:  r.RemoteAddr,
				Details: map[string]interface{}{
					"error": err.Error(),
				},
			})
		}
		writeJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "failed to enable interface: " + err.Error(),
		})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			Timestamp: time.Now(),
			User:      getUser(r),
			Action:    "network.enable_interface",
			Resource:  req.Interface,
			Result:    "success",
			SourceIP:  r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    map[string]interface{}{"message": "interface enabled"},
	})
}

// DisableInterface handles POST /api/v1/network/disable
func (h *NetManagerHandlers) DisableInterface(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	var req struct {
		Interface string `json:"interface"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{
			Success: false,
			Error:   "invalid request body: " + err.Error(),
		})
		return
	}

	if err := h.manager.DisableInterface(req.Interface); err != nil {
		if h.audit != nil {
			h.audit.Log(r.Context(), &audit.Entry{
				Timestamp: time.Now(),
				User:      getUser(r),
				Action:    "network.disable_interface",
				Resource:  req.Interface,
				Result:    "error",
				SourceIP:  r.RemoteAddr,
				Details: map[string]interface{}{
					"error": err.Error(),
				},
			})
		}
		writeJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "failed to disable interface: " + err.Error(),
		})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			Timestamp: time.Now(),
			User:      getUser(r),
			Action:    "network.disable_interface",
			Resource:  req.Interface,
			Result:    "success",
			SourceIP:  r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    map[string]interface{}{"message": "interface disabled"},
	})
}

// ListListeningPorts handles GET /api/v1/network/ports
func (h *NetManagerHandlers) ListListeningPorts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	ports, err := h.manager.ListListeningPorts()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "failed to list ports: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    ports,
	})
}

// GetTrafficStats handles GET /api/v1/network/traffic
func (h *NetManagerHandlers) GetTrafficStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	stats, err := h.manager.GetTrafficStats()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{
			Success: false,
			Error:   "failed to get traffic stats: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    stats,
	})
}
