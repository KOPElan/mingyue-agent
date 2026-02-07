package api

import (
	"net/http"
	"time"

	"github.com/KOPElan/mingyue-agent/internal/audit"
	"github.com/KOPElan/mingyue-agent/internal/monitor"
)

type MonitorAPI struct {
	monitor *monitor.Monitor
	audit   *audit.Logger
}

func NewMonitorAPI(mon *monitor.Monitor, auditLogger *audit.Logger) *MonitorAPI {
	return &MonitorAPI{
		monitor: mon,
		audit:   auditLogger,
	}
}

func (api *MonitorAPI) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/monitor/stats", api.handleStats)
	mux.HandleFunc("/api/v1/monitor/health", api.handleHealth)
	mux.HandleFunc("/healthz", api.handleHealthz)
}

func (api *MonitorAPI) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	stats, err := api.monitor.GetStats()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: stats})
}

func (api *MonitorAPI) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	healthy := api.monitor.IsHealthy()
	status := "healthy"
	if !healthy {
		status = "unhealthy"
	}

	data := map[string]interface{}{
		"status":    status,
		"healthy":   healthy,
		"timestamp": time.Now(),
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: data})
}

func (api *MonitorAPI) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	healthy := api.monitor.IsHealthy()

	resp := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Version:   "1.0.0",
	}

	if !healthy {
		resp.Status = "degraded"
		w.WriteHeader(http.StatusServiceUnavailable)
		writeJSON(w, http.StatusServiceUnavailable, Response{Success: false, Data: resp})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: resp})
}
