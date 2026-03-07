package database

import (
	"database/sql"
)

// Driver defines the interface that all database drivers must implement
type Driver interface {
	// Name returns the driver name
	Name() string

	// Open opens a database connection and returns an opaque handle
	Open(dsn string) (Handle, error)

	// Query executes a SELECT-style query and returns columns + rows
	Query(h Handle, sql string) ([]string, [][]interface{}, error)

	// Execute runs a DML/DDL statement and returns rows affected
	Execute(h Handle, sql string) (int64, error)

	// GetTableNames retrieves all table names
	GetTableNames(h Handle) ([]string, error)

	// GetTableSchema retrieves schema information for a table
	GetTableSchema(h Handle, table string) ([]map[string]interface{}, error)

	// Close closes the connection handle
	Close(h Handle) error
}

// Handle is an opaque database connection handle
type Handle interface{}

// sqlQuery is a shared helper for sql.DB-based drivers
func sqlQuery(db *sql.DB, query string) ([]string, [][]interface{}, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	var result [][]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		ptrs := make([]interface{}, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
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
		result = append(result, row)
	}
	return columns, result, rows.Err()
}

// sqlExecute is a shared helper for sql.DB-based drivers
func sqlExecute(db *sql.DB, query string) (int64, error) {
	res, err := db.Exec(query)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// scanSchemaRows is a helper for schema queries returning map slices
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
