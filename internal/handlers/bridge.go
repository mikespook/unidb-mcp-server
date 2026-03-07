package handlers

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/mikespook/unidb-mcp/internal/store"
)

// BridgeManager manages active bridge connections
type BridgeManager struct {
	mu      sync.RWMutex
	store   *store.Store
	bridges map[string]*store.Bridge // keyed by bridge name (in-memory cache)
}

// NewBridgeManager creates a new bridge manager
func NewBridgeManager(s *store.Store) *BridgeManager {
	return &BridgeManager{
		store:   s,
		bridges: make(map[string]*store.Bridge),
	}
}

// Register adds a new bridge, or reconnects if name+secret already match
func (m *BridgeManager) Register(name, secret, bridgeType string) (*store.Bridge, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check in-memory cache first
	if existing, exists := m.bridges[name]; exists {
		if existing.Secret != secret {
			return nil, ErrInvalidSecret
		}
		return existing, nil
	}

	// Try to create in store
	bridge, err := m.store.CreateBridge(name, secret, bridgeType)
	if err != nil {
		if err != store.ErrBridgeExists {
			return nil, err
		}
		// Bridge already exists in store — verify secret and reconnect
		existing, err := m.store.GetBridgeByName(name)
		if err != nil {
			return nil, err
		}
		if existing.Secret != secret {
			return nil, ErrInvalidSecret
		}
		m.bridges[name] = existing
		return existing, nil
	}

	m.bridges[name] = bridge
	return bridge, nil
}

// Get retrieves a bridge by name
func (m *BridgeManager) Get(name string) (*store.Bridge, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bridge, exists := m.bridges[name]
	if !exists {
		// Try to load from store
		bridge, err := m.store.GetBridgeByName(name)
		if err != nil {
			return nil, ErrBridgeNotFound
		}
		return bridge, nil
	}

	return bridge, nil
}

// Authenticate verifies bridge credentials
func (m *BridgeManager) Authenticate(name, secret string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bridge, exists := m.bridges[name]
	if !exists {
		// Try to load from store
		var err error
		bridge, err = m.store.GetBridgeByName(name)
		if err != nil {
			return ErrBridgeNotFound
		}
	}

	if bridge.Secret != secret {
		return ErrInvalidSecret
	}

	return nil
}

// SetConnected updates the connection status
func (m *BridgeManager) SetConnected(name string, connected bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update in store
	if err := m.store.UpdateBridgeConnection(name, connected); err != nil {
		return err
	}

	// Update cache
	if bridge, exists := m.bridges[name]; exists {
		bridge.Connected = connected
		now := time.Now()
		if connected {
			bridge.ConnectedAt = &now
		} else {
			bridge.ConnectedAt = nil
		}
	}

	return nil
}

// List returns all bridges
func (m *BridgeManager) List() []*store.Bridge {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Load from store
	bridges, err := m.store.ListBridges()
	if err != nil {
		return nil
	}
	return bridges
}

// Update updates a bridge's name and secret
func (m *BridgeManager) Update(oldName, newName, secret string) (*store.Bridge, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	bridge, err := m.store.UpdateBridge(oldName, newName, secret)
	if err != nil {
		return nil, err
	}

	// Update cache: remove old key, add new
	delete(m.bridges, oldName)
	m.bridges[newName] = bridge
	return bridge, nil
}

// Delete removes a bridge
func (m *BridgeManager) Delete(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.store.DeleteBridge(name); err != nil {
		return err
	}

	delete(m.bridges, name)
	return nil
}

// Bridge errors
var (
	ErrBridgeNotFound = &BridgeError{"Bridge not found"}
	ErrBridgeExists   = &BridgeError{"Bridge already exists"}
	ErrInvalidSecret  = &BridgeError{"Invalid secret"}
)

// BridgeError represents a bridge-related error
type BridgeError struct {
	Message string
}

func (e *BridgeError) Error() string {
	return e.Message
}

// BridgeHandler handles bridge-related HTTP requests
type BridgeHandler struct {
	manager *BridgeManager
}

// NewBridgeHandler creates a new bridge handler
func NewBridgeHandler(m *BridgeManager) *BridgeHandler {
	return &BridgeHandler{
		manager: m,
	}
}

// RegisterRequest represents a bridge registration request
type RegisterRequest struct {
	Name   string `json:"name"`
	Secret string `json:"secret"`
	Type   string `json:"type"`
}

