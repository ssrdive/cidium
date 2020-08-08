package queries

import "fmt"

const STATE_ID_FROM_STATE = `
	SELECT S.id FROM state S WHERE S.name = ?`

const WORK_DOCUMENTS = `
	SELECT C.contract_state_id, D.id as document_id, D.name as document_name, CSD.id, CSD.source , CSD.s3bucket, CSD.s3region, SD.compulsory 
	FROM state_document SD LEFT JOIN document D ON D.id = SD.document_id 
	LEFT JOIN contract_state CS ON CS.state_id = SD.state_id 
	LEFT JOIN contract_state_document CSD ON CSD.contract_state_id = CS.id AND CSD.document_id = SD.document_id AND CSD.deleted = 0 
	LEFT JOIN contract C ON C.contract_state_id = CS.id 
	WHERE C.id = ?`

const WORK_QUESTIONS = `
	SELECT C.contract_state_id, Q.id as question_id, Q.name as question, CSQA.id, CSQA.answer, SQ.compulsory
	FROM state_question SQ LEFT JOIN question Q ON Q.id = SQ.question_id 
	LEFT JOIN contract_state CS ON CS.state_id = SQ.state_id 
	LEFT JOIN contract_state_question_answer CSQA ON CSQA.contract_state_id = CS.id AND CSQA.question_id = SQ.question_id AND CSQA.deleted = 0 
	LEFT JOIN contract C ON C.contract_state_id = CS.id 
	WHERE C.id = ?`

const QUESTIONS = `
	SELECT Q.name as question, CSQA.answer
	FROM contract_state_question_answer CSQA
	LEFT JOIN contract_state CS ON CS.id = CSQA.contract_state_id
	LEFT JOIN question Q ON Q.id = CSQA.question_id
	WHERE CS.contract_id = ? AND deleted = 0`

const DOCUMENTS = `
	SELECT D.name as document, CSD.s3region, CSD.s3bucket, CSD.source
	FROM contract_state_document CSD
	LEFT JOIN contract_state CS ON CS.id = CSD.contract_state_id
	LEFT JOIN document D ON D.id = CSD.document_id
	WHERE CS.contract_id = ? AND deleted = 0`

const HISTORY = `
	SELECT S.name as from_state, S2.name as to_state, CST.transition_date
	FROM contract_state_transition CST 
	LEFT JOIN contract_state CS ON CS.id = CST.from_contract_state_id
	LEFT JOIN contract_state CS2 ON CS2.id = CST.to_contract_state_id
	LEFT JOIN state S ON S.id = CS.state_id
	LEFT JOIN state S2 ON S2.id = CS2.state_id
	WHERE CS2.contract_id = ?`

const REJECTED_REQUESTS = `
	SELECT R.id, U.name as user, R.note
		FROM request R
		LEFT JOIN user U ON U.id = R.user_id
		WHERE R.contract_state_id = (
			SELECT C.contract_state_id
			FROM contract C
			WHERE C.id = ?
		) AND R.approved = 0`

const CURRENT_REQUEST_EXISTS = `
	SELECT R.id
	FROM request R
	WHERE R.contract_state_id = (
		SELECT C.contract_state_id
		FROM contract C
		WHERE C.id = ?
	) AND R.approved IS NULL`

const REQUESTS = `
	SELECT R.id AS request_id, C.id as contract_id, R.remarks, C.customer_name, S.name AS contract_state, S1.name AS to_contract_state, U.name AS requested_by, R.datetime AS requested_on
	FROM request R
	LEFT JOIN contract_state CS ON CS.id = R.contract_state_id
	LEFT JOIN contract_state CS1 ON CS1.id = R.to_contract_state_id
	LEFT JOIN state S ON S.id = CS.state_id
	LEFT JOIN state S1 ON S1.id = CS1.state_id
	LEFT JOIN user U ON U.id = R.user_id
	LEFT JOIN contract C ON CS.contract_id = C.id
	WHERE R.approved IS NULL`

const REQUEST_NAME = `
	SELECT R.id, S.name
	FROM request R
	LEFT JOIN contract_state CS ON CS.id = R.to_contract_state_id
	LEFT JOIN state S ON S.id = CS.state_id
	WHERE R.id = ?`

const PARAMS_FOR_CONTRACT_INITIATION = `
	SELECT Q.name as id, CSQA.answer FROM contract_state_question_answer CSQA LEFT JOIN contract_state CS ON CS.id = CSQA.contract_state_id LEFT JOIN contract C ON C.id = CS.contract_id LEFT JOIN question Q ON Q.id = CSQA.question_id WHERE Q.name IN ('Capital', 'Interest Rate', 'Interest Method', 'Installments', 'Installment Interval', 'Initiation Date') AND CSQA.deleted = 0 AND C.id = ( SELECT CS.contract_id FROM request R LEFT JOIN contract_state CS ON CS.id = R.to_contract_state_id WHERE R.id = ? )`

const PARAMS_FOR_CREDIT_WORTHINESS_APPROVAL = `
	SELECT C.id, C.customer_name, C.liaison_contact
	FROM contract C 
	WHERE C.id = (SELECT CS.contract_id FROM request R LEFT JOIN contract_state CS ON CS.id = R.to_contract_state_id WHERE R.id = ?)
`

const CONTRACT_ID_FROM_REUQEST = `
	SELECT CS.contract_id AS id FROM request R LEFT JOIN contract_state CS ON CS.id = R.to_contract_state_id WHERE R.id = ?`

const REQUEST_RAW = `
	SELECT R.id, R.contract_state_id, R.to_contract_state_id, CS.contract_id
	FROM request R 
	LEFT JOIN contract_state CS ON CS.id = R.contract_state_id
	WHERE R.id = ?`

const EXPIRED_COMMITMENTS = `
	SELECT CM.contract_id, DATEDIFF(CM.due_date, NOW()) AS due_in, CM.text
	FROM contract_commitment CM
	WHERE CM.fulfilled IS NULL AND CM.commitment = 1 AND DATEDIFF(CM.due_date, NOW()) <= 0`

const UPCOMING_COMMITMENTS = `
	SELECT CM.contract_id, DATEDIFF(CM.due_date, NOW()) AS due_in, CM.text
	FROM contract_commitment CM
	WHERE CM.fulfilled IS NULL AND CM.commitment = 1 AND DATEDIFF(CM.due_date, NOW()) > 0`

