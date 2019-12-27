package helpers

import (
	"database/sql"
	"net/url"

	sq "github.com/Masterminds/squirrel"
)

func InsertFormFromParams(db *sql.DB, table string, rParams, oParams []string, form url.Values) (int64, error) {
	params := append(rParams, oParams...)
	pLen := len(params)

	p := make([]interface{}, len(params))
	v := make([]interface{}, len(params))

	for _, param := range params {
		p = append(p, "?")
		if _, ok := form[param]; ok {
			v = append(v, form.Get(param))
		} else {
			v = append(v, "")
		}
	}

	stmt, _, err := sq.
		Insert(table).Columns(params...).
		Values(p[len(p)-pLen:]...).
		ToSql()
	if err != nil {
		return 0, err
	}

	r, err := db.Exec(stmt, v[len(v)-pLen:]...)
	if err != nil {
		return 0, err
	}

	id, err := r.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}
