package mysql

import (
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
		return 0, nil
	}

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
	err = mysequel.QueryToStructs(&payables, tx, queries.OVERDUE_INSTALLMENTS, cid, time.Now().Format("2006-01-02"))
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
		err = mysequel.QueryToStructs(&upcoming, tx, queries.UPCOMING_INSTALLMENTS, cid, time.Now().Format("2006-01-02"))
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
		{fmt.Sprintf("%d", officerAccountID), fmt.Sprintf("%f", amount), ""},
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

	resp, err := http.Get(requestURL)
	if err != nil {
		return rid, nil
	}

	defer resp.Body.Close()

	return rid, nil
}
