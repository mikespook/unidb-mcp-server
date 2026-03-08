package database

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLDriver implements the Driver interface for MySQL
type MySQLDriver struct{}

// Name returns the driver name
func (d *MySQLDriver) Name() string {
	return "mysql"
}

// Open opens a MySQL connection
func (d *MySQLDriver) Open(dsn string) (Handle, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open MySQL connection: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}
	return db, nil
}

// Query executes a query and returns columns and rows
func (d *MySQLDriver) Query(h Handle, query string) ([]string, [][]interface{}, error) {
	return sqlQuery(h.(*sql.DB), query)
}

// Execute runs a statement and returns rows affected
func (d *MySQLDriver) Execute(h Handle, query string) (int64, error) {
	return sqlExecute(h.(*sql.DB), query)
}

// GetTableNames retrieves all table names
func (d *MySQLDriver) GetTableNames(h Handle) ([]string, error) {
	rows, err := h.(*sql.DB).Query("SHOW TABLES")
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
func (d *MySQLDriver) GetTableSchema(h Handle, table string) ([]map[string]interface{}, error) {
	return scanSchemaRows(h.(*sql.DB), fmt.Sprintf("DESCRIBE `%s`", table))
}

// Close closes the connection
func (d *MySQLDriver) Close(h Handle) error {
	return h.(*sql.DB).Close()
}
