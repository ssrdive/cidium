package mysql

import (
	"database/sql"
	"encoding/json"
	"net/url"
	"time"

	"github.com/ssrdive/cidium/pkg/models"
	msql "github.com/ssrdive/cidium/pkg/sql"
	"github.com/ssrdive/cidium/pkg/sql/queries"
)

type AccountModel struct {
	DB *sql.DB
}

func (m *AccountModel) CreateAccount(rparams, oparams []string, form url.Values) (int64, error) {
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

	form.Set("datetime", time.Now().Format("2006-01-02 15:04:05"))
	cid, err := msql.Insert(msql.FormTable{
		TableName: "account",
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

func (m *AccountModel) CreateCategory(rparams, oparams []string, form url.Values) (int64, error) {
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

	form.Set("datetime", time.Now().Format("2006-01-02 15:04:05"))
	cid, err := msql.Insert(msql.FormTable{
		TableName: "account_category",
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

func (m *AccountModel) TrialBalance() ([]models.TrialEntry, error) {
	results, err := m.DB.Query(queries.TRIAL_BALANCE)
	if err != nil {
		return nil, err
	}

	var requests []models.TrialEntry
	for results.Next() {
		var r models.TrialEntry
		err = results.Scan(&r.ID, &r.AccountID, &r.AccountName, &r.Debit, &r.Credit, &r.Balance)
		if err != nil {
			return nil, err
		}
		requests = append(requests, r)
	}

	return requests, nil
}

func (m *AccountModel) ChartOfAccounts() ([]models.ChartOfAccount, error) {
	results, err := m.DB.Query(queries.CHART_OF_ACCOUNTS)
	if err != nil {
		return nil, err
	}

	var requests []models.ChartOfAccount
	for results.Next() {
		var r models.ChartOfAccount
		err = results.Scan(&r.MainAccountID, &r.MainAccount, &r.SubAccountID, &r.SubAccount, &r.AccountCategoryID, &r.AccountCategory, &r.AccountID, &r.AccountName)
		if err != nil {
			return nil, err
		}
		requests = append(requests, r)
	}

	return requests, nil
}

func (m *AccountModel) PaymentVoucher(user_id, posting_date, from_account_id, amount, entries, remark, due_date, check_number string) (int64, error) {
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

	var paymentVoucher []models.PaymentVoucherEntry
	json.Unmarshal([]byte(entries), &paymentVoucher)

	tid, err := msql.Insert(msql.Table{
		TableName: "transaction",
		Columns:   []string{"user_id", "datetime", "posting_date", "remark"},
		Vals:      []interface{}{user_id, time.Now().Format("2006-01-02 15:04:05"), posting_date, remark},
		Tx:        tx,
	})
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	_, err = msql.Insert(msql.Table{
		TableName: "payment_voucher",
		Columns:   []string{"transaction_id", "due_date", "check_number"},
		Vals:      []interface{}{tid, due_date, check_number},
		Tx:        tx,
	})
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	_, err = msql.Insert(msql.Table{
		TableName: "account_transaction",
		Columns:   []string{"transaction_id", "account_id", "type", "amount"},
		Vals:      []interface{}{tid, from_account_id, "CR", amount},
		Tx:        tx,
	})
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	for _, entry := range paymentVoucher {
		_, err := msql.Insert(msql.Table{
			TableName: "account_transaction",
			Columns:   []string{"transaction_id", "account_id", "type", "amount"},
			Vals:      []interface{}{tid, entry.Account, "DR", entry.Amount},
			Tx:        tx,
		})
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}
	return tid, nil
}

func (m *AccountModel) Deposit(user_id, posting_date, to_account_id, amount, entries, remark string) (int64, error) {
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

	var paymentVoucher []models.PaymentVoucherEntry
	json.Unmarshal([]byte(entries), &paymentVoucher)

	tid, err := msql.Insert(msql.Table{
		TableName: "transaction",
		Columns:   []string{"user_id", "datetime", "posting_date", "remark"},
		Vals:      []interface{}{user_id, time.Now().Format("2006-01-02 15:04:05"), posting_date, remark},
		Tx:        tx,
	})
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	_, err = msql.Insert(msql.Table{
		TableName: "deposit",
		Columns:   []string{"transaction_id"},
		Vals:      []interface{}{tid},
		Tx:        tx,
	})
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	_, err = msql.Insert(msql.Table{
		TableName: "account_transaction",
		Columns:   []string{"transaction_id", "account_id", "type", "amount"},
		Vals:      []interface{}{tid, to_account_id, "DR", amount},
		Tx:        tx,
	})
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	for _, entry := range paymentVoucher {
		_, err := msql.Insert(msql.Table{
			TableName: "account_transaction",
			Columns:   []string{"transaction_id", "account_id", "type", "amount"},
			Vals:      []interface{}{tid, entry.Account, "CR", entry.Amount},
			Tx:        tx,
		})
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	}
	return tid, nil
}

func (m *AccountModel) JournalEntry(user_id, posting_date, remark, entries string) (int64, error) {
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

	var journalEntries []models.JournalEntry
	json.Unmarshal([]byte(entries), &journalEntries)

	tid, err := msql.Insert(msql.Table{
		TableName: "transaction",
		Columns:   []string{"user_id", "datetime", "posting_date", "remark"},
		Vals:      []interface{}{user_id, time.Now().Format("2006-01-02 15:04:05"), posting_date, remark},
		Tx:        tx,
	})
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	for _, entry := range journalEntries {
		if len(entry.Debit) != 0 {
			_, err := msql.Insert(msql.Table{
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
		if len(entry.Credit) != 0 {
			_, err := msql.Insert(msql.Table{
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
	return tid, nil
}

func (m *AccountModel) Transaction(aid int) ([]models.Transaction, error) {
	results, err := m.DB.Query(queries.TRANSACTION, aid)
	if err != nil {
		return nil, err
	}

	var transaction []models.Transaction
	for results.Next() {
		var t models.Transaction
		err = results.Scan(&t.TransactionID, &t.AccountID, &t.AccountID2, &t.AccountName, &t.Type, &t.Amount)
		if err != nil {
			return nil, err
		}
		transaction = append(transaction, t)
	}

	return transaction, nil
}

func (m *AccountModel) Ledger(aid int) ([]models.LedgerEntry, error) {
	results, err := m.DB.Query(queries.ACCOUNT_LEDGER, aid)
	if err != nil {
		return nil, err
	}

	var ledger []models.LedgerEntry
	for results.Next() {
		var l models.LedgerEntry
		err = results.Scan(&l.Name, &l.TransactionID, &l.PostingDate, &l.Amount, &l.Type, &l.Remark)
		if err != nil {
			return nil, err
		}
		ledger = append(ledger, l)
	}

	return ledger, nil
}
