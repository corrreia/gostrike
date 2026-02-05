// Package database provides a database abstraction module for GoStrike.
// This file contains query builder helpers and utilities.
package database

import (
	"fmt"
	"strings"
)

// QueryBuilder helps build SQL queries
type QueryBuilder struct {
	table      string
	columns    []string
	conditions []string
	args       []interface{}
	orderBy    string
	limit      int
	offset     int
}

// Table sets the table name
func Table(name string) *QueryBuilder {
	return &QueryBuilder{
		table:      name,
		columns:    []string{"*"},
		conditions: make([]string, 0),
		args:       make([]interface{}, 0),
	}
}

// Select sets the columns to select
func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	qb.columns = columns
	return qb
}

// Where adds a WHERE condition
func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	qb.conditions = append(qb.conditions, condition)
	qb.args = append(qb.args, args...)
	return qb
}

// OrderBy sets the ORDER BY clause
func (qb *QueryBuilder) OrderBy(column string, desc bool) *QueryBuilder {
	direction := "ASC"
	if desc {
		direction = "DESC"
	}
	qb.orderBy = column + " " + direction
	return qb
}

// Limit sets the LIMIT clause
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limit = limit
	return qb
}

// Offset sets the OFFSET clause
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offset = offset
	return qb
}

// BuildSelect builds a SELECT query
func (qb *QueryBuilder) BuildSelect() (string, []interface{}) {
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(qb.columns, ", "), qb.table)

	if len(qb.conditions) > 0 {
		query += " WHERE " + strings.Join(qb.conditions, " AND ")
	}

	if qb.orderBy != "" {
		query += " ORDER BY " + qb.orderBy
	}

	if qb.limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", qb.limit)
	}

	if qb.offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", qb.offset)
	}

	return query, qb.args
}

// InsertBuilder helps build INSERT queries
type InsertBuilder struct {
	table   string
	columns []string
	values  []interface{}
}

// Insert creates an insert builder for a table
func Insert(table string) *InsertBuilder {
	return &InsertBuilder{
		table:   table,
		columns: make([]string, 0),
		values:  make([]interface{}, 0),
	}
}

// Set sets a column value
func (ib *InsertBuilder) Set(column string, value interface{}) *InsertBuilder {
	ib.columns = append(ib.columns, column)
	ib.values = append(ib.values, value)
	return ib
}

// Build builds the INSERT query
func (ib *InsertBuilder) Build() (string, []interface{}) {
	placeholders := make([]string, len(ib.columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		ib.table,
		strings.Join(ib.columns, ", "),
		strings.Join(placeholders, ", "),
	)

	return query, ib.values
}

// UpdateBuilder helps build UPDATE queries
type UpdateBuilder struct {
	table      string
	sets       []string
	conditions []string
	args       []interface{}
}

// Update creates an update builder for a table
func Update(table string) *UpdateBuilder {
	return &UpdateBuilder{
		table:      table,
		sets:       make([]string, 0),
		conditions: make([]string, 0),
		args:       make([]interface{}, 0),
	}
}

// Set sets a column value
func (ub *UpdateBuilder) Set(column string, value interface{}) *UpdateBuilder {
	ub.sets = append(ub.sets, column+" = ?")
	ub.args = append(ub.args, value)
	return ub
}

// Where adds a WHERE condition
func (ub *UpdateBuilder) Where(condition string, args ...interface{}) *UpdateBuilder {
	ub.conditions = append(ub.conditions, condition)
	ub.args = append(ub.args, args...)
	return ub
}

// Build builds the UPDATE query
func (ub *UpdateBuilder) Build() (string, []interface{}) {
	query := fmt.Sprintf("UPDATE %s SET %s", ub.table, strings.Join(ub.sets, ", "))

	if len(ub.conditions) > 0 {
		query += " WHERE " + strings.Join(ub.conditions, " AND ")
	}

	return query, ub.args
}

// DeleteBuilder helps build DELETE queries
type DeleteBuilder struct {
	table      string
	conditions []string
	args       []interface{}
}

// Delete creates a delete builder for a table
func Delete(table string) *DeleteBuilder {
	return &DeleteBuilder{
		table:      table,
		conditions: make([]string, 0),
		args:       make([]interface{}, 0),
	}
}

// Where adds a WHERE condition
func (db *DeleteBuilder) Where(condition string, args ...interface{}) *DeleteBuilder {
	db.conditions = append(db.conditions, condition)
	db.args = append(db.args, args...)
	return db
}

// Build builds the DELETE query
func (db *DeleteBuilder) Build() (string, []interface{}) {
	query := fmt.Sprintf("DELETE FROM %s", db.table)

	if len(db.conditions) > 0 {
		query += " WHERE " + strings.Join(db.conditions, " AND ")
	}

	return query, db.args
}

// ScanRow is a helper to scan a row into a map
func ScanRow(rows interface{ Columns() ([]string, error) }, scanner interface{ Scan(...interface{}) error }) (map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	if err := scanner.Scan(valuePtrs...); err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for i, col := range columns {
		result[col] = values[i]
	}

	return result, nil
}
