package mysql

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"time"

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
		RCols:     rparams,
		OCols:     oparams,
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

	_, err = msql.Insert(msql.Table{
		TableName: "contract_state_transition",
		Columns:   []string{"to_contract_state_id", "transition_date"},
		Vals:      []string{strconv.FormatInt(sid, 10), time.Now().Format("2006-01-02 15:04:05")},
		Tx:        tx,
	})

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
		SELECT C.contract_state_id, D.id as document_id, D.name as document_name, CSD.source , CSD.s3bucket, CSD.s3region, SD.compulsory
		FROM contract C 
		LEFT JOIN contract_state CS ON C.contract_state_id = CS.id 
		LEFT JOIN state_document SD ON CS.state_id = SD.state_id
		LEFT JOIN document D ON SD.document_id = D.id 
		LEFT JOIN contract_state_document CSD ON D.id = CSD.document_id AND CSD.contract_state_id = C.contract_state_id
		WHERE C.id = ? AND SD.state_id = CS.state_id`, cid)
	if err != nil {
		return nil, err
	}

	var workDocuments []models.WorkDocument
	for results.Next() {
		var wd models.WorkDocument
		err = results.Scan(&wd.ContractStateID, &wd.DocumentID, &wd.DocumentName, &wd.Source, &wd.S3Bucket, &wd.S3Region, &wd.Compulsory)
		if err != nil {
			return nil, err
		}
		workDocuments = append(workDocuments, wd)
	}

	return workDocuments, nil
}

func (m *ContractModel) WorkQuestions(cid int) ([]models.WorkQuestion, error) {
	results, err := m.DB.Query(`
		SELECT C.contract_state_id, Q.id as question_id, Q.name as question, CSQA.answer, SD.compulsory
		FROM contract C 
		LEFT JOIN contract_state CS ON C.contract_state_id = CS.id 
		LEFT JOIN state_question SD ON CS.state_id = SD.state_id
		LEFT JOIN question Q ON SD.question_id = Q.id
		LEFT JOIN contract_state_question_answer CSQA ON Q.id = CSQA.question_id AND CSQA.contract_state_id = C.contract_state_id
		WHERE C.id = ? AND SD.state_id = CS.state_id`, cid)
	if err != nil {
		return nil, err
	}

	var workQuestions []models.WorkQuestion
	for results.Next() {
		var wq models.WorkQuestion
		err = results.Scan(&wq.ContractStateID, &wq.QuestionID, &wq.Question, &wq.Answer, &wq.Compulsory)
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

func (m *ContractModel) ContractDetail(cid int) (models.ContractDetail, error) {
	var detail models.ContractDetail
	err := m.DB.QueryRow(`
		SELECT C.id, S.name AS contract_state, CB.name as contract_batch, M.name AS model_name, C.chassis_number, C.customer_nic, C.customer_name, C.customer_address, C.customer_contact, C.liaison_name, C.liaison_contact, C.price, C.downpayment
		FROM contract C 
		LEFT JOIN contract_state CS ON CS.id = C.contract_state_id
		LEFT JOIN state S ON S.id = CS.state_id
		LEFT JOIN model M ON M.id = C.model_id
		LEFT JOIN contract_batch CB ON CB.id = C.contract_batch_id
		WHERE C.id = ?`, cid).Scan(&detail.ID, &detail.ContractState, &detail.ContractBatch, &detail.ModelName, &detail.ChassisNumber, &detail.CustomerNic, &detail.CustomerName, &detail.CustomerAddress, &detail.CustomerContact, &detail.LiaisonName, &detail.LiaisonContact, &detail.Price, &detail.Downpayment)
	if err != nil {
		return models.ContractDetail{}, err
	}

	return detail, nil
}

func (m *ContractModel) ContractTransionableStates(cid int) ([]models.Dropdown, error) {
	results, err := m.DB.Query(`
		SELECT TS.transitionable_state_id AS id, S.name AS name
		FROM transitionable_states TS
		LEFT JOIN state S ON S.id = TS.transitionable_state_id
		WHERE TS.state_id = (
			SELECT CS.state_id
			FROM contract C
			LEFT JOIN contract_state CS ON CS.id = C.contract_state_id
			WHERE C.id = ?
		)`, cid)
	if err != nil {
		return nil, err
	}

	var states []models.Dropdown
	for results.Next() {
		var s models.Dropdown
		err = results.Scan(&s.ID, &s.Name)
		if err != nil {
			return nil, err
		}
		states = append(states, s)
	}

	return states, nil
}

func (m *ContractModel) RejectedRequests(cid int) ([]models.RejectedRequest, error) {
	results, err := m.DB.Query(`
	SELECT R.id, U.name as user, R.note
		FROM request R
        LEFT JOIN user U ON U.id = R.user_id
		WHERE R.contract_state_id = (
			SELECT C.contract_state_id
			FROM contract C
			WHERE C.id = ?
		) AND R.approved = 0
	`, cid)
	if err != nil {
		return nil, err
	}

	var requests []models.RejectedRequest
	for results.Next() {
		var r models.RejectedRequest
		err = results.Scan(&r.ID, &r.User, &r.Note)
		if err != nil {
			return nil, err
		}
		requests = append(requests, r)
	}

	return requests, nil
}

func (m *ContractModel) CurrentRequetExists(cid int) (bool, error) {
	result, err := m.DB.Query(`
		SELECT R.id
		FROM request R
		WHERE R.contract_state_id = (
			SELECT C.contract_state_id
			FROM contract C
			WHERE C.id = ?
		) AND R.approved IS NULL
	`, cid)
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

	tcsid, err := msql.Insert(msql.FormTable{
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

	rid, err := msql.Insert(msql.Table{
		TableName: "request",
		Columns:   []string{"contract_state_id", "to_contract_state_id", "user_id", "datetime", "remarks"},
		Vals:      []string{strconv.FormatInt(int64(cs.ID), 10), strconv.FormatInt(tcsid, 10), form.Get("user_id"), time.Now().Format("2006-01-02 15:04:05"), form.Get("remarks")},
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	return rid, nil
}

func (m *ContractModel) Requests(user int) ([]models.Request, error) {
	results, err := m.DB.Query(`
		SELECT R.id AS request_id, C.id as contract_id, R.remarks, C.customer_name, S.name AS contract_state, S1.name AS to_contract_state, U.name AS requested_by, R.datetime AS requested_on
		FROM request R
		LEFT JOIN contract_state CS ON CS.id = R.contract_state_id
		LEFT JOIN contract_state CS1 ON CS1.id = R.to_contract_state_id
		LEFT JOIN state S ON S.id = CS.state_id
		LEFT JOIN state S1 ON S1.id = CS1.state_id
		LEFT JOIN user U ON U.id = R.user_id
		LEFT JOIN contract C ON CS.contract_id = C.id
		WHERE R.approved IS NULL`)
	if err != nil {
		return nil, err
	}

	var requests []models.Request
	for results.Next() {
		var r models.Request
		err = results.Scan(&r.RequestID, &r.ContractID, &r.Remarks, &r.CustomerName, &r.ContractState, &r.ToContractState, &r.RequestedBy, &r.RequestedOn)
		if err != nil {
			return nil, err
		}
		requests = append(requests, r)
	}

	return requests, nil
}

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
	_, err = msql.Update(msql.UpdateTable{
		Table: msql.Table{TableName: "request",
			Columns: []string{"approved", "approved_by", "approved_on", "note"},
			Vals:    []string{action, strconv.FormatInt(int64(user), 10), t, note},
			Tx:      tx},
		WColumns: []string{"id"},
		WVals:    []string{strconv.FormatInt(int64(request), 10)},
	})
	if err != nil {
		return 0, err
	}

	fmt.Println(action == "0")
	if action == "0" {
		return 1, nil
	}

	var r models.RequestRaw
	err = tx.QueryRow(`
		SELECT R.id, R.contract_state_id, R.to_contract_state_id, CS.contract_id
		FROM request R 
		LEFT JOIN contract_state CS ON CS.id = R.contract_state_id
		WHERE R.id = ?`, request).Scan(&r.ID, &r.ContractStateID, &r.ToContractStateID, &r.ContractID)
	if err != nil {
		return 0, err
	}

	_, err = msql.Insert(msql.Table{
		TableName: "contract_state_transition",
		Columns:   []string{"from_contract_state_id", "to_contract_state_id", "request_id", "transition_date"},
		Vals:      []string{strconv.FormatInt(int64(r.ContractStateID), 10), strconv.FormatInt(int64(r.ToContractStateID), 10), strconv.FormatInt(int64(request), 10), t},
		Tx:        tx,
	})
	if err != nil {
		return 0, err
	}

	c, err := msql.Update(msql.UpdateTable{
		Table: msql.Table{TableName: "contract",
			Columns: []string{"contract_state_id"},
			Vals:    []string{strconv.FormatInt(int64(r.ToContractStateID), 10)},
			Tx:      tx},
		WColumns: []string{"id"},
		WVals:    []string{strconv.FormatInt(int64(r.ContractID), 10)},
	})
	if err != nil {
		return 0, err
	}

	return c, nil
}
