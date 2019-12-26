package mysql

import (
	"database/sql"
	"fmt"

	"github.com/ssrdive/cidium/pkg/models"
)

// ModelModel struct holds methods to query user table
type DropdownModel struct {
	DB *sql.DB
}

func (m *DropdownModel) Get(name string) ([]*models.Dropdown, error) {
	stmt := fmt.Sprintf(`SELECT id, name FROM %s`, name)

	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	items := []*models.Dropdown{}
	for rows.Next() {
		i := &models.Dropdown{}

		err = rows.Scan(&i.ID, &i.Name)
		if err != nil {
			return nil, err
		}

		items = append(items, i)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
