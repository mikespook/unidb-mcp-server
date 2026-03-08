package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteDriver implements the Driver interface for SQLite
type SQLiteDriver struct{}

// Name returns the driver name
func (d *SQLiteDriver) Name() string {
	return "sqlite"
}

// Open opens a SQLite connection
func (d *SQLiteDriver) Open(dsn string) (Handle, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite connection: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping SQLite: %w", err)
	}
	return db, nil
}

// Query executes a query and returns columns and rows
func (d *SQLiteDriver) Query(h Handle, query string) ([]string, [][]interface{}, error) {
	return sqlQuery(h.(*sql.DB), query)
}

// Execute runs a statement and returns rows affected
func (d *SQLiteDriver) Execute(h Handle, query string) (int64, error) {
	return sqlExecute(h.(*sql.DB), query)
}

// GetTableNames retrieves all table names
func (d *SQLiteDriver) GetTableNames(h Handle) ([]string, error) {
	rows, err := h.(*sql.DB).Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
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
func (d *SQLiteDriver) GetTableSchema(h Handle, table string) ([]map[string]interface{}, error) {
	return scanSchemaRows(h.(*sql.DB), fmt.Sprintf(`PRAGMA table_info("%s")`, table))
}

// Close closes the connection
func (d *SQLiteDriver) Close(h Handle) error {
	return h.(*sql.DB).Close()
}
