// Package database provides a database abstraction module for GoStrike.
// It supports SQLite and MySQL with a unified interface.
package database

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// Driver represents a database driver type
type Driver string

const (
	DriverSQLite Driver = "sqlite"
	DriverMySQL  Driver = "mysql"
)

// Module implements the database module
type Module struct {
	mu         sync.RWMutex
	db         *sql.DB
	driver     Driver
	config     *Config
	configPath string
	connected  bool
	migrations []Migration
}

// Config represents the database configuration
type Config struct {
	Enabled     bool   `json:"enabled"`
	Driver      string `json:"driver"`      // "sqlite" or "mysql"
	SQLitePath  string `json:"sqlite_path"` // Path to SQLite database file
	MySQLHost   string `json:"mysql_host"`  // MySQL host
	MySQLPort   int    `json:"mysql_port"`  // MySQL port
	MySQLUser   string `json:"mysql_user"`  // MySQL username
	MySQLPass   string `json:"mysql_pass"`  // MySQL password
	MySQLDB     string `json:"mysql_db"`    // MySQL database name
	MaxOpenConn int    `json:"max_open_conn"`
	MaxIdleConn int    `json:"max_idle_conn"`
	MaxLifetime int    `json:"max_lifetime"` // Connection lifetime in seconds
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Enabled:     false,
		Driver:      "sqlite",
		SQLitePath:  "data/gostrike.db",
		MySQLHost:   "localhost",
		MySQLPort:   3306,
		MySQLUser:   "gostrike",
		MySQLPass:   "",
		MySQLDB:     "gostrike",
		MaxOpenConn: 10,
		MaxIdleConn: 5,
		MaxLifetime: 300,
	}
}

// Migration represents a database migration
type Migration struct {
	Version     int
	Description string
	Up          func(db *sql.DB) error
	Down        func(db *sql.DB) error
}

// instance is the singleton instance
var instance *Module

// New creates a new database module
func New() *Module {
	if instance != nil {
		return instance
	}
	instance = &Module{
		config:     DefaultConfig(),
		configPath: "configs/database.json",
		migrations: make([]Migration, 0),
	}
	return instance
}

// Get returns the singleton instance
func Get() *Module {
	return instance
}

// Name returns the module name
func (m *Module) Name() string {
	return "Database"
}

// Version returns the module version
func (m *Module) Version() string {
	return "1.0.0"
}

// Priority returns the module load priority
func (m *Module) Priority() int {
	return 20 // Load early but after permissions
}

// Init initializes the database module
func (m *Module) Init() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load config
	if err := m.loadConfig(); err != nil {
		m.config = DefaultConfig()
	}

	if !m.config.Enabled {
		return nil
	}

	// Connect to database
	if err := m.connectLocked(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Run migrations
	if err := m.runMigrationsLocked(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// Shutdown shuts down the database module
func (m *Module) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.db != nil {
		if err := m.db.Close(); err != nil {
			return err
		}
		m.db = nil
		m.connected = false
	}

	return nil
}

// loadConfig loads the configuration
func (m *Module) loadConfig() error {
	// For now, use default config
	// TODO: Load from file
	m.config = DefaultConfig()
	return nil
}

// connectLocked connects to the database (must be called with lock held)
func (m *Module) connectLocked() error {
	if m.connected {
		return nil
	}

	var db *sql.DB
	var err error

	switch Driver(m.config.Driver) {
	case DriverSQLite:
		db, err = m.connectSQLite()
	case DriverMySQL:
		db, err = m.connectMySQL()
	default:
		return fmt.Errorf("unsupported driver: %s", m.config.Driver)
	}

	if err != nil {
		return err
	}

	// Configure connection pool
	db.SetMaxOpenConns(m.config.MaxOpenConn)
	db.SetMaxIdleConns(m.config.MaxIdleConn)
	db.SetConnMaxLifetime(time.Duration(m.config.MaxLifetime) * time.Second)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	m.db = db
	m.driver = Driver(m.config.Driver)
	m.connected = true

	return nil
}

// connectSQLite connects to SQLite database
func (m *Module) connectSQLite() (*sql.DB, error) {
	// Note: This requires the modernc.org/sqlite driver or github.com/mattn/go-sqlite3
	// For now, we'll use the interface and expect the driver to be registered
	dsn := m.config.SQLitePath
	return sql.Open("sqlite", dsn)
}

// connectMySQL connects to MySQL database
func (m *Module) connectMySQL() (*sql.DB, error) {
	// Note: This requires the github.com/go-sql-driver/mysql driver
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		m.config.MySQLUser,
		m.config.MySQLPass,
		m.config.MySQLHost,
		m.config.MySQLPort,
		m.config.MySQLDB,
	)
	return sql.Open("mysql", dsn)
}

// runMigrationsLocked runs pending migrations
func (m *Module) runMigrationsLocked() error {
	if m.db == nil {
		return nil
	}

	// Create migrations table if not exists
	createTable := `
		CREATE TABLE IF NOT EXISTS gostrike_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	if _, err := m.db.Exec(createTable); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	var currentVersion int
	row := m.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM gostrike_migrations")
	if err := row.Scan(&currentVersion); err != nil {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	// Run pending migrations
	for _, migration := range m.migrations {
		if migration.Version > currentVersion {
			if err := migration.Up(m.db); err != nil {
				return fmt.Errorf("migration %d failed: %w", migration.Version, err)
			}

			_, err := m.db.Exec("INSERT INTO gostrike_migrations (version) VALUES (?)", migration.Version)
			if err != nil {
				return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
			}
		}
	}

	return nil
}

// ============================================================
// Public API
// ============================================================

// IsConnected returns true if connected to the database
func (m *Module) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

// GetDriver returns the current database driver
func (m *Module) GetDriver() Driver {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.driver
}

// DB returns the underlying database connection
func (m *Module) DB() *sql.DB {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.db
}

// Query executes a query and returns rows
func (m *Module) Query(query string, args ...interface{}) (*sql.Rows, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	return m.db.Query(query, args...)
}

// QueryRow executes a query and returns a single row
func (m *Module) QueryRow(query string, args ...interface{}) *sql.Row {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.db == nil {
		return nil
	}

	return m.db.QueryRow(query, args...)
}

// Exec executes a query without returning rows
func (m *Module) Exec(query string, args ...interface{}) (sql.Result, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	return m.db.Exec(query, args...)
}

// Begin starts a transaction
func (m *Module) Begin() (*sql.Tx, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	return m.db.Begin()
}

// RegisterMigration registers a migration
func (m *Module) RegisterMigration(migration Migration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Insert in order by version
	inserted := false
	for i, existing := range m.migrations {
		if migration.Version < existing.Version {
			m.migrations = append(m.migrations[:i], append([]Migration{migration}, m.migrations[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		m.migrations = append(m.migrations, migration)
	}
}

// SetConfig sets the configuration
func (m *Module) SetConfig(config *Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = config
}

// GetConfig returns a copy of the configuration
func (m *Module) GetConfig() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m.config
}
