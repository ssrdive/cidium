package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/ssrdive/cidium/pkg/models"
	"github.com/ssrdive/cidium/pkg/sql/queries"
	"github.com/ssrdive/mysequel"
)

const (
	// UnearnedInterestAccount holds account database id
	UnearnedInterestAccount = 188
	// InterestIncomeAccount holds account database id
	InterestIncomeAccount = 190
	// ReceivableAccount holds account database id
	ReceivableAccount = 185
	// ReceivableArrearsAccount holds account database id
	ReceivableArrearsAccount = 192
	// SuspenseInterestAccount holds account database id
	SuspenseInterestAccount = 194
	// BadDebtProvisionAccount holds account database id
	BadDebtProvisionAccount = 195
	// ProvisionForBadDebtAccount holds account database id
	ProvisionForBadDebtAccount = 196
	// ReceiptsForIncompleteContractsAccount holds account database id
	ReceiptsForIncompleteContractsAccount = 144

	// RecoveryStatusActive holds status database id
	RecoveryStatusActive = 1
	// RecoveryStatusArrears holds status database id
	RecoveryStatusArrears = 2
	// RecoveryStatusNPL holds status database id
	RecoveryStatusNPL = 3
	// RecoveryStatusBDP holds status database id
	RecoveryStatusBDP = 4
)

// ContractFinancial holds financial summary
// related to contracts
type ContractFinancial struct {
	Active             int
	RecoveryStatus     int
	Doubtful           int
	Payment            float64
	CapitalArrears     float64
	InterestArrears    float64
	CapitalProvisioned float64
	ScheduleEndDate    string
}

