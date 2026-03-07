package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// PostgresDriver implements the Driver interface for PostgreSQL
type PostgresDriver struct{}

// Name returns the driver name
func (d *PostgresDriver) Name() string {
	return "postgres"
}

// Open opens a PostgreSQL connection
func (d *PostgresDriver) Open(dsn string) (Handle, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}
	return db, nil
}

// Query executes a query and returns columns and rows
func (d *PostgresDriver) Query(h Handle, query string) ([]string, [][]interface{}, error) {
	return sqlQuery(h.(*sql.DB), query)
}

// Execute runs a statement and returns rows affected
func (d *PostgresDriver) Execute(h Handle, query string) (int64, error) {
	return sqlExecute(h.(*sql.DB), query)
}

// GetTableNames retrieves all table names
func (d *PostgresDriver) GetTableNames(h Handle) ([]string, error) {
	rows, err := h.(*sql.DB).Query("SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename")
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
func (d *PostgresDriver) GetTableSchema(h Handle, table string) ([]map[string]interface{}, error) {
	query := fmt.Sprintf(`
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_name = '%s' AND table_schema = 'public'
		ORDER BY ordinal_position`, table)
	return scanSchemaRows(h.(*sql.DB), query)
}

// Close closes the connection
func (d *PostgresDriver) Close(h Handle) error {
	return h.(*sql.DB).Close()
}
