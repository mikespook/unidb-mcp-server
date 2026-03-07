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
func (d *MSSQLDriver) Open(dsn string) (Handle, error) {
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

// Query executes a query and returns columns and rows
func (d *MSSQLDriver) Query(h Handle, query string) ([]string, [][]interface{}, error) {
	return sqlQuery(h.(*sql.DB), query)
}

// Execute runs a statement and returns rows affected
func (d *MSSQLDriver) Execute(h Handle, query string) (int64, error) {
	return sqlExecute(h.(*sql.DB), query)
}

// GetTableNames retrieves all user table names
func (d *MSSQLDriver) GetTableNames(h Handle) ([]string, error) {
	rows, err := h.(*sql.DB).Query("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE' ORDER BY TABLE_NAME")
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
func (d *MSSQLDriver) GetTableSchema(h Handle, table string) ([]map[string]interface{}, error) {
	query := fmt.Sprintf(`
		SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_DEFAULT
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_NAME = '%s'
		ORDER BY ORDINAL_POSITION`, table)
	return scanSchemaRows(h.(*sql.DB), query)
}

// Close closes the connection
func (d *MSSQLDriver) Close(h Handle) error {
	return h.(*sql.DB).Close()
}
