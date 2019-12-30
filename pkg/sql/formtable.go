// Package sql provides helpers to insert data directly to database
package sql

import (
	"database/sql"
	"fmt"
	"net/url"
)

// FormTable holds data to be inserted
type FormTable struct {
	TableName string
	RCols     []string
	OCols     []string
	Form      url.Values
	Tx        *sql.Tx
}

// Name returns table name
func (t FormTable) Name() string {
	return fmt.Sprintf("`%s`", t.TableName)
}

// Cols returns column names
func (t FormTable) Cols() []string {
	cols := append(t.RCols, t.OCols...)
	return cols
}

// Values returns column values
func (t FormTable) Values() []interface{} {
	values := make([]interface{}, len(t.Cols()))
	for i, col := range t.Cols() {
		if v, ok := t.Form[col]; ok {
			values[i] = NewNullString(v[0])
		} else {
			values[i] = NewNullString("")
		}
	}
	return values
}

// Transaction returns transaction object to query results from
func (t FormTable) Transaction() *sql.Tx {
	return t.Tx
}
