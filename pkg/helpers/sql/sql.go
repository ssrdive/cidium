package sql

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
)

const questionMark = "?"

// Table interface holders to generic table to
// perform INSERT, UPDATE and DELETE queries
type Table interface {
	Name() string
	Cols() ([]string, int)
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

func placeholders(size int) []interface{} {
	p := make([]interface{}, size)
	for i := range p {
		p[i] = questionMark
	}
	return p
}

func prepareInsert(tablename string, cols []string, size int) (string, []interface{}, error) {
	placeholders := placeholders(size)
	return sq.Insert(tablename).Columns(cols...).Values(placeholders...).ToSql()
}

func executeInsert(tx *sql.Tx, stmt string, values []interface{}) (int64, error) {
	result, err := tx.Exec(stmt, values...)
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
func Insert(t Table) (int64, error) {
	cols, ln := t.Cols()
	stmt, _, err := prepareInsert(t.Name(), cols, ln)
	if err != nil {
		return 0, err
	}
	id, err := executeInsert(t.Transaction(), stmt, t.Values())
	if err != nil {
		return 0, err
	}
	return id, nil
}
