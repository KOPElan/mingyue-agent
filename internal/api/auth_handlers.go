package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/KOPElan/mingyue-agent/internal/audit"
	"github.com/KOPElan/mingyue-agent/internal/auth"
)

type AuthHandlers struct {
	auth  *auth.AuthManager
	audit *audit.Logger
}

func NewAuthHandlers(authMgr *auth.AuthManager, auditLogger *audit.Logger) *AuthHandlers {
	return &AuthHandlers{
		auth:  authMgr,
		audit: auditLogger,
	}
}

type CreateTokenRequest struct {
	UserID    string   `json:"user_id"`
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	ExpiresIn int      `json:"expires_in"` // seconds
}

type CreateSessionRequest struct {
	UserID string `json:"user_id"`
}

// CreateToken godoc
// @Summary Create API token
// @Description Creates a new API token for authentication
// @Tags auth
// @Accept json
// @Produce json
// @Param body body CreateTokenRequest true "Token request"
// @Success 200 {object} Response{data=auth.Token}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /auth/tokens/create [post]
// @Security UserAuth
func (h *AuthHandlers) CreateToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	var req CreateTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "invalid request body"})
		return
	}

	expiresAt := time.Now().Add(time.Duration(req.ExpiresIn) * time.Second)
	if req.ExpiresIn == 0 {
		expiresAt = time.Now().Add(365 * 24 * time.Hour) // Default 1 year
	}

	token, err := h.auth.CreateToken(req.UserID, req.Name, req.Scopes, expiresAt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			User:     getUser(r),
			Action:   "create_token",
			Resource: "auth",
			Result:   "success",
			SourceIP: r.RemoteAddr,
			Details:  map[string]interface{}{"user_id": req.UserID, "token_name": req.Name},
		})
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: token})
}

// ListTokens godoc
// @Summary List API tokens
// @Description Lists all API tokens for a user
// @Tags auth
// @Produce json
// @Param user_id query string true "User ID"
// @Success 200 {object} Response{data=[]auth.Token}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /auth/tokens [get]
// @Security UserAuth
func (h *AuthHandlers) ListTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "user_id required"})
		return
	}

	tokens, err := h.auth.ListTokens(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: tokens})
}

// RevokeToken godoc
// @Summary Revoke API token
// @Description Revokes an API token
// @Tags auth
// @Produce json
// @Param id query string true "Token ID"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /auth/tokens/revoke [delete]
// @Security UserAuth
func (h *AuthHandlers) RevokeToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	tokenID := r.URL.Query().Get("id")
	if tokenID == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "token ID required"})
		return
	}

	if err := h.auth.RevokeToken(tokenID); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			User:     getUser(r),
			Action:   "revoke_token",
			Resource: tokenID,
			Result:   "success",
			SourceIP: r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, Response{Success: true})
}

// CreateSession godoc
// @Summary Create session
// @Description Creates a new user session
// @Tags auth
// @Accept json
// @Produce json
// @Param body body CreateSessionRequest true "Session request"
// @Success 200 {object} Response{data=auth.Session}
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /auth/sessions/create [post]
func (h *AuthHandlers) CreateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	var req CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "invalid request body"})
		return
	}

	expiresAt := time.Now().Add(24 * time.Hour) // 24 hour session
	session, err := h.auth.CreateSession(req.UserID, r.RemoteAddr, r.UserAgent(), expiresAt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			User:     getUser(r),
			Action:   "create_session",
			Resource: "auth",
			Result:   "success",
			SourceIP: r.RemoteAddr,
			Details:  map[string]interface{}{"user_id": req.UserID},
		})
	}

	writeJSON(w, http.StatusOK, Response{Success: true, Data: session})
}

// RevokeSession godoc
// @Summary Revoke session
// @Description Revokes a user session
// @Tags auth
// @Produce json
// @Param id query string true "Session ID"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /auth/sessions/revoke [delete]
// @Security UserAuth
func (h *AuthHandlers) RevokeSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, Response{Success: false, Error: "method not allowed"})
		return
	}

	sessionID := r.URL.Query().Get("id")
	if sessionID == "" {
		writeJSON(w, http.StatusBadRequest, Response{Success: false, Error: "session ID required"})
		return
	}

	if err := h.auth.RevokeSession(sessionID); err != nil {
		writeJSON(w, http.StatusInternalServerError, Response{Success: false, Error: err.Error()})
		return
	}

	if h.audit != nil {
		h.audit.Log(r.Context(), &audit.Entry{
			User:     getUser(r),
			Action:   "revoke_session",
			Resource: sessionID,
			Result:   "success",
			SourceIP: r.RemoteAddr,
		})
	}

	writeJSON(w, http.StatusOK, Response{Success: true})
}
