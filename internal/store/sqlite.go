package store

import (
	"database/sql"
	"errors"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/google/uuid"
)

var (
	ErrDSNNotFound    = errors.New("DSN not found")
	ErrDSNExists      = errors.New("DSN name already exists")
	ErrBridgeNotFound = errors.New("Bridge not found")
	ErrBridgeExists   = errors.New("Bridge already exists")
)

// DSN represents a database connection configuration
type DSN struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Driver    string    `json:"driver"`
	DSN       string    `json:"dsn"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Bridge represents a registered SQLite bridge
type Bridge struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Secret      string     `json:"-"` // Never expose in JSON
	Type        string     `json:"type"`
	Connected   bool       `json:"connected"`
	ConnectedAt *time.Time `json:"connected_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Store manages DSN configurations in SQLite
type Store struct {
	db *sql.DB
}

// NewStore creates a new SQLite store
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	store := &Store{db: db}
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// initSchema creates the database schema if it doesn't exist
func (s *Store) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS dsns (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		driver TEXT NOT NULL,
		dsn TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS bridges (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		secret TEXT NOT NULL,
		type TEXT NOT NULL,
		connected INTEGER DEFAULT 0,
		connected_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS settings (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_dsns_name ON dsns(name);
	CREATE INDEX IF NOT EXISTS idx_bridges_name ON bridges(name);
	`
	_, err := s.db.Exec(schema)
	return err
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// List returns all DSN configurations
func (s *Store) List() ([]*DSN, error) {
	rows, err := s.db.Query(`SELECT id, name, driver, dsn, created_at, updated_at FROM dsns ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dsns []*DSN
	for rows.Next() {
		dsn := &DSN{}
		err := rows.Scan(&dsn.ID, &dsn.Name, &dsn.Driver, &dsn.DSN, &dsn.CreatedAt, &dsn.UpdatedAt)
		if err != nil {
			return nil, err
		}
		dsns = append(dsns, dsn)
	}

	return dsns, rows.Err()
}

// Get retrieves a DSN by ID
func (s *Store) Get(id string) (*DSN, error) {
	dsn := &DSN{}
	err := s.db.QueryRow(
		`SELECT id, name, driver, dsn, created_at, updated_at FROM dsns WHERE id = ?`,
		id,
	).Scan(&dsn.ID, &dsn.Name, &dsn.Driver, &dsn.DSN, &dsn.CreatedAt, &dsn.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrDSNNotFound
	}
	return dsn, err
}

// GetByName retrieves a DSN by name
func (s *Store) GetByName(name string) (*DSN, error) {
	dsn := &DSN{}
	err := s.db.QueryRow(
		`SELECT id, name, driver, dsn, created_at, updated_at FROM dsns WHERE name = ?`,
		name,
	).Scan(&dsn.ID, &dsn.Name, &dsn.Driver, &dsn.DSN, &dsn.CreatedAt, &dsn.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrDSNNotFound
	}
	return dsn, err
}

// Create adds a new DSN configuration
func (s *Store) Create(name, driver, dsn string) (*DSN, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := s.db.Exec(
		`INSERT INTO dsns (id, name, driver, dsn, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		id, name, driver, dsn, now, now,
	)

	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrDSNExists
		}
		return nil, err
	}

	return &DSN{
		ID:        id,
		Name:      name,
		Driver:    driver,
		DSN:       dsn,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Update modifies an existing DSN configuration
func (s *Store) Update(id, name, driver, dsn string) (*DSN, error) {
	now := time.Now()

	result, err := s.db.Exec(
		`UPDATE dsns SET name = ?, driver = ?, dsn = ?, updated_at = ? WHERE id = ?`,
		name, driver, dsn, now, id,
	)

	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrDSNExists
		}
		return nil, err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, ErrDSNNotFound
	}

	return &DSN{
		ID:        id,
		Name:      name,
		Driver:    driver,
		DSN:       dsn,
		UpdatedAt: now,
	}, nil
}

// Delete removes a DSN configuration
func (s *Store) Delete(id string) error {
	result, err := s.db.Exec(`DELETE FROM dsns WHERE id = ?`, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrDSNNotFound
	}

	return nil
}

// isUniqueConstraintError checks if the error is a SQLite unique constraint violation
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "UNIQUE constraint failed: dsns.name" || 
		   err.Error() == "UNIQUE constraint failed: bridges.name"
}

// Bridge methods

// ListBridges returns all bridge configurations
func (s *Store) ListBridges() ([]*Bridge, error) {
	rows, err := s.db.Query(`SELECT id, name, secret, type, connected, connected_at, created_at, updated_at FROM bridges ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bridges []*Bridge
	for rows.Next() {
		bridge := &Bridge{}
		var connectedAt sql.NullTime
		err := rows.Scan(&bridge.ID, &bridge.Name, &bridge.Secret, &bridge.Type, &bridge.Connected, &connectedAt, &bridge.CreatedAt, &bridge.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if connectedAt.Valid {
			bridge.ConnectedAt = &connectedAt.Time
		}
		bridges = append(bridges, bridge)
	}

	return bridges, rows.Err()
}

// GetBridge retrieves a bridge by ID
func (s *Store) GetBridge(id string) (*Bridge, error) {
	bridge := &Bridge{}
	var connectedAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, name, secret, type, connected, connected_at, created_at, updated_at FROM bridges WHERE id = ?`,
		id,
	).Scan(&bridge.ID, &bridge.Name, &bridge.Secret, &bridge.Type, &bridge.Connected, &connectedAt, &bridge.CreatedAt, &bridge.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrBridgeNotFound
	}
	
	if connectedAt.Valid {
		bridge.ConnectedAt = &connectedAt.Time
	}
	return bridge, err
}

// GetBridgeByName retrieves a bridge by name
func (s *Store) GetBridgeByName(name string) (*Bridge, error) {
	bridge := &Bridge{}
	var connectedAt sql.NullTime
	err := s.db.QueryRow(
		`SELECT id, name, secret, type, connected, connected_at, created_at, updated_at FROM bridges WHERE name = ?`,
		name,
	).Scan(&bridge.ID, &bridge.Name, &bridge.Secret, &bridge.Type, &bridge.Connected, &connectedAt, &bridge.CreatedAt, &bridge.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrBridgeNotFound
	}
	
	if connectedAt.Valid {
		bridge.ConnectedAt = &connectedAt.Time
	}
	return bridge, err
}

// CreateBridge adds a new bridge configuration
func (s *Store) CreateBridge(name, secret, bridgeType string) (*Bridge, error) {
	id := uuid.New().String()
	now := time.Now()

	_, err := s.db.Exec(
		`INSERT INTO bridges (id, name, secret, type, connected, connected_at, created_at, updated_at) VALUES (?, ?, ?, ?, 0, NULL, ?, ?)`,
		id, name, secret, bridgeType, now, now,
	)

	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrBridgeExists
		}
		return nil, err
	}

	return &Bridge{
		ID:        id,
		Name:      name,
		Secret:    secret,
		Type:      bridgeType,
		Connected: false,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// UpdateBridgeConnection updates the connection status of a bridge
func (s *Store) UpdateBridgeConnection(name string, connected bool) error {
	now := time.Now()
	
	var err error
	if connected {
		_, err = s.db.Exec(
			`UPDATE bridges SET connected = 1, connected_at = ?, updated_at = ? WHERE name = ?`,
			now, now, name,
		)
	} else {
		_, err = s.db.Exec(
			`UPDATE bridges SET connected = 0, updated_at = ? WHERE name = ?`,
			now, name,
		)
	}
	
	return err
}

// UpdateBridge updates a bridge's name and secret
func (s *Store) UpdateBridge(oldName, newName, secret string) (*Bridge, error) {
	now := time.Now()
	result, err := s.db.Exec(
		`UPDATE bridges SET name = ?, secret = ?, updated_at = ? WHERE name = ?`,
		newName, secret, now, oldName,
	)
	if err != nil {
		if isUniqueConstraintError(err) {
			return nil, ErrBridgeExists
		}
		return nil, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, ErrBridgeNotFound
	}
	return s.GetBridgeByName(newName)
}

// DeleteBridge removes a bridge configuration
func (s *Store) DeleteBridge(name string) error {
	result, err := s.db.Exec(`DELETE FROM bridges WHERE name = ?`, name)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrBridgeNotFound
	}

	return nil
}

// GetSetting retrieves a setting value by key. Returns ("", sql.ErrNoRows) if not found.
func (s *Store) GetSetting(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", sql.ErrNoRows
	}
	return value, err
}

// SetSetting stores a setting key/value pair, inserting or replacing as needed.
func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)`, key, value)
	return err
}

// CreateSession stores a session token with an expiry timestamp.
func (s *Store) CreateSession(token string, expiry time.Time) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)`,
		"ui_session_"+token, expiry.Format(time.RFC3339),
	)
	return err
}

// ValidateSession checks if a session token exists and has not expired. Deletes expired sessions.
func (s *Store) ValidateSession(token string) bool {
	key := "ui_session_" + token
	val, err := s.GetSetting(key)
	if err != nil {
		return false
	}
	expiry, err := time.Parse(time.RFC3339, val)
	if err != nil {
		return false
	}
	if time.Now().After(expiry) {
		_ = s.DeleteSetting(key)
		return false
	}
	return true
}

// DeleteSession removes a session token.
func (s *Store) DeleteSession(token string) error {
	return s.DeleteSetting("ui_session_" + token)
}

// DeleteSetting removes a setting by key.
func (s *Store) DeleteSetting(key string) error {
	_, err := s.db.Exec(`DELETE FROM settings WHERE key = ?`, key)
	return err
}