const EXPIRED_COMMITMENTS_BY_OFFICER = `
	SELECT CM.contract_id, DATEDIFF(CM.due_date, NOW()) AS due_in, CM.text
	FROM contract_commitment CM
	LEFT JOIN contract C ON C.id = CM.contract_id
	WHERE CM.fulfilled IS NULL AND CM.commitment = 1 AND DATEDIFF(CM.due_date, NOW()) <= 0 AND C.recovery_officer_id = ?`

const UPCOMING_COMMITMENTS_BY_OFFICER = `
	SELECT CM.contract_id, DATEDIFF(CM.due_date, NOW()) AS due_in, CM.text
	FROM contract_commitment CM
	LEFT JOIN contract C ON C.id = CM.contract_id
	WHERE CM.fulfilled IS NULL AND CM.commitment = 1 AND DATEDIFF(CM.due_date, NOW()) > 0 AND C.recovery_officer_id = ?`

const DEBITS = `
	SELECT CI.id as installment_id, CI.contract_id, COALESCE(CI.capital-COALESCE(SUM(CCP.amount), 0)) as capital_payable, COALESCE(CI.interest-COALESCE(SUM(CIP.amount), 0)) as interest_payable, CI.default_interest, CIT2.unearned_account_id, CIT2.income_account_id
	FROM contract_installment CI
	LEFT JOIN contract_installment_type CIT2 ON CIT2.id = CI.contract_installment_type_id
	LEFT JOIN (
		SELECT CIP.contract_installment_id, COALESCE(SUM(amount), 0) as amount
		FROM contract_interest_payment CIP
		GROUP BY CIP.contract_installment_id
	) CIP ON CIP.contract_installment_id = CI.id
	LEFT JOIN (
		SELECT CCP.contract_installment_id, COALESCE(SUM(amount), 0) as amount
		FROM contract_capital_payment CCP
		GROUP BY CCP.contract_installment_id
	) CCP ON CCP.contract_installment_id = CI.id
	LEFT JOIN contract_installment_type CIT ON CIT.id = CI.contract_installment_type_id
	WHERE CI.contract_id = ? AND CIT.di_chargable = 0
	GROUP BY CI.contract_id, CI.id, CI.capital, CI.interest, CI.default_interest
	ORDER BY CI.due_date ASC
	`

const OVERDUE_INSTALLMENTS = `
	SELECT CI.id as installment_id, CI.contract_id, COALESCE(CI.capital-COALESCE(SUM(CCP.amount), 0)) as capital_payable, COALESCE(CI.interest-COALESCE(SUM(CIP.amount), 0)) as interest_payable, CI.default_interest
	FROM contract_installment CI
	LEFT JOIN (
		SELECT CIP.contract_installment_id, COALESCE(SUM(amount), 0) as amount
		FROM contract_interest_payment CIP
		GROUP BY CIP.contract_installment_id
	) CIP ON CIP.contract_installment_id = CI.id
	LEFT JOIN (
		SELECT CCP.contract_installment_id, COALESCE(SUM(amount), 0) as amount
		FROM contract_capital_payment CCP
		GROUP BY CCP.contract_installment_id
	) CCP ON CCP.contract_installment_id = CI.id
	LEFT JOIN contract_installment_type CIT ON CIT.id = CI.contract_installment_type_id
	WHERE CI.contract_id = ? AND CIT.di_chargable = 1 AND CI.due_date < ?
	GROUP BY CI.contract_id, CI.id, CI.capital, CI.interest, CI.default_interest
	ORDER BY CI.due_date ASC
	`

const UPCOMING_INSTALLMENTS = `
	SELECT CI.id as installment_id, CI.contract_id, COALESCE(CI.capital-COALESCE(SUM(CCP.amount), 0)) as capital_payable, COALESCE(CI.interest-COALESCE(SUM(CIP.amount), 0)) as interest_payable, CI.default_interest
	FROM contract_installment CI
	LEFT JOIN (
		SELECT CIP.contract_installment_id, COALESCE(SUM(amount), 0) as amount
		FROM contract_interest_payment CIP
		GROUP BY CIP.contract_installment_id
	) CIP ON CIP.contract_installment_id = CI.id
	LEFT JOIN (
		SELECT CCP.contract_installment_id, COALESCE(SUM(amount), 0) as amount
		FROM contract_capital_payment CCP
		GROUP BY CCP.contract_installment_id
	) CCP ON CCP.contract_installment_id = CI.id
	LEFT JOIN contract_installment_type CIT ON CIT.id = CI.contract_installment_type_id
	WHERE CI.contract_id = ? AND CIT.di_chargable = 1 AND CI.due_date >= ?
	GROUP BY CI.contract_id, CI.id, CI.capital, CI.interest, CI.default_interest
	ORDER BY CI.due_date ASC
	`

const LEGACY_PAYMENTS = `
	SELECT CI.id as installment_id, CI.contract_id, COALESCE(CI.capital-COALESCE(SUM(CCP.amount), 0)) as capital_payable, COALESCE(CI.interest-COALESCE(SUM(CIP.amount), 0)) as interest_payable, CI.default_interest
	FROM contract_installment CI
	LEFT JOIN (
		SELECT CIP.contract_installment_id, COALESCE(SUM(amount), 0) as amount
		FROM contract_interest_payment CIP
		GROUP BY CIP.contract_installment_id
	) CIP ON CIP.contract_installment_id = CI.id
	LEFT JOIN (
		SELECT CCP.contract_installment_id, COALESCE(SUM(amount), 0) as amount
		FROM contract_capital_payment CCP
		GROUP BY CCP.contract_installment_id
	) CCP ON CCP.contract_installment_id = CI.id
	LEFT JOIN contract_installment_type CIT ON CIT.id = CI.contract_installment_type_id
	WHERE CI.contract_id = ? AND CIT.di_chargable = 1
	GROUP BY CI.contract_id, CI.id, CI.capital, CI.interest, CI.default_interest
	ORDER BY CI.due_date ASC
	`

const INSTALLMENT_INSTALLMENT_TYPE_ID = `
	SELECT CIT.id
	FROM contract_installment_type CIT
	WHERE CIT.name = 'Installment'`

