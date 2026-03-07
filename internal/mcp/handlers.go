package mcp

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mikespook/unidb-mcp/internal/database"
	"github.com/mikespook/unidb-mcp/internal/store"
)

// Request represents an MCP request
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents an MCP response
type Response struct {
	JSONRPC        string      `json:"jsonrpc"`
	ID             interface{} `json:"id"`
	Result         interface{} `json:"result,omitempty"`
	Error          *Error      `json:"error,omitempty"`
	IsNotification bool        `json:"-"`
}

// Error represents an MCP error
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// Handler handles MCP requests
type Handler struct {
	store   *store.Store
	manager *database.DriverManager
}

// NewHandler creates a new MCP handler
func NewHandler(s *store.Store, m *database.DriverManager) *Handler {
	return &Handler{
		store:   s,
		manager: m,
	}
}

// HandleRequest processes an MCP request
func (h *Handler) HandleRequest(req Request) Response {
	switch req.Method {
	case "initialize":
		return h.handleInitialize(req.ID)
	case "notifications/initialized":
		return Response{IsNotification: true}
	case "tools/list":
		return h.handleToolsList(req.ID)
	case "tools/call":
		return h.handleToolsCall(req.ID, req.Params)
	default:
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &Error{
				Code:    MethodNotFound,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

// handleInitialize handles the MCP initialize handshake
func (h *Handler) handleInitialize(id interface{}) Response {
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"protocolVersion": "2025-03-26",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "unidb-mcp",
				"version": "1.0.0",
			},
		},
	}
}

// handleToolsList returns the list of available tools
func (h *Handler) handleToolsList(id interface{}) Response {
	tools := GetTools()
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

// handleToolsCall executes a tool
func (h *Handler) handleToolsCall(id interface{}, params json.RawMessage) Response {
	var callParams struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments,omitempty"`
	}

	if err := json.Unmarshal(params, &callParams); err != nil {
		return Response{
			JSONRPC: "2.0",
			ID:      id,
			Error: &Error{
				Code:    ParseError,
				Message: "Invalid parameters",
				Data:    err.Error(),
			},
		}
	}

	switch callParams.Name {
	case "list_dsns":
		return h.handleListDSNs(id)
	case "connect":
		return h.handleConnect(id, callParams.Arguments)
	case "disconnect":
		return h.handleDisconnect(id, callParams.Arguments)
	case "list_connections":
		return h.handleListConnections(id)
	case "query":
		return h.handleQuery(id, callParams.Arguments)
	case "execute":
		return h.handleExecute(id, callParams.Arguments)
	case "schema":
		return h.handleSchema(id, callParams.Arguments)
	default:
		return Response{
			JSONRPC: "2.0",
			ID:      id,
			Error: &Error{
				Code:    MethodNotFound,
				Message: fmt.Sprintf("Unknown tool: %s", callParams.Name),
			},
		}
	}
}

// handleListDSNs lists all configured DSNs from the store
func (h *Handler) handleListDSNs(id interface{}) Response {
	dsns, err := h.store.List()
	if err != nil {
		return h.internalError(id, err)
	}

	result := make([]map[string]interface{}, 0, len(dsns))
	for _, d := range dsns {
		result = append(result, map[string]interface{}{
			"name":       d.Name,
			"driver":     d.Driver,
			"created_at": d.CreatedAt,
		})
	}

	return toolResult(id, map[string]interface{}{
		"dsns":  result,
		"count": len(result),
	})
}

// handleConnect handles the connect tool
func (h *Handler) handleConnect(id interface{}, args map[string]interface{}) Response {
	dsnName, ok := args["dsn_name"].(string)
	if !ok {
		return Response{
			JSONRPC: "2.0",
			ID:      id,
			Error: &Error{
				Code:    InvalidParams,
				Message: "dsn_name is required and must be a string",
			},
		}
	}

	dsn, err := h.store.GetByName(dsnName)
	if err != nil {
		if errors.Is(err, store.ErrDSNNotFound) {
			return Response{
				JSONRPC: "2.0",
				ID:      id,
				Error: &Error{
					Code:    InvalidParams,
					Message: fmt.Sprintf("DSN not found: %s", dsnName),
				},
			}
		}
		return h.internalError(id, err)
	}

	connID := GenerateID()
	conn, err := h.manager.Connect(connID, dsn.Name, dsn.Driver, dsn.DSN)
	if err != nil {
		return h.internalError(id, err)
	}

	return toolResult(id, map[string]interface{}{
		"success":       true,
		"connection_id": conn.ID,
		"message":       fmt.Sprintf("Connected to %s database", dsn.Driver),
	})
}

// handleDisconnect handles the disconnect tool
func (h *Handler) handleDisconnect(id interface{}, args map[string]interface{}) Response {
	connID, ok := args["connection_id"].(string)
	if !ok {
		return Response{
			JSONRPC: "2.0",
			ID:      id,
			Error: &Error{
				Code:    InvalidParams,
				Message: "connection_id is required and must be a string",
			},
		}
	}

	if err := h.manager.Disconnect(connID); err != nil {
		if errors.Is(err, database.ErrConnectionNotFound) {
			return Response{
				JSONRPC: "2.0",
				ID:      id,
				Error: &Error{
					Code:    InvalidParams,
					Message: fmt.Sprintf("Connection not found: %s", connID),
				},
			}
		}
		return h.internalError(id, err)
	}

	return toolResult(id, map[string]interface{}{
		"success": true,
		"message": "Connection closed",
	})
}

