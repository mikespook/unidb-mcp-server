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
func (d *SQLiteDriver) Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite connection: %w", err)
	}

	// Set connection pool settings for SQLite
	db.SetMaxOpenConns(1) // SQLite doesn't support multiple writers
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping SQLite: %w", err)
	}

	return db, nil
}

// GetTableNames retrieves all table names
func (d *SQLiteDriver) GetTableNames(db *sql.DB) ([]string, error) {
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
func (d *SQLiteDriver) GetTableSchema(db *sql.DB, table string) ([]map[string]interface{}, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", d.QuoteIdentifier(table))
	return scanSchemaRows(db, query)
}

// QuoteIdentifier quotes a SQLite identifier
func (d *SQLiteDriver) QuoteIdentifier(name string) string {
	return `"` + name + `"`
}