const CONTRACT_DETAILS = `
	SELECT C.id, S.name AS state, CB.name AS contract_batch, M.name AS model, C.chassis_number, C.customer_name, C.customer_nic, C.customer_address, C.customer_contact, C.liaison_name, C.liaison_contact, C.price, C.downpayment, U.name AS recovery_officer, SUM(CASE WHEN (CI.due_date < NOW() AND CI.installment_paid < CI.installment) THEN CI.installment - CI.installment_paid ELSE 0 END) AS amount_pending, COALESCE(SUM(CI.installment-CI.installment_paid), 0) AS total_payable, COALESCE(SUM(CI.installment_paid), 0) AS total_paid, ( CASE WHEN (MAX(DATE(CR.datetime)) IS NULL AND MAX(DATE(CRL.legacy_payment_date)) IS NULL) THEN 'N/A' ELSE GREATEST(COALESCE(MAX(DATE(CR.datetime)), '1900-01-01'), COALESCE(MAX(DATE(CRL.legacy_payment_date)), '1900-01-01')) END ) as last_payment_date, COALESCE(ROUND((SUM(CASE WHEN (CI.due_date < NOW() AND CI.installment_paid < CI.installment) THEN CI.installment - CI.installment_paid ELSE 0 END))/(ROUND((COALESCE(SUM(CI.agreed_installment), 0))/(TIMESTAMPDIFF(MONTH, MIN(CI.due_date), MAX(CI.due_date))+TIMESTAMPDIFF(MONTH, MIN(CI.due_date), MIN(CI2.due_date))), 2)), 2), 'N/A') AS overdue_index
	FROM contract C
	LEFT JOIN contract_state CS ON CS.id = C.contract_state_id
	LEFT JOIN state S ON S.id = CS.state_id
	LEFT JOIN contract_batch CB ON CB.id = C.contract_batch_id
	LEFT JOIN model M ON C.model_id = M.id
	LEFT JOIN user U ON U.id = C.recovery_officer_id
	LEFT JOIN (SELECT CR.contract_id, MAX(CR.datetime) AS datetime FROM contract_receipt CR WHERE CR.legacy_payment_date IS NULL GROUP BY CR.contract_id) CR ON CR.contract_id = C.id
	LEFT JOIN (SELECT CI.contract_id, MIN(CI.due_date) AS due_date FROM contract_installment CI WHERE CI.due_date > (SELECT MIN(CI2.due_date) FROM contract_installment CI2 WHERE CI.contract_id = CI2.contract_id) GROUP BY CI.contract_id) CI2 ON CI2.contract_id = C.id
	LEFT JOIN (SELECT CRL.contract_id, MAX(CRL.legacy_payment_date) as legacy_payment_date FROM contract_receipt CRL GROUP BY CRL.contract_id) CRL ON CRL.contract_id = C.id
	LEFT JOIN (SELECT CI.id, CI.contract_id, CI.capital+CI.interest+CI.default_interest AS installment,CI.capital+CI.interest AS agreed_installment,SUM(COALESCE(CCP.amount, 0)+COALESCE(CIP.amount, 0)) AS installment_paid, COALESCE(SUM(CDIP.amount), 0) AS defalut_interest_paid, CI.due_date
	FROM contract_installment CI
	LEFT JOIN (
		SELECT CDIP.contract_installment_id, COALESCE(SUM(amount), 0) AS amount
		FROM contract_default_interest_payment CDIP
		GROUP BY CDIP.contract_installment_id
	) CDIP ON CDIP.contract_installment_id = CI.id
	LEFT JOIN (
		SELECT CIP.contract_installment_id, COALESCE(SUM(amount), 0) AS amount
		FROM contract_interest_payment CIP
		GROUP BY CIP.contract_installment_id
	) CIP ON CIP.contract_installment_id = CI.id
	LEFT JOIN (
		SELECT CCP.contract_installment_id, COALESCE(SUM(amount), 0) AS amount
		FROM contract_capital_payment CCP
		GROUP BY CCP.contract_installment_id
	) CCP ON CCP.contract_installment_id = CI.id
	GROUP BY CI.id, CI.contract_id, CI.capital, CI.interest, CI.interest, CI.default_interest, CI.due_date
	ORDER BY CI.due_date ASC) CI ON CI.contract_id = C.id
	WHERE C.id = ?
	GROUP BY C.id`

const CONTRACT_INSTALLMENTS = `
	SELECT CI.id, CIT.name AS installment_type, CI.capital+CI.interest+CI.default_interest AS installment,SUM(COALESCE(CCP.amount, 0)+COALESCE(CIP.amount, 0)) AS installment_paid, CI.due_date, DATEDIFF(CI.due_date, NOW()) AS due_in
	FROM contract_installment CI
	LEFT JOIN contract_installment_type CIT ON CIT.id = CI.contract_installment_type_id
	LEFT JOIN (
		SELECT CIP.contract_installment_id, COALESCE(SUM(amount), 0) AS amount
		FROM contract_interest_payment CIP
		LEFT JOIN contract_installment CI ON CI.id = CIP.contract_installment_id
		WHERE CI.contract_id = ?
		GROUP BY CIP.contract_installment_id
	) CIP ON CIP.contract_installment_id = CI.id
	LEFT JOIN (
		SELECT CCP.contract_installment_id, COALESCE(SUM(amount), 0) AS amount
		FROM contract_capital_payment CCP
		LEFT JOIN contract_installment CI ON CI.id = CCP.contract_installment_id
		WHERE CI.contract_id = ?
		GROUP BY CCP.contract_installment_id
	) CCP ON CCP.contract_installment_id = CI.id
	WHERE CI.contract_id = ?
	GROUP BY CI.id
	ORDER BY due_date ASC`

const CONTRACT_RECEIPTS = `
	SELECT CR.id, CR.datetime, CR.amount, CR.notes
	FROM contract_receipt CR
	WHERE CR.contract_id = ?
`

const CONTRACT_RECEIPTS_V2 = `
	SELECT CR.id, CR.datetime, CR.amount, CR.notes, CRT.name as type
	FROM contract_receipt CR
	LEFT JOIN contract_receipt_type CRT ON CRT.id = CR.contract_receipt_type_id
	WHERE CR.contract_id = ?
`

const CONTRACT_OFFICER_RECEIPTS = `
	SELECT CR.id, CR.datetime, CR.amount, CR.notes
	FROM contract_receipt CR
	WHERE CR.user_id = ? AND DATE(CR.datetime) = ?;
`

