package handlers

import (
	"encoding/json"
	"net/http"

	gorbac "github.com/mikespook/gorbac/v3"

	apprbac "github.com/mikespook/unidb-mcp/internal/rbac"
	"github.com/mikespook/unidb-mcp/internal/store"
)

// TeamHandler handles team management API endpoints.
type TeamHandler struct {
	store *store.Store
	rbac  *gorbac.RBAC[string]
}

// NewTeamHandler creates a new TeamHandler.
func NewTeamHandler(s *store.Store, r *gorbac.RBAC[string]) *TeamHandler {
	return &TeamHandler{store: s, rbac: r}
}

func (h *TeamHandler) currentUserRole(r *http.Request) (string, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", err
	}
	userID, err := h.store.GetSessionUser(cookie.Value)
	if err != nil {
		return "", err
	}
	isAdmin, err := h.store.IsUserAdmin(userID)
	if err != nil {
		return "", err
	}
	if isAdmin {
		return "admin", nil
	}
	return "member", nil
}

func (h *TeamHandler) requirePerm(w http.ResponseWriter, r *http.Request, perm string) bool {
	role, err := h.currentUserRole(r)
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return false
	}
	if !apprbac.IsGranted(h.rbac, role, perm) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return false
	}
	return true
}

// List returns all teams (GET /api/teams).
func (h *TeamHandler) List(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, apprbac.PermTeamRead) {
		return
	}
	teams, err := h.store.ListTeams()
	if err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"teams": teams})
}

// Create adds a new team (POST /api/teams).
func (h *TeamHandler) Create(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, apprbac.PermTeamWrite) {
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, `{"error":"name required"}`, http.StatusBadRequest)
		return
	}
	team, err := h.store.CreateTeam(req.Name)
	if err != nil {
		if err == store.ErrTeamExists {
			http.Error(w, `{"error":"team already exists"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "team": team})
}

// Delete removes a team (DELETE /api/teams?id=...). The default team cannot be deleted.
func (h *TeamHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, apprbac.PermTeamDelete) {
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, `{"error":"id required"}`, http.StatusBadRequest)
		return
	}
	team, err := h.store.GetTeam(id)
	if err != nil {
		if err == store.ErrTeamNotFound {
			http.Error(w, `{"error":"team not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	if team.Name == "admin" {
		http.Error(w, `{"error":"cannot delete the admin team"}`, http.StatusForbidden)
		return
	}
	if err := h.store.DeleteTeam(id); err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// GetUsers returns users in a team (GET /api/teams/{id}/users).
func (h *TeamHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, apprbac.PermTeamRead) {
		return
	}
	teamID := r.PathValue("id")
	users, err := h.store.GetTeamUsers(teamID)
	if err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	if users == nil {
		users = []*store.User{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"users": users})
}

// GetDSNs returns DSNs assigned to a team (GET /api/teams/{id}/dsns).
func (h *TeamHandler) GetDSNs(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, apprbac.PermTeamRead) {
		return
	}
	teamID := r.PathValue("id")
	dsns, err := h.store.GetTeamDSNs(teamID)
	if err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	if dsns == nil {
		dsns = []*store.DSN{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"dsns": dsns})
}

// AddUser assigns a user to a team (POST /api/teams/{id}/users).
func (h *TeamHandler) AddUser(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, apprbac.PermTeamWrite) {
		return
	}
	teamID := r.PathValue("id")
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		http.Error(w, `{"error":"user_id required"}`, http.StatusBadRequest)
		return
	}
	if err := h.store.AddUserToTeam(req.UserID, teamID); err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// RemoveUser removes a user from a team (DELETE /api/teams/{id}/users).
func (h *TeamHandler) RemoveUser(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, apprbac.PermTeamWrite) {
		return
	}
	teamID := r.PathValue("id")
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		http.Error(w, `{"error":"user_id required"}`, http.StatusBadRequest)
		return
	}
	if err := h.store.RemoveUserFromTeam(req.UserID, teamID); err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// requireNotAdminTeam returns false and writes a 400 error if the team is the admin team.
func (h *TeamHandler) requireNotAdminTeam(w http.ResponseWriter, teamID string) bool {
	team, err := h.store.GetTeam(teamID)
	if err != nil {
		http.Error(w, `{"error":"team not found"}`, http.StatusNotFound)
		return false
	}
	if team.Name == "admin" {
		http.Error(w, `{"error":"admin team has access to all DSNs and cannot be modified"}`, http.StatusBadRequest)
		return false
	}
	return true
}

// AddDSN assigns a DSN to a team (POST /api/teams/{id}/dsns).
func (h *TeamHandler) AddDSN(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, apprbac.PermTeamWrite) {
		return
	}
	teamID := r.PathValue("id")
	if !h.requireNotAdminTeam(w, teamID) {
		return
	}
	var req struct {
		DSNID string `json:"dsn_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DSNID == "" {
		http.Error(w, `{"error":"dsn_id required"}`, http.StatusBadRequest)
		return
	}
	if err := h.store.AddDSNToTeam(req.DSNID, teamID); err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// RemoveDSN removes a DSN from a team (DELETE /api/teams/{id}/dsns).
func (h *TeamHandler) RemoveDSN(w http.ResponseWriter, r *http.Request) {
	if !h.requirePerm(w, r, apprbac.PermTeamWrite) {
		return
	}
	teamID := r.PathValue("id")
	if !h.requireNotAdminTeam(w, teamID) {
		return
	}
	var req struct {
		DSNID string `json:"dsn_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DSNID == "" {
		http.Error(w, `{"error":"dsn_id required"}`, http.StatusBadRequest)
		return
	}
	if err := h.store.RemoveDSNFromTeam(req.DSNID, teamID); err != nil {
		http.Error(w, `{"error":"server error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
