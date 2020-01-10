// Package sql provides helpers to insert data directly to database
package sql

import (
	"database/sql"
	"fmt"
)

// Table holds data to be inserted
type Table struct {
	TableName string
	Columns   []string
	Vals      []interface{}
	Tx        *sql.Tx
}

// Name returns table name
func (t Table) Name() string {
	return fmt.Sprintf("`%s`", t.TableName)
}

// Cols returns column names
func (t Table) Cols() []string {
	return t.Columns
}

// Values returns column values
func (t Table) Values() []interface{} {
	cols := t.Cols()
	values := make([]interface{}, len(cols))
	for i := range cols {
		values[i] = NewNullString(fmt.Sprintf("%v", t.Vals[i]))
	}
	return values
}

// Transaction returns transaction object to query results from
func (t Table) Transaction() *sql.Tx {
	return t.Tx
}

// UpdateTable holds data to be updated
type UpdateTable struct {
	Table
	WColumns []string
	WVals    []string
}

// WhereCols returns columns included in where clause
func (t UpdateTable) WhereCols() []string {
	return t.WColumns
}

// WhereValues returns values included in where clause
func (t UpdateTable) WhereValues() []interface{} {
	wvals := make([]interface{}, len(t.WColumns))
	for i := range t.WColumns {
		wvals[i] = t.WVals[i]
	}
	return wvals
}
