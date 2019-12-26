package mysql

import (
	"database/sql"
	"fmt"

	"net/url"
	sq "github.com/Masterminds/squirrel"
	// "github.com/ssrdive/cidium/pkg/models"
)

// ModelModel struct holds methods to query user table
type ContractModel struct {
	DB *sql.DB
}

func (m *ContractModel) Insert(params []string, form url.Values) (int, error) {
	p := make([]interface{}, len(params))
	for i := 0; i < len(params); i++ {
		p = append(p, "?")
	}

	stmt, _, _ := sq.
    Insert("p").Columns(params...).
    Values(p...).
	ToSql()

	fmt.Println(form)
	v := make([]interface{}, len(params))
	for _, param := range params {
		fmt.Printf("%s\t-\t%T\n", param, form.Get(param))
		v = append(v, form.Get(param))
	}

	fmt.Println(v)

	fmt.Println(stmt)

	return 1, nil
	// stmt := `INSERT INTO user (group_id, username, password, name, created_at) VALUES (?, ?, ?, ?, NOW())`

	// username := fmt.Sprintf("%s.%s%s", commonName, string([]rune(firstName)[0]), string([]rune(lastName)[0]))
	// name := fmt.Sprintf("%s %s %s", firstName, middleName, lastName)

	// ps, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	// if err != nil {
	// 	return 0, err
	// }

	// result, err := m.DB.Exec(stmt, groupID, username, ps, name)
	// if err != nil {
	// 	return 0, err
	// }

	// id, err := result.LastInsertId()
	// if err != nil {
	// 	return 0, err
	// }

	// return int(id), nil
}