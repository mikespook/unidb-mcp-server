package bridge

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// BridgeConfig holds the bridge configuration
type BridgeConfig struct {
	Name       string
	Secret     string
	FilePath   string
	UniDBURL   string
	Reconnect  bool
	ReconnectDelay time.Duration
}

// Client represents an SSE client that connects to UniDB server
type Client struct {
	config   BridgeConfig
	bridge   *SQLiteBridge
	mu       sync.RWMutex
	running  bool
	ctx      context.Context
	cancel   context.CancelFunc
}

// MCPRequest represents an MCP request
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPResponse represents an MCP response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewClient creates a new bridge client
func NewClient(config BridgeConfig) (*Client, error) {
	bridge, err := NewSQLiteBridge(config.FilePath)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		config: config,
		bridge: bridge,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Start connects to UniDB server and starts listening for MCP commands
func (c *Client) Start() error {
	c.mu.Lock()
	c.running = true
	c.mu.Unlock()

	for {
		if err := c.connectAndListen(); err != nil {
			if !c.config.Reconnect {
				return err
			}
			
			select {
			case <-c.ctx.Done():
				return nil
			case <-time.After(c.config.ReconnectDelay):
				// Reconnect
			}
		} else {
			// Connection closed normally
			if !c.config.Reconnect {
				return nil
			}
			
			select {
			case <-c.ctx.Done():
				return nil
			case <-time.After(c.config.ReconnectDelay):
				// Reconnect
			}
		}
	}
}

// connectAndListen establishes SSE connection and listens for commands
func (c *Client) connectAndListen() error {
	// Connect to SSE endpoint
	sseURL := fmt.Sprintf("%s/sse?name=%s&secret=%s", 
		c.config.UniDBURL, 
		url.QueryEscape(c.config.Name),
		url.QueryEscape(c.config.Secret))

	req, err := http.NewRequest("GET", sseURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SSE connection failed: %s - %s", resp.Status, string(body))
	}

	return c.readSSE(resp.Body)
}

// readSSE reads SSE events and processes MCP commands
func (c *Client) readSSE(reader io.Reader) error {
	scanner := bufio.NewScanner(reader)
	var eventType, eventData string

	for scanner.Scan() {
		select {
		case <-c.ctx.Done():
			return nil
		default:
		}

		line := scanner.Text()
		
		if line == "" {
			// Empty line means end of event
			if eventData != "" {
				if err := c.processEvent(eventType, eventData); err != nil {
					return err
				}
				eventType, eventData = "", ""
			}
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			eventData = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}

	return scanner.Err()
}

// processEvent processes an SSE event
func (c *Client) processEvent(eventType, eventData string) error {
	switch eventType {
	case "message", "mcp":
		return c.handleMCPRequest(eventData)
	default:
		return nil
	}
}

// handleMCPRequest handles an MCP request
func (c *Client) handleMCPRequest(data string) error {
	var req MCPRequest
	if err := json.Unmarshal([]byte(data), &req); err != nil {
		return err
	}

	resp := c.handleRequest(req)

	// Send response back to UniDB
	return c.sendResponse(resp)
}

// handleRequest processes an MCP request and returns a response
func (c *Client) handleRequest(req MCPRequest) MCPResponse {
	switch req.Method {
	case "tools/list":
		return c.handleToolsList(req.ID)
	case "tools/call":
		return c.handleToolsCall(req.ID, req.Params)
	default:
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

// handleToolsList returns available tools
func (c *Client) handleToolsList(id interface{}) MCPResponse {
	tools := []map[string]interface{}{
		{
			"name":        "query",
			"description": "Execute a SELECT query on the SQLite database",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sql": map[string]interface{}{
						"type":        "string",
						"description": "SQL SELECT query",
					},
				},
				"required": []string{"sql"},
			},
		},
		{
			"name":        "execute",
			"description": "Execute a write operation (INSERT/UPDATE/DELETE)",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sql": map[string]interface{}{
						"type":        "string",
						"description": "SQL write operation",
					},
				},
				"required": []string{"sql"},
			},
		},
		{
			"name":        "schema",
			"description": "Get schema information for tables",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"table": map[string]interface{}{
						"type":        "string",
						"description": "Table name (optional, returns all tables if not specified)",
					},
				},
			},
		},
	}

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