const CONTRACT_COMMITMENTS = `
	SELECT CM.id, U.name AS created_by, CM.created, CM.commitment, CM.fulfilled, DATEDIFF(CM.due_date, NOW()) AS due_in, CM.text, U2.name AS fulfilled_by, CM.fulfilled_on
	FROM contract_commitment CM
	LEFT JOIN user U ON U.id = CM.user_id
	LEFT JOIN user U2 ON U2.id = CM.fulfilled_by
	WHERE CM.contract_id = ?
	ORDER BY created DESC
`

const TRANSITIONABLE_STATES = `
	SELECT TS.transitionable_state_id AS id, S.name AS name
	FROM transitionable_states TS
	LEFT JOIN state S ON S.id = TS.transitionable_state_id
	WHERE TS.state_id = (
		SELECT CS.state_id
		FROM contract C
		LEFT JOIN contract_state CS ON CS.id = C.contract_state_id
		WHERE C.id = ?
	)`

const SEARCH_OLD = `
	SELECT C.id, C.agrivest, U.name as recovery_officer, S.name as state, M.name as model, C.chassis_number, C.customer_name, C.customer_address, C.customer_contact, SUM(CASE WHEN (CI.due_date < NOW() AND CI.installment_paid < CI.installment) THEN CI.installment - CI.installment_paid ELSE 0 END) as amount_pending, COALESCE(SUM(CI.installment-CI.installment_paid), 0) AS total_payable,  COALESCE(SUM(CI.agreed_installment), 0) AS total_agreement, COALESCE(SUM(CI.installment_paid), 0) AS total_paid, COALESCE(SUM(CI.defalut_interest_paid), 0) AS total_di_paid
	FROM contract C
	LEFT JOIN user U ON U.id = C.recovery_officer_id
	LEFT JOIN contract_state CS ON CS.id = C.contract_state_id
	LEFT JOIN state S ON S.id = CS.state_id
	LEFT JOIN model M ON C.model_id = M.id
	LEFT JOIN (SELECT CI.id, CI.contract_id, CI.capital+CI.interest+CI.default_interest AS installment, CI.capital+CI.interest AS agreed_installment, SUM(COALESCE(CCP.amount, 0)+COALESCE(CIP.amount, 0)) AS installment_paid, COALESCE(SUM(CDIP.amount), 0) as defalut_interest_paid, CI.due_date
	FROM contract_installment CI
	LEFT JOIN (
		SELECT CDIP.contract_installment_id, COALESCE(SUM(amount), 0) as amount
		FROM contract_default_interest_payment CDIP
		GROUP BY CDIP.contract_installment_id
	) CDIP ON CDIP.contract_installment_id = CI.id
	LEFT JOIN (
		SELECT CIP.contract_installment_id, COALESCE(SUM(amount), 0) as amount
		FROM contract_interest_payment CIP
		GROUP BY CIP.contract_installment_id
	) CIP ON CIP.contract_installment_id = CI.id
	LEFT JOIN (
		SELECT CCP.contract_installment_id, COALESCE(SUM(amount), 0) as amount
		FROM contract_capital_payment CCP
		GROUP BY CCP.contract_installment_id
	) CCP ON CCP.contract_installment_id = CI.id
	GROUP BY CI.id, CI.contract_id, CI.capital, CI.interest, CI.interest, CI.default_interest, CI.due_date
	ORDER BY CI.due_date ASC) CI ON CI.contract_id = C.id
	WHERE (? IS NULL OR CONCAT(C.id, C.customer_name, C.chassis_number) LIKE ?) AND (? IS NULL OR S.id = ?) AND (? IS NULL OR C.recovery_officer_id = ?) AND (? IS NULL OR C.contract_batch_id = ?)
	GROUP BY C.id`

const SEARCH = `
	SELECT C.id, C.agrivest, U.name as recovery_officer, S.name as state, M.name as model, CB.name as batch, C.chassis_number, C.customer_name, C.customer_address, C.customer_contact, SUM(CASE WHEN (CI.due_date < NOW() AND CI.installment_paid < CI.installment) THEN CI.installment - CI.installment_paid ELSE 0 END) as amount_pending, COALESCE(SUM(CI.installment-CI.installment_paid), 0) AS total_payable,  COALESCE(SUM(CI.agreed_installment), 0) AS total_agreement, COALESCE(SUM(CI.installment_paid), 0) AS total_paid, COALESCE(SUM(CI.defalut_interest_paid), 0) AS total_di_paid, 
	( CASE WHEN (MAX(DATE(CR.datetime)) IS NULL AND MAX(DATE(CRL.legacy_payment_date)) IS NULL) THEN 'N/A' ELSE GREATEST(COALESCE(MAX(DATE(CR.datetime)), '1900-01-01'), COALESCE(MAX(DATE(CRL.legacy_payment_date)), '1900-01-01')) END ) as last_payment_date
	FROM contract C
	LEFT JOIN user U ON U.id = C.recovery_officer_id
	LEFT JOIN contract_state CS ON CS.id = C.contract_state_id
	LEFT JOIN contract_batch CB ON CB.id = C.contract_batch_id
	LEFT JOIN state S ON S.id = CS.state_id
	LEFT JOIN model M ON C.model_id = M.id
	LEFT JOIN (SELECT CR.contract_id, MAX(CR.datetime) AS datetime FROM contract_receipt CR WHERE CR.legacy_payment_date IS NULL GROUP BY CR.contract_id) CR ON CR.contract_id = C.id
	LEFT JOIN (SELECT CRL.contract_id, MAX(CRL.legacy_payment_date) as legacy_payment_date FROM contract_receipt CRL GROUP BY CRL.contract_id) CRL ON CRL.contract_id = C.id
	LEFT JOIN (SELECT CI.id, CI.contract_id, CI.capital+CI.interest+CI.default_interest AS installment, CI.capital+CI.interest AS agreed_installment, SUM(COALESCE(CCP.amount, 0)+COALESCE(CIP.amount, 0)) AS installment_paid, COALESCE(SUM(CDIP.amount), 0) as defalut_interest_paid, CI.due_date
	FROM contract_installment CI
	LEFT JOIN (
		SELECT CDIP.contract_installment_id, COALESCE(SUM(amount), 0) as amount
		FROM contract_default_interest_payment CDIP
		GROUP BY CDIP.contract_installment_id
	) CDIP ON CDIP.contract_installment_id = CI.id
	LEFT JOIN (
		SELECT CIP.contract_installment_id, COALESCE(SUM(amount), 0) as amount
		FROM contract_interest_payment CIP
		GROUP BY CIP.contract_installment_id
	) CIP ON CIP.contract_installment_id = CI.id
	LEFT JOIN (
		SELECT CCP.contract_installment_id, COALESCE(SUM(amount), 0) as amount
		FROM contract_capital_payment CCP
		GROUP BY CCP.contract_installment_id
	) CCP ON CCP.contract_installment_id = CI.id
	GROUP BY CI.id, CI.contract_id, CI.capital, CI.interest, CI.interest, CI.default_interest, CI.due_date
	ORDER BY CI.due_date ASC) CI ON CI.contract_id = C.id
	WHERE (? IS NULL OR CONCAT(C.id, C.customer_name, C.chassis_number, C.customer_nic, C.customer_contact) LIKE ?) AND (? IS NULL OR S.id = ?) AND (? IS NULL OR C.recovery_officer_id = ?) AND (? IS NULL OR C.contract_batch_id = ?) AND C.non_performing = 0
	GROUP BY C.id`

