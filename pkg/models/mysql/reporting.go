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

// AchievementSummary returns achievement summary
func (m *ReportingModel) AchievementSummary() ([]models.AchievementSummaryItem, error) {
	var res []models.AchievementSummaryItem
	err := mysequel.QueryToStructs(&res, m.DB, queries.ACHIEVEMENT_SUMMARY)
	if err != nil {
		return nil, err
	}

	return res, nil
}
