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
	"github.com/ssrdive/cidium/pkg/sql/queries"
	"github.com/ssrdive/mysequel"
)

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

	var managedByAgrivest int
	var lcas17Compliant int
	var telephone string
	err = tx.QueryRow(queries.MANAGED_BY_AGRIVEST_LCAS17_COMPLIANT, cid).Scan(&lcas17Compliant, &managedByAgrivest, &telephone)
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
	err = tx.QueryRow(queries.CONTRACT_PAYABLE, cid).Scan(&contractTotalPayable)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	var officerAccountID int
	err = tx.QueryRow(queries.OFFICER_ACC_NO, userID).Scan(&officerAccountID)

	if lcas17Compliant == 1 {
		rid, err := issueLKAS17Receipt(tx, userID, cid, amount, notes, dueDate)
		if err != nil {
			tx.Rollback()
			return 0, err
		}

		return rid, nil
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
			{fmt.Sprintf("%d", officerAccountID), fmt.Sprintf("%f", amount), ""},
			{fmt.Sprintf("%d", 144), "", fmt.Sprintf("%f", amount)},
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

		return frid, err
	}

	var debits []models.DebitsPayable
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
			if *balance-p.CapitalPayable >= 0 {
				payments = append(payments, models.ContractPayment{ContractInstallmentID: p.InstallmentID, ContractReceiptID: rid, Amount: p.CapitalPayable})
				*balance = math.Round((*balance-p.CapitalPayable)*100) / 100
			} else {
				payments = append(payments, models.ContractPayment{ContractInstallmentID: p.InstallmentID, ContractReceiptID: rid, Amount: *balance})
				*balance = 0
			}
		}
	}
	return payments
}

func issueLKAS17Receipt(tx *sql.Tx, userID, cid int, amount float64, notes, dueDate string) (int64, error) {
	fBalance := amount

	rid, err := mysequel.Insert(mysequel.Table{
		TableName: "contract_receipt",
		Columns:   []string{"lcas_17", "user_id", "contract_id", "datetime", "amount", "notes", "due_date"},
		Vals:      []interface{}{1, userID, cid, time.Now().Format("2006-01-02 15:04:05"), amount, notes, dueDate},
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	// today := time.Now().Format("2006-01-02")
	today := "2021-11-01"

	var fArrears []models.ContractPayable
	err = mysequel.QueryToStructs(&fArrears, tx, queries.FINANCIAL_OVERDUE_INSTALLMENTS_LKAS_17, cid, today)
	if err != nil {
		return 0, err
	}

	var fInts []models.ContractPayment
	var fCaps []models.ContractPayment

	if fBalance != 0 {
		fInts = payments("I", rid, &fBalance, fArrears, fInts)
	}

	if fBalance != 0 {
		fCaps = payments("C", rid, &fBalance, fArrears, fCaps)
	}

	if fBalance != 0 {
		var fUpcoming []models.ContractPayable
		err = mysequel.QueryToStructs(&fUpcoming, tx, queries.FINANCIAL_UPCOMING_INSTALLMENTS_LKAS_17, cid, today)
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

	financialInterestPaid := float64(0)
	for _, intPayment := range fInts {
		financialInterestPaid = financialInterestPaid + intPayment.Amount
		_, err := mysequel.Insert(mysequel.Table{
			TableName: "contract_financial_payment",
			Columns:   []string{"contract_payment_type_id", "contract_schedule_id", "contract_receipt_id", "amount"},
			Vals:      []interface{}{2, intPayment.ContractInstallmentID, intPayment.ContractReceiptID, intPayment.Amount},
			Tx:        tx,
		})
		if err != nil {
			return 0, err
		}

		_, err = tx.Exec("UPDATE contract_schedule SET interest_paid = interest_paid + ? WHERE id = ?", intPayment.Amount, intPayment.ContractInstallmentID)
		if err != nil {
			return 0, err
		}
	}

	financialCapitalPaid := float64(0)
	for _, capPayment := range fCaps {
		financialCapitalPaid = financialCapitalPaid + capPayment.Amount
		_, err := mysequel.Insert(mysequel.Table{
			TableName: "contract_financial_payment",
			Columns:   []string{"contract_payment_type_id", "contract_schedule_id", "contract_receipt_id", "amount"},
			Vals:      []interface{}{1, capPayment.ContractInstallmentID, capPayment.ContractReceiptID, capPayment.Amount},
			Tx:        tx,
		})
		if err != nil {
			return 0, err
		}

		_, err = tx.Exec("UPDATE contract_schedule SET capital_paid = capital_paid + ? WHERE id = ?", capPayment.Amount, capPayment.ContractInstallmentID)
		if err != nil {
			return 0, err
		}
	}

	mBalance := amount

	var mArrears []models.ContractPayable
	err = mysequel.QueryToStructs(&mArrears, tx, queries.MARKETED_OVERDUE_INSTALLMENTS_LKAS_17, cid, today)
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
		err = mysequel.QueryToStructs(&mUpcoming, tx, queries.MARKETED_UPCOMING_INSTALLMENTS_LKAS_17, cid, today)
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

	_, err = tx.Exec("UPDATE contract_financial SET capital_paid = capital_paid + ?, interest_paid = interest_paid + ?, capital_arrears = capital_arrears - ?, interest_arrears = interest_arrears - ?", financialCapitalPaid, financialInterestPaid, financialCapitalPaid, financialInterestPaid)
	if err != nil {
		return 0, err
	}

	return rid, err
}