const SEARCH_V2 = `
	SELECT C.id, C.agrivest, U.name as recovery_officer, S.name as state, DATEDIFF(NOW(), CST.transition_date) as in_state_for, M.name as model, CB.name as batch, C.chassis_number, C.customer_name, C.customer_address, C.customer_contact, SUM(CASE WHEN (CI.due_date < NOW() AND CI.installment_paid < CI.installment) THEN CI.installment - CI.installment_paid ELSE 0 END) as amount_pending, COALESCE(SUM(CI.installment-CI.installment_paid), 0) AS total_payable,  COALESCE(SUM(CI.agreed_installment), 0) AS total_agreement, COALESCE(SUM(CI.installment_paid), 0) AS total_paid, COALESCE(SUM(CI.defalut_interest_paid), 0) AS total_di_paid, ( CASE WHEN (MAX(DATE(CR.datetime)) IS NULL AND MAX(DATE(CRL.legacy_payment_date)) IS NULL) THEN 'N/A' ELSE GREATEST(COALESCE(MAX(DATE(CR.datetime)), '1900-01-01'), COALESCE(MAX(DATE(CRL.legacy_payment_date)), '1900-01-01')) END ) as last_payment_date, COALESCE(ROUND((SUM(CASE WHEN (CI.due_date < NOW() AND CI.installment_paid < CI.installment) THEN CI.installment - CI.installment_paid ELSE 0 END))/(ROUND((COALESCE(SUM(CI.agreed_installment), 0))/(TIMESTAMPDIFF(MONTH, MIN(CI.due_date), MAX(CI.due_date))+TIMESTAMPDIFF(MONTH, MIN(CI.due_date), MIN(CI2.due_date))), 2)), 2), 'N/A') AS overdue_index
	FROM contract C
	LEFT JOIN user U ON U.id = C.recovery_officer_id
	LEFT JOIN contract_state CS ON CS.id = C.contract_state_id
	LEFT JOIN contract_batch CB ON CB.id = C.contract_batch_id
	LEFT JOIN state S ON S.id = CS.state_id
	LEFT JOIN model M ON C.model_id = M.id
	LEFT JOIN contract_state_transition CST ON CST.to_contract_state_id = C.contract_state_id
	LEFT JOIN (SELECT CR.contract_id, MAX(CR.datetime) AS datetime FROM contract_receipt CR WHERE CR.legacy_payment_date IS NULL AND CR.is_customer_payment = 1 GROUP BY CR.contract_id) CR ON CR.contract_id = C.id
	LEFT JOIN (SELECT CRL.contract_id, MAX(CRL.legacy_payment_date) as legacy_payment_date FROM contract_receipt CRL WHERE CRL.is_customer_payment = 1 GROUP BY CRL.contract_id) CRL ON CRL.contract_id = C.id
	LEFT JOIN (SELECT CI.id, CI.contract_id, CI.capital+CI.interest+CI.default_interest AS installment, CI.capital+CI.interest AS agreed_installment, SUM(COALESCE(CCP.amount, 0)+COALESCE(CIP.amount, 0)) AS installment_paid, COALESCE(SUM(CDIP.amount), 0) as defalut_interest_paid, CI.due_date
	FROM contract_installment CI
	LEFT JOIN (
		SELECT CDIP.contract_installment_id, COALESCE(SUM(amount), 0) as amount
		FROM contract_default_interest_payment CDIP
		GROUP BY CDIP.contract_installment_id
	) CDIP ON CDIP.contract_installment_id = CI.id
	LEFT JOIN (
		SELECT CIP.contract_installment_id, COALESCE(SUM(amount), 0) as amount
		FROM contract_interest_payment CIP
		GROUP BY CIP.contract_installment_id
	) CIP ON CIP.contract_installment_id = CI.id
	LEFT JOIN (
		SELECT CCP.contract_installment_id, COALESCE(SUM(amount), 0) as amount
		FROM contract_capital_payment CCP
		GROUP BY CCP.contract_installment_id
	) CCP ON CCP.contract_installment_id = CI.id
	GROUP BY CI.id, CI.contract_id, CI.capital, CI.interest, CI.interest, CI.default_interest, CI.due_date
	ORDER BY CI.due_date ASC) CI ON CI.contract_id = C.id
	LEFT JOIN (SELECT CI.contract_id, MIN(CI.due_date) AS due_date FROM contract_installment CI WHERE CI.due_date > (SELECT MIN(CI2.due_date) FROM contract_installment CI2 WHERE CI.contract_id = CI2.contract_id) GROUP BY CI.contract_id) CI2 ON CI2.contract_id = C.id
	WHERE (? IS NULL OR CONCAT(C.id, C.customer_name, C.chassis_number, C.customer_nic, C.customer_contact) LIKE ?) AND (? IS NULL OR S.id = ?) AND (? IS NULL OR C.recovery_officer_id = ?) AND (? IS NULL OR C.contract_batch_id = ?) AND (? IS NULL OR C.non_performing = ?)
	GROUP BY C.id, in_state_for
`

