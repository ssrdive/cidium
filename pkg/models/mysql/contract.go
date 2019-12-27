package mysql

import (
	"database/sql"
	"net/url"

	msql "github.com/ssrdive/cidium/pkg/helpers/sql"
)

// ContractModel struct holds database instance
type ContractModel struct {
	DB *sql.DB
}

// Insert creates a new contract
func (m *ContractModel) Insert(table string, rparams, oparams []string, form url.Values) (int64, error) {
	tx, err := m.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		_ = tx.Commit()
	}()

	contract := msql.FormTable{
		TableName: "group",
		RCols:     []string{"id", "name"},
		OCols:     []string{},
		Form:      form,
		Tx:        tx,
	}
	id, err := msql.Insert(contract)
	if err != nil {
		return 0, err
	}

	return id, nil
}