// Receipt issues a receipt
func (m *ContractModel) Receipt(userID, cid int, amount float64, notes, dueDate, rAPIKey, aAPIKey, runtimeEnv string) (int64, error) {
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

	if amount <= 0 {
		tx.Rollback()
		return 0, err
	}

	var managedByAgrivest int
	var lkas17Compliant int
	var telephone string
	err = tx.QueryRow(queries.MANAGED_BY_AGRIVEST_LKAS17_COMPLIANT, cid).Scan(&lkas17Compliant, &managedByAgrivest, &telephone)
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
	message := fmt.Sprintf("Hithawath paribhogikaya, obage giwisum anka %d wetha gewu mudala Rs. %s. Sthuthiyi.", cid, humanize.Comma(int64(amount)))
	if runtimeEnv == "dev" {
		telephone = fmt.Sprintf("%s", "768237192")
	} else {
		telephone = fmt.Sprintf("%s,%s,%s,%s,%s,%s", telephone, "768237192", "703524330", "703524420", "775607777", "703524278")
	}
	requestURL := fmt.Sprintf("https://cpsolutions.dialog.lk/index.php/cbs/sms/send?destination=%s&q=%s&message=%s", telephone, apiKey, url.QueryEscape(message))

	var contractTotalPayable float64
	if lkas17Compliant == 1 {
		err = tx.QueryRow(queries.CONTRACT_PAYABLE_LKAS_17, cid).Scan(&contractTotalPayable)
	} else {
		err = tx.QueryRow(queries.CONTRACT_PAYABLE, cid).Scan(&contractTotalPayable)
	}
	if err != nil {
		fmt.Println(err)
		tx.Rollback()
		return 0, err
	}

	var officerAccountID int
	err = tx.QueryRow(queries.OFFICER_ACC_NO, userID).Scan(&officerAccountID)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// Issue receipt to float
	if contractTotalPayable < amount {
		frid, err := mysequel.Insert(mysequel.Table{
			TableName: "contract_receipt_float",
			Columns:   []string{"user_id", "contract_id", "datetime", "amount"},
			Vals:      []interface{}{userID, cid, time.Now().Format("2006-01-02 15:04:05"), amount, notes, dueDate},
			Tx:        tx,
		})

		resp, err := http.Get(requestURL)
		if err != nil {
			return frid, nil
		}

		defer resp.Body.Close()

		tid, err := mysequel.Insert(mysequel.Table{
			TableName: "transaction",
			Columns:   []string{"user_id", "datetime", "posting_date", "contract_id", "remark"},
			Vals:      []interface{}{userID, time.Now().Format("2006-01-02 15:04:05"), time.Now().Format("2006-01-02"), cid, fmt.Sprintf("FLOAT RECEIPT %d", frid)},
			Tx:        tx,
		})
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		journalEntries := []models.JournalEntry{
			{Account: fmt.Sprintf("%d", officerAccountID), Debit: fmt.Sprintf("%f", amount), Credit: ""},
			{Account: fmt.Sprintf("%d", 144), Debit: "", Credit: fmt.Sprintf("%f", amount)},
		}

		err = IssueJournalEntries(tx, tid, journalEntries)
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		return frid, err
	}

	if lkas17Compliant == 1 {
		rid, err := m.IssueLKAS17Receipt(tx, userID, cid, amount, notes, dueDate, "REGULAR", time.Now())
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		return rid, nil
	}

	var debits []models.DebitPayable
	err = mysequel.QueryToStructs(&debits, tx, queries.DEBITS, cid)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	var intPayments []models.ContractPayment
	var capPayments []models.ContractPayment
	var debitPayments []models.DebitPayment

	balance := amount

	rid, err := mysequel.Insert(mysequel.Table{
		TableName: "contract_receipt",
		Columns:   []string{"user_id", "contract_id", "datetime", "amount", "notes", "due_date"},
		Vals:      []interface{}{userID, cid, time.Now().Format("2006-01-02 15:04:05"), amount, notes, dueDate},
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
					debitPayments = append(debitPayments, models.DebitPayment{ContractInstallmentID: p.InstallmentID, ContractReceiptID: rid, Amount: p.CapitalPayable, UnearnedAccountID: p.UnearnedAccountID, IncomeAccountID: p.IncomeAccountID})
					balance = math.Round((balance-p.CapitalPayable)*100) / 100
				} else {
					debitPayments = append(debitPayments, models.DebitPayment{ContractInstallmentID: p.InstallmentID, ContractReceiptID: rid, Amount: balance, UnearnedAccountID: p.UnearnedAccountID, IncomeAccountID: p.IncomeAccountID})
					balance = 0
				}
			}
		}
	}

	var payables []models.ContractPayable
	err = mysequel.QueryToStructs(&payables, tx, queries.OVERDUE_INSTALLMENTS, cid, time.Now().Format("2006-01-02"))
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if balance != 0 {
		intPayments = payments("I", rid, &balance, payables, intPayments)
	}

	if balance != 0 {
		capPayments = payments("C", rid, &balance, payables, capPayments)
	}

	if balance != 0 {
		var upcoming []models.ContractPayable
		err = mysequel.QueryToStructs(&upcoming, tx, queries.UPCOMING_INSTALLMENTS, cid, time.Now().Format("2006-01-02"))
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		for _, p := range upcoming {
			intPayments = payments("I", rid, &balance, []models.ContractPayable{p}, intPayments)
			capPayments = payments("C", rid, &balance, []models.ContractPayable{p}, capPayments)
		}
	}

	if balance != 0 {
		tx.Rollback()
		return 0, errors.New("Error: Payment exceeds payables")
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
		Vals:      []interface{}{userID, time.Now().Format("2006-01-02 15:04:05"), time.Now().Format("2006-01-02"), cid, fmt.Sprintf("RECEIPT %d", rid)},
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
		{Account: fmt.Sprintf("%d", officerAccountID), Debit: fmt.Sprintf("%f", amount), Credit: ""},
		{Account: fmt.Sprintf("%d", 25), Debit: "", Credit: fmt.Sprintf("%f", amount)},
		{Account: fmt.Sprintf("%d", 46), Debit: "", Credit: fmt.Sprintf("%f", interestAmount)},
		{Account: fmt.Sprintf("%d", 78), Debit: fmt.Sprintf("%f", interestAmount), Credit: ""},
	}

	err = IssueJournalEntries(tx, tid, journalEntries)

	if err != nil {
		return 0, err
	}

	resp, err := http.Get(requestURL)
	if err != nil {
		return rid, nil
	}

	defer resp.Body.Close()

	return rid, nil
}

