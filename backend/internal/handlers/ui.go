package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
	gorbac "github.com/mikespook/gorbac/v3"

	"github.com/mikespook/unidb-mcp/internal/database"
	apprbac "github.com/mikespook/unidb-mcp/internal/rbac"
	"github.com/mikespook/unidb-mcp/internal/store"
)

const sessionCookieName = "ui_session"
const sessionDuration = 24 * time.Hour

// UIHandler handles web UI requests
type UIHandler struct {
	store         *store.Store
	manager       *database.DriverManager
	frontendPath  string
	rbac          *gorbac.RBAC[string]
	bridgeManager *BridgeManager
}

// NewUIHandler creates a new UI handler
func NewUIHandler(s *store.Store, m *database.DriverManager, frontendPath string, r *gorbac.RBAC[string], bm *BridgeManager) *UIHandler {
	return &UIHandler{
		store:         s,
		manager:       m,
		frontendPath:  frontendPath,
		rbac:          r,
		bridgeManager: bm,
	}
}

func (h *UIHandler) requirePerm(w http.ResponseWriter, r *http.Request, perm string) bool {
	u, err := h.sessionUser(r)
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return false
	}
	isAdmin, _ := h.store.IsUserAdmin(u.ID)
	role := "member"
	if isAdmin {
		role = "admin"
	}
	if !apprbac.IsGranted(h.rbac, role, perm) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return false
	}
	return true
}

// sessionUser retrieves the User associated with the current session cookie.
func (h *UIHandler) sessionUser(r *http.Request) (*store.User, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, err
	}
	if !h.store.ValidateSession(cookie.Value) {
		return nil, sql.ErrNoRows
	}
	userID, err := h.store.GetSessionUser(cookie.Value)
	if err != nil {
		return nil, err
	}
	return h.store.GetUser(userID)
}

// SessionMiddleware checks the session cookie for UI routes.
func (h *UIHandler) SessionMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := h.sessionUser(r); err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// Me returns session info if the session is valid.
func (h *UIHandler) Me(w http.ResponseWriter, r *http.Request) {
	u, err := h.sessionUser(r)
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	initAdminID, _ := h.store.GetInitialUserID()
	isAdmin, _ := h.store.IsUserAdmin(u.ID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": true,
		"username":      u.Username,
		"init_admin_id": initAdminID,
		"is_admin":      isAdmin,
	})
}

// Login verifies username+password and sets a session cookie.
func (h *UIHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		http.Error(w, `{"error":"username and password required"}`, http.StatusBadRequest)
		return
	}

	u, err := h.store.GetUserByUsername(req.Username)
	if err != nil {
		// Don't distinguish "not found" from "wrong password"
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	// Generate session token
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	token := base64.RawURLEncoding.EncodeToString(buf)

	if err := h.store.CreateSessionWithUser(token, time.Now().Add(sessionDuration), u.ID); err != nil {
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
		_ = h.store.DeleteSessionWithUser(cookie.Value)
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

// ChangePassword updates the current user's password (requires valid session).
func (h *UIHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	u, err := h.sessionUser(r)
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Current == "" || req.New == "" {
		http.Error(w, `{"error":"current and new password required"}`, http.StatusBadRequest)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Current)); err != nil {
		http.Error(w, `{"error":"current password is incorrect"}`, http.StatusUnauthorized)
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.New), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}

	if err := h.store.UpdateUserPassword(u.ID, string(newHash)); err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// Index serves the Vite-built SPA entry point
func (h *UIHandler) Index(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, h.frontendPath+"/index.html")
}

// ListDSNs returns DSN configurations. Admins see all; members see only DSNs in their teams.
func (h *UIHandler) ListDSNs(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, apprbac.PermDSNRead) {
		return
	}
	u, _ := h.sessionUser(r)
	isAdmin, _ := h.store.IsUserAdmin(u.ID)
	var dsns []*store.DSN
	var err error
	if isAdmin {
		dsns, err = h.store.List()
	} else {
		dsns, err = h.store.ListDSNsForUser(u.ID)
	}
	if err != nil {
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	for _, dsn := range dsns {
		if dsn.Driver == "sqlite-bridge" {
			dsn.Connected = h.bridgeManager.IsConnected(dsn.Name)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dsns": dsns,
	})
}

// CreateDSN adds a new DSN configuration
func (h *UIHandler) CreateDSN(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, apprbac.PermDSNWrite) {
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
	if !h.requirePerm(w, r, apprbac.PermDSNWrite) {
		return
	}
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
	if !h.requirePerm(w, r, apprbac.PermDSNDelete) {
		return
	}
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
	if !h.requirePerm(w, r, apprbac.PermDSNTest) {
		return
	}
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

	if dsn.Driver == "sqlite-bridge" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "connection test not supported for sqlite-bridge",
		})
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
