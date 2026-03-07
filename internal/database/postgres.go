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
func (d *PostgresDriver) Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	return db, nil
}

// GetTableNames retrieves all table names
func (d *PostgresDriver) GetTableNames(db *sql.DB) ([]string, error) {
	query := "SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename"
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
func (d *PostgresDriver) GetTableSchema(db *sql.DB, table string) ([]map[string]interface{}, error) {
	query := fmt.Sprintf(`
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_name = %s AND table_schema = 'public'
		ORDER BY ordinal_position`, d.QuoteIdentifier(table))
	return scanSchemaRows(db, query)
}

// QuoteIdentifier quotes a PostgreSQL identifier
func (d *PostgresDriver) QuoteIdentifier(name string) string {
	return `"` + name + `"`
}
