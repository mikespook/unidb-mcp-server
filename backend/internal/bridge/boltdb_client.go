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

// BoltDBClient represents an SSE client that connects to UniDB server for BoltDB
type BoltDBClient struct {
	config  BridgeConfig
	bridge  *BoltDBBridge
	mu      sync.RWMutex
	running bool
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewBoltDBClient creates a new BoltDB bridge client
func NewBoltDBClient(config BridgeConfig) (*BoltDBClient, error) {
	bridge, err := NewBoltDBBridge(config.FilePath)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &BoltDBClient{
		config: config,
		bridge: bridge,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Start connects to UniDB server and starts listening for MCP commands
func (c *BoltDBClient) Start() error {
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
			}
		} else {
			if !c.config.Reconnect {
				return nil
			}

			select {
			case <-c.ctx.Done():
				return nil
			case <-time.After(c.config.ReconnectDelay):
			}
		}
	}
}

// connectAndListen establishes SSE connection and listens for commands
func (c *BoltDBClient) connectAndListen() error {
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
func (c *BoltDBClient) readSSE(reader io.Reader) error {
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
func (c *BoltDBClient) processEvent(eventType, eventData string) error {
	switch eventType {
	case "message", "mcp":
		return c.handleMCPRequest(eventData)
	default:
		return nil
	}
}

// handleMCPRequest handles an MCP request
func (c *BoltDBClient) handleMCPRequest(data string) error {
	var req MCPRequest
	if err := json.Unmarshal([]byte(data), &req); err != nil {
		return err
	}

	resp := c.handleRequest(req)
	return c.sendResponse(resp)
}

// handleRequest processes an MCP request and returns a response
func (c *BoltDBClient) handleRequest(req MCPRequest) MCPResponse {
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
func (c *BoltDBClient) handleToolsList(id interface{}) MCPResponse {
	tools := []map[string]interface{}{
		{
			"name":        "buckets",
			"description": "List all top-level buckets in the BoltDB database",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "get",
			"description": "Get a value by bucket and key",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"bucket": map[string]interface{}{
						"type":        "string",
						"description": "Bucket name",
					},
					"key": map[string]interface{}{
						"type":        "string",
						"description": "Key to retrieve",
					},
				},
				"required": []string{"bucket", "key"},
			},
		},
		{
			"name":        "put",
			"description": "Set a key-value pair in a bucket (creates bucket if it does not exist)",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"bucket": map[string]interface{}{
						"type":        "string",
						"description": "Bucket name",
					},
					"key": map[string]interface{}{
						"type":        "string",
						"description": "Key to set",
					},
					"value": map[string]interface{}{
						"type":        "string",
						"description": "Value to store",
					},
				},
				"required": []string{"bucket", "key", "value"},
			},
		},
		{
			"name":        "delete",
			"description": "Delete a key from a bucket",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"bucket": map[string]interface{}{
						"type":        "string",
						"description": "Bucket name",
					},
					"key": map[string]interface{}{
						"type":        "string",
						"description": "Key to delete",
					},
				},
				"required": []string{"bucket", "key"},
			},
		},
		{
			"name":        "scan",
			"description": "Scan all key-value pairs in a bucket, optionally filtered by key prefix",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"bucket": map[string]interface{}{
						"type":        "string",
						"description": "Bucket name",
					},
					"prefix": map[string]interface{}{
						"type":        "string",
						"description": "Key prefix filter (optional, returns all keys if not specified)",
					},
				},
				"required": []string{"bucket"},
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
func (c *BoltDBClient) handleToolsCall(id interface{}, params json.RawMessage) MCPResponse {
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
	case "buckets":
		return c.handleBuckets(id)
	case "get":
		return c.handleGet(id, callParams.Arguments)
	case "put":
		return c.handlePut(id, callParams.Arguments)
	case "delete":
		return c.handleDelete(id, callParams.Arguments)
	case "scan":
		return c.handleScan(id, callParams.Arguments)
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

// handleBuckets lists all top-level buckets
func (c *BoltDBClient) handleBuckets(id interface{}) MCPResponse {
	buckets, err := c.bridge.GetBuckets()
	if err != nil {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32603,
				Message: "Failed to list buckets",
				Data:    err.Error(),
			},
		}
	}

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"success": true,
			"buckets": buckets,
		},
	}
}

// handleGet retrieves a value by bucket and key
func (c *BoltDBClient) handleGet(id interface{}, args map[string]interface{}) MCPResponse {
	bucket, ok := args["bucket"].(string)
	if !ok || bucket == "" {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "bucket is required and must be a string"},
		}
	}

	key, ok := args["key"].(string)
	if !ok || key == "" {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "key is required and must be a string"},
		}
	}

	value, err := c.bridge.Get(bucket, key)
	if err != nil {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32603,
				Message: "Get failed",
				Data:    err.Error(),
			},
		}
	}

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"success": true,
			"bucket":  bucket,
			"key":     key,
			"value":   value,
		},
	}
}

// handlePut sets a key-value pair in a bucket
func (c *BoltDBClient) handlePut(id interface{}, args map[string]interface{}) MCPResponse {
	bucket, ok := args["bucket"].(string)
	if !ok || bucket == "" {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "bucket is required and must be a string"},
		}
	}

	key, ok := args["key"].(string)
	if !ok || key == "" {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "key is required and must be a string"},
		}
	}

	value, ok := args["value"].(string)
	if !ok {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "value is required and must be a string"},
		}
	}

	if err := c.bridge.Put(bucket, key, value); err != nil {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32603,
				Message: "Put failed",
				Data:    err.Error(),
			},
		}
	}

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"success": true,
			"bucket":  bucket,
			"key":     key,
		},
	}
}

// handleDelete removes a key from a bucket
func (c *BoltDBClient) handleDelete(id interface{}, args map[string]interface{}) MCPResponse {
	bucket, ok := args["bucket"].(string)
	if !ok || bucket == "" {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "bucket is required and must be a string"},
		}
	}

	key, ok := args["key"].(string)
	if !ok || key == "" {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "key is required and must be a string"},
		}
	}

	if err := c.bridge.Delete(bucket, key); err != nil {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32603,
				Message: "Delete failed",
				Data:    err.Error(),
			},
		}
	}

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"success": true,
			"bucket":  bucket,
			"key":     key,
		},
	}
}

// handleScan scans all key-value pairs in a bucket
func (c *BoltDBClient) handleScan(id interface{}, args map[string]interface{}) MCPResponse {
	bucket, ok := args["bucket"].(string)
	if !ok || bucket == "" {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &MCPError{Code: -32602, Message: "bucket is required and must be a string"},
		}
	}

	prefix, _ := args["prefix"].(string)

	entries, err := c.bridge.Scan(bucket, prefix)
	if err != nil {
		return MCPResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &MCPError{
				Code:    -32603,
				Message: "Scan failed",
				Data:    err.Error(),
			},
		}
	}

	return MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"success": true,
			"bucket":  bucket,
			"entries": entries,
			"count":   len(entries),
		},
	}
}

// sendResponse sends a response back to UniDB server
func (c *BoltDBClient) sendResponse(resp MCPResponse) error {
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

// Stop stops the BoltDB bridge client
func (c *BoltDBClient) Stop() {
	c.mu.Lock()
	c.running = false
	c.mu.Unlock()
	c.cancel()
	c.bridge.Close()
}

// IsRunning returns whether the client is running
func (c *BoltDBClient) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}
