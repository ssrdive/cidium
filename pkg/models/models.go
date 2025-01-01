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
	ContractStateID int            `json:"contract_state_id"`
	DocumentID      int            `json:"document_id"`
	DocumentName    string         `json:"document_name"`
	ID              sql.NullInt32  `json:"id"`
	Source          sql.NullString `json:"source"`
	S3Bucket        sql.NullString `json:"s3bucket"`
	S3Region        sql.NullString `json:"s3region"`
	Compulsory      int            `json:"compulsory"`
}

type WorkQuestion struct {
	ContractStateID int            `json:"contract_state_id"`
	QuestionID      int            `json:"question_id"`
	Question        string         `json:"question"`
	ID              sql.NullInt32  `json:"id"`
	Answer          sql.NullString `json:"answer"`
	Compulsory      int            `json:"compulsory"`
}

type ContractDetail struct {
	ID                 int            `json:"id"`
	HoldDefault        int            `json:"hold_default"`
	ContractState      string         `json:"contract_state"`
	ContractBatch      string         `json:"contract_batch"`
	ModelName          string         `json:"model_name"`
	ChassisNumber      string         `json:"chassis_number"`
	CustomerName       string         `json:"customer_name"`
	CustomerNic        string         `json:"customer_nic"`
	CustomerAddress    string         `json:"customer_address"`
	CustomerContact    int            `json:"customer_contact"`
	LiaisonName        sql.NullString `json:"liaison_name"`
	LiaisonContact     sql.NullInt32  `json:"liaison_contact"`
	Price              int            `json:"price"`
	Downpayment        sql.NullInt32  `json:"downpayment"`
	IntroducingOfficer string         `json:"introducing_officer"`
	CreditOfficer      string         `json:"credit_officer"`
	RecoveryOfficer    string         `json:"recovery_officer"`
	AmountPending      float64        `json:"amount_pending"`
	TotalPayable       float64        `json:"total_payable"`
	DefaultCharges     float64        `json:"default_charges"`
	TotalPaid          float64        `json:"total_paid"`
	LastPaymentDate    string         `json:"last_payment_date"`
	OverdueIndex       string         `json:"overdue_index"`
}

type ContractRequestable struct {
	Requestable           bool              `json:"transitionalble"`
	NonRequestableMessage string            `json:"non_requestable_message"`
	States                []Dropdown        `json:"states"`
	RejectedRequests      []RejectedRequest `json:"rejected_requests"`
}

