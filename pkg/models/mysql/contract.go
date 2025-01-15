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
	"github.com/ssrdive/scribe"
	smodels "github.com/ssrdive/scribe/models"
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

	journalEntries := []smodels.JournalEntry{
		{fmt.Sprintf("%d", 95), "", fmt.Sprintf("%f", capital)},
		{fmt.Sprintf("%d", 78), "", fmt.Sprintf("%f", interestAmount)},
		{fmt.Sprintf("%d", 25), fmt.Sprintf("%f", fullRecievables), ""},
	}

	err = scribe.IssueJournalEntries(tx, tid, journalEntries)
	if err != nil {
		tx.Rollback()
		return err
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
	err = tx.QueryRow(detailsQuery, cid).Scan(&detail.ID, &detail.HoldDefault, &detail.ContractState,
		&detail.ContractBatch, &detail.ModelName, &detail.ChassisNumber, &detail.CustomerName, &detail.CustomerNic,
		&detail.CustomerAddress, &detail.CustomerContact, &detail.LiaisonName, &detail.LiaisonContact,
		&detail.Price, &detail.Downpayment, &detail.IntroducingOfficer, &detail.CreditOfficer, &detail.RecoveryOfficer,
		&detail.AmountPending, &detail.TotalPayable, &detail.DefaultCharges, &detail.TotalPaid, &detail.LastPaymentDate,
		&detail.OverdueIndex)
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

// MicroLoanDetails returns params for micro loan agreement
func (m *ContractModel) MicroLoanDetails(cid int) ([]models.Question, error) {
	var res []models.Question
	err := mysequel.QueryToStructs(&res, m.DB, queries.CONTRACT_MICRO_LOAN_DETAILS, cid)
	if err != nil {
		return nil, err
	}

	results, err := m.DB.Query(queries.PARAMS_FOR_CONTRACT_INITIATION_BY_ID, cid)
	if err != nil {
		return nil, err
	}

	var params []models.Dropdown
	for results.Next() {
		var p models.Dropdown
		err = results.Scan(&p.ID, &p.Name)
		if err != nil {
			return nil, err
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
		return nil, err
	}

	_, financialSchedule, err := loan.Create(capital, rate, installments, installmentInterval, structuredMonthlyRental, initiationDate.Format("2006-01-02"), method)
	if err != nil {
		return nil, err
	}

	intstallment := financialSchedule[0].Capital + financialSchedule[0].Interest
	installmentStr := strconv.FormatFloat(intstallment, 'f', -1, 64)

	res = append(res, models.Question{
		Question: "Installment",
		Answer:   installmentStr,
	})
	res = append(res, models.Question{
		Question: "First Installment Date",
		Answer:   financialSchedule[0].MarketedDueDate,
	})
	res = append(res, models.Question{
		Question: "Last Installment Date",
		Answer:   financialSchedule[len(financialSchedule)-1].MarketedDueDate,
	})

	return res, nil
}

// DocGen returns document generation options for state
func (m *ContractModel) DocGen(cid int) ([]models.DocGen, error) {
	var res []models.DocGen
	err := mysequel.QueryToStructs(&res, m.DB, queries.CONTRACT_STATE_DOC_GEN, cid)
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
func (m *ContractModel) OfficerReceipts(oid int, date string) ([]models.AndroidReceipt, error) {
	var res []models.AndroidReceipt
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

// TemporaryAssignment returns contract temporary assignment
func (m *ContractModel) TemporaryAssignment(cid int) ([]models.TemporaryAssignment, error) {
	var res []models.TemporaryAssignment
	err := mysequel.QueryToStructs(&res, m.DB, queries.TEMPORARY_ASSIGNMENT, cid)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// LegalStatus returns contract legal status
func (m *ContractModel) LegalCaseStatus(cid int) ([]models.LegalCaseStatusResponse, error) {
	var res []models.LegalCaseStatusResponse
	err := mysequel.QueryToStructs(&res, m.DB, queries.LEGAL_CASE_STATUS, cid)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Timeline returns contract timeline
func (m *ContractModel) Timeline(cid int) ([]models.TimelineRow, error) {
	var contractChanges []models.ContractBalanceChangeRow
	err := mysequel.QueryToStructs(&contractChanges, m.DB, queries.CONTRACT_CHANGES, cid, cid, cid)
	if err != nil {
		return nil, err
	}

	if len(contractChanges) <= 1 {
		return []models.TimelineRow{}, nil
	}

	var lines []models.TimelineRow

	location, _ := time.LoadLocation("Asia/Colombo")

	for _, line := range contractChanges {
		parsedDate, _ := time.ParseInLocation("2006-01-02", line.Date[0:10], location)
		lines = append(lines, models.TimelineRow{
			ContractID: line.ContractID,
			Type:       line.Type,
			Amount:     line.Amount,
			Date:       parsedDate,
		})
	}

	lines[0].Grouping = 0

	for i := 1; i < len(lines); i++ {
		if lines[i-1].Date.Equal(lines[i].Date) {
			lines[i].Grouping = lines[i-1].Grouping
		} else {
			lines[i].Grouping = lines[i-1].Grouping + 1
		}
	}

	due := 0.0
	payments := 0.0

	var returnLines []models.TimelineRow

	if lines[0].Type == "Receipt" {
		payments = math.Round((payments+lines[0].Amount)*100) / 100
	} else {
		due = math.Round((due+lines[0].Amount)*100) / 100
	}

	returnLines = append(returnLines, lines[0])

	cumulativeArrears := 0
	prevArrears := 0.0

	for i := 1; i < len(lines); i++ {
		if lines[i-1].Grouping == lines[i].Grouping {
			if lines[i].Type == "Receipt" {
				payments = math.Round((payments+lines[i].Amount)*100) / 100
			} else {
				due = math.Round((due+lines[i].Amount)*100) / 100
			}
			//fmt.Println(lines[i])
			returnLines = append(returnLines, lines[i])
		} else {
			duration := lines[i].Date.Sub(lines[i-1].Date)
			days := int(duration.Hours() / 24)

			balance := math.Round((due-payments)*100) / 100

			if balance > 0 {
				cumulativeArrears += days
				arrearsChange := math.Round((balance-prevArrears)*100) / 100
				returnLines = append(returnLines, models.TimelineRow{
					Grouping:       0,
					ContractID:     lines[i].ContractID,
					Type:           "Overdue",
					Amount:         balance,
					Date:           time.Time{},
					Change:         arrearsChange,
					Days:           days,
					DaysCumulative: cumulativeArrears,
				})
				//fmt.Println("Arrears amount: ", balance, " for ", days, " days cumalative arrears = ", cumulativeArrears)
				//fmt.Println("Arrears change: ", arrearsChange)
				prevArrears = balance
			} else if balance < 0 {
				//fmt.Println("Overpayment: ", balance, " for ", days, " days")
				returnLines = append(returnLines, models.TimelineRow{
					Grouping:       0,
					ContractID:     lines[i].ContractID,
					Type:           "Overpayment",
					Amount:         balance,
					Date:           time.Time{},
					Change:         0,
					Days:           days,
					DaysCumulative: 0,
				})
				cumulativeArrears = 0
				prevArrears = 0
			} else {
				//fmt.Println("Zero balance", " for ", days, " days")
				returnLines = append(returnLines, models.TimelineRow{
					Grouping:       0,
					ContractID:     lines[i].ContractID,
					Type:           "Zero Balance",
					Amount:         0,
					Date:           time.Time{},
					Change:         0,
					Days:           days,
					DaysCumulative: 0,
				})
				cumulativeArrears = 0
				prevArrears = 0
			}

			if lines[i].Type == "Receipt" {
				payments = math.Round((payments+lines[i].Amount)*100) / 100
			} else {
				due = math.Round((due+lines[i].Amount)*100) / 100
			}

			returnLines = append(returnLines, lines[i])
		}
	}

	currentBalance := due - payments
	duration := time.Now().Sub(lines[len(lines)-1].Date)
	days := int(duration.Hours() / 24)
	if currentBalance > 0 {
		arrersChange := currentBalance - prevArrears
		cumulativeArrears += days
		//fmt.Println("Arrears amount: ", currentBalance, " for ", days, " days cumalative arrears = ", cumulativeArrears)
		//fmt.Println("Arrears change: ", arrersChange)
		returnLines = append(returnLines, models.TimelineRow{
			Grouping:       0,
			ContractID:     lines[0].ContractID,
			Type:           "Overdue",
			Amount:         currentBalance,
			Date:           time.Time{},
			Change:         arrersChange,
			Days:           days,
			DaysCumulative: cumulativeArrears,
		})
	} else if currentBalance > 0 {
		//fmt.Println("Overpayment: ", currentBalance, " for ", days, " days")
		returnLines = append(returnLines, models.TimelineRow{
			Grouping:       0,
			ContractID:     lines[0].ContractID,
			Type:           "Overpayment",
			Amount:         currentBalance,
			Date:           time.Time{},
			Change:         0,
			Days:           days,
			DaysCumulative: 0,
		})
	} else {
		//fmt.Println("Zero balance", " for ", days, " days")
		returnLines = append(returnLines, models.TimelineRow{
			Grouping:       0,
			ContractID:     lines[0].ContractID,
			Type:           "Zero Balance",
			Amount:         0,
			Date:           time.Time{},
			Change:         0,
			Days:           days,
			DaysCumulative: 0,
		})
	}

	return returnLines, nil
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
	err := m.DB.QueryRow(queries.SEASONAL_INCENTIVE, user).Scan(&r.Amount)
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

	var external int
	err = tx.QueryRow(queries.CONTRACT_LEAD_TYPE, cid).Scan(&external)

	if external == 1 {
		payableAccount = 314
	}

	journalEntries := []smodels.JournalEntry{
		{fmt.Sprintf("%d", receivableAccount), fmt.Sprintf("%f", fullRecievables), ""},
		{fmt.Sprintf("%d", payableAccount), "", fmt.Sprintf("%f", capital)},
		{fmt.Sprintf("%d", unearnedInterestAccount), "", fmt.Sprintf("%f", interestAmount)},
	}

	err = scribe.IssueJournalEntries(tx, tid, journalEntries)
	if err != nil {
		tx.Rollback()
		return err
	}

	var floatReceipts []models.FloatReceipts
	mysequel.QueryToStructs(&floatReceipts, tx, queries.FLOAT_RECEIPTS, cid)

	// Issue receipts in float
	if len(floatReceipts) > 0 {
		for _, r := range floatReceipts {
			_, _, err = sprinter.Run(r.Date, fmt.Sprintf("%d", cid), true, tx)
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
	_, _, err = sprinter.Run(time.Now().Format("2006-01-02"), fmt.Sprintf("%d", cid), true, tx)
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

func (m *ContractModel) SetLegalCaseStatus(rparams, oparams []string, form url.Values) (int64, error) {
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
			TableName: "contract",
			Columns:   []string{"legal_case"},
			Vals:      []interface{}{form.Get("legal_case_status")},
			Tx:        tx,
		},
		WColumns: []string{"id"},
		WVals:    []string{form.Get("contract_id")},
	})
	if err != nil {
		return 0, err
	}

	return 0, nil
}

func (m *ContractModel) SetTemporaryOfficer(rparams, oparams []string, form url.Values) (int64, error) {
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
			TableName: "contract",
			Columns:   []string{"temporary_officer"},
			Vals:      []interface{}{form.Get("temporary_officer")},
			Tx:        tx,
		},
		WColumns: []string{"id"},
		WVals:    []string{form.Get("contract_id")},
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

		journalEntries := []smodels.JournalEntry{
			{Account: fmt.Sprintf("%d", expenseAccount), Debit: form.Get("capital"), Credit: ""},
			{Account: fmt.Sprintf("%d", receivableAccount), Debit: "", Credit: form.Get("capital")},
		}

		err = scribe.IssueJournalEntries(tx, tid, journalEntries)
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

	journalEntries := []smodels.JournalEntry{
		{Account: fmt.Sprintf("%d", 25), Debit: form.Get("capital"), Credit: ""},
		{Account: fmt.Sprintf("%d", unearnedAccountID), Debit: "", Credit: form.Get("capital")},
	}

	err = scribe.IssueJournalEntries(tx, tid, journalEntries)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	return dnid, nil
}

func (m *ContractModel) DebitNoteWithtTx(tx *sql.Tx, rparams, oparams []string, form url.Values) (int64, error) {
	var lkas17Compliant int
	err := tx.QueryRow(queries.LKAS_17_COMPLIANT, form.Get("contract_id")).Scan(&lkas17Compliant)
	if err != nil {
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
			return 0, err
		}

		_, err = tx.Exec("UPDATE contract_financial SET charges_debits_arrears = charges_debits_arrears + ? WHERE contract_id = ?", form.Get("capital"), form.Get("contract_id"))
		if err != nil {
			return 0, err
		}

		tid, err := mysequel.Insert(mysequel.Table{
			TableName: "transaction",
			Columns:   []string{"user_id", "datetime", "posting_date", "contract_id", "remark"},
			Vals:      []interface{}{form.Get("user_id"), time.Now().Format("2006-01-02 15:04:05"), time.Now().Format("2006-01-02"), form.Get("contract_id"), fmt.Sprintf("DEBIT NOTE %d [%s]", dnid, form.Get("contract_id"))},
			Tx:        tx,
		})
		if err != nil {
			return 0, err
		}

		var expenseAccount, receivableAccount int
		err = tx.QueryRow(queries.GET_DEBIT_TYPE_EXPENSE_RECEIVABLE_ACCOUNT, form.Get("contract_installment_type_id")).Scan(&expenseAccount, &receivableAccount)
		if err != nil {
			return 0, err
		}

		journalEntries := []smodels.JournalEntry{
			{Account: fmt.Sprintf("%d", expenseAccount), Debit: form.Get("capital"), Credit: ""},
			{Account: fmt.Sprintf("%d", receivableAccount), Debit: "", Credit: form.Get("capital")},
		}

		err = scribe.IssueJournalEntries(tx, tid, journalEntries)
		if err != nil {
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
		return 0, err
	}

	journalEntries := []smodels.JournalEntry{
		{Account: fmt.Sprintf("%d", 25), Debit: form.Get("capital"), Credit: ""},
		{Account: fmt.Sprintf("%d", unearnedAccountID), Debit: "", Credit: form.Get("capital")},
	}

	err = scribe.IssueJournalEntries(tx, tid, journalEntries)
	if err != nil {
		return 0, err
	}

	return dnid, nil
}

func (m *ContractModel) LKAS17Rebate(userID, cid int, amount float64) (int64, error) {
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

	fBalance := amount

	rid, err := mysequel.Insert(mysequel.Table{
		TableName: "contract_receipt",
		Columns:   []string{"lkas_17", "contract_receipt_type_id", "user_id", "contract_id", "datetime", "amount"},
		Vals:      []interface{}{1, 3, userID, cid, time.Now().Format("2006-01-02 15:04:05"), amount},
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}
	m.ReceiptLogger.Printf("REBATE RID %d", rid)

	var fInterestPayables []models.ContractPayable
	err = mysequel.QueryToStructs(&fInterestPayables, tx, queries.FINANCIAL_INTEREST_PAYABLES_FOR_REBATES, cid)
	if err != nil {
		return 0, err
	}

	var fInts []models.ContractPayment

	if len(fInterestPayables) > 0 && fBalance != 0 {
		fInts = payments("I", rid, &fBalance, fInterestPayables, fInts)
	}

	if fBalance != 0 {
		return 0, errors.New("Error: Payment exceeds payables")
	}

	fIntPaid := float64(0)
	for _, intPayment := range fInts {
		fIntPaid = fIntPaid + intPayment.Amount
		_, err := mysequel.Insert(mysequel.Table{
			TableName: "contract_financial_payment",
			Columns:   []string{"contract_payment_type_id", "contract_schedule_id", "contract_receipt_id", "amount"},
			Vals:      []interface{}{2, intPayment.ContractInstallmentID, intPayment.ContractReceiptID, intPayment.Amount},
			Tx:        tx,
		})
		if err != nil {
			return 0, err
		}

		_, err = tx.Exec("UPDATE contract_schedule SET interest_paid = interest_paid + ?, installment_paid = installment_paid + ? WHERE id = ?", intPayment.Amount, intPayment.Amount, intPayment.ContractInstallmentID)
		if err != nil {
			return 0, err
		}
	}

	mBalance := amount

	var mPayables []models.ContractPayable
	err = mysequel.QueryToStructs(&mPayables, tx, queries.MARKETED_PAYABLES_FOR_REBATE, cid)
	if err != nil {
		return 0, err
	}

	var mInts []models.ContractPayment
	var mCaps []models.ContractPayment

	if mBalance != 0 {
		mInts = payments("I", rid, &mBalance, mPayables, mInts)
	}

	if mBalance != 0 {
		mCaps = payments("C", rid, &mBalance, mPayables, mCaps)
	}

	if mBalance != 0 {
		return 0, errors.New("Error: Payment exceeds payables")
	}

	for _, intPayment := range mInts {
		_, err := mysequel.Insert(mysequel.Table{
			TableName: "contract_marketed_payment",
			Columns:   []string{"contract_payment_type_id", "contract_schedule_id", "contract_receipt_id", "amount"},
			Vals:      []interface{}{2, intPayment.ContractInstallmentID, intPayment.ContractReceiptID, intPayment.Amount},
			Tx:        tx,
		})
		if err != nil {
			return 0, err
		}

		_, err = tx.Exec("UPDATE contract_schedule SET marketed_interest_paid = marketed_interest_paid + ? WHERE id = ?", intPayment.Amount, intPayment.ContractInstallmentID)
		if err != nil {
			return 0, err
		}
	}

	for _, capPayment := range mCaps {
		_, err := mysequel.Insert(mysequel.Table{
			TableName: "contract_marketed_payment",
			Columns:   []string{"contract_payment_type_id", "contract_schedule_id", "contract_receipt_id", "amount"},
			Vals:      []interface{}{1, capPayment.ContractInstallmentID, capPayment.ContractReceiptID, capPayment.Amount},
			Tx:        tx,
		})
		if err != nil {
			return 0, err
		}

		_, err = tx.Exec("UPDATE contract_schedule SET marketed_capital_paid = marketed_capital_paid + ? WHERE id = ?", capPayment.Amount, capPayment.ContractInstallmentID)
		if err != nil {
			return 0, err
		}
	}

	// Obtain active / period over, arrears status, last installment date
	var cF ContractFinancial
	err = tx.QueryRow(queries.ContractFinancial, cid).Scan(&cF.Active, &cF.RecoveryStatus, &cF.Doubtful, &cF.Payment, &cF.CapitalArrears, &cF.InterestArrears, &cF.CapitalProvisioned, &cF.ScheduleEndDate)
	if err != nil {
		return 0, err
	}
	m.ReceiptLogger.Printf("REBATE RID %d \t %+v", rid, cF)

	_, err = tx.Exec("UPDATE contract_financial SET interest_paid = interest_paid + ?, interest_arrears = interest_arrears - ? WHERE contract_id = ?", fIntPaid, fIntPaid, cid)
	if err != nil {
		return 0, err
	}

	tid, err := mysequel.Insert(mysequel.Table{
		TableName: "transaction",
		Columns:   []string{"user_id", "datetime", "posting_date", "contract_id", "remark"},
		Vals:      []interface{}{userID, time.Now().Format("2006-01-02 15:04:05"), time.Now().Format("2006-01-02"), cid, fmt.Sprintf("INTEREST REBATE %d [%d]", rid, cid)},
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	rebateJEs := []smodels.JournalEntry{
		{Account: fmt.Sprintf("%d", RebateExpenseAccount), Debit: fmt.Sprintf("%f", amount), Credit: ""},
		{Account: fmt.Sprintf("%d", ReceivableArrearsAccount), Debit: "", Credit: fmt.Sprintf("%f", amount)},
	}

	arrears := cF.CapitalArrears + cF.InterestArrears
	nAge := arrears / cF.Payment

	if nAge <= 0 && cF.Doubtful == 0 {
		m.ReceiptLogger.Printf("REBATE RID %d \t %s", rid, "nAge <= 0 && cF.Doubtful == 0")
		// db_txn, txn_id, interest, capital_provisioned
		rebateJEs, err = addBadDebtJEsUpdateStatus(tx, int64(cid), tid, 0, cF.CapitalProvisioned, rebateJEs, `UPDATE contract_financial SET recovery_status_id = ?, doubtful = ? WHERE contract_id = ?`, RecoveryStatusActive, 0, cid)
		if err != nil {
			return 0, err
		}
	} else if nAge <= 0 && cF.Doubtful == 1 {
		m.ReceiptLogger.Printf("REBATE RID %d \t %s", rid, "nAge <= 0 && cF.Doubtful == 1")
		// db_txn, txn_id, interest, capital_provisioned
		rebateJEs, err = addBadDebtJEsUpdateStatus(tx, int64(cid), tid, cF.InterestArrears, cF.CapitalProvisioned, rebateJEs, `UPDATE contract_financial SET recovery_status_id = ?, doubtful = ? WHERE contract_id = ?`, RecoveryStatusActive, 0, cid)
		if err != nil {
			return 0, err
		}
	} else if (cF.RecoveryStatus == RecoveryStatusArrears && nAge > 0 && cF.Doubtful == 1) || (cF.RecoveryStatus == RecoveryStatusNPL && nAge < 6) ||
		(cF.RecoveryStatus == RecoveryStatusBDP && nAge < 6) {
		m.ReceiptLogger.Printf("REBATE RID %d \t %s", rid, `(cF.RecoveryStatus == RecoveryStatusArrears && nAge > 0 && cF.Doubtful == 1) || (cF.RecoveryStatus == RecoveryStatusNPL && nAge < 6) ||
		(cF.RecoveryStatus == RecoveryStatusBDP && nAge < 6)`)
		// db_txn, txn_id, interest, capital_provisioned
		rebateJEs, err = addBadDebtJEsUpdateStatus(tx, int64(cid), tid, fIntPaid, cF.CapitalProvisioned, rebateJEs, `UPDATE contract_financial SET recovery_status_id = ? WHERE contract_id = ?`, RecoveryStatusArrears, cid)
		if err != nil {
			return 0, err
		}
	} else if (cF.RecoveryStatus == RecoveryStatusNPL && nAge >= 6) || (cF.RecoveryStatus == RecoveryStatusBDP && nAge >= 12) {
		m.ReceiptLogger.Printf("REBATE RID %d \t %s", rid, "nAge >= 6 || nAge >= 12")
		bdJEs, err := badDebtReceiptJEProvision(tx, int64(cid), tid, fIntPaid, 0)
		if err != nil {
			return 0, err
		}
		rebateJEs = append(rebateJEs, bdJEs...)
	} else if cF.RecoveryStatus == RecoveryStatusBDP && nAge < 12 {
		m.ReceiptLogger.Printf("REBATE RID %d \t %s", rid, "cF.RecoveryStatus == RecoveryStatusBDP && nAge < 12")
		var capitalProvision float64
		err = tx.QueryRow(queries.NplCapitalProvision, cid).Scan(&capitalProvision)
		if err != nil {
			return 0, err
		}
		capitalProvisionRemoval := math.Round((cF.CapitalProvisioned-capitalProvision)*100) / 100

		// db_txn, txn_id, interest, capital_provisioned
		rebateJEs, err = addBadDebtJEsUpdateStatus(tx, int64(cid), tid, fIntPaid, capitalProvisionRemoval, rebateJEs, `UPDATE contract_financial SET recovery_status_id = ? WHERE contract_id = ?`, RecoveryStatusNPL, cid)
		if err != nil {
			return 0, err
		}
	}

	err = scribe.IssueJournalEntries(tx, tid, rebateJEs)
	if err != nil {
		return 0, err
	}

	m.ReceiptLogger.Printf("REBATE RID %d \t %s", rid, "LKAS 17 function complete")
	return rid, err
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

	journalEntries := []smodels.JournalEntry{
		{fmt.Sprintf("%d", 78), fmt.Sprintf("%f", amount), ""},
		{fmt.Sprintf("%d", 25), "", fmt.Sprintf("%f", amount)},
	}

	err = scribe.IssueJournalEntries(tx, tid, journalEntries)

	if err != nil {
		tx.Rollback()
		return 0, err
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
func (m *ContractModel) SearchV2(searchType, search, state, officer, batch, npl, lkas17, external, startOd, endOd, removeDeleted string) ([]models.SearchResultV2, error) {
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
	e := mysequel.NewNullString(external)
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
	if searchType == "default" {
		err = mysequel.QueryToStructs(&res, m.DB, queries.SEARCH_V2, k, k, s, s, o, o, b, b, n, n, l, l, e, e, sod, eod, sod, eod, rd, k, k, s, s, o, o, b, b, n, n, l, l, e, e, sod, eod, sod, eod, rd)
	} else if searchType == "archived" {
		err = mysequel.QueryToStructs(&res, m.DB, queries.SEARCH_V2_ARCHIVED, k, k, s, s, o, o, b, b, n, n, l, l, e, e, sod, eod, sod, eod, rd, k, k, s, s, o, o, b, b, n, n, l, l, e, e, sod, eod, sod, eod, rd)
	} else if searchType == "micro" {
		err = mysequel.QueryToStructs(&res, m.DB, queries.SEARCH_V2_MICRO, k, k, s, s, o, o, b, b, n, n, l, l, e, e, sod, eod, sod, eod, rd, k, k, s, s, o, o, b, b, n, n, l, l, e, e, sod, eod, sod, eod, rd)
	}

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
	err := mysequel.QueryToStructs(&res, m.DB, queries.SEARCH, k, k, s, s, o, o, b, b, o, o, k, k, s, s, o, o, b, b, o, o)
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
