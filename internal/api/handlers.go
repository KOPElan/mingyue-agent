package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/KOPElan/mingyue-agent/internal/audit"
	"github.com/KOPElan/mingyue-agent/internal/config"
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

func RegisterHTTPHandlers(mux *http.ServeMux, auditLogger *audit.Logger, cfg *config.Config) {
	mux.HandleFunc("/api/v1/register", registrationHandler(auditLogger, cfg))
	mux.HandleFunc("/api/v1/status", statusHandler)
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
func registrationHandler(auditLogger *audit.Logger, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, Response{
				Success: false,
				Error:   "method not allowed",
			})
			return
		}

		hostname, _ := getHostname()
		apiURLs := buildAPIURLs(cfg, hostname)

		info := RegistrationInfo{
			AgentID:   fmt.Sprintf("agent-%s-%d", hostname, time.Now().Unix()),
			Hostname:  hostname,
			Version:   "1.0.0",
			StartTime: time.Now(),
			APIURLs:   apiURLs,
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
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		return "localhost", err
	}
	return hostname, nil
}

func buildAPIURLs(cfg *config.Config, hostname string) []string {
	if cfg == nil || !cfg.API.EnableHTTP {
		return nil
	}

	host := cfg.Server.ListenAddr
	if host == "" || host == "0.0.0.0" || host == "::" {
		if hostname != "" {
			host = hostname
		} else {
			host = "localhost"
		}
	}

	scheme := "http"
	if cfg.API.TLSCert != "" && cfg.API.TLSKey != "" {
		scheme = "https"
	}

	return []string{fmt.Sprintf("%s://%s:%d/api/v1", scheme, host, cfg.Server.HTTPPort)}
}
