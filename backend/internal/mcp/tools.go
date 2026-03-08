package mcp

import "github.com/google/uuid"

// ToolDefinition represents an MCP tool
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"inputSchema"`
}

// InputSchema defines the JSON schema for tool input
type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

// Property defines a single input property
type Property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// GetTools returns all available MCP tools
func GetTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "connect",
			Description: "Establish connection to a database using stored DSN",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"dsn_name": {
						Type:        "string",
						Description: "Name of the stored DSN configuration",
					},
				},
				Required: []string{"dsn_name"},
			},
		},
		{
			Name:        "disconnect",
			Description: "Close and remove an active database connection",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"connection_id": {
						Type:        "string",
						Description: "ID of the active connection",
					},
				},
				Required: []string{"connection_id"},
			},
		},
		{
			Name:        "list_dsns",
			Description: "List all configured DSN connections available to connect to",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "list_connections",
			Description: "List all active database connections",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "query",
			Description: "Execute a SELECT query on the specified database",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"connection_id": {
						Type:        "string",
						Description: "ID of the active connection",
					},
					"sql": {
						Type:        "string",
						Description: "SQL SELECT query",
					},
				},
				Required: []string{"connection_id", "sql"},
			},
		},
		{
			Name:        "execute",
			Description: "Execute a write operation (INSERT/UPDATE/DELETE)",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"connection_id": {
						Type:        "string",
						Description: "ID of the active connection",
					},
					"sql": {
						Type:        "string",
						Description: "SQL write statement",
					},
				},
				Required: []string{"connection_id", "sql"},
			},
		},
		{
			Name:        "schema",
			Description: "Get schema information for tables in the database",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"connection_id": {
						Type:        "string",
						Description: "ID of the active connection",
					},
					"table": {
						Type:        "string",
						Description: "Optional: specific table name",
					},
				},
				Required: []string{"connection_id"},
			},
		},
	}
}

// GenerateID generates a unique ID for connections
func GenerateID() string {
	return uuid.New().String()
}
