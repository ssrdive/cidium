// Package sql provides helpers to insert data directly to database
package sql

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
)

// InsertTable interface define table structure
// perform INSERT queries
type InsertTable interface {
	Name() string
	Cols() []string
	Values() []interface{}
	Transaction() *sql.Tx
}

// NewNullString fuctions returns a NULL if the passed string is empty
func NewNullString(s string) sql.NullString {
	if len(s) == 0 {
		return sql.NullString{}
	}
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}

func prepareInsert(tablename string, cols []string, values []interface{}) sq.InsertBuilder {
	return sq.Insert(tablename).Columns(cols...).Values(values...)
}

func executeInsert(tx *sql.Tx, ib sq.InsertBuilder) (int64, error) {
	result, err := ib.RunWith(tx).Exec()
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	return id, nil
}

// Insert prepares INSERT statement and executes it
func Insert(t InsertTable) (int64, error) {
	ib := prepareInsert(t.Name(), t.Cols(), t.Values())
	id, err := executeInsert(t.Transaction(), ib)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// UdpateTable interface define table structure
// perform UPDATE queries
type UdpateTable interface {
	InsertTable
	WhereCols() []string
	WhereValues() []interface{}
}

func prepareUpdate(tablename string, cols []string, values []interface{}, wcols []string, wvalues []interface{}) sq.UpdateBuilder {
	ub := sq.Update(tablename)
	for i, c := range cols {
		ub = ub.Set(c, values[i])
	}
	for i, c := range wcols {
		ub = ub.Where(sq.Eq{c: wvalues[i]})
	}
	return ub
}

func executeUpdate(tx *sql.Tx, ub sq.UpdateBuilder) (int64, error) {
	result, err := ub.RunWith(tx).Exec()
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	r, err := result.RowsAffected()
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	return r, nil
}

// Update prepares UPDATE statement and executes it
func Update(t UpdateTable) (int64, error) {
	ub := prepareUpdate(t.Name(), t.Cols(), t.Values(), t.WhereCols(), t.WhereValues())
	r, err := executeUpdate(t.Transaction(), ub)
	if err != nil {
		return 0, err
	}
	return r, err
}