type ContractDetailFinancial struct {
	LKAS17               bool    `json:"lkas_17"`
	Active               int     `json:"active"`
	RecoveryStatus       string  `json:"recovery_status"`
	Doubtful             int     `json:"doubtful"`
	Payment              float64 `json:"payment"`
	ContractArrears      float64 `json:"contract_arrears"`
	ChargesDebitsArrears float64 `json:"charges_debits_arrears"`
	OverdueIndex         float64 `json:"overdue_index"`
	CapitalProvisioned   float64 `json:"capital_provisioned"`
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

type DebitPayable struct {
	InstallmentID     int
	ContractID        int
	CapitalPayable    float64
	InterestPayable   float64
	DefaultInterest   float64
	UnearnedAccountID int
	IncomeAccountID   int
}

type DebitPayableLKAS17 struct {
	InstallmentID       int
	ContractID          int
	CapitalPayable      float64
	InterestPayable     float64
	DefaultInterest     float64
	ExpenseAccountID    int
	ReceivableAccountID int
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

type DebitPaymentLKAS17 struct {
	ContractInstallmentID int
	ContractReceiptID     int64
	Amount                float64
	ExpenseAccountID      int
	ReceivableAccountID   int
}

type SearchResultOld struct {
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

type SearchResult struct {
	ID              int     `json:"id"`
	Agrivest        int     `json:"agrivest"`
	RecoveryOfficer string  `json:"recovery_officer"`
	State           string  `json:"state"`
	Model           string  `json:"model"`
	Batch           string  `json:"batch"`
	ChassisNumber   string  `json:"chassis_number"`
	CustomerName    string  `json:"customer_name"`
	CustomerAddress string  `json:"customer_address"`
	CustomerContact string  `json:"customer_contact"`
	AmountPending   float64 `json:"amount_pending"`
	TotalPayable    float64 `json:"total_payable"`
	DefaultCharges  float64 `json:"default_charges"`
	TotalAgreement  float64 `json:"total_agreement"`
	TotalPaid       float64 `json:"total_paid"`
	TotalDIPaid     float64 `json:"total_di_paid"`
	LastPaymentDate string  `json:"last_payment_date"`
}

type SearchResultV2 struct {
	ID              int            `json:"id"`
	Agrivest        int            `json:"agrivest"`
	RecoveryOfficer string         `json:"recovery_officer"`
	State           string         `json:"state"`
	InStateFor      sql.NullString `json:"in_state_for"`
	Model           string         `json:"model"`
	Batch           string         `json:"batch"`
	ChassisNumber   string         `json:"chassis_number"`
	CustomerName    string         `json:"customer_name"`
	CustomerAddress string         `json:"customer_address"`
	CustomerContact string         `json:"customer_contact"`
	AmountPending   float64        `json:"amount_pending"`
	TotalPayable    float64        `json:"total_payable"`
	DefaultCharges  float64        `json:"default_charges"`
	TotalAgreement  float64        `json:"total_agreement"`
	TotalPaid       float64        `json:"total_paid"`
	TotalDIPaid     float64        `json:"total_di_paid"`
	LastPaymentDate string         `json:"last_payment_date"`
	OverdueIndex    string         `json:"overdue_index"`
}

type PerformanceReview struct {
	ID                        int     `json:"id"`
	Agrivest                  int     `json:"agrivest"`
	RecoveryOfficer           string  `json:"recovery_officer"`
	State                     string  `json:"state"`
	Model                     string  `json:"model"`
	Batch                     string  `json:"batch"`
	ChassisNumber             string  `json:"chassis_number"`
	CustomerName              string  `json:"customer_name"`
	CustomerAddress           string  `json:"customer_address"`
	CustomerContact           string  `json:"customer_contact"`
	AmountPending             float64 `json:"amount_pending"`
	StartAmountPending        float64 `json:"start_amount_pending"`
	EndAmountPending          float64 `json:"end_amount_pending"`
	StartBetweenAmountPending float64 `json:"start_between_amount_pending"`
	EndBetweenAmountPending   float64 `json:"end_between_amount_pending"`
	TotalPayable              float64 `json:"total_payable"`
	TotalAgreement            float64 `json:"total_agreement"`
	TotalPaid                 float64 `json:"total_paid"`
	TotalDIPaid               float64 `json:"total_di_paid"`
	LastPaymentDate           string  `json:"last_payment_date"`
	StartOverdueIndex         string  `json:"start_overdue_index"`
	EndOverdueIndex           string  `json:"end_overdue_index"`
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

type AndroidReceipt struct {
	ID         int            `json:"id"`
	ContractID int            `json:"contract_id"`
	Date       time.Time      `json:"date"`
	Amount     float64        `json:"amount"`
	Notes      sql.NullString `json:"notes"`
}

type DocGen struct {
	StateID       int    `json:"state_id"`
	Name          string `json:"name"`
	GenerationURL string `json:"generation_url"`
}

type ReceiptV2 struct {
	ID     int            `json:"id"`
	Date   time.Time      `json:"date"`
	Amount float64        `json:"amount"`
	Notes  sql.NullString `json:"notes"`
	Type   string         `json:"type"`
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

type TimelineRow struct {
	Grouping       int16
	ContractID     int16
	Type           string
	Amount         float64
	Date           time.Time
	Change         float64
	Days           int
	DaysCumulative int
}

type ContractBalanceChangeRow struct {
	ContractID int16
	Type       string
	Amount     float64
	Date       string
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
	Approver       sql.NullString `json:"approver"`
	TransitionDate time.Time      `json:"transition_date"`
}

type CSQASearchResult struct {
	ID              int            `json:"id"`
	RecoveryOfficer string         `json:"recovery_officer"`
	State           string         `json:"state"`
	Answer          sql.NullString `json:"answer"`
	CreatedAgo      sql.NullInt32  `json:"created_ago"`
	StateAtAnswer   sql.NullString `json:"state_at_answer"`
}

type FloatReceipts struct {
	ID       int       `json:"id"`
	UserID   int       `json:"user_id"`
	Amount   float64   `json:"amount"`
	Date     string    `json:"date"`
	Datetime time.Time `json:"datetime"`
}

type FloatReceiptsClient struct {
	ID       int     `json:"id"`
	Datetime string  `json:"datetime"`
	Amount   float64 `json:"amount"`
}

type SeasonalIncentive struct {
	Amount float64 `json:"amount"`
}

type AchievementSummaryItem struct {
	UserID               int     `json:"user_id"`
	Officer              string  `json:"officer"`
	Month                string  `json:"month"`
	Target               float64 `json:"target"`
	Collection           float64 `json:"collection"`
	CollectionPercentage float64 `json:"collection_percentage"`
}

type ArrearsAnalysisItem struct {
	Officer                              string  `json:"officer"`
	StartDateArrears                     float64 `json:"start_date_arrears"`
	StartDateArrearsAtEndDate            float64 `json:"start_date_arrears_at_end_date"`
	ArrearsCollectionAmountFromStartDate float64 `json:"arrears_collection_amount_from_start_date"`
	EndDateArrears                       float64 `json:"end_date_arrears"`
	StartDateDueForPeriod                float64 `json:"start_date_due_for_period"`
	EndDateDueForPeriod                  float64 `json:"end_date_due_for_period"`
	CurrentArrears                       float64 `json:"current_arrears"`
}

type ReceiptSearchItem struct {
	ID         int            `json:"id"`
	ContractID int            `json:"contract_id"`
	Officer    string         `json:"officer"`
	Issuer     string         `json:"issuer"`
	Datetime   string         `json:"datetime"`
	Amount     float64        `json:"amount"`
	Notes      sql.NullString `json:"notes"`
}

type ContractDetailFinancialRaw struct {
	ID                         int     `json:"id"`
	ContractID                 int     `json:"contract_id"`
	Active                     int     `json:"active"`
	RecoveryStatusID           int     `json:"recovery_status_id"`
	Doubtful                   int     `json:"doubtful"`
	Payment                    float64 `json:"payment"`
	AgreedCapital              float64 `json:"agreed_capital"`
	AgreedInterest             float64 `json:"agreed_interest"`
	CapitalPaid                float64 `json:"capital_paid"`
	InterestPaid               float64 `json:"interest_paid"`
	ChargesDebitsPaid          float64 `json:"charges_debits_paid"`
	CapitalArrears             float64 `json:"capital_arrears"`
	InterestArrears            float64 `json:"interest_arrears"`
	ChargesDebitsArrears       float64 `json:"charges_debits_arrears"`
	CapitalProvisioned         float64 `json:"capital_provisioned"`
	FinancialScheduleStartDate string  `json:"financial_schedule_start_date"`
	FinancialScheduleEndDate   string  `json:"financial_schedule_end_date"`
	MarketedScheduleStartDate  string  `json:"marketed_schedule_start_date"`
	MarketedScheduleEndDate    string  `json:"marketed_schedule_end_date"`
	PaymentInterval            float64 `json:"payment_interval"`
	Payments                   float64 `json:"payments"`
}

type ContractLegacyFinancial struct {
	InstallmentID   int     `json:"installment_id"`
	InstallmentType string  `json:"installment_type"`
	Capital         float64 `json:"capital"`
	Interest        float64 `json:"interest"`
	CapitalPaid     float64 `json:"capital_paid"`
	InterestPaid    float64 `json:"interest_paid"`
	CapitalPayable  float64 `json:"capital_payable"`
	InterestPayable float64 `json:"interest_payable"`
	DueDate         string  `json:"due_date"`
	DueIn           string  `json:"due_in"`
}