func payments(payablesType string, rid int64, balance *float64, payables []models.ContractPayable, payments []models.ContractPayment) []models.ContractPayment {
	for _, p := range payables {
		if payablesType == "I" {
			if p.InterestPayable != 0 && *balance != 0 {
				if *balance-p.InterestPayable >= 0 {
					payments = append(payments, models.ContractPayment{ContractInstallmentID: p.InstallmentID, ContractReceiptID: rid, Amount: p.InterestPayable})
					*balance = math.Round((*balance-p.InterestPayable)*100) / 100
				} else {
					payments = append(payments, models.ContractPayment{ContractInstallmentID: p.InstallmentID, ContractReceiptID: rid, Amount: *balance})
					*balance = 0
				}
			}
		} else if payablesType == "C" {
			if p.CapitalPayable != 0 && *balance != 0 {
				if *balance-p.CapitalPayable >= 0 {
					payments = append(payments, models.ContractPayment{ContractInstallmentID: p.InstallmentID, ContractReceiptID: rid, Amount: p.CapitalPayable})
					*balance = math.Round((*balance-p.CapitalPayable)*100) / 100
				} else {
					payments = append(payments, models.ContractPayment{ContractInstallmentID: p.InstallmentID, ContractReceiptID: rid, Amount: *balance})
					*balance = 0
				}
			}
		}
	}
	return payments
}

func debitPayments(rid int64, balance *float64, payables []models.DebitPayableLKAS17, payments []models.DebitPaymentLKAS17) []models.DebitPaymentLKAS17 {
	for _, p := range payables {
		if p.CapitalPayable != 0 && *balance != 0 {
			if *balance-p.CapitalPayable >= 0 {
				payments = append(payments, models.DebitPaymentLKAS17{ContractInstallmentID: p.InstallmentID, ContractReceiptID: rid, Amount: p.CapitalPayable, ExpenseAccountID: p.ExpenseAccountID, ReceivableAccountID: p.ReceivableAccountID})
				*balance = math.Round((*balance-p.CapitalPayable)*100) / 100
			} else {
				payments = append(payments, models.DebitPaymentLKAS17{ContractInstallmentID: p.InstallmentID, ContractReceiptID: rid, Amount: *balance, ExpenseAccountID: p.ExpenseAccountID, ReceivableAccountID: p.ReceivableAccountID})
				*balance = 0
			}
		}
	}
	return payments
}

