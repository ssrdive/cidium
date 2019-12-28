package mysql

import (
	"database/sql"
	"net/url"

	msql "github.com/ssrdive/cidium/pkg/sql"
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

	// contract := msql.Table{
	// 	TableName: "group",
	// 	Columns:   []string{"id", "name"},
	// 	Vals:      []string{"4", "Love Quinn"},
	// 	Tx:        tx,
	// }

	group := msql.UpdateTable{
		Table: msql.Table{
			TableName: "group",
			Columns:   []string{"name"},
			Vals:      []string{"Hello World from Go!"},
			Tx:        tx,
		},
		WColumns: []string{"id"},
		WVals:    []string{"23423"},
	}

	id, err := msql.Update(group)
	if err != nil {
		return 0, err
	}

	return id, nil
}
