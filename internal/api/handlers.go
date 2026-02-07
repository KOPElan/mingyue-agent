package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/KOPElan/mingyue-agent/internal/audit"
)

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Version   string    `json:"version"`
}

type RegistrationInfo struct {
	AgentID   string    `json:"agent_id"`
	Hostname  string    `json:"hostname"`
	Version   string    `json:"version"`
	StartTime time.Time `json:"start_time"`
	APIURLs   []string  `json:"api_urls"`
}

func RegisterHTTPHandlers(mux *http.ServeMux, auditLogger *audit.Logger) {
	mux.HandleFunc("/api/v1/register", registrationHandler(auditLogger))
	mux.HandleFunc("/api/v1/status", statusHandler)
	mux.HandleFunc("/healthz", healthHandler)
}

// healthHandler godoc
// @Summary Health check endpoint
// @Description Returns the current health status of the agent
// @Tags health
// @Produce json
// @Success 200 {object} Response{data=HealthResponse}
// @Router /healthz [get]
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	health := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Version:   "1.0.0",
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: health})
}

// registrationHandler godoc
// @Summary Register agent with WebUI
// @Description Registers the agent and returns registration information
// @Tags registration
// @Accept json
// @Produce json
// @Success 200 {object} Response{data=RegistrationInfo}
// @Failure 405 {object} Response
// @Router /register [post]
// @Security UserAuth
func registrationHandler(auditLogger *audit.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, Response{
				Success: false,
				Error:   "method not allowed",
			})
			return
		}

		hostname, _ := getHostname()

		info := RegistrationInfo{
			AgentID:   fmt.Sprintf("agent-%s-%d", hostname, time.Now().Unix()),
			Hostname:  hostname,
			Version:   "1.0.0",
			StartTime: time.Now(),
			APIURLs:   []string{"http://localhost:8080/api/v1"},
		}

		if auditLogger != nil {
			auditLogger.Log(r.Context(), &audit.Entry{
				Timestamp: time.Now(),
				User:      "webui",
				Action:    "register",
				Resource:  "agent",
				Result:    "success",
				SourceIP:  r.RemoteAddr,
				Details: map[string]interface{}{
					"agent_id": info.AgentID,
					"hostname": info.Hostname,
				},
			})
		}

		writeJSON(w, http.StatusOK, Response{Success: true, Data: info})
	}
}

// statusHandler godoc
// @Summary Get agent status
// @Description Returns the current status and uptime of the agent
// @Tags status
// @Produce json
// @Success 200 {object} Response
// @Failure 405 {object} Response
// @Router /status [get]
func statusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{
			Success: false,
			Error:   "method not allowed",
		})
		return
	}

	hostname, _ := getHostname()

	status := map[string]interface{}{
		"hostname": hostname,
		"uptime":   time.Since(time.Now()).Seconds(),
		"status":   "running",
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: status})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func getHostname() (string, error) {
	return "localhost", nil
}
