// Package sql provides helpers to insert form data directly to database
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
func (s FormTable) Name() string {
	return fmt.Sprintf("`%s`", s.TableName)
}

// Cols returns column names
func (s FormTable) Cols() ([]string, int) {
	cols := append(s.RCols, s.OCols...)
	return cols, len(cols)
}

// Values returns column values
func (s FormTable) Values() []interface{} {
	cols, len := s.Cols()
	values := make([]interface{}, len)
	for i, col := range cols {
		if v, ok := s.Form[col]; ok {
			values[i] = NewNullString(v[0])
		} else {
			values[i] = NewNullString("")
		}
	}
	return values
}

// Transaction returns transaction object to query results from
func (s FormTable) Transaction() *sql.Tx {
	return s.Tx
}