// handleToolsCall executes a tool
func (c *Client) handleToolsCall(id interface{}, params json.RawMessage) MCPResponse {
	var callParams struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments,omitempty"`
	}

	if err := json.Unmarshal(params, &callParams); err != nil {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32700,
				Message: "Invalid parameters",
				Data:    err.Error(),
			},
		}
	}

	switch callParams.Name {
	case "query":
		return c.handleQuery(id, callParams.Arguments)
	case "execute":
		return c.handleExecute(id, callParams.Arguments)
	case "schema":
		return c.handleSchema(id, callParams.Arguments)
	default:
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32601,
				Message: fmt.Sprintf("Unknown tool: %s", callParams.Name),
			},
		}
	}
}

// handleQuery handles query execution
func (c *Client) handleQuery(id interface{}, args map[string]interface{}) MCPResponse {
	sqlQuery, ok := args["sql"].(string)
	if !ok {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "sql is required and must be a string",
			},
		}
	}

	columns, rows, err := c.bridge.ExecuteQuery(sqlQuery)
	if err != nil {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32603,
				Message: "Query execution failed",
				Data:    err.Error(),
			},
		}
	}

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"success":   true,
			"columns":   columns,
			"rows":      rows,
			"row_count": len(rows),
		},
	}
}

// handleExecute handles statement execution
func (c *Client) handleExecute(id interface{}, args map[string]interface{}) MCPResponse {
	sqlQuery, ok := args["sql"].(string)
	if !ok {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32602,
				Message: "sql is required and must be a string",
			},
		}
	}

	rowsAffected, lastInsertID, err := c.bridge.ExecuteStatement(sqlQuery)
	if err != nil {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32603,
				Message: "Execute failed",
				Data:    err.Error(),
			},
		}
	}

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"success":        true,
			"rows_affected":  rowsAffected,
			"last_insert_id": lastInsertID,
		},
	}
}

// handleSchema handles schema retrieval
func (c *Client) handleSchema(id interface{}, args map[string]interface{}) MCPResponse {
	table, _ := args["table"].(string)

	tables := make(map[string]interface{})

	if table != "" {
		schema, err := c.bridge.GetTableSchema(c.bridge.db, table)
		if err != nil {
			return MCPResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error: &MCPError{
					Code:    -32603,
					Message: "Failed to get schema",
					Data:    err.Error(),
				},
			}
		}
		tables[table] = schema
	} else {
		tableNames, err := c.bridge.GetTableNames(c.bridge.db)
		if err != nil {
			return MCPResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error: &MCPError{
					Code:    -32603,
					Message: "Failed to get table names",
					Data:    err.Error(),
				},
			}
		}

		for _, tableName := range tableNames {
			schema, err := c.bridge.GetTableSchema(c.bridge.db, tableName)
			if err != nil {
				continue
			}
			tables[tableName] = schema
		}
	}

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"success": true,
			"tables":  tables,
		},
	}
}

// sendResponse sends a response back to UniDB server
func (c *Client) sendResponse(resp MCPResponse) error {
	jsonData, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	respURL := fmt.Sprintf("%s/api/bridges/response?name=%s&secret=%s",
		c.config.UniDBURL,
		url.QueryEscape(c.config.Name),
		url.QueryEscape(c.config.Secret))

	req, err := http.NewRequest("POST", respURL, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.Secret)

	client := &http.Client{Timeout: 10 * time.Second}
	respHTTP, err := client.Do(req)
	if err != nil {
		return err
	}
	defer respHTTP.Body.Close()

	if respHTTP.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(respHTTP.Body)
		return fmt.Errorf("failed to send response: %s - %s", respHTTP.Status, string(body))
	}

	return nil
}

// Stop stops the bridge client
func (c *Client) Stop() {
	c.mu.Lock()
	c.running = false
	c.mu.Unlock()
	c.cancel()
	c.bridge.Close()
}

// IsRunning returns whether the client is running
func (c *Client) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}
