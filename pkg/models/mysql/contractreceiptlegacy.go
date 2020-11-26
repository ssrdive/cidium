package mysql

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/ssrdive/cidium/pkg/models"
	"github.com/ssrdive/cidium/pkg/sql/queries"
	"github.com/ssrdive/mysequel"
)

// LegacyReceipt issues a legacy receipt
func (m *ContractModel) LegacyReceipt(userID, cid int, amount float64, notes string) (int64, error) {
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

	var diUpdates []models.ContractDefaultInterestUpdate
	var diLogs []models.ContractDefaultInterestChangeHistory
	var diPayments []models.ContractPayment
	var intPayments []models.ContractPayment
	var capPayments []models.ContractPayment

	balance := amount

	rid, err := mysequel.Insert(mysequel.Table{
		TableName: "contract_receipt",
		Columns:   []string{"user_id", "contract_id", "datetime", "amount", "notes"},
		Vals:      []interface{}{userID, cid, time.Now().Format("2006-01-02 15:04:05"), amount, notes},
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

	return rid, nil
}
