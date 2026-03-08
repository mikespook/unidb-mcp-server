package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/mikespook/unidb-mcp/internal/store"
)

// InitHandler handles the first-run initialization flow.
type InitHandler struct {
	store *store.Store
}

// NewInitHandler creates a new InitHandler.
func NewInitHandler(s *store.Store) *InitHandler {
	return &InitHandler{store: s}
}

// Status returns whether the system has been initialized (GET /api/ui/init-status).
func (h *InitHandler) Status(w http.ResponseWriter, r *http.Request) {
	initialized, err := h.store.IsInitialized()
	if err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"initialized": initialized})
}

// Setup creates the admin user and default team (POST /init).
// Returns 409 if already initialized.
func (h *InitHandler) Setup(w http.ResponseWriter, r *http.Request) {
	initialized, err := h.store.IsInitialized()
	if err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	if initialized {
		http.Error(w, `{"error":"already initialized"}`, http.StatusConflict)
		return
	}

	var req struct {
		Username  string `json:"username"`
		Password  string `json:"password"`
		JWTSecret string `json:"jwt_secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		http.Error(w, `{"error":"username and password required"}`, http.StatusBadRequest)
		return
	}

	// Use client-provided JWT secret or generate one
	jwtSecret := req.JWTSecret
	if jwtSecret == "" {
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
			return
		}
		jwtSecret = base64.RawURLEncoding.EncodeToString(buf)
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}

	user, err := h.store.CreateUser(req.Username, string(hash), jwtSecret, "admin")
	if err != nil {
		if err == store.ErrUserExists {
			http.Error(w, `{"error":"username already exists"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}

	// Create default team and add admin to it
	team, err := h.store.EnsureDefaultTeam()
	if err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	_ = h.store.AddUserToTeam(user.ID, team.ID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"username":   user.Username,
		"jwt_secret": jwtSecret,
		"created_at": user.CreatedAt.Format(time.RFC3339),
	})
}
