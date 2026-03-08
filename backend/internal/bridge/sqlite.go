package bridge

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteBridge implements a bridge to a local SQLite file
type SQLiteBridge struct {
	filePath string
	db       *sql.DB
}

// NewSQLiteBridge creates a new SQLite bridge
func NewSQLiteBridge(filePath string) (*SQLiteBridge, error) {
	db, err := sql.Open("sqlite3", filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite file: %w", err)
	}

	// Set connection pool settings for SQLite
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping SQLite: %w", err)
	}

	return &SQLiteBridge{
		filePath: filePath,
		db:       db,
	}, nil
}

// Close closes the bridge connection
func (b *SQLiteBridge) Close() error {
	if b.db != nil {
		return b.db.Close()
	}
	return nil
}

// Name returns the driver name
func (b *SQLiteBridge) Name() string {
	return "sqlite"
}

// Open opens a SQLite connection (not used in bridge mode)
func (b *SQLiteBridge) Open(dsn string) (*sql.DB, error) {
	return b.db, nil
}

// GetTableNames retrieves all table names
func (b *SQLiteBridge) GetTableNames(db *sql.DB) ([]string, error) {
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}

	return tables, rows.Err()
}

// GetTableSchema retrieves schema information for a table
func (b *SQLiteBridge) GetTableSchema(db *sql.DB, table string) ([]map[string]interface{}, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", b.QuoteIdentifier(table))
	return scanSchemaRows(db, query)
}

// QuoteIdentifier quotes a SQLite identifier
func (b *SQLiteBridge) QuoteIdentifier(name string) string {
	return `"` + name + `"`
}

// ExecuteQuery executes a query and returns results
func (b *SQLiteBridge) ExecuteQuery(query string) ([]string, [][]interface{}, error) {
	rows, err := b.db.Query(query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	results := make([][]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, nil, err
		}

		row := make([]interface{}, len(columns))
		for i, v := range values {
			if b, ok := v.([]byte); ok {
				row[i] = string(b)
			} else {
				row[i] = v
			}
		}
		results = append(results, row)
	}

	return columns, results, rows.Err()
}

// ExecuteStatement executes a statement (INSERT/UPDATE/DELETE) and returns affected rows
func (b *SQLiteBridge) ExecuteStatement(query string) (int64, int64, error) {
	result, err := b.db.Exec(query)
	if err != nil {
		return 0, 0, err
	}

	rowsAffected, _ := result.RowsAffected()
	lastInsertID, _ := result.LastInsertId()

	return rowsAffected, lastInsertID, nil
}

// scanSchemaRows is a helper function to scan schema query results
func scanSchemaRows(db *sql.DB, query string) ([]map[string]interface{}, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	schema := make([]map[string]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			if b, ok := values[i].([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = values[i]
			}
		}
		schema = append(schema, row)
	}

	return schema, rows.Err()
}