// handleListConnections handles the list_connections tool
func (h *Handler) handleListConnections(id interface{}) Response {
	conns := h.manager.List()
	result := make([]map[string]interface{}, 0, len(conns))

	for _, conn := range conns {
		result = append(result, map[string]interface{}{
			"connection_id": conn.ID,
			"dsn_name":      conn.DSNName,
			"driver":        conn.Driver,
			"connected_at":  conn.ConnectedAt,
		})
	}

	return toolResult(id, map[string]interface{}{
		"connections": result,
	})
}

// handleQuery handles the query tool
func (h *Handler) handleQuery(id interface{}, args map[string]interface{}) Response {
	connID, ok := args["connection_id"].(string)
	if !ok {
		return Response{
			JSONRPC: "2.0",
			ID:      id,
			Error: &Error{
				Code:    InvalidParams,
				Message: "connection_id is required and must be a string",
			},
		}
	}

	sqlQuery, ok := args["sql"].(string)
	if !ok {
		return Response{
			JSONRPC: "2.0",
			ID:      id,
			Error: &Error{
				Code:    InvalidParams,
				Message: "sql is required and must be a string",
			},
		}
	}

	conn, err := h.manager.Get(connID)
	if err != nil {
		if errors.Is(err, database.ErrConnectionNotFound) {
			return Response{
				JSONRPC: "2.0",
				ID:      id,
				Error: &Error{
					Code:    InvalidParams,
					Message: fmt.Sprintf("Connection not found: %s", connID),
				},
			}
		}
		return h.internalError(id, err)
	}

	rows, err := conn.DB.Query(sqlQuery)
	if err != nil {
		return Response{
			JSONRPC: "2.0",
			ID:      id,
			Error: &Error{
				Code:    InternalError,
				Message: "Query execution failed",
				Data:    err.Error(),
			},
		}
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return h.internalError(id, err)
	}

	result := make([][]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return h.internalError(id, err)
		}

		row := make([]interface{}, len(columns))
		for i, v := range values {
			if b, ok := v.([]byte); ok {
				row[i] = string(b)
			} else {
				row[i] = v
			}
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return h.internalError(id, err)
	}

	return toolResult(id, map[string]interface{}{
		"success":   true,
		"columns":   columns,
		"rows":      result,
		"row_count": len(result),
	})
}

// handleExecute handles the execute tool
func (h *Handler) handleExecute(id interface{}, args map[string]interface{}) Response {
	connID, ok := args["connection_id"].(string)
	if !ok {
		return Response{
			JSONRPC: "2.0",
			ID:      id,
			Error: &Error{
				Code:    InvalidParams,
				Message: "connection_id is required and must be a string",
			},
		}
	}

	sqlQuery, ok := args["sql"].(string)
	if !ok {
		return Response{
			JSONRPC: "2.0",
			ID:      id,
			Error: &Error{
				Code:    InvalidParams,
				Message: "sql is required and must be a string",
			},
		}
	}

	conn, err := h.manager.Get(connID)
	if err != nil {
		if errors.Is(err, database.ErrConnectionNotFound) {
			return Response{
				JSONRPC: "2.0",
				ID:      id,
				Error: &Error{
					Code:    InvalidParams,
					Message: fmt.Sprintf("Connection not found: %s", connID),
				},
			}
		}
		return h.internalError(id, err)
	}

	result, err := conn.DB.Exec(sqlQuery)
	if err != nil {
		return Response{
			JSONRPC: "2.0",
			ID:      id,
			Error: &Error{
				Code:    InternalError,
				Message: "Execute failed",
				Data:    err.Error(),
			},
		}
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()

	return toolResult(id, map[string]interface{}{
		"success":        true,
		"rows_affected":  rowsAffected,
		"last_insert_id": lastInsertID,
	})
}

// handleSchema handles the schema tool
func (h *Handler) handleSchema(id interface{}, args map[string]interface{}) Response {
	connID, ok := args["connection_id"].(string)
	if !ok {
		return Response{
			JSONRPC: "2.0",
			ID:      id,
			Error: &Error{
				Code:    InvalidParams,
				Message: "connection_id is required and must be a string",
			},
		}
	}

	table, _ := args["table"].(string)

	conn, err := h.manager.Get(connID)
	if err != nil {
		if errors.Is(err, database.ErrConnectionNotFound) {
			return Response{
				JSONRPC: "2.0",
				ID:      id,
				Error: &Error{
					Code:    InvalidParams,
					Message: fmt.Sprintf("Connection not found: %s", connID),
				},
			}
		}
		return h.internalError(id, err)
	}

	tables := make(map[string]interface{})
	driver := conn.GetDriver()

	if table != "" {
		schema, err := driver.GetTableSchema(conn.DB, table)
		if err != nil {
			return h.internalError(id, err)
		}
		tables[table] = schema
	} else {
		tableNames, err := driver.GetTableNames(conn.DB)
		if err != nil {
			return h.internalError(id, err)
		}

		for _, tableName := range tableNames {
			schema, err := driver.GetTableSchema(conn.DB, tableName)
			if err != nil {
				continue
			}
			tables[tableName] = schema
		}
	}

	return toolResult(id, map[string]interface{}{
		"success": true,
		"tables":  tables,
	})
}

// toolResult wraps a result value in the MCP content format
func toolResult(id interface{}, result interface{}) Response {
	text, _ := json.Marshal(result)
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": string(text)},
			},
		},
	}
}

// internalError creates an internal error response
func (h *Handler) internalError(id interface{}, err error) Response {
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    InternalError,
			Message: "Internal error",
			Data:    err.Error(),
		},
	}
}
