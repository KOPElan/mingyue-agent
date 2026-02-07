package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

// Token represents an API token
type Token struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token,omitempty"` // Only shown on creation
	Hash      string    `json:"-"`
	Name      string    `json:"name"`
	Scopes    []string  `json:"scopes"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
}

// Session represents a user session
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
}

// AuthManager handles authentication and authorization
type AuthManager struct {
	db       *sql.DB
	mu       sync.RWMutex
	tokens   map[string]*Token
	sessions map[string]*Session
}

// Config holds auth configuration
type Config struct {
	DBPath        string
	TokenExpiry   time.Duration
	SessionExpiry time.Duration
	RequireAuth   bool
	AllowedIPs    []string
	EnableMTLS    bool
	MTLSCertPath  string
	MTLSKeyPath   string
	MTLSCAPath    string
}

// New creates a new AuthManager
func New(config Config) (*AuthManager, error) {
	if config.DBPath == "" {
		config.DBPath = "/var/lib/mingyue-agent/auth.db"
	}

	db, err := sql.Open("sqlite3", config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	am := &AuthManager{
		db:       db,
		tokens:   make(map[string]*Token),
		sessions: make(map[string]*Session),
	}

	if err := am.initDB(); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize database: %w", err)
	}

	// Load tokens
	if err := am.loadTokens(); err != nil {
		db.Close()
		return nil, fmt.Errorf("load tokens: %w", err)
	}

	return am, nil
}

func (am *AuthManager) initDB() error {
	schema := `
	CREATE TABLE IF NOT EXISTS api_tokens (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		token_hash TEXT NOT NULL,
		name TEXT,
		scopes TEXT,
		expires_at INTEGER,
		created_at INTEGER,
		last_used INTEGER
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		token_hash TEXT NOT NULL,
		expires_at INTEGER,
		created_at INTEGER,
		ip TEXT,
		user_agent TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_token_hash ON api_tokens(token_hash);
	CREATE INDEX IF NOT EXISTS idx_session_token ON sessions(token_hash);
	CREATE INDEX IF NOT EXISTS idx_user_id ON api_tokens(user_id);
	`

	_, err := am.db.Exec(schema)
	return err
}

func (am *AuthManager) loadTokens() error {
	am.mu.Lock()
	defer am.mu.Unlock()

	rows, err := am.db.Query(`
		SELECT id, user_id, token_hash, name, scopes, expires_at, created_at, last_used
		FROM api_tokens
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var token Token
		var scopesStr string
		var expiresAt, createdAt, lastUsed int64

		err := rows.Scan(&token.ID, &token.UserID, &token.Hash, &token.Name,
			&scopesStr, &expiresAt, &createdAt, &lastUsed)
		if err != nil {
			continue
		}

		token.ExpiresAt = time.Unix(expiresAt, 0)
		token.CreatedAt = time.Unix(createdAt, 0)
		token.LastUsed = time.Unix(lastUsed, 0)

		// Parse scopes (simplified)
		if scopesStr != "" {
			token.Scopes = []string{scopesStr}
		}

		am.tokens[token.Hash] = &token
	}

	return rows.Err()
}

