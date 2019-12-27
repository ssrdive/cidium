package mysql

import (
	"database/sql"
	"net/url"

	"github.com/ssrdive/cidium/pkg/helpers"
)

// ContractModel struct holds database instance
type ContractModel struct {
	DB *sql.DB
}

func (m *ContractModel) Insert(table string, rParams, oParams []string, form url.Values) (int64, error) {
	id, err := helpers.InsertFormFromParams(m.DB, table, rParams, oParams, form)
	if err != nil {
		return 0, err
	}

	return id, nil
}