// IssueLKAS17Receipt issues receipts for contracts that are compliant with LKAS 17
func (m *ContractModel) IssueLKAS17Receipt(tx *sql.Tx, userID, cid int, amount float64, notes, dueDate, rType string, date time.Time) (int64, error) {
	fBalance := amount

	rid, err := mysequel.Insert(mysequel.Table{
		TableName: "contract_receipt",
		Columns:   []string{"lkas_17", "user_id", "contract_id", "datetime", "amount", "notes", "due_date"},
		Vals:      []interface{}{1, userID, cid, date.Format("2006-01-02 15:04:05"), amount, notes, dueDate},
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	var debits []models.DebitPayableLKAS17
	err = mysequel.QueryToStructs(&debits, tx, queries.DEBITS_LKAS_17, cid)
	if err != nil {
		return 0, err
	}

	var debitPymnts []models.DebitPaymentLKAS17
	if len(debits) != 0 {
		debitPymnts = debitPayments(rid, &fBalance, debits, debitPymnts)
	}

	var fArrears []models.ContractPayable
	err = mysequel.QueryToStructs(&fArrears, tx, queries.FINANCIAL_OVERDUE_INSTALLMENTS_LKAS_17, cid)
	if err != nil {
		return 0, err
	}

	var fInts []models.ContractPayment
	var fCaps []models.ContractPayment

	if len(fArrears) > 0 {
		if fBalance != 0 {
			fInts = payments("I", rid, &fBalance, fArrears, fInts)
		}

		if fBalance != 0 {
			fCaps = payments("C", rid, &fBalance, fArrears, fCaps)
		}
	}

	if fBalance != 0 {
		var fUpcoming []models.ContractPayable
		err = mysequel.QueryToStructs(&fUpcoming, tx, queries.FINANCIAL_UPCOMING_INSTALLMENTS_LKAS_17, cid)
		if err != nil {
			return 0, err
		}

		for _, p := range fUpcoming {
			fInts = payments("I", rid, &fBalance, []models.ContractPayable{p}, fInts)
			fCaps = payments("C", rid, &fBalance, []models.ContractPayable{p}, fCaps)
		}
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

	fCapPaid := float64(0)
	for _, capPayment := range fCaps {
		fCapPaid = fCapPaid + capPayment.Amount
		_, err := mysequel.Insert(mysequel.Table{
			TableName: "contract_financial_payment",
			Columns:   []string{"contract_payment_type_id", "contract_schedule_id", "contract_receipt_id", "amount"},
			Vals:      []interface{}{1, capPayment.ContractInstallmentID, capPayment.ContractReceiptID, capPayment.Amount},
			Tx:        tx,
		})
		if err != nil {
			return 0, err
		}

		_, err = tx.Exec("UPDATE contract_schedule SET capital_paid = capital_paid + ?, installment_paid = installment_paid + ? WHERE id = ?", capPayment.Amount, capPayment.Amount, capPayment.ContractInstallmentID)
		if err != nil {
			return 0, err
		}
	}

	debitJEs := []models.JournalEntry{}

	debitsPaid := float64(0)
	for _, debPayment := range debitPymnts {
		debitsPaid = debitsPaid + debPayment.Amount

		debitJEs = append(debitJEs, models.JournalEntry{Account: fmt.Sprintf("%d", debPayment.ExpenseAccountID), Debit: "", Credit: fmt.Sprintf("%f", debPayment.Amount)})

		_, err := mysequel.Insert(mysequel.Table{
			TableName: "contract_financial_payment",
			Columns:   []string{"contract_payment_type_id", "contract_schedule_id", "contract_receipt_id", "amount"},
			Vals:      []interface{}{3, debPayment.ContractInstallmentID, debPayment.ContractReceiptID, debPayment.Amount},
			Tx:        tx,
		})
		_, err = mysequel.Insert(mysequel.Table{
			TableName: "contract_marketed_payment",
			Columns:   []string{"contract_payment_type_id", "contract_schedule_id", "contract_receipt_id", "amount"},
			Vals:      []interface{}{3, debPayment.ContractInstallmentID, debPayment.ContractReceiptID, debPayment.Amount},
			Tx:        tx,
		})
		if err != nil {
			return 0, err
		}

		_, err = tx.Exec("UPDATE contract_schedule SET capital_paid = capital_paid + ?, installment_paid = installment_paid + ?, marketed_capital_paid = marketed_capital_paid + ? WHERE id = ?", debPayment.Amount, debPayment.Amount, debPayment.Amount, debPayment.ContractInstallmentID)
		if err != nil {
			return 0, err
		}
	}

	mBalance := amount - debitsPaid

	var mArrears []models.ContractPayable
	err = mysequel.QueryToStructs(&mArrears, tx, queries.MARKETED_OVERDUE_INSTALLMENTS_LKAS_17, cid)
	if err != nil {
		return 0, err
	}

	var mInts []models.ContractPayment
	var mCaps []models.ContractPayment

	if mBalance != 0 {
		mInts = payments("I", rid, &mBalance, mArrears, mInts)
	}

	if mBalance != 0 {
		mCaps = payments("C", rid, &mBalance, mArrears, mCaps)
	}

	if mBalance != 0 {
		var mUpcoming []models.ContractPayable
		err = mysequel.QueryToStructs(&mUpcoming, tx, queries.MARKETED_UPCOMING_INSTALLMENTS_LKAS_17, cid)
		if err != nil {
			return 0, err
		}

		for _, p := range mUpcoming {
			mInts = payments("I", rid, &mBalance, []models.ContractPayable{p}, mInts)
			mCaps = payments("C", rid, &mBalance, []models.ContractPayable{p}, mCaps)
		}
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

	_, err = tx.Exec("UPDATE contract_financial SET capital_paid = capital_paid + ?, interest_paid = interest_paid + ?, charges_debits_paid = charges_debits_paid + ?, capital_arrears = capital_arrears - ?, interest_arrears = interest_arrears - ?, charges_debits_arrears = charges_debits_arrears - ? WHERE contract_id = ?", fCapPaid, fIntPaid, debitsPaid, fCapPaid, fIntPaid, debitsPaid, cid)
	if err != nil {
		return 0, err
	}

	tid, err := mysequel.Insert(mysequel.Table{
		TableName: "transaction",
		Columns:   []string{"user_id", "datetime", "posting_date", "contract_id", "remark"},
		Vals:      []interface{}{1, time.Now().Format("2006-01-02 15:04:05"), time.Now().Format("2006-01-02"), cid, fmt.Sprintf("RECEIPT %d [%d]", rid, cid)},
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	receiptJEs := []models.JournalEntry{}

	cashJEs, err := cashInHandJE(tx, int64(userID), amount, amount-debitsPaid, debitJEs, rType)
	if err != nil {
		return 0, err
	}
	receiptJEs = append(receiptJEs, cashJEs...)

	arrears := cF.CapitalArrears + cF.InterestArrears
	nAge := (arrears - (amount - debitsPaid)) / cF.Payment

	if nAge <= 0 && cF.Doubtful == 1 {
		fmt.Println("nAge <= 0 && cF.Doubtful == 1")
		receiptJEs, err = addBadDebtJEsUpdateStatus(tx, int64(cid), tid, cF.InterestArrears, cF.CapitalProvisioned, receiptJEs, `UPDATE contract_financial SET recovery_status_id = ?, doubtful = ? WHERE contract_id = ?`, RecoveryStatusActive, 0, cid)
		if err != nil {
			return 0, err
		}
	} else if (cF.RecoveryStatus == RecoveryStatusArrears && nAge > 0 && cF.Doubtful == 1) || (cF.RecoveryStatus == RecoveryStatusNPL && nAge < 6) ||
		(cF.RecoveryStatus == RecoveryStatusBDP && nAge < 6) {
		fmt.Println(`(cF.RecoveryStatus == RecoveryStatusArrears && nAge > 0 && cF.Doubtful == 1) || (cF.RecoveryStatus == RecoveryStatusNPL && nAge < 6) ||
		(cF.RecoveryStatus == RecoveryStatusBDP && nAge < 6)`)
		receiptJEs, err = addBadDebtJEsUpdateStatus(tx, int64(cid), tid, fIntPaid, cF.CapitalProvisioned, receiptJEs, `UPDATE contract_financial SET recovery_status_id = ? WHERE contract_id = ?`, RecoveryStatusArrears, cid)
		if err != nil {
			return 0, err
		}
	} else if (cF.RecoveryStatus == RecoveryStatusNPL && nAge >= 6) || (cF.RecoveryStatus == RecoveryStatusBDP && nAge >= 12) {
		fmt.Println("nAge >= 6 || nAge >= 12")
		bdJEs, err := badDebtReceiptJEProvision(tx, int64(cid), tid, fIntPaid, fCapPaid)
		if err != nil {
			return 0, err
		}
		receiptJEs = append(receiptJEs, bdJEs...)
	} else if cF.RecoveryStatus == RecoveryStatusBDP && nAge < 12 {
		fmt.Println("cF.RecoveryStatus == RecoveryStatusBDP && nAge < 12")
		var capitalProvision float64
		err = tx.QueryRow(queries.NplCapitalProvision, cid).Scan(&capitalProvision)
		if err != nil {
			return 0, err
		}
		capitalProvisionRemoval := math.Round((cF.CapitalProvisioned-capitalProvision)*100) / 100

		receiptJEs, err = addBadDebtJEsUpdateStatus(tx, int64(cid), tid, fIntPaid, capitalProvisionRemoval, receiptJEs, `UPDATE contract_financial SET recovery_status_id = ? WHERE contract_id = ?`, RecoveryStatusNPL, cid)
		if err != nil {
			return 0, err
		}
	}

	err = IssueJournalEntries(tx, tid, receiptJEs)
	if err != nil {
		return 0, err
	}

	return rid, err
}

func addBadDebtJEsUpdateStatus(tx *sql.Tx, cid, tid int64, interest, capitalProvisioned float64, receiptJEs []models.JournalEntry, query string, queryArgs ...interface{}) ([]models.JournalEntry, error) {
	bdJEs, err := badDebtReceiptJEProvision(tx, int64(cid), tid, interest, capitalProvisioned)
	if err != nil {
		return nil, err
	}
	receiptJEs = append(receiptJEs, bdJEs...)
	_, err = tx.Exec(query, queryArgs...)
	if err != nil {
		return nil, err
	}
	return receiptJEs, nil
}

func badDebtReceiptJEProvision(tx *sql.Tx, cid, tid int64, interest, capital float64) ([]models.JournalEntry, error) {
	journalEntries := []models.JournalEntry{}
	if interest != 0 {
		journalEntries = append(journalEntries, models.JournalEntry{Account: fmt.Sprintf("%d", SuspenseInterestAccount), Debit: fmt.Sprintf("%f", interest), Credit: ""},
			models.JournalEntry{Account: fmt.Sprintf("%d", InterestIncomeAccount), Debit: "", Credit: fmt.Sprintf("%f", interest)})
	}
	if capital != 0 {
		// Reverse capital provisioned
		journalEntries = append(journalEntries, models.JournalEntry{Account: fmt.Sprintf("%d", ProvisionForBadDebtAccount), Debit: fmt.Sprintf("%f", capital), Credit: ""},
			models.JournalEntry{Account: fmt.Sprintf("%d", BadDebtProvisionAccount), Debit: "", Credit: fmt.Sprintf("%f", capital)})

		_, err := tx.Exec(`UPDATE contract_financial SET capital_provisioned = capital_provisioned - ? WHERE contract_id = ?`, capital, cid)
		if err != nil {
			return nil, err
		}
	}
	return journalEntries, nil
}

func cashInHandJE(tx *sql.Tx, userID int64, receiptAmount, arrearsDeduction float64, debits []models.JournalEntry, rType string) ([]models.JournalEntry, error) {
	var cashAccountID int
	if rType == "REGULAR" {
		err := tx.QueryRow(queries.OFFICER_ACC_NO, userID).Scan(&cashAccountID)
		if err != nil {
			return nil, err
		}
	} else {
		cashAccountID = ReceiptsForIncompleteContractsAccount
	}

	journalEntries := []models.JournalEntry{
		{Account: fmt.Sprintf("%d", cashAccountID), Debit: fmt.Sprintf("%f", receiptAmount), Credit: ""},
	}

	if len(debits) > 0 {
		for _, debit := range debits {
			journalEntries = append(journalEntries, debit)
		}
	}

	if arrearsDeduction > 0 {
		journalEntries = append(journalEntries, models.JournalEntry{Account: fmt.Sprintf("%d", ReceivableArrearsAccount), Debit: "", Credit: fmt.Sprintf("%f", arrearsDeduction)})
	}

	return journalEntries, nil
}