func PERFORMANCE_REVIEW(startDate, endDate string) string {
	return fmt.Sprintf(`
	SELECT C.id, C.agrivest, U.name as recovery_officer, S.name as state, M.name as model, CB.name as batch, C.chassis_number, C.customer_name, C.customer_address, C.customer_contact, SUM(CASE WHEN (CI.due_date <= NOW() AND CI.installment_paid < CI.installment) THEN CI.installment - CI.installment_paid ELSE 0 END) as amount_pending, SUM(CASE WHEN (DATE(CI.due_date) <= '%s' AND CI.sd_installment_paid < CI.installment) THEN CI.installment - CI.sd_installment_paid ELSE 0 END) as start_amount_pending, SUM(CASE WHEN (DATE(CI.due_date) <= '%s' AND CI.ed_installment_paid < CI.installment) THEN CI.installment - CI.ed_installment_paid ELSE 0 END) as end_amount_pending, SUM(CASE WHEN (DATE(CI.due_date) BETWEEN '%s' AND '%s' AND CI.sd_installment_paid < CI.installment) THEN CI.installment - CI.sd_installment_paid ELSE 0 END) as start_between_amount_pending, SUM(CASE WHEN (DATE(CI.due_date) BETWEEN '%s' AND '%s' AND CI.ed_installment_paid < CI.installment) THEN CI.installment - CI.ed_installment_paid ELSE 0 END) as end_between_amount_pending, COALESCE(SUM(CI.installment-CI.installment_paid), 0) AS total_payable,  COALESCE(SUM(CI.agreed_installment), 0) AS total_agreement, COALESCE(SUM(CI.installment_paid), 0) AS total_paid, COALESCE(SUM(CI.defalut_interest_paid), 0) AS total_di_paid, ( CASE WHEN (MAX(DATE(CR.datetime)) IS NULL AND MAX(DATE(CRL.legacy_payment_date)) IS NULL) THEN 'N/A' ELSE GREATEST(COALESCE(MAX(DATE(CR.datetime)), '1900-01-01'), COALESCE(MAX(DATE(CRL.legacy_payment_date)), '1900-01-01')) END ) as last_payment_date, 
	COALESCE(ROUND((SUM(CASE WHEN (CI.due_date <= DATE('%s') AND CI.sd_installment_paid < CI.installment) THEN CI.installment - CI.sd_installment_paid ELSE 0 END))/(ROUND((COALESCE(SUM(CI.agreed_installment), 0))/(TIMESTAMPDIFF(MONTH, MIN(CI.due_date), MAX(CI.due_date))+TIMESTAMPDIFF(MONTH, MIN(CI.due_date), MIN(CI2.due_date))), 2)), 2), 'N/A') AS start_overdue_index,
	COALESCE(ROUND((SUM(CASE WHEN (CI.due_date <= DATE('%s') AND CI.ed_installment_paid < CI.installment) THEN CI.installment - CI.ed_installment_paid ELSE 0 END))/(ROUND((COALESCE(SUM(CI.agreed_installment), 0))/(TIMESTAMPDIFF(MONTH, MIN(CI.due_date), MAX(CI.due_date))+TIMESTAMPDIFF(MONTH, MIN(CI.due_date), MIN(CI2.due_date))), 2)), 2), 'N/A') AS end_overdue_index
		FROM contract C
		LEFT JOIN user U ON U.id = C.recovery_officer_id
		LEFT JOIN contract_state CS ON CS.id = C.contract_state_id
		LEFT JOIN contract_batch CB ON CB.id = C.contract_batch_id
		LEFT JOIN state S ON S.id = CS.state_id
		LEFT JOIN model M ON C.model_id = M.id
		LEFT JOIN (SELECT CR.contract_id, MAX(CR.datetime) AS datetime FROM contract_receipt CR WHERE CR.legacy_payment_date IS NULL AND CR.is_customer_payment = 1 GROUP BY CR.contract_id) CR ON CR.contract_id = C.id
		LEFT JOIN (SELECT CRL.contract_id, MAX(CRL.legacy_payment_date) as legacy_payment_date FROM contract_receipt CRL WHERE CRL.is_customer_payment = 1 GROUP BY CRL.contract_id) CRL ON CRL.contract_id = C.id
		LEFT JOIN (SELECT CI.id, CI.contract_id, CI.capital+CI.interest+CI.default_interest AS installment, CI.capital+CI.interest AS agreed_installment, SUM(COALESCE(CCP.amount, 0)+COALESCE(CIP.amount, 0)) AS installment_paid, SUM(COALESCE(CCP_SD.sd_amount, 0)+COALESCE(CIP_SD.sd_amount, 0)) AS sd_installment_paid, SUM(COALESCE(CCP_ED.ed_amount, 0)+COALESCE(CIP_ED.ed_amount, 0)) AS ed_installment_paid, COALESCE(SUM(CDIP.amount), 0) as defalut_interest_paid, CI.due_date
		FROM contract_installment CI
		
		/* Default payments */
		
		LEFT JOIN (
			SELECT CDIP.contract_installment_id, COALESCE(SUM(CDIP.amount), 0) as amount
			FROM contract_default_interest_payment CDIP
			GROUP BY CDIP.contract_installment_id
		) CDIP ON CDIP.contract_installment_id = CI.id
		LEFT JOIN (
			SELECT CDIP.contract_installment_id, COALESCE(SUM(CDIP.amount), 0) as sd_amount
			FROM contract_default_interest_payment CDIP
			LEFT JOIN contract_receipt CR ON CR.id = CDIP.contract_receipt_id
			WHERE DATE(CR.datetime) <= '%s'
			GROUP BY CDIP.contract_installment_id
		) CDIP_SD ON CDIP_SD.contract_installment_id = CI.id
		LEFT JOIN (
			SELECT CDIP.contract_installment_id, COALESCE(SUM(CDIP.amount), 0) as ed_amount
			FROM contract_default_interest_payment CDIP
			LEFT JOIN contract_receipt CR ON CR.id = CDIP.contract_receipt_id
			WHERE DATE(CR.datetime) <= '%s'
			GROUP BY CDIP.contract_installment_id
		) CDIP_ED ON CDIP_ED.contract_installment_id = CI.id
		
		/* Interest payments */
			
		LEFT JOIN (
			SELECT CIP.contract_installment_id, COALESCE(SUM(CIP.amount), 0) as amount
			FROM contract_interest_payment CIP
			GROUP BY CIP.contract_installment_id
		) CIP ON CIP.contract_installment_id = CI.id
		LEFT JOIN (
			SELECT CIP.contract_installment_id, COALESCE(SUM(CIP.amount), 0) as sd_amount
			FROM contract_interest_payment CIP
			LEFT JOIN contract_receipt CR ON CR.id = CIP.contract_receipt_id
			WHERE DATE(CR.datetime) <= '%s'
			GROUP BY CIP.contract_installment_id
		) CIP_SD ON CIP_SD.contract_installment_id = CI.id
		LEFT JOIN (
			SELECT CIP.contract_installment_id, COALESCE(SUM(CIP.amount), 0) as ed_amount
			FROM contract_interest_payment CIP
			LEFT JOIN contract_receipt CR ON CR.id = CIP.contract_receipt_id
			WHERE DATE(CR.datetime) <= '%s'
			GROUP BY CIP.contract_installment_id
		) CIP_ED ON CIP_ED.contract_installment_id = CI.id
		
		/* Capital payments */
		
		LEFT JOIN (
			SELECT CCP.contract_installment_id, COALESCE(SUM(CCP.amount), 0) as amount
			FROM contract_capital_payment CCP
			GROUP BY CCP.contract_installment_id
		) CCP ON CCP.contract_installment_id = CI.id
		LEFT JOIN (
			SELECT CCP.contract_installment_id, COALESCE(SUM(CCP.amount), 0) as sd_amount
			FROM contract_capital_payment CCP
			LEFT JOIN contract_receipt CR ON CR.id = CCP.contract_receipt_id
			WHERE DATE(CR.datetime) <= '%s'
			GROUP BY CCP.contract_installment_id
		) CCP_SD ON CCP_SD.contract_installment_id = CI.id
		LEFT JOIN (
			SELECT CCP.contract_installment_id, COALESCE(SUM(CCP.amount), 0) as ed_amount
			FROM contract_capital_payment CCP
			LEFT JOIN contract_receipt CR ON CR.id = CCP.contract_receipt_id
			WHERE DATE(CR.datetime) <= '%s'
			GROUP BY CCP.contract_installment_id
		) CCP_ED ON CCP_ED.contract_installment_id = CI.id
		GROUP BY CI.id, CI.contract_id, CI.capital, CI.interest, CI.interest, CI.default_interest, CI.due_date
		ORDER BY CI.due_date ASC) CI ON CI.contract_id = C.id
		LEFT JOIN (SELECT CI.contract_id, MIN(CI.due_date) AS due_date FROM contract_installment CI WHERE CI.due_date > (SELECT MIN(CI2.due_date) FROM contract_installment CI2 WHERE CI.contract_id = CI2.contract_id) GROUP BY CI.contract_id) CI2 ON CI2.contract_id = C.id
		WHERE (? IS NULL OR S.id = ?) AND (? IS NULL OR C.recovery_officer_id = ?) AND (? IS NULL OR C.contract_batch_id = ?) AND (? IS NULL OR C.non_performing = ?)
		GROUP BY C.id
	`, startDate, endDate, startDate, endDate, startDate, endDate, startDate, endDate, startDate, endDate, startDate, endDate, startDate, endDate)
}

