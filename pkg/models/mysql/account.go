package mysql

import (
	"database/sql"
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
