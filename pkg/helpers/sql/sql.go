package sql

import (
	"database/sql"
)

// QuestionMark holds ?
const QuestionMark = "?"

type Table interface {
	name() string
	// cols() []string{}
	// values() []interface{}
	transaction() *sql.Tx
}

func Insert(t Table) int64, error {
	cols := t.cols()
	pholders := make([]interface{}, ln)
	values := t.getValues()

	for _, param := range cols {
		pholders = append(pholders, QuestionMark)
		if v, ok := form[param]; ok {
			values = append(values, NewNullString(v[0]))
		} else {
			values = append(values, NewNullString(""))
		}
	}

	ftable := fmt.Sprintf("`%s`", table)
	stmt, _, err := sq.
		Insert(ftable).Columns(cols...).
		Values(values...).
		ToSql()
	if err != nil {
		return 0, err
	}
	r, err := tx.Exec(stmt, values[len(values)-ln:]...)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	return id, nil
}