const CHART_OF_ACCOUNTS = `
	SELECT MA.account_id AS main_account_id, MA.name AS main_account, SA.account_id AS sub_account_id, SA.name AS sub_account, AC.account_id AS account_category_id, AC.name AS account_category, A.account_id, A.name AS account_name
	FROM account A
	RIGHT JOIN account_category AC ON AC.id = A.account_category_id
	RIGHT JOIN sub_account SA ON SA.id = AC.sub_account_id
	RIGHT JOIN main_account MA ON MA.id = SA.main_account_id
`
const MANAGED_BY_AGRIVEST = `
	SELECT C.agrivest, C.customer_contact FROM contract C WHERE C.id = ?
`
const ACCOUNT_LEDGER = `
	SELECT A.name, AT.transaction_id, DATE_FORMAT(T.posting_date, '%Y-%m-%d') as posting_date, AT.amount, AT.type, T.remark
	FROM account_transaction AT
	LEFT JOIN account A ON A.id = AT.account_id
	LEFT JOIN transaction T ON T.id = AT.transaction_id
	WHERE AT.account_id = ?
`

const TRANSACTION = `
	SELECT AT.transaction_id, A.account_id, A.id AS account_id2, A.name AS account_name, AT.type, AT.amount
	FROM account_transaction AT
	LEFT JOIN account A ON A.id = AT.account_id
	WHERE AT.transaction_id = ?
`

const TRIAL_BALANCE = `
	SELECT A.id, A.account_id, A.name, COALESCE(AT.debit, 0) AS debit, COALESCE(AT.credit, 0) AS credit, COALESCE(AT.debit-AT.credit, 0) AS balance
	FROM account A
	LEFT JOIN (
		SELECT AT.account_id, SUM(CASE WHEN AT.type = "DR" THEN AT.amount ELSE 0 END) AS debit, SUM(CASE WHEN AT.type = "CR" THEN AT.amount ELSE 0 END) AS credit 
		FROM account_transaction AT
		GROUP BY AT.account_id
	) AT ON AT.account_id = A.id
	ORDER BY account_id ASC
`

const OFFICER_ACC_NO = `
	SELECT account_id FROM user WHERE id = ?
`

const DEBIT_NOTE_UNEARNED_ACC_NO = `
	SELECT CIT.unearned_account_id FROM contract_installment_type CIT WHERE CIT.id = ?
`

const PAYMENT_VOUCHERS = `
	SELECT PV.id, T.datetime, T.posting_date, A.name AS from_account, U.name AS user
	FROM payment_voucher PV
	LEFT JOIN transaction T ON T.id = PV.transaction_id
	LEFT JOIN account_transaction AT ON AT.transaction_id = T.id AND AT.type = 'CR'
	LEFT JOIN account A ON A.id = AT.account_id
	LEFT JOIN user U ON T.user_id = U.id
	ORDER BY T.datetime DESC
`

const PAYMENT_VOUCHER_DETAILS = `
	SELECT A.account_id, A.name AS account_name, AT.amount, DATE(T.posting_date) as posting_date
	FROM payment_voucher PV
	LEFT JOIN transaction T ON T.id = PV.transaction_id
	LEFT JOIN account_transaction AT ON AT.transaction_id = T.id AND AT.type = 'DR'
	LEFT JOIN account A ON A.id = AT.account_id
	WHERE PV.id = ?
`

