package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/mikespook/unidb-mcp/internal/config"
	"github.com/mikespook/unidb-mcp/internal/database"
	"github.com/mikespook/unidb-mcp/internal/store"
)

const sessionCookieName = "ui_session"
const sessionDuration = 24 * time.Hour

// UIHandler handles web UI requests
type UIHandler struct {
	store     *store.Store
	manager   *database.DriverManager
	uiPassCfg *config.UIPasswordConfig
}

// NewUIHandler creates a new UI handler
func NewUIHandler(s *store.Store, m *database.DriverManager, p *config.UIPasswordConfig) *UIHandler {
	return &UIHandler{
		store:     s,
		manager:   m,
		uiPassCfg: p,
	}
}

// SessionMiddleware checks the session cookie for UI routes. Passes through if auth is disabled.
func (h *UIHandler) SessionMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.uiPassCfg.Disabled {
			next(w, r)
			return
		}
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || !h.store.ValidateSession(cookie.Value) {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// Me returns 200 if the session is valid, 401 otherwise.
func (h *UIHandler) Me(w http.ResponseWriter, r *http.Request) {
	if h.uiPassCfg.Disabled {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"authenticated": true, "disabled": true})
		return
	}
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || !h.store.ValidateSession(cookie.Value) {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"authenticated": true})
}

// Login verifies the password and sets a session cookie.
func (h *UIHandler) Login(w http.ResponseWriter, r *http.Request) {
	if h.uiPassCfg.Disabled {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" {
		http.Error(w, `{"error":"password required"}`, http.StatusBadRequest)
		return
	}

	hash, err := h.store.GetSetting("ui_password_hash")
	if err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		http.Error(w, `{"error":"invalid password"}`, http.StatusUnauthorized)
		return
	}

	// Generate session token
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	token := base64.RawURLEncoding.EncodeToString(buf)

	if err := h.store.CreateSession(token, time.Now().Add(sessionDuration)); err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(sessionDuration.Seconds()),
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// Logout invalidates the session cookie.
func (h *UIHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		_ = h.store.DeleteSession(cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:   sessionCookieName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// ChangePassword updates the UI password (requires valid session).
func (h *UIHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Current == "" || req.New == "" {
		http.Error(w, `{"error":"current and new password required"}`, http.StatusBadRequest)
		return
	}

	hash, err := h.store.GetSetting("ui_password_hash")
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"password not set"}`, http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Current)); err != nil {
		http.Error(w, `{"error":"current password is incorrect"}`, http.StatusUnauthorized)
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.New), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}

	if err := h.store.SetSetting("ui_password_hash", string(newHash)); err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// Index serves the main UI page
func (h *UIHandler) Index(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "frontend/index.html")
}

// AppJS serves the app.js file
func (h *UIHandler) AppJS(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "frontend/app.js")
}

// ListDSNs returns all DSN configurations
func (h *UIHandler) ListDSNs(w http.ResponseWriter, r *http.Request) {
	dsns, err := h.store.List()
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dsns": dsns,
	})
}

// CreateDSN adds a new DSN configuration
func (h *UIHandler) CreateDSN(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string `json:"name"`
		Driver string `json:"driver"`
		DSN    string `json:"dsn"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Driver == "" || req.DSN == "" {
		http.Error(w, `{"error": "name, driver, and dsn are required"}`, http.StatusBadRequest)
		return
	}

	dsn, err := h.store.Create(req.Name, req.Driver, req.DSN)
	if err != nil {
		if err == store.ErrDSNExists {
			http.Error(w, `{"error": "DSN name already exists"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"dsn":     dsn,
	})
}

// UpdateDSN modifies an existing DSN configuration
func (h *UIHandler) UpdateDSN(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, `{"error": "id is required"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Name   string `json:"name"`
		Driver string `json:"driver"`
		DSN    string `json:"dsn"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Driver == "" || req.DSN == "" {
		http.Error(w, `{"error": "name, driver, and dsn are required"}`, http.StatusBadRequest)
		return
	}

	dsn, err := h.store.Update(id, req.Name, req.Driver, req.DSN)
	if err != nil {
		if err == store.ErrDSNNotFound {
			http.Error(w, `{"error": "DSN not found"}`, http.StatusNotFound)
			return
		}
		if err == store.ErrDSNExists {
			http.Error(w, `{"error": "DSN name already exists"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"dsn":     dsn,
	})
}

// DeleteDSN removes a DSN configuration
func (h *UIHandler) DeleteDSN(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, `{"error": "id is required"}`, http.StatusBadRequest)
		return
	}

	if err := h.store.Delete(id); err != nil {
		if err == store.ErrDSNNotFound {
			http.Error(w, `{"error": "DSN not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}

// TestDSN tests a DSN connection
func (h *UIHandler) TestDSN(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}
	if id == "" {
		http.Error(w, `{"error": "id is required"}`, http.StatusBadRequest)
		return
	}

	dsn, err := h.store.Get(id)
	if err != nil {
		if err == store.ErrDSNNotFound {
			http.Error(w, `{"error": "DSN not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	start := time.Now()
	err = database.TestConnection(dsn.Driver, dsn.DSN)
	duration := time.Since(start)

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"duration": duration.String(),
	})
}
