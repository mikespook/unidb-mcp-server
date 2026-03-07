package database

import (
	"database/sql"
)

// Driver defines the interface that all database drivers must implement
type Driver interface {
	// Name returns the driver name
	Name() string

	// Open opens a database connection
	Open(dsn string) (*sql.DB, error)

	// GetTableNames retrieves all table names
	GetTableNames(db *sql.DB) ([]string, error)

	// GetTableSchema retrieves schema information for a table
	GetTableSchema(db *sql.DB, table string) ([]map[string]interface{}, error)

	// QuoteIdentifier quotes a table or column name for the specific database
	QuoteIdentifier(name string) string
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
