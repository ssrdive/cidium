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
func (t FormTable) Cols() ([]string, int) {
	cols := append(t.RCols, t.OCols...)
	return cols, len(cols)
}

// Values returns column values
func (t FormTable) Values() []interface{} {
	cols, len := t.Cols()
	values := make([]interface{}, len)
	for i, col := range cols {
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
