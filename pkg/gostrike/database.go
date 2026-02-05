// Package gostrike provides the public SDK for GoStrike plugins.
// This file provides database access for plugins.
package gostrike

import (
	"database/sql"

	"github.com/corrreia/gostrike/internal/modules/database"
)

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
