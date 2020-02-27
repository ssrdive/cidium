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
	Type     string
}

type Dropdown struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type DropdownAccount struct {
	ID        string `json:"id"`
	AccountID int    `json:"account_id"`
	Name      string `json:"name"`
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
	CustomerName    string         `json:"customer_name"`
	CustomerNic     string         `json:"customer_nic"`
	CustomerAddress string         `json:"customer_address"`
	CustomerContact int            `json:"customer_contact"`
	LiaisonName     sql.NullString `json:"liaison_name"`
	LiaisonContact  sql.NullInt32  `json:"liaison_contact"`
	Price           int            `json:"price"`
	Downpayment     sql.NullInt32  `json:"downpayment"`
	RecoveryOfficer string         `json:"recovery_officer"`
	AmountPending   float64        `json:"amount_pending"`
	TotalPayable    float64        `json:"total_payable"`
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

type ContractPayable struct {
	InstallmentID   int
	ContractID      int
	CapitalPayable  float64
	InterestPayable float64
	DefaultInterest float64
}

type DebitsPayable struct {
	InstallmentID     int
	ContractID        int
	CapitalPayable    float64
	InterestPayable   float64
	DefaultInterest   float64
	UnearnedAccountID int
	IncomeAccountID   int
}

type ContractDefaultInterestChangeHistory struct {
	ContractInstallmentID int
	ContractReceiptID     int64
	DefaultInterest       float64
}

type ContractDefaultInterestUpdate struct {
	ContractInstallmentID int
	DefaultInterest       float64
}

type ContractPayment struct {
	ContractInstallmentID int
	ContractReceiptID     int64
	Amount                float64
}

type DebitPayment struct {
	ContractInstallmentID int
	ContractReceiptID     int64
	Amount                float64
	UnearnedAccountID     int
	IncomeAccountID       int
}

type SearchResult struct {
	ID              int     `json:"id"`
	Agrivest        int     `json:"agrivest"`
	RecoveryOfficer string  `json:"recovery_officer"`
	State           string  `json:"state"`
	Model           string  `json:"model"`
	ChassisNumber   string  `json:"chassis_number"`
	CustomerName    string  `json:"customer_name"`
	CustomerAddress string  `json:"customer_address"`
	CustomerContact string  `json:"customer_contact"`
	AmountPending   float64 `json:"amount_pending"`
	TotalPayable    float64 `json:"total_payable"`
	TotalAgreement  float64 `json:"total_agreement"`
	TotalPaid       float64 `json:"total_paid"`
	TotalDIPaid     float64 `json:"total_di_paid"`
}

type ActiveInstallment struct {
	ID              int       `json:"id"`
	InstallmentType string    `json:"installment_type"`
	Installment     float64   `json:"installment"`
	InstallmentPaid float64   `json:"installment_paid"`
	DueDate         time.Time `json:"due_date"`
	DueIn           int       `json:"due_in"`
}

type Receipt struct {
	ID     int            `json:"id"`
	Date   time.Time      `json:"date"`
	Amount float64        `json:"amount"`
	Notes  sql.NullString `json:"notes"`
}

type Commitment struct {
	ID          int            `json:"id"`
	CreatedBy   string         `json:"created_by"`
	Created     time.Time      `json:"created"`
	Commitment  int            `json:"commitment"`
	Fulfilled   sql.NullInt32  `json:"fulfilled"`
	DueIn       sql.NullInt32  `json:"due_in"`
	Text        string         `json:"text"`
	FulfilledBy sql.NullString `json:"fulfilled_by"`
	FulfilledOn sql.NullTime   `json:"fulfilled_on"`
}

type PaymentVoucherList struct {
	ID          int       `json:"id"`
	Datetime    time.Time `json:"date_time"`
	PostingDate string    `json:"posting_date"`
	FromAccount string    `json:"from_account"`
	User        string    `json:"user"`
}

type PaymentVoucherDetails struct {
	AccountID   int     `json:"account_id"`
	AccountName string  `json:"account_name"`
	Amount      float64 `json:"amount"`
}

type PaymentVoucherSummary struct {
	DueDate               sql.NullString          `json:"due_date"`
	CheckNumber           sql.NullString          `json:"check_number"`
	PaymentVoucherDetails []PaymentVoucherDetails `json:"payment_voucher_details"`
}

type DashboardCommitment struct {
	ContractID int    `json:"contract_id"`
	DueIn      int    `json:"due_in"`
	Text       string `json:"text"`
}

type Question struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type Document struct {
	Document string `json:"document"`
	S3Region string `json:"s3region"`
	S3Bucket string `json:"s3bucket"`
	Source   string `json:"source"`
}

type History struct {
	FromState      sql.NullString `json:"from_state"`
	ToState        string         `json:"to_state"`
	TransitionDate time.Time      `json:"transition_date"`
}

type ChartOfAccount struct {
	MainAccountID     int            `json:"main_account_id"`
	MainAccount       string         `json:"main_account"`
	SubAccountID      int            `json:"sub_account_id"`
	SubAccount        string         `json:"sub_account"`
	AccountCategoryID sql.NullInt32  `json:"account_category_id"`
	AccountCategory   sql.NullString `json:"account_category"`
	AccountID         sql.NullInt32  `json:"account_id"`
	AccountName       sql.NullString `json:"account_name"`
}

type TrialEntry struct {
	ID          int     `json:"id"`
	AccountID   int     `json:"account_id"`
	AccountName string  `json:"account_name"`
	Debit       float64 `json:"debit"`
	Credit      float64 `json:"credit"`
	Balance     float64 `json:"balance"`
}

type JournalEntry struct {
	Account string
	Debit   string
	Credit  string
}

type PaymentVoucherEntry struct {
	Account string
	Amount  string
}

type LedgerEntry struct {
	Name          string  `json:"account_name"`
	TransactionID int     `json:"transaction_id"`
	PostingDate   string  `json:"posting_date"`
	Amount        float64 `json:"amount"`
	Type          string  `json:"type"`
	Remark        string  `json:"remark"`
}

type Transaction struct {
	TransactionID int     `json:"transaction_id"`
	AccountID     int     `json:"account_id"`
	AccountID2    int     `json:"account_id2"`
	AccountName   string  `json:"account_name"`
	Type          string  `json:"type"`
	Amount        float64 `json:"amount"`
}
