// Package gostrike provides the public SDK for GoStrike plugins.
// This file provides database access for plugins.
package gostrike

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/corrreia/gostrike/internal/modules/database"
)

// pluginDBs holds all plugin database instances
var (
	pluginDBs   = make(map[string]*PluginDB)
	pluginDBsMu sync.RWMutex
)

// PluginDB provides isolated database access for a plugin
// Each plugin gets its own SQLite database file
type PluginDB struct {
	slug string
	db   *sql.DB
	path string
	mu   sync.RWMutex
}

// GetPluginDB returns or creates an isolated database for a plugin
// Database files are stored in data/plugins/[slug].db
func GetPluginDB(slug string) (*PluginDB, error) {
	pluginDBsMu.Lock()
	defer pluginDBsMu.Unlock()

	// Return existing connection if available
	if pdb, exists := pluginDBs[slug]; exists {
		return pdb, nil
	}

	// Create the plugins data directory if it doesn't exist
	pluginsDir := filepath.Join("data", "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create plugins data directory: %w", err)
	}

	// Create database path
	dbPath := filepath.Join(pluginsDir, slug+".db")

	// Open SQLite database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to plugin database: %w", err)
	}

	// Configure connection pool (smaller than main DB since it's per-plugin)
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)

	pdb := &PluginDB{
		slug: slug,
		db:   db,
		path: dbPath,
	}

	pluginDBs[slug] = pdb
	return pdb, nil
}

// ClosePluginDB closes and removes a plugin's database connection
func ClosePluginDB(slug string) error {
	pluginDBsMu.Lock()
	defer pluginDBsMu.Unlock()

	pdb, exists := pluginDBs[slug]
	if !exists {
		return nil
	}

	if err := pdb.db.Close(); err != nil {
		return err
	}

	delete(pluginDBs, slug)
	return nil
}

// CloseAllPluginDBs closes all plugin database connections
func CloseAllPluginDBs() {
	pluginDBsMu.Lock()
	defer pluginDBsMu.Unlock()

	for slug, pdb := range pluginDBs {
		pdb.db.Close()
		delete(pluginDBs, slug)
	}
}

// Slug returns the plugin's slug
func (pdb *PluginDB) Slug() string {
	return pdb.slug
}

// Path returns the database file path
func (pdb *PluginDB) Path() string {
	return pdb.path
}

// DB returns the underlying sql.DB for advanced use
func (pdb *PluginDB) DB() *sql.DB {
	pdb.mu.RLock()
	defer pdb.mu.RUnlock()
	return pdb.db
}

// IsConnected returns true if connected to the database
func (pdb *PluginDB) IsConnected() bool {
	pdb.mu.RLock()
	defer pdb.mu.RUnlock()
	return pdb.db != nil
}

// Query executes a query and returns rows
func (pdb *PluginDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	pdb.mu.RLock()
	defer pdb.mu.RUnlock()

	if pdb.db == nil {
		return nil, fmt.Errorf("plugin database not connected")
	}
	return pdb.db.Query(query, args...)
}

// QueryRow executes a query and returns a single row
func (pdb *PluginDB) QueryRow(query string, args ...interface{}) *sql.Row {
	pdb.mu.RLock()
	defer pdb.mu.RUnlock()

	if pdb.db == nil {
		return nil
	}
	return pdb.db.QueryRow(query, args...)
}

// Exec executes a query without returning rows
func (pdb *PluginDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	pdb.mu.RLock()
	defer pdb.mu.RUnlock()

	if pdb.db == nil {
		return nil, fmt.Errorf("plugin database not connected")
	}
	return pdb.db.Exec(query, args...)
}

// Begin starts a transaction
func (pdb *PluginDB) Begin() (*sql.Tx, error) {
	pdb.mu.RLock()
	defer pdb.mu.RUnlock()

	if pdb.db == nil {
		return nil, fmt.Errorf("plugin database not connected")
	}
	return pdb.db.Begin()
}

// Close closes the plugin database connection
func (pdb *PluginDB) Close() error {
	return ClosePluginDB(pdb.slug)
}

// Database interface for plugins
type Database interface {
	// Query executes a query and returns rows
	Query(query string, args ...interface{}) (*sql.Rows, error)
	// QueryRow executes a query and returns a single row
	QueryRow(query string, args ...interface{}) *sql.Row
	// Exec executes a query without returning rows
	Exec(query string, args ...interface{}) (sql.Result, error)
	// Begin starts a transaction
	Begin() (*sql.Tx, error)
	// IsConnected returns true if connected to the database
	IsConnected() bool
}

// GetDatabase returns the database module instance
func GetDatabase() *database.Module {
	return database.Get()
}

// IsDatabaseEnabled returns true if the database is connected
func IsDatabaseEnabled() bool {
	mod := database.Get()
	return mod != nil && mod.IsConnected()
}

// DB returns the underlying sql.DB instance for advanced use
func DB() *sql.DB {
	mod := database.Get()
	if mod != nil {
		return mod.DB()
	}
	return nil
}

// Query executes a query and returns rows
func Query(query string, args ...interface{}) (*sql.Rows, error) {
	mod := database.Get()
	if mod != nil {
		return mod.Query(query, args...)
	}
	return nil, sql.ErrConnDone
}

// QueryRow executes a query and returns a single row
func QueryRow(query string, args ...interface{}) *sql.Row {
	mod := database.Get()
	if mod != nil {
		return mod.QueryRow(query, args...)
	}
	return nil
}

// Exec executes a query without returning rows
func Exec(query string, args ...interface{}) (sql.Result, error) {
	mod := database.Get()
	if mod != nil {
		return mod.Exec(query, args...)
	}
	return nil, sql.ErrConnDone
}

// Begin starts a transaction
func Begin() (*sql.Tx, error) {
	mod := database.Get()
	if mod != nil {
		return mod.Begin()
	}
	return nil, sql.ErrConnDone
}

// RegisterMigration registers a database migration
func RegisterMigration(version int, description string, up, down func(*sql.DB) error) {
	mod := database.Get()
	if mod != nil {
		mod.RegisterMigration(database.Migration{
			Version:     version,
			Description: description,
			Up:          up,
			Down:        down,
		})
	}
}

// Query builders - re-export from database module

// Table starts a SELECT query builder
func Table(name string) *database.QueryBuilder {
	return database.Table(name)
}

// Insert starts an INSERT query builder
func Insert(table string) *database.InsertBuilder {
	return database.Insert(table)
}

// Update starts an UPDATE query builder
func Update(table string) *database.UpdateBuilder {
	return database.Update(table)
}

// Delete starts a DELETE query builder
func Delete(table string) *database.DeleteBuilder {
	return database.Delete(table)
}
