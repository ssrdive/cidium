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
	ID              sql.NullInt32  `json:"id"`
	ContractStateID int            `json:"contract_state_id"`
	DocumentID      int            `json:"document_id"`
	DocumentName    string         `json:"document_name"`
	Source          sql.NullString `json:"source"`
	S3Bucket        sql.NullString `json:"s3bucket"`
	S3Region        sql.NullString `json:"s3region"`
	Compulsory      int            `json:"compulsory"`
}

type WorkQuestion struct {
	ID              sql.NullInt32  `json:"id"`
	ContractStateID int            `json:"contract_state_id"`
	QuestionID      int            `json:"question_id"`
	Question        string         `json:"question"`
	Answer          sql.NullString `json:"answer"`
	Compulsory      int            `json:"compulsory"`
}

type ContractDetail struct {
	ID              int            `json:"id"`
	ContractState   string         `json:"contract_state"`
	ContractBatch   string         `json:"contract_batch"`
	ModelName       string         `json:"model_name"`
	ChassisNumber   string         `json:"chassis_number"`
	CustomerNic     string         `json:"customer_nic"`
	CustomerName    string         `json:"customer_name"`
	CustomerAddress string         `json:"customer_address"`
	CustomerContact int            `json:"customer_contact"`
	LiaisonName     sql.NullString `json:"liaison_name"`
	LiaisonContact  sql.NullInt32  `json:"liaison_contact"`
	Price           int            `json:"price"`
	Downpayment     sql.NullInt32  `json:"downpayment"`
}

type ContractRequestable struct {
	Requestable           bool              `json:"transitionalble"`
	NonRequestableMessage string            `json:"non_requestable_message"`
	States                []Dropdown        `json:"states"`
	RejectedRequests      []RejectedRequest `json:"rejected_requests"`
}

type ID struct {
	ID int `json:"id"`
}

type Request struct {
	RequestID       int            `json:"request_id"`
	ContractID      int            `json:"contract_id"`
	Remarks         sql.NullString `json:"remarks"`
	CustomerName    string         `json:"customer_name"`
	ContractState   string         `json:"contract_state"`
	ToContractState string         `json:"to_contract_state"`
	RequestedBy     string         `json:"requested_by"`
	RequestedOn     time.Time      `json:"requested_on"`
}

type RequestRaw struct {
	ID                int
	ContractStateID   int
	ToContractStateID int
	ContractID        int
}

type RejectedRequest struct {
	ID   int            `json:"id"`
	User string         `json:"user"`
	Note sql.NullString `json:"note"`
}
