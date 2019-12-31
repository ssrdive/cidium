package models

import (
	"database/sql"
	"errors"
	"time"
)

var ErrNoRecord = errors.New("models: no matching record found")

type UserResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Token    string `json:"token"`
}

type User struct {
	ID        int
	GroupID   int
	Username  string
	Password  string
	Name      string
	CreatedAt time.Time
}

type JWTUser struct {
	ID       int
	Username string
	Password string
	Name     string
}

type Dropdown struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type WorkDocument struct {
	ContractStateID int            `json:"contract_state_id"`
	DocumentID      int            `json:"document_id"`
	DocumentName    string         `json:"document_name"`
	Source          sql.NullString `json:"source"`
	S3Bucket        sql.NullString `json:"s3bucket"`
	S3Region        sql.NullString `json:"s3region"`
}

type WorkQuestion struct {
	ContractStateID int            `json:"contract_state_id"`
	QuestionID      int            `json:"question_id"`
	Question        string         `json:"question"`
	Answer          sql.NullString `json:"answer"`
}
