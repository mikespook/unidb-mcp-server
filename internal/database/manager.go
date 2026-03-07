package database

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrConnectionNotFound = errors.New("connection not found")
	ErrUnsupportedDriver  = errors.New("unsupported database driver")
	ErrConnectionExists   = errors.New("connection already exists")
)

// driverRegistry holds all registered drivers
var driverRegistry = map[string]Driver{
	"mysql":      &MySQLDriver{},
	"postgres":   &PostgresDriver{},
	"postgresql": &PostgresDriver{},
	"sqlite":     &SQLiteDriver{},
	"sqlite3":    &SQLiteDriver{},
	"mssql":      &MSSQLDriver{},
	"sqlserver":  &MSSQLDriver{},
	"mongodb":    &MongoDriver{},
}

// RegisterDriver registers a new database driver
func RegisterDriver(driver Driver) {
	driverRegistry[driver.Name()] = driver
}

// GetDriver retrieves a driver by name
func GetDriver(name string) (Driver, error) {
	driver, exists := driverRegistry[name]
	if !exists {
		return nil, ErrUnsupportedDriver
	}
	return driver, nil
}

// ListDrivers returns all registered driver names
func ListDrivers() []string {
	names := make([]string, 0, len(driverRegistry))
	for name := range driverRegistry {
		names = append(names, name)
	}
	return names
}

// Connection represents an active database connection
type Connection struct {
	ID          string    `json:"connection_id"`
	DSNName     string    `json:"dsn_name"`
	Driver      string    `json:"driver"`
	Handle      Handle    `json:"-"`
	driver      Driver    `json:"-"`
	ConnectedAt time.Time `json:"connected_at"`
}

// GetDriver returns the driver instance for this connection
func (c *Connection) GetDriver() Driver {
	return c.driver
}

// DriverManager manages active database connections
type DriverManager struct {
	mu          sync.RWMutex
	connections map[string]*Connection
}

// NewDriverManager creates a new connection manager
func NewDriverManager() *DriverManager {
	return &DriverManager{
		connections: make(map[string]*Connection),
	}
}

// Connect establishes a new database connection
func (m *DriverManager) Connect(id, name, driver, dsn string) (*Connection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.connections[id]; exists {
		return nil, ErrConnectionExists
	}

	drv, err := GetDriver(driver)
	if err != nil {
		return nil, err
	}

	h, err := drv.Open(dsn)
	if err != nil {
		return nil, err
	}

	conn := &Connection{
		ID:          id,
		DSNName:     name,
		Driver:      driver,
		Handle:      h,
		driver:      drv,
		ConnectedAt: time.Now(),
	}

	m.connections[id] = conn
	return conn, nil
}

// Disconnect closes and removes a database connection
func (m *DriverManager) Disconnect(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, exists := m.connections[id]
	if !exists {
		return ErrConnectionNotFound
	}

	if err := conn.driver.Close(conn.Handle); err != nil {
		return err
	}

	delete(m.connections, id)
	return nil
}

// Get retrieves a connection by ID
func (m *DriverManager) Get(id string) (*Connection, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conn, exists := m.connections[id]
	if !exists {
		return nil, ErrConnectionNotFound
	}
	return conn, nil
}

// List returns all active connections
func (m *DriverManager) List() []*Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()

	conns := make([]*Connection, 0, len(m.connections))
	for _, conn := range m.connections {
		conns = append(conns, conn)
	}
	return conns
}

// TestConnection tests if a DSN is valid without storing the connection
func TestConnection(driver, dsn string) error {
	drv, err := GetDriver(driver)
	if err != nil {
		return err
	}

	h, err := drv.Open(dsn)
	if err != nil {
		return err
	}
	return drv.Close(h)
}
