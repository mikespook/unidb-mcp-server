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
func (d *MySQLDriver) Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open MySQL connection: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}

	return db, nil
}

// GetTableNames retrieves all table names
func (d *MySQLDriver) GetTableNames(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SHOW TABLES")
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
func (d *MySQLDriver) GetTableSchema(db *sql.DB, table string) ([]map[string]interface{}, error) {
	query := fmt.Sprintf("DESCRIBE %s", d.QuoteIdentifier(table))
	return scanSchemaRows(db, query)
}

// QuoteIdentifier quotes a MySQL identifier
func (d *MySQLDriver) QuoteIdentifier(name string) string {
	return "`" + name + "`"
}