// CreateToken creates a new API token
func (am *AuthManager) CreateToken(userID, name string, scopes []string, expiresAt time.Time) (*Token, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Generate token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	tokenStr := base64.URLEncoding.EncodeToString(tokenBytes)

	// Hash token
	hash, err := bcrypt.GenerateFromPassword([]byte(tokenStr), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash token: %w", err)
	}

	token := &Token{
		ID:        generateID(),
		UserID:    userID,
		Token:     tokenStr, // Only shown on creation
		Hash:      string(hash),
		Name:      name,
		Scopes:    scopes,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	scopesStr := ""
	if len(scopes) > 0 {
		scopesStr = scopes[0]
	}

	_, err = am.db.Exec(`
		INSERT INTO api_tokens (id, user_id, token_hash, name, scopes, expires_at, created_at, last_used)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, token.ID, token.UserID, token.Hash, token.Name, scopesStr,
		token.ExpiresAt.Unix(), token.CreatedAt.Unix(), token.LastUsed.Unix())
	if err != nil {
		return nil, err
	}

	am.tokens[token.Hash] = token
	return token, nil
}

// ValidateToken validates an API token
func (am *AuthManager) ValidateToken(tokenStr string) (*Token, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	// Find matching token
	for _, token := range am.tokens {
		if err := bcrypt.CompareHashAndPassword([]byte(token.Hash), []byte(tokenStr)); err == nil {
			// Check expiration
			if time.Now().After(token.ExpiresAt) {
				return nil, fmt.Errorf("token expired")
			}

			// Update last used
			go am.updateTokenLastUsed(token.ID)

			return token, nil
		}
	}

	return nil, fmt.Errorf("invalid token")
}

func (am *AuthManager) updateTokenLastUsed(tokenID string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	_, err := am.db.Exec("UPDATE api_tokens SET last_used = ? WHERE id = ?", time.Now().Unix(), tokenID)
	if err == nil {
		for _, token := range am.tokens {
			if token.ID == tokenID {
				token.LastUsed = time.Now()
				break
			}
		}
	}
}

// RevokeToken revokes an API token
func (am *AuthManager) RevokeToken(tokenID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	_, err := am.db.Exec("DELETE FROM api_tokens WHERE id = ?", tokenID)
	if err != nil {
		return err
	}

	// Remove from cache
	for hash, token := range am.tokens {
		if token.ID == tokenID {
			delete(am.tokens, hash)
			break
		}
	}

	return nil
}

// CreateSession creates a new user session
func (am *AuthManager) CreateSession(userID, ip, userAgent string, expiresAt time.Time) (*Session, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Generate session token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	tokenStr := base64.URLEncoding.EncodeToString(tokenBytes)

	// Hash token
	hash, err := bcrypt.GenerateFromPassword([]byte(tokenStr), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash token: %w", err)
	}

	session := &Session{
		ID:        generateID(),
		UserID:    userID,
		Token:     tokenStr,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
		IP:        ip,
		UserAgent: userAgent,
	}

	_, err = am.db.Exec(`
		INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at, ip, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, session.ID, session.UserID, string(hash), session.ExpiresAt.Unix(),
		session.CreatedAt.Unix(), session.IP, session.UserAgent)
	if err != nil {
		return nil, err
	}

	am.sessions[string(hash)] = session
	return session, nil
}

// ValidateSession validates a session token
func (am *AuthManager) ValidateSession(tokenStr string) (*Session, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	for hash, session := range am.sessions {
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(tokenStr)); err == nil {
			if time.Now().After(session.ExpiresAt) {
				return nil, fmt.Errorf("session expired")
			}
			return session, nil
		}
	}

	// Try to load from database
	rows, err := am.db.Query("SELECT id, user_id, token_hash, expires_at, created_at, ip, user_agent FROM sessions")
	if err != nil {
		return nil, fmt.Errorf("invalid session")
	}
	defer rows.Close()

	for rows.Next() {
		var session Session
		var tokenHash string
		var expiresAt, createdAt int64

		err := rows.Scan(&session.ID, &session.UserID, &tokenHash, &expiresAt, &createdAt, &session.IP, &session.UserAgent)
		if err != nil {
			continue
		}

		if err := bcrypt.CompareHashAndPassword([]byte(tokenHash), []byte(tokenStr)); err == nil {
			session.ExpiresAt = time.Unix(expiresAt, 0)
			session.CreatedAt = time.Unix(createdAt, 0)

			if time.Now().After(session.ExpiresAt) {
				return nil, fmt.Errorf("session expired")
			}

			return &session, nil
		}
	}

	return nil, fmt.Errorf("invalid session")
}

// RevokeSession revokes a session
func (am *AuthManager) RevokeSession(sessionID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	_, err := am.db.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	if err != nil {
		return err
	}

	// Remove from cache
	for hash, session := range am.sessions {
		if session.ID == sessionID {
			delete(am.sessions, hash)
			break
		}
	}

	return nil
}

// ListTokens lists all API tokens for a user
func (am *AuthManager) ListTokens(userID string) ([]*Token, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var tokens []*Token
	for _, token := range am.tokens {
		if token.UserID == userID {
			tokens = append(tokens, token)
		}
	}

	return tokens, nil
}

// Close closes the auth manager
func (am *AuthManager) Close() error {
	return am.db.Close()
}

// CompareSecure performs constant-time string comparison
func CompareSecure(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
