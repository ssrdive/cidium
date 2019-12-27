// Package sqlform provides helpers to insert form data directly to database
package sql

import (
	"database/sql"
	"net/url"
)

type SqlForm struct {
	TableName string
	RCols     []string
	OCols     []string
	Form      url.Values
}

// QuestionMark holds ?
const QuestionMark = "?"

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

func InsertForm(tx *sql.Tx, table string, rcols, ocols []string, form url.Values) (int64, error) {
	return 0, nil
}

// InsertForm Inserts a row to the table.
// ocols slice can be empty while rcols should at least have one element.
// func InsertForm(tx *sql.Tx, table string, rcols, ocols []string, form url.Values) (int64, error) {
// 	cols := append(rcols, ocols...)
// 	ln := len(cols)
// 	pholders := make([]interface{}, ln)
// 	values := make([]interface{}, ln)

// 	for _, param := range cols {
// 		pholders = append(pholders, QuestionMark)
// 		if v, ok := form[param]; ok {
// 			values = append(values, NewNullString(v[0]))
// 		} else {
// 			values = append(values, NewNullString(""))
// 		}
// 	}

// 	ftable := fmt.Sprintf("`%s`", table)
// 	stmt, _, err := sq.
// 		Insert(ftable).Columns(cols...).
// 		Values(pholders[len(pholders)-ln:]...).
// 		ToSql()
// 	if err != nil {
// 		return 0, err
// 	}
// 	r, err := tx.Exec(stmt, values[len(values)-ln:]...)
// 	if err != nil {
// 		tx.Rollback()
// 		return 0, err
// 	}
// 	id, err := r.LastInsertId()
// 	if err != nil {
// 		tx.Rollback()
// 		return 0, err
// 	}

// 	return id, nil
// }
