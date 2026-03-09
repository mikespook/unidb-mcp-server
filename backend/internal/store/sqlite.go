package store

import (
	"database/sql"
	"errors"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/google/uuid"
)

var (
	ErrDSNNotFound  = errors.New("DSN not found")
	ErrDSNExists    = errors.New("DSN name already exists")
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("username already exists")
	ErrTeamNotFound = errors.New("team not found")
	ErrTeamExists   = errors.New("team already exists")
)

// User represents a UI/API user
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	JWTSecret    string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Team represents a group that users and DSNs can belong to
type Team struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// DSN represents a database connection configuration
type DSN struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Driver    string    `json:"driver"`
	DSN       string    `json:"dsn"`
	Connected bool      `json:"connected,omitempty"` // populated from in-memory state, not DB
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
	PRAGMA foreign_keys = ON;
	CREATE TABLE IF NOT EXISTS users (
		id            TEXT PRIMARY KEY,
		username      TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		jwt_secret    TEXT NOT NULL,
		created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS teams (
		id         TEXT PRIMARY KEY,
		name       TEXT UNIQUE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS user_teams (
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
		PRIMARY KEY (user_id, team_id)
	);
	CREATE TABLE IF NOT EXISTS dsns (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		driver TEXT NOT NULL,
		dsn TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS dsn_teams (
		dsn_id  TEXT NOT NULL REFERENCES dsns(id) ON DELETE CASCADE,
		team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
		PRIMARY KEY (dsn_id, team_id)
	);
	CREATE TABLE IF NOT EXISTS settings (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_teams_name ON teams(name);
	CREATE INDEX IF NOT EXISTS idx_dsns_name ON dsns(name);
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
	return err.Error() == "UNIQUE constraint failed: dsns.name"
}

// GetByNameAndDriver retrieves a DSN by name and driver (used by SSE auth for sqlite-bridge).
func (s *Store) GetByNameAndDriver(name, driver string) (*DSN, error) {
	dsn := &DSN{}
	err := s.db.QueryRow(
		`SELECT id, name, driver, dsn, created_at, updated_at FROM dsns WHERE name = ? AND driver = ?`,
		name, driver,
	).Scan(&dsn.ID, &dsn.Name, &dsn.Driver, &dsn.DSN, &dsn.CreatedAt, &dsn.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrDSNNotFound
	}
	return dsn, err
}

// TouchUpdatedAt bumps the updated_at timestamp for a DSN (used on bridge connect/disconnect).
func (s *Store) TouchUpdatedAt(id string) error {
	_, err := s.db.Exec(`UPDATE dsns SET updated_at = ? WHERE id = ?`, time.Now(), id)
	return err
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

// --- User methods ---

// IsInitialized returns true if the initial admin user has been recorded in settings.
func (s *Store) IsInitialized() (bool, error) {
	id, err := s.GetSetting("init_admin_user_id")
	if err != nil {
		return false, nil
	}
	return id != "", nil
}

// GetInitialUserID returns the ID of the initial admin user recorded at setup time.
func (s *Store) GetInitialUserID() (string, error) {
	id, err := s.GetSetting("init_admin_user_id")
	if err != nil {
		return "", ErrUserNotFound
	}
	return id, nil
}

// CreateUser inserts a new user record.
func (s *Store) CreateUser(username, passwordHash, jwtSecret string) (*User, error) {
	id := uuid.New().String()
	now := time.Now()
	_, err := s.db.Exec(
		`INSERT INTO users (id, username, password_hash, jwt_secret, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		id, username, passwordHash, jwtSecret, now, now,
	)
	if err != nil {
		if isUserUniqueError(err) {
			return nil, ErrUserExists
		}
		return nil, err
	}
	return &User{ID: id, Username: username, PasswordHash: passwordHash, JWTSecret: jwtSecret, CreatedAt: now, UpdatedAt: now}, nil
}

// GetUser retrieves a user by ID.
func (s *Store) GetUser(id string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		`SELECT id, username, password_hash, jwt_secret, created_at, updated_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.JWTSecret, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	return u, err
}

// GetUserByUsername retrieves a user by username.
func (s *Store) GetUserByUsername(username string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		`SELECT id, username, password_hash, jwt_secret, created_at, updated_at FROM users WHERE username = ?`, username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.JWTSecret, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	return u, err
}

// ListUsers returns all users ordered by username.
func (s *Store) ListUsers() ([]*User, error) {
	rows, err := s.db.Query(`SELECT id, username, created_at, updated_at FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// UpdateUserPassword updates the bcrypt password hash for a user.
func (s *Store) UpdateUserPassword(id, passwordHash string) error {
	result, err := s.db.Exec(`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`, passwordHash, time.Now(), id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

// DeleteUser removes a user by ID.
func (s *Store) DeleteUser(id string) error {
	result, err := s.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

// ListAllJWTSecrets returns all per-user JWT secrets (for MCP middleware).
func (s *Store) ListAllJWTSecrets() ([]string, error) {
	rows, err := s.db.Query(`SELECT jwt_secret FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var secrets []string
	for rows.Next() {
		var secret string
		if err := rows.Scan(&secret); err != nil {
			return nil, err
		}
		secrets = append(secrets, secret)
	}
	return secrets, rows.Err()
}

// GetUserJWTSecret returns the JWT secret for a specific user.
func (s *Store) GetUserJWTSecret(id string) (string, error) {
	var secret string
	err := s.db.QueryRow(`SELECT jwt_secret FROM users WHERE id = ?`, id).Scan(&secret)
	if err == sql.ErrNoRows {
		return "", ErrUserNotFound
	}
	return secret, err
}

// CreateSessionWithUser stores session expiry and associated user ID.
func (s *Store) CreateSessionWithUser(token string, expiry time.Time, userID string) error {
	if err := s.CreateSession(token, expiry); err != nil {
		return err
	}
	return s.SetSetting("ui_session_user_"+token, userID)
}

// GetSessionUser returns the user ID associated with a session token.
func (s *Store) GetSessionUser(token string) (string, error) {
	return s.GetSetting("ui_session_user_" + token)
}

// DeleteSessionWithUser removes both session keys.
func (s *Store) DeleteSessionWithUser(token string) error {
	_ = s.DeleteSetting("ui_session_user_" + token)
	return s.DeleteSession(token)
}

func isUserUniqueError(err error) bool {
	return err != nil && err.Error() == "UNIQUE constraint failed: users.username"
}

// --- Team methods ---

// EnsureDefaultTeam creates the default team if it doesn't exist, then returns it.
func (s *Store) EnsureDefaultTeam() (*Team, error) {
	t, err := s.GetTeamByName("default")
	if err == nil {
		return t, nil
	}
	if err != ErrTeamNotFound {
		return nil, err
	}
	return s.CreateTeam("default")
}

// CreateTeam inserts a new team.
func (s *Store) CreateTeam(name string) (*Team, error) {
	id := uuid.New().String()
	now := time.Now()
	_, err := s.db.Exec(`INSERT INTO teams (id, name, created_at) VALUES (?, ?, ?)`, id, name, now)
	if err != nil {
		if isTeamUniqueError(err) {
			return nil, ErrTeamExists
		}
		return nil, err
	}
	return &Team{ID: id, Name: name, CreatedAt: now}, nil
}

// GetTeam retrieves a team by ID.
func (s *Store) GetTeam(id string) (*Team, error) {
	t := &Team{}
	err := s.db.QueryRow(`SELECT id, name, created_at FROM teams WHERE id = ?`, id).Scan(&t.ID, &t.Name, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrTeamNotFound
	}
	return t, err
}

// GetTeamByName retrieves a team by name.
func (s *Store) GetTeamByName(name string) (*Team, error) {
	t := &Team{}
	err := s.db.QueryRow(`SELECT id, name, created_at FROM teams WHERE name = ?`, name).Scan(&t.ID, &t.Name, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, ErrTeamNotFound
	}
	return t, err
}

// ListTeams returns all teams ordered by name.
func (s *Store) ListTeams() ([]*Team, error) {
	rows, err := s.db.Query(`SELECT id, name, created_at FROM teams ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var teams []*Team
	for rows.Next() {
		t := &Team{}
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

// DeleteTeam removes a team by ID.
func (s *Store) DeleteTeam(id string) error {
	result, err := s.db.Exec(`DELETE FROM teams WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrTeamNotFound
	}
	return nil
}

func isTeamUniqueError(err error) bool {
	return err != nil && err.Error() == "UNIQUE constraint failed: teams.name"
}

// --- User-Team membership ---

// AddUserToTeam assigns a user to a team.
func (s *Store) AddUserToTeam(userID, teamID string) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO user_teams (user_id, team_id) VALUES (?, ?)`, userID, teamID)
	return err
}

// RemoveUserFromTeam removes a user from a team.
func (s *Store) RemoveUserFromTeam(userID, teamID string) error {
	_, err := s.db.Exec(`DELETE FROM user_teams WHERE user_id = ? AND team_id = ?`, userID, teamID)
	return err
}

// GetUserTeams returns all teams a user belongs to.
func (s *Store) GetUserTeams(userID string) ([]*Team, error) {
	rows, err := s.db.Query(
		`SELECT t.id, t.name, t.created_at FROM teams t JOIN user_teams ut ON t.id = ut.team_id WHERE ut.user_id = ? ORDER BY t.name`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var teams []*Team
	for rows.Next() {
		t := &Team{}
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

// GetTeamUsers returns all users in a team.
func (s *Store) GetTeamUsers(teamID string) ([]*User, error) {
	rows, err := s.db.Query(
		`SELECT u.id, u.username, u.created_at, u.updated_at FROM users u JOIN user_teams ut ON u.id = ut.user_id WHERE ut.team_id = ? ORDER BY u.username`,
		teamID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*User
	for rows.Next() {
		u := &User{}
		if err := rows.Scan(&u.ID, &u.Username, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// IsUserAdmin returns true if the user is a member of the admin team.
func (s *Store) IsUserAdmin(userID string) (bool, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM user_teams ut JOIN teams t ON t.id = ut.team_id WHERE ut.user_id = ? AND t.name = 'admin'`,
		userID,
	).Scan(&count)
	return count > 0, err
}

// --- DSN-Team membership ---

// AddDSNToTeam assigns a DSN to a team.
func (s *Store) AddDSNToTeam(dsnID, teamID string) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO dsn_teams (dsn_id, team_id) VALUES (?, ?)`, dsnID, teamID)
	return err
}

// RemoveDSNFromTeam removes a DSN from a team.
func (s *Store) RemoveDSNFromTeam(dsnID, teamID string) error {
	_, err := s.db.Exec(`DELETE FROM dsn_teams WHERE dsn_id = ? AND team_id = ?`, dsnID, teamID)
	return err
}

// GetDSNTeams returns all teams a DSN belongs to.
func (s *Store) GetDSNTeams(dsnID string) ([]*Team, error) {
	rows, err := s.db.Query(
		`SELECT t.id, t.name, t.created_at FROM teams t JOIN dsn_teams dt ON t.id = dt.team_id WHERE dt.dsn_id = ? ORDER BY t.name`,
		dsnID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var teams []*Team
	for rows.Next() {
		t := &Team{}
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

// GetTeamDSNs returns all DSNs assigned to a team.
func (s *Store) GetTeamDSNs(teamID string) ([]*DSN, error) {
	rows, err := s.db.Query(
		`SELECT d.id, d.name, d.driver, d.dsn, d.created_at, d.updated_at FROM dsns d JOIN dsn_teams dt ON d.id = dt.dsn_id WHERE dt.team_id = ? ORDER BY d.name`,
		teamID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var dsns []*DSN
	for rows.Next() {
		d := &DSN{}
		if err := rows.Scan(&d.ID, &d.Name, &d.Driver, &d.DSN, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		dsns = append(dsns, d)
	}
	return dsns, rows.Err()
}
