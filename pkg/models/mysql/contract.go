package mysql

import (
	"database/sql"
	"net/url"
	"strconv"

	"github.com/ssrdive/cidium/pkg/models"
	msql "github.com/ssrdive/cidium/pkg/sql"
)

// ContractModel struct holds database instance
type ContractModel struct {
	DB *sql.DB
}

// Insert creates a new contract
func (m *ContractModel) Insert(rparams, oparams []string, form url.Values) (int64, error) {
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
		RCols:     oparams,
		OCols:     rparams,
		Form:      form,
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	sid, err := msql.Insert(msql.Table{
		TableName: "contract_state",
		Columns:   []string{"contract_id", "state_id"},
		Vals:      []string{strconv.FormatInt(cid, 10), strconv.FormatInt(int64(1), 10)},
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	_, err = msql.Update(msql.UpdateTable{
		Table: msql.Table{
			TableName: "contract",
			Columns:   []string{"contract_state_id"},
			Vals:      []string{strconv.FormatInt(sid, 10)},
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

func (m *ContractModel) WorkDocuments(cid int) ([]models.WorkDocument, error) {
	results, err := m.DB.Query(`
		SELECT C.contract_state_id, D.id as document_id, D.name as document_name, CSD.source , CSD.s3bucket, CSD.s3region
		FROM contract C 
		LEFT JOIN contract_state CS ON C.contract_state_id = CS.id 
		LEFT JOIN state_document SD ON CS.state_id = CS.state_id = SD.state_id 
		LEFT JOIN document D ON SD.document_id = D.id 
		LEFT JOIN contract_state_document CSD ON D.id = CSD.document_id 
		WHERE C.id = ?`, cid)
	if err != nil {
		return nil, err
	}

	var workDocuments []models.WorkDocument
	for results.Next() {
		var wd models.WorkDocument
		err = results.Scan(&wd.ContractStateID, &wd.DocumentID, &wd.DocumentName, &wd.Source, &wd.S3Bucket, &wd.S3Region)
		if err != nil {
			return nil, err
		}
		workDocuments = append(workDocuments, wd)
	}

	return workDocuments, nil
}

func (m *ContractModel) WorkQuestions(cid int) ([]models.WorkQuestion, error) {
	results, err := m.DB.Query(`
		SELECT C.contract_state_id, Q.id as question_id, Q.name as question, CSQA.answer 
		FROM contract C 
		LEFT JOIN contract_state CS ON C.contract_state_id = CS.id 
		LEFT JOIN state_question SD ON CS.state_id = CS.state_id = SD.state_id 
		LEFT JOIN question Q ON SD.question_id = Q.id
		LEFT JOIN contract_state_question_answer CSQA ON Q.id = CSQA.question_id
		WHERE C.id = ?`, cid)
	if err != nil {
		return nil, err
	}

	var workQuestions []models.WorkQuestion
	for results.Next() {
		var wq models.WorkQuestion
		err = results.Scan(&wq.ContractStateID, &wq.QuestionID, &wq.Question, &wq.Answer)
		if err != nil {
			return nil, err
		}
		workQuestions = append(workQuestions, wq)
	}

	return workQuestions, nil
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
		RCols:     oparams,
		OCols:     rparams,
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
		RCols:     oparams,
		OCols:     rparams,
		Form:      form,
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	return cid, nil
}