// Register handles bridge registration
func (h *BridgeHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Secret == "" || req.Type == "" {
		http.Error(w, `{"error": "name, secret, and type are required"}`, http.StatusBadRequest)
		return
	}

	bridge, err := h.manager.Register(req.Name, req.Secret, req.Type)
	if err != nil {
		if err == ErrInvalidSecret {
			http.Error(w, `{"error": "Invalid secret"}`, http.StatusUnauthorized)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"bridge": map[string]interface{}{
			"id":         bridge.ID,
			"name":       bridge.Name,
			"type":       bridge.Type,
			"created_at": bridge.CreatedAt,
		},
	})
}

// bridgeResponse is the JSON representation of a bridge, including the secret
type bridgeResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Secret      string     `json:"secret"`
	Type        string     `json:"type"`
	Connected   bool       `json:"connected"`
	ConnectedAt *time.Time `json:"connected_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// List handles listing all bridges
func (h *BridgeHandler) List(w http.ResponseWriter, r *http.Request) {
	bridges := h.manager.List()

	resp := make([]bridgeResponse, 0, len(bridges))
	for _, b := range bridges {
		resp = append(resp, bridgeResponse{
			ID:          b.ID,
			Name:        b.Name,
			Secret:      b.Secret,
			Type:        b.Type,
			Connected:   b.Connected,
			ConnectedAt: b.ConnectedAt,
			CreatedAt:   b.CreatedAt,
			UpdatedAt:   b.UpdatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"bridges": resp,
	})
}

// Delete handles bridge deletion
func (h *BridgeHandler) Delete(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, `{"error": "name is required"}`, http.StatusBadRequest)
		return
	}

	if err := h.manager.Delete(name); err != nil {
		if err == ErrBridgeNotFound || err == store.ErrBridgeNotFound {
			http.Error(w, `{"error": "Bridge not found"}`, http.StatusNotFound)
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

// Update handles bridge update (name + secret)
func (h *BridgeHandler) Update(w http.ResponseWriter, r *http.Request) {
	oldName := r.URL.Query().Get("name")
	if oldName == "" {
		http.Error(w, `{"error": "name is required"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Name   string `json:"name"`
		Secret string `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Secret == "" {
		http.Error(w, `{"error": "name and secret are required"}`, http.StatusBadRequest)
		return
	}

	bridge, err := h.manager.Update(oldName, req.Name, req.Secret)
	if err != nil {
		if err == ErrBridgeNotFound || err == store.ErrBridgeNotFound {
			http.Error(w, `{"error": "Bridge not found"}`, http.StatusNotFound)
			return
		}
		if err == ErrBridgeExists || err == store.ErrBridgeExists {
			http.Error(w, `{"error": "Bridge name already exists"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"bridge": map[string]interface{}{
			"id":   bridge.ID,
			"name": bridge.Name,
		},
	})
}

// SSEHandler handles SSE connections from bridges
type SSEHandler struct {
	manager *BridgeManager
}

// NewSSEHandler creates a new SSE handler
func NewSSEHandler(m *BridgeManager) *SSEHandler {
	return &SSEHandler{manager: m}
}

// ServeHTTP handles SSE connections
func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	secret := r.URL.Query().Get("secret")

	if name == "" || secret == "" {
		http.Error(w, "name and secret are required", http.StatusBadRequest)
		return
	}

	// Authenticate
	if err := h.manager.Authenticate(name, secret); err != nil {
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	// Set connection status
	if err := h.manager.SetConnected(name, true); err != nil {
		http.Error(w, "Failed to update connection status", http.StatusInternalServerError)
		return
	}

	// Ensure connection status is reset on disconnect
	defer h.manager.SetConnected(name, false)

	// Setup SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send initial connection event
	sendSSE(w, flusher, "connected", `{"status": "connected", "name": "`+name+`"}`)

	// Keep connection alive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Monitor connection
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			h.manager.SetConnected(name, false)
			return
		case <-ticker.C:
			sendSSE(w, flusher, "ping", `{"type": "ping"}`)
		}
	}
}

// sendSSE sends an SSE event
func sendSSE(w http.ResponseWriter, flusher http.Flusher, event, data string) {
	w.Write([]byte("event: " + event + "\n"))
	w.Write([]byte("data: " + data + "\n\n"))
	flusher.Flush()
}
