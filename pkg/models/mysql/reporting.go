package mysql

import (
	"database/sql"

	"github.com/ssrdive/cidium/pkg/models"
	"github.com/ssrdive/cidium/pkg/sql/queries"
	"github.com/ssrdive/mysequel"
)

// ReportingModel struct holds database instance
type ReportingModel struct {
	DB *sql.DB
}

// ArrearsAnalysis returns achievement summary
func (m *ReportingModel) ArrearsAnalysis(startDate, endDate string) ([]models.ArrearsAnalysisItem, error) {
	var res []models.ArrearsAnalysisItem
	err := mysequel.QueryToStructs(&res, m.DB, queries.ARREARS_ANALYSIS(startDate, endDate))
	if err != nil {
		return nil, err
	}

	return res, nil
}

// AchievementSummary returns achievement summary
func (m *ReportingModel) AchievementSummary() ([]models.AchievementSummaryItem, error) {
	var res []models.AchievementSummaryItem
	err := mysequel.QueryToStructs(&res, m.DB, queries.ACHIEVEMENT_SUMMARY)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// CreditAchievementSummary returns achievement summary
func (m *ReportingModel) CreditAchievementSummary() ([]models.AchievementSummaryItem, error) {
	var res []models.AchievementSummaryItem
	err := mysequel.QueryToStructs(&res, m.DB, queries.CREDIT_ACHIEVEMENT_SUMMARY)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ReceiptSearch returns receipt search
func (m *ReportingModel) ReceiptSearch(startDate, endDate, officer string) ([]models.ReceiptSearchItem, error) {
	o := mysequel.NewNullString(officer)

	var res []models.ReceiptSearchItem
	err := mysequel.QueryToStructs(&res, m.DB, queries.RECEIPT_SEARCH, o, o, startDate, endDate, o, o, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return res, nil
}
