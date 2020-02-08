package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/ssrdive/cidium/pkg/models"
	msql "github.com/ssrdive/cidium/pkg/sql"
	"github.com/ssrdive/cidium/pkg/sql/queries"
)

// ContractModel struct holds database instance
type ContractModel struct {
	DB *sql.DB
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

	cid, err := msql.Insert(msql.FormTable{
		TableName: "contract",
		RCols:     rparams,
		OCols:     oparams,
		Form:      form,
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	var isid int
	err = tx.QueryRow(queries.STATE_ID_FROM_STATE, initialState).Scan(&isid)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	sid, err := msql.Insert(msql.Table{
		TableName: "contract_state",
		Columns:   []string{"contract_id", "state_id"},
		Vals:      []interface{}{cid, isid},
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	_, err = msql.Insert(msql.Table{
		TableName: "contract_state_transition",
		Columns:   []string{"to_contract_state_id", "transition_date"},
		Vals:      []interface{}{sid, time.Now().Format("2006-01-02 15:04:05")},
		Tx:        tx,
	})

	_, err = msql.Update(msql.UpdateTable{
		Table: msql.Table{
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

	schedule, err := models.Create(capital, rate, installments, installmentInterval, initiationDate, method)
	if err != nil {
		return err
	}

	var citid int
	err = tx.QueryRow(queries.INSTALLMENT_INSTALLMENT_TYPE_ID).Scan(&citid)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, inst := range schedule {
		_, err = msql.Insert(msql.Table{
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

	return nil
}

func (m *ContractModel) WorkDocuments(cid int) ([]models.WorkDocument, error) {
	results, err := m.DB.Query(queries.WORK_DOCUMENTS, cid)
	if err != nil {
		return nil, err
	}

	var workDocuments []models.WorkDocument
	for results.Next() {
		var wd models.WorkDocument
		err = results.Scan(&wd.ContractStateID, &wd.DocumentID, &wd.DocumentName, &wd.ID, &wd.Source, &wd.S3Bucket, &wd.S3Region, &wd.Compulsory)
		if err != nil {
			return nil, err
		}
		workDocuments = append(workDocuments, wd)
	}

	return workDocuments, nil
}

func (m *ContractModel) WorkQuestions(cid int) ([]models.WorkQuestion, error) {
	results, err := m.DB.Query(queries.WORK_QUESTIONS, cid)
	if err != nil {
		return nil, err
	}

	var workQuestions []models.WorkQuestion
	for results.Next() {
		var wq models.WorkQuestion
		err = results.Scan(&wq.ContractStateID, &wq.QuestionID, &wq.Question, &wq.ID, &wq.Answer, &wq.Compulsory)
		if err != nil {
			return nil, err
		}
		workQuestions = append(workQuestions, wq)
	}

	return workQuestions, nil
}

func (m *ContractModel) Questions(cid int) ([]models.Question, error) {
	results, err := m.DB.Query(queries.QUESTIONS, cid)
	if err != nil {
		return nil, err
	}

	var questions []models.Question
	for results.Next() {
		var q models.Question
		err = results.Scan(&q.Question, &q.Answer)
		if err != nil {
			return nil, err
		}
		questions = append(questions, q)
	}

	return questions, nil
}

func (m *ContractModel) Documents(cid int) ([]models.Document, error) {
	results, err := m.DB.Query(queries.DOCUMENTS, cid)
	if err != nil {
		return nil, err
	}

	var documents []models.Document
	for results.Next() {
		var d models.Document
		err = results.Scan(&d.Document, &d.S3Region, &d.S3Bucket, &d.Source)
		if err != nil {
			return nil, err
		}
		documents = append(documents, d)
	}

	return documents, nil
}

func (m *ContractModel) History(cid int) ([]models.History, error) {
	results, err := m.DB.Query(queries.HISTORY, cid)
	if err != nil {
		return nil, err
	}

	var history []models.History
	for results.Next() {
		var h models.History
		err = results.Scan(&h.FromState, &h.ToState, &h.TransitionDate)
		if err != nil {
			return nil, err
		}
		history = append(history, h)
	}

	return history, nil
}

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

	cid, err := msql.Insert(msql.FormTable{
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

	cid, err := msql.Insert(msql.FormTable{
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

func (m *ContractModel) ContractDetail(cid int) (models.ContractDetail, error) {
	var detail models.ContractDetail
	err := m.DB.QueryRow(queries.CONTRACT_DETAILS, cid).Scan(&detail.ID, &detail.ContractState, &detail.ContractBatch, &detail.ModelName, &detail.ChassisNumber, &detail.CustomerName, &detail.CustomerNic, &detail.CustomerAddress, &detail.CustomerContact, &detail.LiaisonName, &detail.LiaisonContact, &detail.Price, &detail.Downpayment, &detail.RecoveryOfficer, &detail.AmountPending, &detail.TotalPayable)
	if err != nil {
		return models.ContractDetail{}, err
	}

	return detail, nil
}

func (m *ContractModel) ContractInstallments(cid int) ([]models.ActiveInstallment, error) {
	results, err := m.DB.Query(queries.CONTRACT_INSTALLMENTS, cid, cid, cid)
	if err != nil {
		return nil, err
	}
	var installments []models.ActiveInstallment
	for results.Next() {
		var installment models.ActiveInstallment
		err = results.Scan(&installment.ID, &installment.Installment, &installment.InstallmentPaid, &installment.DueDate, &installment.DueIn)
		if err != nil {
			return nil, err
		}
		installments = append(installments, installment)
	}

	return installments, nil
}

func (m *ContractModel) ContractReceipts(cid int) ([]models.Receipt, error) {
	results, err := m.DB.Query(queries.CONTRACT_RECEIPTS, cid)
	if err != nil {
		return nil, err
	}
	var receipts []models.Receipt
	for results.Next() {
		var receipt models.Receipt
		err = results.Scan(&receipt.ID, &receipt.Date, &receipt.Amount, &receipt.Notes)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, receipt)
	}

	return receipts, nil
}

func (m *ContractModel) ContractOfficerReceipts(oid int, date string) ([]models.Receipt, error) {
	results, err := m.DB.Query(queries.CONTRACT_OFFICER_RECEIPTS, oid, date)
	if err != nil {
		return nil, err
	}
	var receipts []models.Receipt
	for results.Next() {
		var receipt models.Receipt
		err = results.Scan(&receipt.ID, &receipt.Date, &receipt.Amount, &receipt.Notes)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, receipt)
	}

	return receipts, nil
}

func (m *ContractModel) Commitments(cid int) ([]models.Commitment, error) {
	results, err := m.DB.Query(queries.CONTRACT_COMMITMENTS, cid)
	if err != nil {
		return nil, err
	}
	var commitments []models.Commitment
	for results.Next() {
		var commitment models.Commitment
		err = results.Scan(&commitment.ID, &commitment.CreatedBy, &commitment.Created, &commitment.Commitment, &commitment.Fulfilled, &commitment.DueIn, &commitment.Text, &commitment.FulfilledBy, &commitment.FulfilledOn)
		if err != nil {
			return nil, err
		}
		commitments = append(commitments, commitment)
	}

	return commitments, nil
}

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

	var commitments []models.DashboardCommitment
	for results.Next() {
		var commitment models.DashboardCommitment
		err = results.Scan(&commitment.ContractID, &commitment.DueIn, &commitment.Text)
		if err != nil {
			return nil, err
		}
		commitments = append(commitments, commitment)
	}

	return commitments, nil
}

func (m *ContractModel) ContractTransionableStates(cid int) ([]models.Dropdown, error) {
	results, err := m.DB.Query(queries.TRANSITIONABLE_STATES, cid)
	if err != nil {
		return nil, err
	}

	var states []models.Dropdown
	for results.Next() {
		var s models.Dropdown
		err = results.Scan(&s.ID, &s.Name)
		if err != nil {
			return nil, err
		}
		states = append(states, s)
	}

	return states, nil
}

func (m *ContractModel) RejectedRequests(cid int) ([]models.RejectedRequest, error) {
	results, err := m.DB.Query(queries.REJECTED_REQUESTS, cid)
	if err != nil {
		return nil, err
	}

	var requests []models.RejectedRequest
	for results.Next() {
		var r models.RejectedRequest
		err = results.Scan(&r.ID, &r.User, &r.Note)
		if err != nil {
			return nil, err
		}
		requests = append(requests, r)
	}

	return requests, nil
}

func (m *ContractModel) CurrentRequetExists(cid int) (bool, error) {
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

	tcsid, err := msql.Insert(msql.FormTable{
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

	rid, err := msql.Insert(msql.Table{
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

func (m *ContractModel) Requests(user int) ([]models.Request, error) {
	results, err := m.DB.Query(queries.REQUESTS)
	if err != nil {
		return nil, err
	}

	var requests []models.Request
	for results.Next() {
		var r models.Request
		err = results.Scan(&r.RequestID, &r.ContractID, &r.Remarks, &r.CustomerName, &r.ContractState, &r.ToContractState, &r.RequestedBy, &r.RequestedOn)
		if err != nil {
			return nil, err
		}
		requests = append(requests, r)
	}

	return requests, nil
}

func (m *ContractModel) RequestName(request int) (string, error) {
	var r models.Dropdown
	err := m.DB.QueryRow(queries.REQUEST_NAME, request).Scan(&r.ID, &r.Name)
	if err != nil {
		return "", nil
	}
	return r.Name, nil
}

func (m *ContractModel) InitiateContract(request int) error {
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
	if err != nil {
		return err
	}

	schedule, err := models.Create(capital, rate, installments, installmentInterval, time.Now().Format("2006-01-02"), method)
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

	for _, inst := range schedule {
		_, err = msql.Insert(msql.Table{
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

	return nil

}

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

	c, err := msql.Update(msql.UpdateTable{
		Table: msql.Table{TableName: "contract_commitment",
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

func (m *ContractModel) RequestAction(user, request int, action, note string) (int64, error) {
	fmt.Println(msql.NewNullString(action))
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
	_, err = msql.Update(msql.UpdateTable{
		Table: msql.Table{TableName: "request",
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

	_, err = msql.Insert(msql.Table{
		TableName: "contract_state_transition",
		Columns:   []string{"from_contract_state_id", "to_contract_state_id", "request_id", "transition_date"},
		Vals:      []interface{}{r.ContractStateID, r.ToContractStateID, request, t},
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	c, err := msql.Update(msql.UpdateTable{
		Table: msql.Table{TableName: "contract",
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

	_, err = msql.Update(msql.UpdateTable{
		Table: msql.Table{
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

func (m *ContractModel) Commitment(rparams, oparams []string, form url.Values) (int64, error) {
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

	comid, err := msql.Insert(msql.FormTable{
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

	return comid, nil
}

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

	dnid, err := msql.Insert(msql.FormTable{
		TableName: "contract_installment",
		RCols:     rparams,
		OCols:     oparams,
		Form:      form,
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	form.Set("contract_installment_id", fmt.Sprintf("%d", dnid))
	_, err = msql.Insert(msql.FormTable{
		TableName: "contract_installment_details",
		RCols:     []string{"contract_installment_id", "user_id", "notes"},
		OCols:     []string{},
		Form:      form,
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	return dnid, nil
}

func (m *ContractModel) Receipt(user_id, cid int, amount float64, notes, due_date, rAPIKey, aAPIKey string) (int64, error) {
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

	results, err := tx.Query(queries.DEBITS, cid)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	var debits []models.ContractPayable
	for results.Next() {
		var d models.ContractPayable
		err = results.Scan(&d.InstallmentID, &d.ContractID, &d.CapitalPayable, &d.InterestPayable, &d.DefaultInterest)
		if err != nil {
			return 0, err
		}
		debits = append(debits, d)
	}

	fmt.Println(debits)

	var diUpdates []models.ContractDefaultInterestUpdate
	var diLogs []models.ContractDefaultInterestChangeHistory
	var diPayments []models.ContractPayment
	var intPayments []models.ContractPayment
	var capPayments []models.ContractPayment

	balance := amount

	rid, err := msql.Insert(msql.Table{
		TableName: "contract_receipt",
		Columns:   []string{"user_id", "contract_id", "datetime", "amount", "notes", "due_date"},
		Vals:      []interface{}{user_id, cid, time.Now().Format("2006-01-02 15:04:05"), amount, notes, due_date},
		Tx:        tx,
	})
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if balance != 0 {
		for _, p := range debits {
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

	results, err = tx.Query(queries.OVERDUE_INSTALLMENTS, cid)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	var payables []models.ContractPayable
	for results.Next() {
		var p models.ContractPayable
		err = results.Scan(&p.InstallmentID, &p.ContractID, &p.CapitalPayable, &p.InterestPayable, &p.DefaultInterest)
		if err != nil {
			return 0, err
		}
		payables = append(payables, p)
	}

	fmt.Println(payables)

	if balance != 0 {
		for _, p := range payables {
			if p.DefaultInterest != 0 && balance != 0 {
				if balance-p.DefaultInterest >= 0 {
					diUpdates = append(diUpdates, models.ContractDefaultInterestUpdate{p.InstallmentID, float64(0)})
					diLogs = append(diLogs, models.ContractDefaultInterestChangeHistory{p.InstallmentID, rid, p.DefaultInterest})
					diPayments = append(diPayments, models.ContractPayment{p.InstallmentID, rid, p.DefaultInterest})
					balance = math.Round((balance-p.DefaultInterest)*100) / 100
				} else {
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
		results, err = tx.Query(queries.UPCOMING_INSTALLMENTS, cid)
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		var upcoming []models.ContractPayable
		for results.Next() {
			var u models.ContractPayable
			err = results.Scan(&u.InstallmentID, &u.ContractID, &u.CapitalPayable, &u.InterestPayable, &u.DefaultInterest)
			if err != nil {
				return 0, err
			}
			upcoming = append(upcoming, u)
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
		_, err = msql.Update(msql.UpdateTable{
			Table: msql.Table{TableName: "contract_installment",
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
		_, err := msql.Insert(msql.Table{
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
		_, err := msql.Insert(msql.Table{
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

	for _, intPayment := range intPayments {
		_, err := msql.Insert(msql.Table{
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
		_, err := msql.Insert(msql.Table{
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

	var managedByAgrivest int
	var telephone string
	err = tx.QueryRow(queries.MANAGED_BY_AGRIVEST, cid).Scan(&managedByAgrivest, &telephone)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	apiKey := ""
	if managedByAgrivest == 0 {
		apiKey = rAPIKey
	} else {
		apiKey = aAPIKey
	}
	message := fmt.Sprintf("Hithawath paribhogikaya, obage giwisum anka %d wetha gewu mudala Rs. %s. Niyamitha dinayata pera wadi mudalak gewa polee wasi laba ganna. Sthuthiyi.", cid, humanize.Comma(int64(amount)))
	telephone = fmt.Sprintf("%s,%s,%s,%s,%s", telephone, "768237192", "703524330", "703524420", "775607777")
	requestURL := fmt.Sprintf("https://cpsolutions.dialog.lk/index.php/cbs/sms/send?destination=%s&q=%s&message=%s", telephone, apiKey, url.QueryEscape(message))
	resp, err := http.Get(requestURL)
	if err != nil {
		return rid, nil
	}

	defer resp.Body.Close()

	return rid, nil
}

func (m *ContractModel) LegacyReceipt(user_id, cid int, amount float64, notes string) (int64, error) {
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

	results, err := tx.Query(queries.DEBITS, cid)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	var debits []models.ContractPayable
	for results.Next() {
		var d models.ContractPayable
		err = results.Scan(&d.InstallmentID, &d.ContractID, &d.CapitalPayable, &d.InterestPayable, &d.DefaultInterest)
		if err != nil {
			return 0, err
		}
		debits = append(debits, d)
	}

	fmt.Println(debits)

	var diUpdates []models.ContractDefaultInterestUpdate
	var diLogs []models.ContractDefaultInterestChangeHistory
	var diPayments []models.ContractPayment
	var intPayments []models.ContractPayment
	var capPayments []models.ContractPayment

	balance := amount

	rid, err := msql.Insert(msql.Table{
		TableName: "contract_receipt",
		Columns:   []string{"user_id", "contract_id", "datetime", "amount", "notes"},
		Vals:      []interface{}{user_id, cid, time.Now().Format("2006-01-02 15:04:05"), amount, notes},
		Tx:        tx,
	})
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if balance != 0 {
		for _, p := range debits {
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
		results, err = tx.Query(queries.LEGACY_PAYMENTS, cid)
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		var upcoming []models.ContractPayable
		for results.Next() {
			var u models.ContractPayable
			err = results.Scan(&u.InstallmentID, &u.ContractID, &u.CapitalPayable, &u.InterestPayable, &u.DefaultInterest)
			if err != nil {
				return 0, err
			}
			upcoming = append(upcoming, u)
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
		_, err = msql.Update(msql.UpdateTable{
			Table: msql.Table{TableName: "contract_installment",
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
		_, err := msql.Insert(msql.Table{
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
		_, err := msql.Insert(msql.Table{
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

	for _, intPayment := range intPayments {
		_, err := msql.Insert(msql.Table{
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
		_, err := msql.Insert(msql.Table{
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

	return rid, nil
}

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
	s := msql.NewNullString(state)
	o := msql.NewNullString(officer)
	b := msql.NewNullString(batch)

	results, err := m.DB.Query(queries.SEARCH, k, k, s, s, o, o, b, b)
	if err != nil {
		return nil, err
	}

	var res []models.SearchResult
	for results.Next() {
		var r models.SearchResult
		err = results.Scan(&r.ID, &r.Agrivest, &r.RecoveryOfficer, &r.State, &r.Model, &r.ChassisNumber, &r.CustomerName, &r.CustomerContact, &r.AmountPending, &r.TotalPayable, &r.TotalAgreement, &r.TotalPaid, &r.TotalDIPaid)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}

	return res, nil
}
