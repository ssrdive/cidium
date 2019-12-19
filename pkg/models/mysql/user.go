package mysql

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/ssrdive/cidium/pkg/models"
	"golang.org/x/crypto/bcrypt"
)

// UserModel struct holds methods to query user table
type UserModel struct {
	DB *sql.DB
}

// Insert method insert a user
func (m *UserModel) Insert(groupID int, firstName, middleName, lastName, commonName, password string) (int, error) {
	stmt := `INSERT INTO user (group_id, username, password, name, created_at) VALUES (?, ?, ?, ?, NOW())`

	username := fmt.Sprintf("%s.%s%s", commonName, string([]rune(firstName)[0]), string([]rune(lastName)[0]))
	name := fmt.Sprintf("%s %s %s", firstName, middleName, lastName)

	ps, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return 0, err
	}

	result, err := m.DB.Exec(stmt, groupID, username, ps, name)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

// Get method retrieves a user for given username and password
func (m *UserModel) Get(username, password string) (*models.JWTUser, error) {
	u := &models.JWTUser{}

	err := m.DB.QueryRow("SELECT id, username, password, name FROM user WHERE username = ?", username).Scan(&u.ID, &u.Username, &u.Password, &u.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrNoRecord
		} else {
			return nil, err
		}
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		return nil, err
	}

	return u, nil
}