const PAYMENT_VOUCHER_CHECK_DETAILS = `
	SELECT PV.due_date, PV.check_number, PV.payee, T.remark, A.name AS account_name, T.datetime
	FROM payment_voucher PV
	LEFT JOIN transaction T ON T.id = PV.transaction_id
	LEFT JOIN account_transaction AT ON AT.transaction_id = T.id AND AT.type = 'CR'
	LEFT JOIN account A ON A.id = AT.account_id
	WHERE PV.id = ?
`

const CSQA_SEARCH = `
	SELECT C.id, U.name as recovery_officer, S.name as state, CSQA.answer as answer, DATEDIFF(NOW(), CSQA.created) as created_ago, CSQA.state_at_answer
	FROM contract C 
	LEFT JOIN (SELECT CS.contract_id, CSQA.question_id, CSQA.created, CSQA.answer, S.name as state_at_answer FROM contract_state_question_answer 	CSQA LEFT JOIN contract_state CS ON CS.id = CSQA.contract_state_id LEFT JOIN state S ON S.id = CS.state_id WHERE CSQA.deleted = 0 AND 			CSQA.question_id = ?) CSQA ON CSQA.contract_id = C.id 
	LEFT JOIN contract_state CS ON CS.id = C.contract_state_id
	LEFT JOIN state S ON S.id = CS.state_id
	LEFT JOIN user U ON U.id = C.recovery_officer_id
	WHERE C.legacy = 0 AND S.name <> 'Deleted' AND (CASE WHEN ? = 0 THEN ? IS NULL OR CSQA.answer LIKE ? ELSE CSQA.answer IS NULL END)
`

const CONTRACT_PAYABLE = `
	SELECT COALESCE(SUM(CI.installment-CI.installment_paid), 0) AS total_payable
	FROM contract C
	LEFT JOIN (SELECT CI.contract_id, MIN(CI.due_date) AS due_date FROM contract_installment CI WHERE CI.due_date > (SELECT MIN(CI2.due_date) FROM contract_installment CI2 WHERE CI.contract_id = CI2.contract_id) GROUP BY CI.contract_id) CI2 ON CI2.contract_id = C.id
	LEFT JOIN (SELECT CRL.contract_id, MAX(CRL.legacy_payment_date) as legacy_payment_date FROM contract_receipt CRL GROUP BY CRL.contract_id) CRL ON CRL.contract_id = C.id
	LEFT JOIN (SELECT CI.id, CI.contract_id, CI.capital+CI.interest+CI.default_interest AS installment,CI.capital+CI.interest AS agreed_installment,SUM(COALESCE(CCP.amount, 0)+COALESCE(CIP.amount, 0)) AS installment_paid, COALESCE(SUM(CDIP.amount), 0) AS defalut_interest_paid, CI.due_date
	FROM contract_installment CI
	LEFT JOIN (
		SELECT CDIP.contract_installment_id, COALESCE(SUM(amount), 0) AS amount
		FROM contract_default_interest_payment CDIP
		GROUP BY CDIP.contract_installment_id
	) CDIP ON CDIP.contract_installment_id = CI.id
	LEFT JOIN (
		SELECT CIP.contract_installment_id, COALESCE(SUM(amount), 0) AS amount
		FROM contract_interest_payment CIP
		GROUP BY CIP.contract_installment_id
	) CIP ON CIP.contract_installment_id = CI.id
	LEFT JOIN (
		SELECT CCP.contract_installment_id, COALESCE(SUM(amount), 0) AS amount
		FROM contract_capital_payment CCP
		GROUP BY CCP.contract_installment_id
	) CCP ON CCP.contract_installment_id = CI.id
	GROUP BY CI.id, CI.contract_id, CI.capital, CI.interest, CI.interest, CI.default_interest, CI.due_date
	ORDER BY CI.due_date ASC) CI ON CI.contract_id = C.id
	WHERE C.id = ?
	GROUP BY C.id
`

const FLOAT_RECEIPTS = `
	SELECT CRF.id, CRF.user_id, CRF.amount, DATE(CRF.datetime) AS date, CRF.datetime
	FROM contract_receipt_float CRF
	WHERE CRF.contract_id = ? AND CRF.cleared = 0
	ORDER BY CRF.datetime ASC
`

const FLOAT_RECEIPTS_CLIENT = `
	SELECT CRF.id, CRF.datetime, CRF.amount
	FROM contract_receipt_float CRF
	WHERE CRF.contract_id = ? AND CRF.cleared = 0
`

const OFFICER_NAME = `
	SELECT name FROM user WHERE id = ?
`

const SENDER_MOBILE = `
	SELECT U.mobile
	FROM contract C 
	LEFT JOIN user U ON C.recovery_officer_id = U.id
	WHERE C.id = ?
`

const SEASONAL_INCENTIVE = `
	SELECT ROUND(SUM(CIP.amount)*(1.6/100), 2) as seasonal_incentive
	FROM contract_interest_payment CIP
	WHERE CIP.contract_receipt_id IN (SELECT CR.id
	FROM contract_receipt CR 
	LEFT JOIN contract C ON C.id = CR.contract_id
	WHERE CR.contract_receipt_type_id = 1  AND C.recovery_officer_id = ? AND DATE(CR.datetime) BETWEEN '2020-07-01' AND '2020-12-31')
`

const ACHIEVEMENT_SUMMARY = `
	SELECT T.user_id, U.name AS officer, DATE_FORMAT(T.month, "%Y-%m") AS month, T.amount AS target, COALESCE(SUM(CR.amount), 0) AS collection, COALESCE(ROUND(SUM(CR.amount)*100/T.amount, 2), 0) AS collection_percentage
	FROM target T
	LEFT JOIN user U ON U.id = T.user_id
	LEFT JOIN contract C ON C.recovery_officer_id = T.user_id
	LEFT JOIN contract_receipt CR ON CR.contract_id = C.id AND YEAR(CR.datetime) = YEAR(T.month) AND MONTH(CR.datetime) = MONTH(T.month) AND CR.contract_receipt_type_id = 1
	WHERE T.target_batch_id = (SELECT TB.id
	FROM target_batch TB
	WHERE DATE(NOW()) BETWEEN TB.start AND TB.end)
	GROUP BY T.user_id, U.name, month, T.amount
	ORDER BY month ASC
`
