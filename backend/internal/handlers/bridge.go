package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/mikespook/unidb-mcp/internal/store"
)

// BridgeManager manages active bridge connections in memory
type BridgeManager struct {
	mu        sync.RWMutex
	store     *store.Store
	connected map[string]bool // keyed by bridge name
}

// NewBridgeManager creates a new bridge manager.
// All bridges are considered disconnected on startup since any previous SSE
// connections were lost when the server stopped.
func NewBridgeManager(s *store.Store) *BridgeManager {
	return &BridgeManager{
		store:     s,
		connected: make(map[string]bool),
	}
}

// Authenticate verifies bridge credentials by looking up the DSN entry.
// For sqlite-bridge entries the secret is stored in the dsn field.
func (m *BridgeManager) Authenticate(name, secret string) error {
	dsn, err := m.store.GetByNameAndDriver(name, "sqlite-bridge")
	if err != nil {
		return ErrBridgeNotFound
	}
	if dsn.DSN != secret {
		return ErrInvalidSecret
	}
	return nil
}

// SetConnected updates the in-memory connection status and bumps updated_at in the DB.
func (m *BridgeManager) SetConnected(name string, connected bool) error {
	m.mu.Lock()
	m.connected[name] = connected
	m.mu.Unlock()

	dsn, err := m.store.GetByNameAndDriver(name, "sqlite-bridge")
	if err != nil {
		return err
	}
	return m.store.TouchUpdatedAt(dsn.ID)
}

// IsConnected returns whether the named bridge is currently connected.
func (m *BridgeManager) IsConnected(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected[name]
}

// Bridge errors
var (
	ErrBridgeNotFound = &BridgeError{"Bridge not found"}
	ErrInvalidSecret  = &BridgeError{"Invalid secret"}
)

// BridgeError represents a bridge-related error
type BridgeError struct {
	Message string
}

func (e *BridgeError) Error() string {
	return e.Message
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

	// Keep connection alive with periodic pings
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

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
