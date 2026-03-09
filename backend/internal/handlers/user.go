package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"
	gorbac "github.com/mikespook/gorbac/v3"

	apprbac "github.com/mikespook/unidb-mcp/internal/rbac"
	"github.com/mikespook/unidb-mcp/internal/store"
)

// UserHandler handles user management API endpoints.
type UserHandler struct {
	store *store.Store
	rbac  *gorbac.RBAC[string]
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(s *store.Store, r *gorbac.RBAC[string]) *UserHandler {
	return &UserHandler{store: s, rbac: r}
}

// currentUserRole extracts the session user's role from the request cookie.
func (h *UserHandler) currentUser(r *http.Request) (*store.User, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return nil, err
	}
	userID, err := h.store.GetSessionUser(cookie.Value)
	if err != nil {
		return nil, err
	}
	return h.store.GetUser(userID)
}

func (h *UserHandler) requirePerm(w http.ResponseWriter, r *http.Request, perm string) (*store.User, bool) {
	u, err := h.currentUser(r)
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return nil, false
	}
	isAdmin, _ := h.store.IsUserAdmin(u.ID)
	role := "member"
	if isAdmin {
		role = "admin"
	}
	if !apprbac.IsGranted(h.rbac, role, perm) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return nil, false
	}
	return u, true
}

// List returns all users (GET /api/users).
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requirePerm(w, r, apprbac.PermUserRead); !ok {
		return
	}
	users, err := h.store.ListUsers()
	if err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"users": users})
}

// Create adds a new user (POST /api/users). Returns jwt_secret once.
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requirePerm(w, r, apprbac.PermUserWrite); !ok {
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		http.Error(w, `{"error":"username and password required"}`, http.StatusBadRequest)
		return
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	jwtSecret := base64.RawURLEncoding.EncodeToString(buf)

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}

	user, err := h.store.CreateUser(req.Username, string(hash), jwtSecret)
	if err != nil {
		if err == store.ErrUserExists {
			http.Error(w, `{"error":"username already exists"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"user":       user,
		"jwt_secret": jwtSecret,
	})
}

// Update changes a user's password (PUT /api/users?id=...).
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requirePerm(w, r, apprbac.PermUserWrite); !ok {
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, `{"error":"id required"}`, http.StatusBadRequest)
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Password == "" {
		http.Error(w, `{"error":"password required"}`, http.StatusBadRequest)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	if err := h.store.UpdateUserPassword(id, string(hash)); err != nil {
		if err == store.ErrUserNotFound {
			http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// Delete removes a user (DELETE /api/users?id=...). Cannot delete self.
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.requirePerm(w, r, apprbac.PermUserDelete)
	if !ok {
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, `{"error":"id required"}`, http.StatusBadRequest)
		return
	}
	if id == caller.ID {
		http.Error(w, `{"error":"cannot delete yourself"}`, http.StatusBadRequest)
		return
	}
	// Block deletion of the initial user
	initialID, err := h.store.GetInitialUserID()
	if err == nil && id == initialID {
		http.Error(w, `{"error":"cannot delete the initial admin user"}`, http.StatusBadRequest)
		return
	}
	if err := h.store.DeleteUser(id); err != nil {
		if err == store.ErrUserNotFound {
			http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// GetJWTSecret returns the jwt_secret for a user (GET /api/users/{id}/jwt-secret).
func (h *UserHandler) GetJWTSecret(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requirePerm(w, r, apprbac.PermUserRead); !ok {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		id = r.URL.Query().Get("id")
	}
	if id == "" {
		http.Error(w, `{"error":"id required"}`, http.StatusBadRequest)
		return
	}
	secret, err := h.store.GetUserJWTSecret(id)
	if err != nil {
		if err == store.ErrUserNotFound {
			http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"jwt_secret": secret})
}
