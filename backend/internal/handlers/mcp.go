package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/mikespook/unidb-mcp/internal/mcp"
)

// MCPHandler handles MCP protocol requests
type MCPHandler struct {
	handler *mcp.Handler
}

// NewMCPHandler creates a new MCP handler
func NewMCPHandler(h *mcp.Handler) *MCPHandler {
	return &MCPHandler{
		handler: h,
	}
}

// HandleMCP processes MCP JSON-RPC requests
func (h *MCPHandler) HandleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req mcp.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp := mcp.Response{
			JSONRPC: "2.0",
			ID:      nil,
			Error: &mcp.Error{
				Code:    mcp.ParseError,
				Message: "Invalid JSON-RPC request",
				Data:    err.Error(),
			},
		}
		h.writeResponse(w, resp)
		return
	}

	resp := h.handler.HandleRequest(req)
	h.writeResponse(w, resp)
}

// writeResponse writes an MCP response
func (h *MCPHandler) writeResponse(w http.ResponseWriter, resp mcp.Response) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
