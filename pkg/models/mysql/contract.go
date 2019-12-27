package mysql

import (
	"database/sql"
	"net/url"

	"github.com/ssrdive/cidium/pkg/helpers/sql"
)

// ContractModel struct holds database instance
type ContractModel struct {
	DB *sql.DB
}

// Insert creates a new contract
func (m *ContractModel) Insert(table string, rParams, oParams []string, form url.Values) (int64, error) {
	tx, err := m.DB.Begin()
	if err != nil {
		return 0, err
	}

	id, err := sql.InsertForm(tx, "group", []string{"id", "name"}, []string{}, form)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}

	return id, nil
}
