package database

import (
	"database/sql"
	"fmt"

	_ "github.com/microsoft/go-mssqldb"
)

// MSSQLDriver implements the Driver interface for Microsoft SQL Server
type MSSQLDriver struct{}

// Name returns the driver name
func (d *MSSQLDriver) Name() string {
	return "mssql"
}

// Open opens a SQL Server connection
func (d *MSSQLDriver) Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mssql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQL Server connection: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping SQL Server: %w", err)
	}

	return db, nil
}

// GetTableNames retrieves all user table names
func (d *MSSQLDriver) GetTableNames(db *sql.DB) ([]string, error) {
	query := "SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE' ORDER BY TABLE_NAME"
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
func (d *MSSQLDriver) GetTableSchema(db *sql.DB, table string) ([]map[string]interface{}, error) {
	query := fmt.Sprintf(`
		SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_DEFAULT
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_NAME = '%s'
		ORDER BY ORDINAL_POSITION`, table)
	return scanSchemaRows(db, query)
}

// QuoteIdentifier quotes a SQL Server identifier
func (d *MSSQLDriver) QuoteIdentifier(name string) string {
	return "[" + name + "]"
}
