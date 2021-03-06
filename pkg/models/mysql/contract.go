package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ssrdive/cidium/pkg/loan"
	"github.com/ssrdive/cidium/pkg/models"
	"github.com/ssrdive/cidium/pkg/sql/queries"
	"github.com/ssrdive/mysequel"
	"github.com/ssrdive/sprinter"
)

// ContractModel struct holds database instance
type ContractModel struct {
	DB            *sql.DB
	ReceiptLogger *log.Logger
}

// Insert creates a new contract
func (m *ContractModel) Insert(initialState string, rparams, oparams []string, form url.Values) (int64, error) {
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

	cid, err := mysequel.Insert(mysequel.FormTable{
		TableName: "contract",
		RCols:     rparams,
		OCols:     oparams,
		Form:      form,
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	_, err = mysequel.Insert(mysequel.Table{
		TableName: "contract_financial",
		Columns:   []string{"contract_id"},
		Vals:      []interface{}{cid},
		Tx:        tx,
	})
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	var isid int
	err = tx.QueryRow(queries.STATE_ID_FROM_STATE, initialState).Scan(&isid)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	sid, err := mysequel.Insert(mysequel.Table{
		TableName: "contract_state",
		Columns:   []string{"contract_id", "state_id"},
		Vals:      []interface{}{cid, isid},
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	_, err = mysequel.Insert(mysequel.Table{
		TableName: "contract_state_transition",
		Columns:   []string{"to_contract_state_id", "transition_date"},
		Vals:      []interface{}{sid, time.Now().Format("2006-01-02 15:04:05")},
		Tx:        tx,
	})

	_, err = mysequel.Update(mysequel.UpdateTable{
		Table: mysequel.Table{
			TableName: "contract",
			Columns:   []string{"contract_state_id"},
			Vals:      []interface{}{sid},
			Tx:        tx,
		},
		WColumns: []string{"id"},
		WVals:    []string{strconv.FormatInt(cid, 10)},
	})
	if err != nil {
		return 0, err
	}

	return cid, nil
}

// Legacy creates a new legacy contract
func (m *ContractModel) Legacy(cid int, form url.Values) error {
	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		_ = tx.Commit()
	}()

	capital, err := strconv.ParseFloat(form.Get("capital"), 32)
	rate, err := strconv.ParseFloat(form.Get("rate"), 32)
	installments, err := strconv.Atoi(form.Get("installments"))
	installmentInterval, err := strconv.Atoi(form.Get("installment_interval"))
	method := form.Get("method")
	initiationDate := form.Get("initiation_date")
	if err != nil {
		return err
	}

	marketedSchedule, _, err := loan.Create(capital, rate, installments, installmentInterval, 0, initiationDate, method)
	if err != nil {
		return err
	}

	var citid int
	err = tx.QueryRow(queries.INSTALLMENT_INSTALLMENT_TYPE_ID).Scan(&citid)
	if err != nil {
		tx.Rollback()
		return err
	}

	capitalAmount := 0.0
	interestAmount := 0.0
	for _, inst := range marketedSchedule {
		capitalAmount += inst.Capital
		interestAmount += inst.Interest
		_, err = mysequel.Insert(mysequel.Table{
			TableName: "contract_installment",
			Columns:   []string{"contract_id", "contract_installment_type_id", "capital", "interest", "default_interest", "due_date"},
			Vals:      []interface{}{cid, citid, inst.Capital, inst.Interest, inst.DefaultInterest, inst.DueDate},
			Tx:        tx,
		})
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	fullRecievables := capitalAmount + interestAmount

	tid, err := mysequel.Insert(mysequel.Table{
		TableName: "transaction",
		Columns:   []string{"user_id", "datetime", "posting_date", "contract_id", "remark"},
		Vals:      []interface{}{form.Get("user_id"), time.Now().Format("2006-01-02 15:04:05"), time.Now().Format("2006-01-02"), cid, fmt.Sprintf("LEGACY CONTRACT CREATION %d", cid)},
		Tx:        tx,
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	journalEntries := []models.JournalEntry{
		{fmt.Sprintf("%d", 95), "", fmt.Sprintf("%f", capital)},
		{fmt.Sprintf("%d", 78), "", fmt.Sprintf("%f", interestAmount)},
		{fmt.Sprintf("%d", 25), fmt.Sprintf("%f", fullRecievables), ""},
	}

	for _, entry := range journalEntries {
		if len(entry.Debit) != 0 {
			_, err := mysequel.Insert(mysequel.Table{
				TableName: "account_transaction",
				Columns:   []string{"transaction_id", "account_id", "type", "amount"},
				Vals:      []interface{}{tid, entry.Account, "DR", entry.Debit},
				Tx:        tx,
			})
			if err != nil {
				tx.Rollback()
				return err
			}
		}
		if len(entry.Credit) != 0 {
			_, err := mysequel.Insert(mysequel.Table{
				TableName: "account_transaction",
				Columns:   []string{"transaction_id", "account_id", "type", "amount"},
				Vals:      []interface{}{tid, entry.Account, "CR", entry.Credit},
				Tx:        tx,
			})
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}

	return nil
}

// WorkDocuments returns documents to be completed at the current stage of the contract
func (m *ContractModel) WorkDocuments(cid int) ([]models.WorkDocument, error) {
	var res []models.WorkDocument
	err := mysequel.QueryToStructs(&res, m.DB, queries.WORK_DOCUMENTS, cid)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// WorkQuestions returns questions to be answered at the current stage of the contract
func (m *ContractModel) WorkQuestions(cid int) ([]models.WorkQuestion, error) {
	var res []models.WorkQuestion
	err := mysequel.QueryToStructs(&res, m.DB, queries.WORK_QUESTIONS, cid)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Questions returns all the answered questions of the contract
func (m *ContractModel) Questions(cid int) ([]models.Question, error) {
	var res []models.Question
	err := mysequel.QueryToStructs(&res, m.DB, queries.QUESTIONS, cid)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Documents returns all the documents of the contract
func (m *ContractModel) Documents(cid int) ([]models.Document, error) {
	var res []models.Document
	err := mysequel.QueryToStructs(&res, m.DB, queries.DOCUMENTS, cid)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// History returns the state history of the contract
func (m *ContractModel) History(cid int) ([]models.History, error) {
	var res []models.History
	err := mysequel.QueryToStructs(&res, m.DB, queries.HISTORY, cid)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// StateAnswer adds an answer to a question in the current contract state
func (m *ContractModel) StateAnswer(rparams, oparams []string, form url.Values) (int64, error) {
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

	cid, err := mysequel.Insert(mysequel.FormTable{
		TableName: "contract_state_question_answer",
		RCols:     rparams,
		OCols:     oparams,
		Form:      form,
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	return cid, nil
}

// StateDocument adds a document in the current contract state
func (m *ContractModel) StateDocument(rparams, oparams []string, form url.Values) (int64, error) {
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

	cid, err := mysequel.Insert(mysequel.FormTable{
		TableName: "contract_state_document",
		RCols:     rparams,
		OCols:     oparams,
		Form:      form,
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	return cid, nil
}

// DetailFinancial returns contract details
func (m *ContractModel) DetailFinancial(cid int) (models.ContractDetailFinancial, error) {
	tx, err := m.DB.Begin()
	if err != nil {
		return models.ContractDetailFinancial{}, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		_ = tx.Commit()
	}()

	var lkas17Compliant int
	err = tx.QueryRow(queries.LKAS_17_COMPLIANT, cid).Scan(&lkas17Compliant)
	if err != nil {
		tx.Rollback()
		return models.ContractDetailFinancial{}, err
	}

	if lkas17Compliant == 1 {
		var detailFinancial models.ContractDetailFinancial
		detailFinancial.LKAS17 = true
		err = tx.QueryRow(queries.CONTRACT_DETAILS_FINANCIAL, cid).Scan(&detailFinancial.Active, &detailFinancial.RecoveryStatus, &detailFinancial.Doubtful, &detailFinancial.Payment, &detailFinancial.ContractArrears, &detailFinancial.ChargesDebitsArrears, &detailFinancial.OverdueIndex, &detailFinancial.CapitalProvisioned)
		if err != nil {
			return models.ContractDetailFinancial{}, err
		}

		return detailFinancial, nil
	}
	return models.ContractDetailFinancial{LKAS17: false}, nil
}

// DetailFinancialRaw returns contract details raw
func (m *ContractModel) DetailFinancialRaw(cid int) (models.ContractDetailFinancialRaw, error) {
	tx, err := m.DB.Begin()
	if err != nil {
		return models.ContractDetailFinancialRaw{}, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		_ = tx.Commit()
	}()

	var lkas17Compliant int
	err = tx.QueryRow(queries.LKAS_17_COMPLIANT, cid).Scan(&lkas17Compliant)
	if err != nil {
		tx.Rollback()
		return models.ContractDetailFinancialRaw{}, err
	}

	if lkas17Compliant == 1 {
		var detailFinancial models.ContractDetailFinancialRaw
		err = tx.QueryRow(queries.ContractFinancialRaw, cid).Scan(&detailFinancial.ID, &detailFinancial.ContractID, &detailFinancial.Active, &detailFinancial.RecoveryStatusID, &detailFinancial.Doubtful, &detailFinancial.Payment, &detailFinancial.AgreedCapital, &detailFinancial.AgreedInterest, &detailFinancial.CapitalPaid, &detailFinancial.InterestPaid, &detailFinancial.ChargesDebitsPaid, &detailFinancial.CapitalArrears, &detailFinancial.InterestArrears, &detailFinancial.ChargesDebitsArrears, &detailFinancial.CapitalProvisioned, &detailFinancial.FinancialScheduleStartDate, &detailFinancial.FinancialScheduleEndDate, &detailFinancial.MarketedScheduleStartDate, &detailFinancial.MarketedScheduleEndDate, &detailFinancial.PaymentInterval, &detailFinancial.Payments)
		if err != nil {
			return models.ContractDetailFinancialRaw{}, err
		}

		return detailFinancial, nil
	}
	return models.ContractDetailFinancialRaw{}, nil
}

func (m *ContractModel) DetailLegacyFinancialRaw(cid int) ([]models.ContractLegacyFinancial, error) {
	tx, err := m.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		_ = tx.Commit()
	}()

	var lkas17Compliant int
	err = tx.QueryRow(queries.LKAS_17_COMPLIANT, cid).Scan(&lkas17Compliant)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	if lkas17Compliant == 0 {
		var detailFinancial []models.ContractLegacyFinancial
		err := mysequel.QueryToStructs(&detailFinancial, tx, queries.ContractLegacyFinancials, cid)
		if err != nil {
			return nil, err
		}
		if err != nil {
			return nil, err
		}

		return detailFinancial, nil
	}
	return nil, nil
}

// Detail returns contract details
func (m *ContractModel) Detail(cid int) (models.ContractDetail, error) {
	tx, err := m.DB.Begin()
	if err != nil {
		return models.ContractDetail{}, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		_ = tx.Commit()
	}()

	var lkas17Compliant int
	err = tx.QueryRow(queries.LKAS_17_COMPLIANT, cid).Scan(&lkas17Compliant)
	if err != nil {
		tx.Rollback()
		return models.ContractDetail{}, err
	}

	var detailsQuery string
	if lkas17Compliant == 1 {
		detailsQuery = queries.CONTRACT_DETAILS_LKAS_17
	} else {
		detailsQuery = queries.CONTRACT_DETAILS
	}

	var detail models.ContractDetail
	err = tx.QueryRow(detailsQuery, cid).Scan(&detail.ID, &detail.ContractState, &detail.ContractBatch, &detail.ModelName, &detail.ChassisNumber, &detail.CustomerName, &detail.CustomerNic, &detail.CustomerAddress, &detail.CustomerContact, &detail.LiaisonName, &detail.LiaisonContact, &detail.Price, &detail.Downpayment, &detail.CreditOfficer, &detail.RecoveryOfficer, &detail.AmountPending, &detail.TotalPayable, &detail.TotalPaid, &detail.LastPaymentDate, &detail.OverdueIndex)
	if err != nil {
		return models.ContractDetail{}, err
	}

	return detail, nil
}

// Installment returns installments
func (m *ContractModel) Installment(cid int) ([]models.ActiveInstallment, error) {
	tx, err := m.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		_ = tx.Commit()
	}()

	var lkas17Compliant int
	err = tx.QueryRow(queries.LKAS_17_COMPLIANT, cid).Scan(&lkas17Compliant)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	var res []models.ActiveInstallment
	if lkas17Compliant == 1 {
		err := mysequel.QueryToStructs(&res, tx, queries.CONTRACT_INSTALLMENTS_LKAS_17, cid)
		if err != nil {
			return nil, err
		}
		return res, err
	}
	err = mysequel.QueryToStructs(&res, tx, queries.CONTRACT_INSTALLMENTS, cid, cid, cid)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ReceiptsV2 returns v2 of receipts
func (m *ContractModel) ReceiptsV2(cid int) ([]models.ReceiptV2, error) {
	var res []models.ReceiptV2
	err := mysequel.QueryToStructs(&res, m.DB, queries.CONTRACT_RECEIPTS_V2, cid)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// FloatReceipts returns receipts in float
func (m *ContractModel) FloatReceipts(cid int) ([]models.FloatReceiptsClient, error) {
	var res []models.FloatReceiptsClient
	err := mysequel.QueryToStructs(&res, m.DB, queries.FLOAT_RECEIPTS_CLIENT, cid)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Receipts returns receipts
func (m *ContractModel) Receipts(cid int) ([]models.Receipt, error) {
	var res []models.Receipt
	err := mysequel.QueryToStructs(&res, m.DB, queries.CONTRACT_RECEIPTS, cid)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// OfficerReceipts returns receipts issued on a date
func (m *ContractModel) OfficerReceipts(oid int, date string) ([]models.Receipt, error) {
	var res []models.Receipt
	err := mysequel.QueryToStructs(&res, m.DB, queries.CONTRACT_OFFICER_RECEIPTS, oid, date, oid, date)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Commitments returns contract commitments
func (m *ContractModel) Commitments(cid int) ([]models.Commitment, error) {
	var res []models.Commitment
	err := mysequel.QueryToStructs(&res, m.DB, queries.CONTRACT_COMMITMENTS, cid)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// DashboardCommitmentsByOfficer returns commitments related to an officer
func (m *ContractModel) DashboardCommitmentsByOfficer(ctype, officer string) ([]models.DashboardCommitment, error) {
	var results *sql.Rows
	var err error
	if ctype == "expired" {
		results, err = m.DB.Query(queries.EXPIRED_COMMITMENTS_BY_OFFICER, officer)
	} else if ctype == "upcoming" {
		results, err = m.DB.Query(queries.UPCOMING_COMMITMENTS_BY_OFFICER, officer)
	} else {
		return nil, errors.New("Invalid commitment type")
	}
	if err != nil {
		return nil, err
	}

	var res []models.DashboardCommitment
	for results.Next() {
		var commitment models.DashboardCommitment
		err = results.Scan(&commitment.ContractID, &commitment.DueIn, &commitment.Text)
		if err != nil {
			return nil, err
		}
		res = append(res, commitment)
	}

	return res, nil
}

// DashboardCommitments returns web application dashboard commitments
func (m *ContractModel) DashboardCommitments(ctype string) ([]models.DashboardCommitment, error) {
	var results *sql.Rows
	var err error
	if ctype == "expired" {
		results, err = m.DB.Query(queries.EXPIRED_COMMITMENTS)
	} else if ctype == "upcoming" {
		results, err = m.DB.Query(queries.UPCOMING_COMMITMENTS)
	} else {
		return nil, errors.New("Invalid commitment type")
	}
	if err != nil {
		return nil, err
	}

	var res []models.DashboardCommitment
	for results.Next() {
		var commitment models.DashboardCommitment
		err = results.Scan(&commitment.ContractID, &commitment.DueIn, &commitment.Text)
		if err != nil {
			return nil, err
		}
		res = append(res, commitment)
	}

	return res, nil
}

// TransionableStates returns the list of states a contract can be transition into
func (m *ContractModel) TransionableStates(cid int) ([]models.Dropdown, error) {
	var res []models.Dropdown
	err := mysequel.QueryToStructs(&res, m.DB, queries.TRANSITIONABLE_STATES, cid)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// RejectedRequests returns rejected requests
func (m *ContractModel) RejectedRequests(cid int) ([]models.RejectedRequest, error) {
	var res []models.RejectedRequest
	err := mysequel.QueryToStructs(&res, m.DB, queries.REJECTED_REQUESTS, cid)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// CurrentRequestExists returns whether a current request exists or not
func (m *ContractModel) CurrentRequestExists(cid int) (bool, error) {
	result, err := m.DB.Query(queries.CURRENT_REQUEST_EXISTS, cid)
	if err != nil {
		return false, err
	}

	count := 0
	for result.Next() {
		count++
	}

	if count == 0 {
		return false, nil
	}
	return true, nil
}

// Request issues a request
func (m *ContractModel) Request(rparams, oparams []string, form url.Values) (int64, error) {
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

	tcsid, err := mysequel.Insert(mysequel.FormTable{
		TableName: "contract_state",
		RCols:     rparams,
		OCols:     oparams,
		Form:      form,
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	var cs models.ID
	err = tx.QueryRow(`
		SELECT C.contract_state_id 
		FROM contract C 
		WHERE C.id = ?`, form.Get("contract_id")).Scan(&cs.ID)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	rid, err := mysequel.Insert(mysequel.Table{
		TableName: "request",
		Columns:   []string{"contract_state_id", "to_contract_state_id", "user_id", "datetime", "remarks"},
		Vals:      []interface{}{cs.ID, tcsid, form.Get("user_id"), time.Now().Format("2006-01-02 15:04:05"), form.Get("remarks")},
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	return rid, nil
}

// Requests returns a list of requests made
func (m *ContractModel) Requests(user int) ([]models.Request, error) {
	var res []models.Request
	err := mysequel.QueryToStructs(&res, m.DB, queries.REQUESTS)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// SeasonalIncentive returns the seasonal incentive for the given user
func (m *ContractModel) SeasonalIncentive(user int) (models.SeasonalIncentive, error) {
	var r models.SeasonalIncentive
	err := m.DB.QueryRow(queries.SEASONAL_INCENTIVE, user, user).Scan(&r.Amount)
	if err != nil {
		return models.SeasonalIncentive{}, nil
	}
	return r, nil
}

// RequestName returns the name of the request from the given id
func (m *ContractModel) RequestName(request int) (string, error) {
	var r models.Dropdown
	err := m.DB.QueryRow(queries.REQUEST_NAME, request).Scan(&r.ID, &r.Name)
	if err != nil {
		return "", nil
	}
	return r.Name, nil
}

// CreditWorthinessApproved sends SMS message to customer, liaison upon credit worthiness approval
func (m *ContractModel) CreditWorthinessApproved(user, request int, aAPIKey string) error {
	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		_ = tx.Commit()
	}()

	var cid int
	var customerName string
	var liaisonContact sql.NullInt32
	err = tx.QueryRow(queries.PARAMS_FOR_CREDIT_WORTHINESS_APPROVAL, request).Scan(&cid, &customerName, &liaisonContact)
	if err != nil {
		return err
	}

	if !liaisonContact.Valid {
		return errors.New("Liaison contact not provided")
	}

	message := fmt.Sprintf("Customer %s bearing contract number %d has obtained credit worthiness approval.", customerName, cid)
	var telephone string
	if liaisonContact.Int32 > 100000000 && liaisonContact.Int32 < 999999999 {
		telephone = fmt.Sprintf("%d,768237192,703524330,703524420,775607777,703524300,703524333,703524408", liaisonContact.Int32)
	} else {
		telephone = "768237192,703524330,703524420,775607777,703524300,703524333,703524408"
	}
	requestURL := fmt.Sprintf("https://richcommunication.dialog.lk/api/sms/inline/send.php?destination=%s&q=%s&message=%s", telephone, aAPIKey, url.QueryEscape(message))
	resp, err := http.Get(requestURL)
	if err != nil {

	}
	defer resp.Body.Close()
	if err != nil {
		return nil
	}
	return nil
}

// InitiateContract initiates the financials in of a contract in the system
// This includes creating installments with capital and interest,
// adding journal entries for financial accounts
func (m *ContractModel) InitiateContract(user, request int) error {
	tx, err := m.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		_ = tx.Commit()
	}()

	results, err := tx.Query(queries.PARAMS_FOR_CONTRACT_INITIATION, request)
	if err != nil {
		tx.Rollback()
		return err
	}

	var params []models.Dropdown
	for results.Next() {
		var p models.Dropdown
		err = results.Scan(&p.ID, &p.Name)
		if err != nil {
			return err
		}
		params = append(params, p)
	}

	details := make(map[string]string)
	for _, param := range params {
		details[param.ID] = param.Name
	}

	capital, err := strconv.ParseFloat(details["Capital"], 32)
	rate, err := strconv.ParseFloat(details["Interest Rate"], 32)
	installments, err := strconv.Atoi(details["Installments"])
	installmentInterval, err := strconv.Atoi(details["Installment Interval"])
	method := details["Interest Method"]
	initiationDate, err := time.Parse("2006-01-02", details["Initiation Date"])
	structuredMonthlyRental, err := strconv.Atoi(details["Structured Monthly Rental"])
	if err != nil {
		return err
	}

	marketedSchedule, financialSchedule, err := loan.Create(capital, rate, installments, installmentInterval, structuredMonthlyRental, initiationDate.Format("2006-01-02"), method)
	if err != nil {
		return err
	}

	var cid int
	err = tx.QueryRow(queries.CONTRACT_ID_FROM_REUQEST, request).Scan(&cid)
	if err != nil {
		tx.Rollback()
		return err
	}

	var citid int
	err = tx.QueryRow(queries.INSTALLMENT_INSTALLMENT_TYPE_ID).Scan(&citid)
	if err != nil {
		tx.Rollback()
		return err
	}

	capitalAmount := 0.0
	interestAmount := 0.0
	for _, inst := range financialSchedule {
		capitalAmount += inst.Capital
		interestAmount += inst.Interest
		_, err = mysequel.Insert(mysequel.Table{
			TableName: "contract_schedule",
			Columns:   []string{"contract_id", "contract_installment_type_id", "capital", "interest", "installment", "monthly_date", "marketed_installment", "marketed_capital", "marketed_interest", "marketed_due_date"},
			Vals:      []interface{}{cid, citid, inst.Capital, inst.Interest, inst.Capital + inst.Interest, inst.MonthlyDate, inst.MarketedInstallment, inst.MarketedCapital, inst.MarketedInterest, inst.MarketedDueDate},
			Tx:        tx,
		})
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	fullRecievables := capitalAmount + interestAmount

	_, err = mysequel.Update(mysequel.UpdateTable{
		Table: mysequel.Table{
			TableName: "contract_financial",
			Columns:   []string{"payment", "agreed_capital", "agreed_interest", "financial_schedule_start_date", "financial_schedule_end_date", "marketed_schedule_start_date", "marketed_schedule_end_date", "payment_interval", "payments"},
			Vals:      []interface{}{financialSchedule[0].Capital + financialSchedule[0].Interest, capitalAmount, interestAmount, financialSchedule[0].MonthlyDate, financialSchedule[len(financialSchedule)-1].MonthlyDate, marketedSchedule[0].DueDate, marketedSchedule[len(marketedSchedule)-1].DueDate, installmentInterval, installments},
			Tx:        tx,
		},
		WColumns: []string{"contract_id"},
		WVals:    []string{strconv.FormatInt(int64(cid), 10)},
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	tid, err := mysequel.Insert(mysequel.Table{
		TableName: "transaction",
		Columns:   []string{"user_id", "datetime", "posting_date", "contract_id", "remark"},
		Vals:      []interface{}{user, time.Now().Format("2006-01-02 15:04:05"), time.Now().Format("2006-01-02"), cid, fmt.Sprintf("CONTRACT INITIATION %d", cid)},
		Tx:        tx,
	})
	if err != nil {
		tx.Rollback()
		return err
	}

	receivableAccount := 185
	unearnedInterestAccount := 188
	payableAccount := 189

	journalEntries := []models.JournalEntry{
		{fmt.Sprintf("%d", receivableAccount), fmt.Sprintf("%f", fullRecievables), ""},
		{fmt.Sprintf("%d", payableAccount), "", fmt.Sprintf("%f", capital)},
		{fmt.Sprintf("%d", unearnedInterestAccount), "", fmt.Sprintf("%f", interestAmount)},
	}

	err = IssueJournalEntries(tx, tid, journalEntries)
	if err != nil {
		tx.Rollback()
		return err
	}

	var floatReceipts []models.FloatReceipts
	mysequel.QueryToStructs(&floatReceipts, tx, queries.FLOAT_RECEIPTS, cid)

	// Issue receipts in float
	if len(floatReceipts) > 0 {
		for _, r := range floatReceipts {
			err = sprinter.Run(r.Date, fmt.Sprintf("%d", cid), true, tx)
			if err != nil {
				tx.Rollback()
				return err
			}
			_, err := m.IssueLKAS17Receipt(tx, r.UserID, cid, r.Amount, "", "", "FLOAT", r.Datetime)
			if err != nil {
				tx.Rollback()
				return err
			}

			_, err = mysequel.Update(mysequel.UpdateTable{
				Table: mysequel.Table{TableName: "contract_receipt_float",
					Columns: []string{"cleared"},
					Vals:    []interface{}{1},
					Tx:      tx},
				WColumns: []string{"id"},
				WVals:    []string{strconv.FormatInt(int64(r.ID), 10)},
			})
			if err != nil {
				tx.Rollback()
				return err
			}
		}
	}
	err = sprinter.Run(time.Now().Format("2006-01-02"), fmt.Sprintf("%d", cid), true, tx)
	if err != nil {
		tx.Rollback()
		return err
	}

	return nil

}

// CommitmentAction sets whether a commitment was fulfilled or expired
func (m *ContractModel) CommitmentAction(comid, fulfilled, user int) (int64, error) {
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

	c, err := mysequel.Update(mysequel.UpdateTable{
		Table: mysequel.Table{TableName: "contract_commitment",
			Columns: []string{"fulfilled", "fulfilled_by", "fulfilled_on"},
			Vals:    []interface{}{fulfilled, user, time.Now().Format("2006-01-02 15:04:05")},
			Tx:      tx},
		WColumns: []string{"id"},
		WVals:    []string{strconv.FormatInt(int64(comid), 10)},
	})
	if err != nil {
		return 0, err
	}

	return c, nil
}

// RequestAction approves or rejects a request
func (m *ContractModel) RequestAction(user, request int, action, note string) (int64, error) {
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

	t := time.Now().Format("2006-01-02 15:04:05")
	_, err = mysequel.Update(mysequel.UpdateTable{
		Table: mysequel.Table{TableName: "request",
			Columns: []string{"approved", "approved_by", "approved_on", "note"},
			Vals:    []interface{}{action, user, t, note},
			Tx:      tx},
		WColumns: []string{"id"},
		WVals:    []string{strconv.FormatInt(int64(request), 10)},
	})
	if err != nil {
		return 0, err
	}

	if action == "0" {
		return 1, nil
	}

	var r models.RequestRaw
	err = tx.QueryRow(queries.REQUEST_RAW, request).Scan(&r.ID, &r.ContractStateID, &r.ToContractStateID, &r.ContractID)
	if err != nil {
		return 0, err
	}

	_, err = mysequel.Insert(mysequel.Table{
		TableName: "contract_state_transition",
		Columns:   []string{"from_contract_state_id", "to_contract_state_id", "request_id", "transition_date"},
		Vals:      []interface{}{r.ContractStateID, r.ToContractStateID, request, t},
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	c, err := mysequel.Update(mysequel.UpdateTable{
		Table: mysequel.Table{TableName: "contract",
			Columns: []string{"contract_state_id"},
			Vals:    []interface{}{r.ToContractStateID},
			Tx:      tx},
		WColumns: []string{"id"},
		WVals:    []string{strconv.FormatInt(int64(r.ContractID), 10)},
	})
	if err != nil {
		return 0, err
	}

	return c, nil
}

// DeleteStateInfo marks a question or document deleted
func (m *ContractModel) DeleteStateInfo(form url.Values) (int64, error) {
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

	_, err = mysequel.Update(mysequel.UpdateTable{
		Table: mysequel.Table{
			TableName: form.Get("table"),
			Columns:   []string{"deleted"},
			Vals:      []interface{}{1},
			Tx:        tx,
		},
		WColumns: []string{"id"},
		WVals:    []string{form.Get("id")},
	})
	if err != nil {
		return 0, err
	}
	return 0, nil
}

// Commitment adds a commitment
func (m *ContractModel) Commitment(rparams, oparams []string, form url.Values, specialMessage, aAPIKey string) (int64, error) {
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

	comid, err := mysequel.Insert(mysequel.FormTable{
		TableName: "contract_commitment",
		RCols:     rparams,
		OCols:     oparams,
		Form:      form,
		Tx:        tx,
	})
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if specialMessage == "1" {
		var officerName string
		err = tx.QueryRow(queries.OFFICER_NAME, form.Get("user_id")).Scan(&officerName)

		var senderMobile string
		err = tx.QueryRow(queries.SENDER_MOBILE, form.Get("contract_id")).Scan(&senderMobile)

		message := fmt.Sprintf("%s left a special comment on your contract %s", officerName, form.Get("contract_id"))

		requestURL := fmt.Sprintf("https://cpsolutions.dialog.lk/index.php/cbs/sms/send?destination=%s&q=%s&message=%s", senderMobile, aAPIKey, url.QueryEscape(message))

		resp, err := http.Get(requestURL)

		if err == nil {
			defer resp.Body.Close()
		}
	}

	return comid, nil
}

// DebitNote issues a debit note
func (m *ContractModel) DebitNote(rparams, oparams []string, form url.Values) (int64, error) {
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

	var lkas17Compliant int
	err = tx.QueryRow(queries.LKAS_17_COMPLIANT, form.Get("contract_id")).Scan(&lkas17Compliant)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if lkas17Compliant == 1 {
		dnid, err := mysequel.Insert(mysequel.Table{
			TableName: "contract_schedule",
			Columns:   []string{"contract_id", "contract_installment_type_id", "capital", "installment", "monthly_date", "daily_entry_issued", "marketed_installment", "marketed_capital", "marketed_due_date"},
			Vals:      []interface{}{form.Get("contract_id"), form.Get("contract_installment_type_id"), form.Get("capital"), form.Get("capital"), time.Now().Format("2006-01-02"), 1, 1, form.Get("capital"), time.Now().Format("2006-01-02")},
			Tx:        tx,
		})
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		form.Set("contract_schedule_id", fmt.Sprintf("%d", dnid))
		_, err = mysequel.Insert(mysequel.FormTable{
			TableName: "contract_schedule_charges_debits_details",
			RCols:     []string{"contract_schedule_id", "user_id", "notes"},
			OCols:     []string{},
			Form:      form,
			Tx:        tx,
		})
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		_, err = tx.Exec("UPDATE contract_financial SET charges_debits_arrears = charges_debits_arrears + ? WHERE contract_id = ?", form.Get("capital"), form.Get("contract_id"))
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		tid, err := mysequel.Insert(mysequel.Table{
			TableName: "transaction",
			Columns:   []string{"user_id", "datetime", "posting_date", "contract_id", "remark"},
			Vals:      []interface{}{form.Get("user_id"), time.Now().Format("2006-01-02 15:04:05"), time.Now().Format("2006-01-02"), form.Get("contract_id"), fmt.Sprintf("DEBIT NOTE %d [%s]", dnid, form.Get("contract_id"))},
			Tx:        tx,
		})
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		var expenseAccount, receivableAccount int
		err = tx.QueryRow(queries.GET_DEBIT_TYPE_EXPENSE_RECEIVABLE_ACCOUNT, form.Get("contract_installment_type_id")).Scan(&expenseAccount, &receivableAccount)
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		journalEntries := []models.JournalEntry{
			{Account: fmt.Sprintf("%d", expenseAccount), Debit: form.Get("capital"), Credit: ""},
			{Account: fmt.Sprintf("%d", receivableAccount), Debit: "", Credit: form.Get("capital")},
		}

		err = IssueJournalEntries(tx, tid, journalEntries)
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		return dnid, nil
	}

	dnid, err := mysequel.Insert(mysequel.FormTable{
		TableName: "contract_installment",
		RCols:     rparams,
		OCols:     oparams,
		Form:      form,
		Tx:        tx,
	})
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	form.Set("contract_installment_id", fmt.Sprintf("%d", dnid))
	_, err = mysequel.Insert(mysequel.FormTable{
		TableName: "contract_installment_details",
		RCols:     []string{"contract_installment_id", "user_id", "notes"},
		OCols:     []string{},
		Form:      form,
		Tx:        tx,
	})
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	var unearnedAccountID int
	err = tx.QueryRow(queries.DEBIT_NOTE_UNEARNED_ACC_NO, form.Get("contract_installment_type_id")).Scan(&unearnedAccountID)

	tid, err := mysequel.Insert(mysequel.Table{
		TableName: "transaction",
		Columns:   []string{"user_id", "datetime", "posting_date", "contract_id", "remark"},
		Vals:      []interface{}{form.Get("user_id"), time.Now().Format("2006-01-02 15:04:05"), time.Now().Format("2006-01-02"), form.Get("contract_id"), fmt.Sprintf("DEBIT NOTE %d", dnid)},
		Tx:        tx,
	})
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	journalEntries := []models.JournalEntry{
		{Account: fmt.Sprintf("%d", 25), Debit: form.Get("capital"), Credit: ""},
		{Account: fmt.Sprintf("%d", unearnedAccountID), Debit: "", Credit: form.Get("capital")},
	}

	err = IssueJournalEntries(tx, tid, journalEntries)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	return dnid, nil
}

// LegacyRebate issues a legacy rebate
func (m *ContractModel) LegacyRebate(userID, cid int, amount float64) (int64, error) {
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

	balance := amount

	var intPayments []models.ContractPayment

	results, err := tx.Query(queries.LEGACY_PAYMENTS, cid)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	var payables []models.ContractPayable
	for results.Next() {
		var u models.ContractPayable
		err = results.Scan(&u.InstallmentID, &u.ContractID, &u.CapitalPayable, &u.InterestPayable, &u.DefaultInterest)
		if err != nil {
			return 0, err
		}
		payables = append(payables, u)
	}

	rid, err := mysequel.Insert(mysequel.Table{
		TableName: "contract_receipt",
		Columns:   []string{"contract_receipt_type_id", "user_id", "contract_id", "datetime", "amount"},
		Vals:      []interface{}{3, userID, cid, time.Now().Format("2006-01-02 15:04:05"), amount},
		Tx:        tx,
	})
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	for i := len(payables); i > 0; i-- {
		fmt.Println(payables[i-1].InterestPayable)
		if payables[i-1].InterestPayable != 0 && balance != 0 {
			if balance-payables[i-1].InterestPayable >= 0 {
				intPayments = append(intPayments, models.ContractPayment{payables[i-1].InstallmentID, rid, payables[i-1].InterestPayable})
				balance = math.Round((balance-payables[i-1].InterestPayable)*100) / 100
			} else {
				intPayments = append(intPayments, models.ContractPayment{payables[i-1].InstallmentID, rid, balance})
				balance = 0
			}
		}
	}

	if balance != 0 {
		tx.Rollback()
		return 0, errors.New("Rebate exceeds payable interest")
	}

	for _, intPayment := range intPayments {
		_, err := mysequel.Insert(mysequel.Table{
			TableName: "contract_interest_payment",
			Columns:   []string{"contract_installment_id", "contract_receipt_id", "amount"},
			Vals:      []interface{}{intPayment.ContractInstallmentID, intPayment.ContractReceiptID, intPayment.Amount},
			Tx:        tx,
		})
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	tid, err := mysequel.Insert(mysequel.Table{
		TableName: "transaction",
		Columns:   []string{"user_id", "datetime", "posting_date", "contract_id", "remark"},
		Vals:      []interface{}{userID, time.Now().Format("2006-01-02 15:04:05"), time.Now().Format("2006-01-02"), cid, fmt.Sprintf("INTEREST REBATE %d", rid)},
		Tx:        tx,
	})
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	journalEntries := []models.JournalEntry{
		{fmt.Sprintf("%d", 78), fmt.Sprintf("%f", amount), ""},
		{fmt.Sprintf("%d", 25), "", fmt.Sprintf("%f", amount)},
	}

	for _, entry := range journalEntries {
		if val, _ := strconv.ParseFloat(entry.Debit, 64); len(entry.Debit) != 0 && val != 0 {
			_, err := mysequel.Insert(mysequel.Table{
				TableName: "account_transaction",
				Columns:   []string{"transaction_id", "account_id", "type", "amount"},
				Vals:      []interface{}{tid, entry.Account, "DR", entry.Debit},
				Tx:        tx,
			})
			if err != nil {
				tx.Rollback()
				return 0, err
			}
		}
		if val, _ := strconv.ParseFloat(entry.Credit, 64); len(entry.Credit) != 0 && val != 0 {
			_, err := mysequel.Insert(mysequel.Table{
				TableName: "account_transaction",
				Columns:   []string{"transaction_id", "account_id", "type", "amount"},
				Vals:      []interface{}{tid, entry.Account, "CR", entry.Credit},
				Tx:        tx,
			})
			if err != nil {
				tx.Rollback()
				return 0, err
			}
		}
	}

	return rid, nil

}

// ReceiptV2 issues a receipt by taking the receipted date in to account
// Hence it will be used to issue receipts that was stored before a contract was initiated.
func (m *ContractModel) ReceiptV2(fid, userID, cid int, amount float64, notes, dueDate, date string, datetime time.Time, tx *sql.Tx) (int64, error) {

	var debits []models.DebitPayable
	err := mysequel.QueryToStructs(&debits, tx, queries.DEBITS, cid)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	var diUpdates []models.ContractDefaultInterestUpdate
	var diLogs []models.ContractDefaultInterestChangeHistory
	var diPayments []models.ContractPayment
	var intPayments []models.ContractPayment
	var capPayments []models.ContractPayment
	var debitPayments []models.DebitPayment

	balance := amount

	rid, err := mysequel.Insert(mysequel.Table{
		TableName: "contract_receipt",
		Columns:   []string{"user_id", "contract_id", "datetime", "amount", "notes", "due_date"},
		Vals:      []interface{}{userID, cid, datetime.Format("2006-01-02 15:04:05"), amount, notes, dueDate},
		Tx:        tx,
	})
	if err != nil {
		fmt.Println(err)
		tx.Rollback()
		return 0, err
	}

	if balance != 0 {
		for _, p := range debits {
			if p.CapitalPayable != 0 && balance != 0 {
				if balance-p.CapitalPayable >= 0 {
					debitPayments = append(debitPayments, models.DebitPayment{p.InstallmentID, rid, p.CapitalPayable, p.UnearnedAccountID, p.IncomeAccountID})
					balance = math.Round((balance-p.CapitalPayable)*100) / 100
				} else {
					debitPayments = append(debitPayments, models.DebitPayment{p.InstallmentID, rid, balance, p.UnearnedAccountID, p.IncomeAccountID})
					balance = 0
				}
			}
		}
	}

	var payables []models.ContractPayable
	err = mysequel.QueryToStructs(&payables, tx, queries.OVERDUE_INSTALLMENTS, cid, date)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	diAmount := 0.0

	if balance != 0 {
		for _, p := range payables {
			if p.DefaultInterest != 0 && balance != 0 {
				if balance-p.DefaultInterest >= 0 {
					diAmount += p.DefaultInterest
					diUpdates = append(diUpdates, models.ContractDefaultInterestUpdate{p.InstallmentID, float64(0)})
					diLogs = append(diLogs, models.ContractDefaultInterestChangeHistory{p.InstallmentID, rid, p.DefaultInterest})
					diPayments = append(diPayments, models.ContractPayment{p.InstallmentID, rid, p.DefaultInterest})
					balance = math.Round((balance-p.DefaultInterest)*100) / 100
				} else {
					diAmount += math.Round((p.DefaultInterest-balance)*100) / 100
					diUpdates = append(diUpdates, models.ContractDefaultInterestUpdate{p.InstallmentID, math.Round((p.DefaultInterest-balance)*100) / 100})
					diLogs = append(diLogs, models.ContractDefaultInterestChangeHistory{p.InstallmentID, rid, p.DefaultInterest})
					diPayments = append(diPayments, models.ContractPayment{p.InstallmentID, rid, balance})
					balance = 0
				}
			}
		}
	}

	if balance != 0 {
		for _, p := range payables {
			if p.InterestPayable != 0 && balance != 0 {
				if balance-p.InterestPayable >= 0 {
					intPayments = append(intPayments, models.ContractPayment{p.InstallmentID, rid, p.InterestPayable})
					balance = math.Round((balance-p.InterestPayable)*100) / 100
				} else {
					intPayments = append(intPayments, models.ContractPayment{p.InstallmentID, rid, balance})
					balance = 0
				}
			}
		}
	}

	if balance != 0 {
		for _, p := range payables {
			if p.CapitalPayable != 0 && balance != 0 {
				if balance-p.CapitalPayable >= 0 {
					capPayments = append(capPayments, models.ContractPayment{p.InstallmentID, rid, p.CapitalPayable})
					balance = math.Round((balance-p.CapitalPayable)*100) / 100
				} else {
					capPayments = append(capPayments, models.ContractPayment{p.InstallmentID, rid, balance})
					balance = 0
				}
			}
		}
	}

	if balance != 0 {
		var upcoming []models.ContractPayable
		err = mysequel.QueryToStructs(&upcoming, tx, queries.UPCOMING_INSTALLMENTS, cid, date)
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		for _, p := range upcoming {
			if p.InterestPayable != 0 && balance != 0 {
				if balance-p.InterestPayable >= 0 {
					intPayments = append(intPayments, models.ContractPayment{p.InstallmentID, rid, p.InterestPayable})
					balance = math.Round((balance-p.InterestPayable)*100) / 100
				} else {
					intPayments = append(intPayments, models.ContractPayment{p.InstallmentID, rid, balance})
					balance = 0
				}
			}
			if p.CapitalPayable != 0 && balance != 0 {
				if balance-p.CapitalPayable >= 0 {
					capPayments = append(capPayments, models.ContractPayment{p.InstallmentID, rid, p.CapitalPayable})
					balance = math.Round((balance-p.CapitalPayable)*100) / 100
				} else {
					capPayments = append(capPayments, models.ContractPayment{p.InstallmentID, rid, balance})
					balance = 0
				}
			}
		}
	}

	if balance != 0 {
		tx.Rollback()
		return 0, errors.New("Error: Payment exceeds payables")
	}

	for _, diUpdate := range diUpdates {
		_, err = mysequel.Update(mysequel.UpdateTable{
			Table: mysequel.Table{TableName: "contract_installment",
				Columns: []string{"default_interest"},
				Vals:    []interface{}{diUpdate.DefaultInterest},
				Tx:      tx},
			WColumns: []string{"id"},
			WVals:    []string{fmt.Sprintf("%d", diUpdate.ContractInstallmentID)},
		})
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	for _, diLog := range diLogs {
		_, err := mysequel.Insert(mysequel.Table{
			TableName: "contract_default_interest_change_history",
			Columns:   []string{"contract_installment_id", "contract_receipt_id", "default_interest"},
			Vals:      []interface{}{diLog.ContractInstallmentID, diLog.ContractReceiptID, diLog.DefaultInterest},
			Tx:        tx,
		})
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	for _, intPayment := range diPayments {
		_, err := mysequel.Insert(mysequel.Table{
			TableName: "contract_default_interest_payment",
			Columns:   []string{"contract_installment_id", "contract_receipt_id", "amount"},
			Vals:      []interface{}{intPayment.ContractInstallmentID, intPayment.ContractReceiptID, intPayment.Amount},
			Tx:        tx,
		})
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	interestAmount := 0.0

	for _, intPayment := range intPayments {
		interestAmount += intPayment.Amount
		_, err := mysequel.Insert(mysequel.Table{
			TableName: "contract_interest_payment",
			Columns:   []string{"contract_installment_id", "contract_receipt_id", "amount"},
			Vals:      []interface{}{intPayment.ContractInstallmentID, intPayment.ContractReceiptID, intPayment.Amount},
			Tx:        tx,
		})
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	for _, capPayment := range capPayments {
		_, err := mysequel.Insert(mysequel.Table{
			TableName: "contract_capital_payment",
			Columns:   []string{"contract_installment_id", "contract_receipt_id", "amount"},
			Vals:      []interface{}{capPayment.ContractInstallmentID, capPayment.ContractReceiptID, capPayment.Amount},
			Tx:        tx,
		})
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	tid, err := mysequel.Insert(mysequel.Table{
		TableName: "transaction",
		Columns:   []string{"user_id", "datetime", "posting_date", "contract_id", "remark"},
		Vals:      []interface{}{userID, time.Now().Format("2006-01-02 15:04:05"), time.Now().Format("2006-01-02"), cid, fmt.Sprintf("RECEIPT %d (FLOAT RECEIPT REMOVAL %d)", rid, fid)},
		Tx:        tx,
	})
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	for _, capPayment := range debitPayments {
		_, err := mysequel.Insert(mysequel.Table{
			TableName: "account_transaction",
			Columns:   []string{"transaction_id", "account_id", "type", "amount"},
			Vals:      []interface{}{tid, capPayment.UnearnedAccountID, "DR", capPayment.Amount},
			Tx:        tx,
		})
		if err != nil {
			tx.Rollback()
			return 0, err
		}
		_, err = mysequel.Insert(mysequel.Table{
			TableName: "account_transaction",
			Columns:   []string{"transaction_id", "account_id", "type", "amount"},
			Vals:      []interface{}{tid, capPayment.IncomeAccountID, "CR", capPayment.Amount},
			Tx:        tx,
		})
		if err != nil {
			tx.Rollback()
			return 0, err
		}
		_, err = mysequel.Insert(mysequel.Table{
			TableName: "contract_capital_payment",
			Columns:   []string{"contract_installment_id", "contract_receipt_id", "amount"},
			Vals:      []interface{}{capPayment.ContractInstallmentID, capPayment.ContractReceiptID, capPayment.Amount},
			Tx:        tx,
		})
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	journalEntries := []models.JournalEntry{
		{fmt.Sprintf("%d", 144), fmt.Sprintf("%f", amount), ""},
		{fmt.Sprintf("%d", 25), "", fmt.Sprintf("%f", amount)},
		{fmt.Sprintf("%d", 46), "", fmt.Sprintf("%f", interestAmount)},
		{fmt.Sprintf("%d", 78), fmt.Sprintf("%f", interestAmount), ""},
		{fmt.Sprintf("%d", 48), "", fmt.Sprintf("%f", diAmount)},
		{fmt.Sprintf("%d", 79), fmt.Sprintf("%f", diAmount), ""},
	}

	for _, entry := range journalEntries {
		if val, _ := strconv.ParseFloat(entry.Debit, 64); len(entry.Debit) != 0 && val != 0 {
			_, err := mysequel.Insert(mysequel.Table{
				TableName: "account_transaction",
				Columns:   []string{"transaction_id", "account_id", "type", "amount"},
				Vals:      []interface{}{tid, entry.Account, "DR", entry.Debit},
				Tx:        tx,
			})
			if err != nil {
				tx.Rollback()
				return 0, err
			}
		}
		if val, _ := strconv.ParseFloat(entry.Credit, 64); len(entry.Credit) != 0 && val != 0 {
			_, err := mysequel.Insert(mysequel.Table{
				TableName: "account_transaction",
				Columns:   []string{"transaction_id", "account_id", "type", "amount"},
				Vals:      []interface{}{tid, entry.Account, "CR", entry.Credit},
				Tx:        tx,
			})
			if err != nil {
				tx.Rollback()
				return 0, err
			}
		}
	}

	return rid, nil
}

// PerformanceReview returns contract performance review
func (m *ContractModel) PerformanceReview(startDate, endDate, state, officer, batch, npl string) ([]models.PerformanceReview, error) {
	s := mysequel.NewNullString(state)
	o := mysequel.NewNullString(officer)
	b := mysequel.NewNullString(batch)
	n := mysequel.NewNullString(npl)

	var res []models.PerformanceReview
	err := mysequel.QueryToStructs(&res, m.DB, queries.PERFORMANCE_REVIEW(startDate, endDate), s, s, o, o, b, b, n, n, s, s, o, o, b, b, n, n)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// SearchV2 returns V2 search results
// Multiple search methods are implemented to support
// different web and mobile versions
func (m *ContractModel) SearchV2(search, state, officer, batch, npl, lkas17, startOd, endOd, removeDeleted string) ([]models.SearchResultV2, error) {
	var k sql.NullString
	if search == "" {
		k = sql.NullString{}
	} else {
		k = sql.NullString{
			Valid:  true,
			String: "%" + search + "%",
		}
	}
	s := mysequel.NewNullString(state)
	o := mysequel.NewNullString(officer)
	b := mysequel.NewNullString(batch)
	n := mysequel.NewNullString(npl)
	l := mysequel.NewNullString(lkas17)
	var sod, eod sql.NullFloat64
	if startOd == "" {
		sod = sql.NullFloat64{}
	} else {
		v, _ := strconv.ParseFloat(startOd, 64)
		sod = sql.NullFloat64{
			Valid:   true,
			Float64: v,
		}
	}
	if endOd == "" {
		eod = sql.NullFloat64{}
	} else {
		v, _ := strconv.ParseFloat(endOd, 64)
		eod = sql.NullFloat64{
			Valid:   true,
			Float64: v,
		}
	}

	rd, err := strconv.Atoi(removeDeleted)
	if err != nil {
		rd = 0
	}

	var res []models.SearchResultV2
	err = mysequel.QueryToStructs(&res, m.DB, queries.SEARCH_V2, k, k, s, s, o, o, b, b, n, n, l, l, sod, eod, sod, eod, rd, k, k, s, s, o, o, b, b, n, n, l, l, sod, eod, sod, eod, rd)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// SearchOld returns old search results
// Multiple search methods are implemented to support
// different web and mobile versions
func (m *ContractModel) SearchOld(search, state, officer, batch string) ([]models.SearchResultOld, error) {
	var k sql.NullString
	if search == "" {
		k = sql.NullString{}
	} else {
		k = sql.NullString{
			Valid:  true,
			String: "%" + search + "%",
		}
	}
	s := mysequel.NewNullString(state)
	o := mysequel.NewNullString(officer)
	b := mysequel.NewNullString(batch)

	var res []models.SearchResultOld
	err := mysequel.QueryToStructs(&res, m.DB, queries.SEARCH_OLD, k, k, s, s, o, o, b, b)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Search returns search results for mobile application
// Multiple search methods are implemented to support
// different web and mobile versions
func (m *ContractModel) Search(search, state, officer, batch string) ([]models.SearchResult, error) {
	var k sql.NullString
	if search == "" {
		k = sql.NullString{}
	} else {
		k = sql.NullString{
			Valid:  true,
			String: "%" + search + "%",
		}
	}
	s := mysequel.NewNullString(state)
	o := mysequel.NewNullString(officer)
	b := mysequel.NewNullString(batch)

	var res []models.SearchResult
	err := mysequel.QueryToStructs(&res, m.DB, queries.SEARCH, k, k, s, s, o, o, b, b, k, k, s, s, o, o, b, b)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// CSQASearch returns CSQA search results
func (m *ContractModel) CSQASearch(search, question, empty string) ([]models.CSQASearchResult, error) {
	var k sql.NullString
	if search == "" {
		k = sql.NullString{}
	} else {
		k = sql.NullString{
			Valid:  true,
			String: "%" + search + "%",
		}
	}

	var res []models.CSQASearchResult
	err := mysequel.QueryToStructs(&res, m.DB, queries.CSQA_SEARCH, question, empty, k, k)
	if err != nil {
		return nil, err
	}

	return res, nil
}
