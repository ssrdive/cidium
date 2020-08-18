package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/ssrdive/cidium/pkg/models"
	"golang.org/x/crypto/bcrypt"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	// user := app.extractUser(r)

	if app.runtimeEnv == "dev" {
		fmt.Fprintf(w, "It works! [dev]")
	} else {
		fmt.Fprintf(w, "It works!")
	}
}

func (app *application) authenticate(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")

	u, err := app.user.Get(username, password)
	if err != nil {
		if errors.Is(err, models.ErrNoRecord) || errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			app.notFound(w)
		} else {
			app.serverError(w, err)
		}
		return
	}

	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)

	claims["username"] = u.Username
	claims["name"] = u.Name
	claims["exp"] = time.Now().Add(time.Minute * 180).Unix()

	ts, err := token.SignedString(app.secret)
	if err != nil {
		app.serverError(w, err)
		return
	}

	user := models.UserResponse{u.ID, u.Username, u.Name, u.Type, ts}
	js, err := json.Marshal(user)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func (app *application) dropdownHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	if name == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	items, err := app.dropdown.Get(name)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)

}

func (app *application) achievementSummary(w http.ResponseWriter, r *http.Request) {
	items, err := app.reporting.AchievementSummary()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (app *application) paymentVouchers(w http.ResponseWriter, r *http.Request) {
	items, err := app.account.PaymentVouchers()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)

}

func (app *application) paymentVoucherDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pid, err := strconv.Atoi(vars["pid"])
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	items, err := app.account.PaymentVoucherDetails(pid)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)

}

func (app *application) dropdownConditionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	where := vars["where"]
	value := vars["value"]
	if name == "" || where == "" || value == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	items, err := app.dropdown.ConditionGet(name, where, value)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)

}

func (app *application) dropdownConditionAccountsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	where := vars["where"]
	value := vars["value"]
	if name == "" || where == "" || value == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	items, err := app.dropdown.ConditionAccountsGet(name, where, value)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)

}

func (app *application) newAccountCategory(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	requiredParams := []string{"sub_account_id", "user_id", "account_id", "name"}
	optionalParams := []string{"datetime"}
	for _, param := range requiredParams {
		if v := r.PostForm.Get(param); v == "" {
			fmt.Println(param)
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	id, err := app.account.CreateCategory(requiredParams, optionalParams, r.PostForm)
	if err != nil {
		app.serverError(w, err)
		return
	}

	fmt.Fprintf(w, "%d", id)
}

func (app *application) newAccount(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	requiredParams := []string{"account_category_id", "user_id", "account_id", "name"}
	optionalParams := []string{"datetime"}
	for _, param := range requiredParams {
		if v := r.PostForm.Get(param); v == "" {
			fmt.Println(param)
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	id, err := app.account.CreateAccount(requiredParams, optionalParams, r.PostForm)
	if err != nil {
		app.serverError(w, err)
		return
	}

	fmt.Fprintf(w, "%d", id)
}

func (app *application) newContract(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	requiredParams := []string{"user_id", "recovery_officer_id", "contract_type_id", "institute_dealer_id", "contract_batch_id", "model_id", "chassis_number", "customer_nic", "customer_name", "customer_address", "customer_contact", "price"}
	optionalParams := []string{"institute_id", "liaison_name", "liaison_contact", "liaison_comment", "downpayment"}

	for _, param := range requiredParams {
		if v := r.PostForm.Get(param); v == "" {
			fmt.Println(param)
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	id, err := app.contract.Insert("Start", requiredParams, optionalParams, r.PostForm)
	if err != nil {
		app.serverError(w, err)
		return
	}

	fmt.Fprintf(w, "%d", id)
}

func (app *application) newLegacyContract(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	requiredContractParams := []string{"user_id", "recovery_officer_id", "contract_type_id", "institute_dealer_id", "contract_batch_id", "model_id", "chassis_number", "customer_nic", "customer_name", "customer_address", "customer_contact", "price"}
	requiredLoanParams := []string{"capital", "rate", "installments", "installment_interval", "method", "initiation_date"}
	requiredParams := append(requiredContractParams, requiredLoanParams...)
	optionalParams := []string{"institute_id", "liaison_name", "liaison_contact", "liaison_comment", "downpayment"}

	for _, param := range requiredParams {
		if v := r.PostForm.Get(param); v == "" {
			fmt.Println(param)
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	id, err := app.contract.Insert("Active", requiredContractParams, optionalParams, r.PostForm)
	if err != nil {
		app.serverError(w, err)
		return
	}

	err = app.contract.Legacy(int(id), r.PostForm)
	if err != nil {
		app.serverError(w, err)
	}

	fmt.Fprintf(w, "%d", id)
}

func (app *application) searchContractOld(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	state := r.URL.Query().Get("state")
	officer := r.URL.Query().Get("officer")
	batch := r.URL.Query().Get("batch")

	results, err := app.contract.SearchOld(search, state, officer, batch)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (app *application) searchContractV2(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	state := r.URL.Query().Get("state")
	officer := r.URL.Query().Get("officer")
	batch := r.URL.Query().Get("batch")
	npl := r.URL.Query().Get("npl")
	startOd := r.URL.Query().Get("startod")
	endOd := r.URL.Query().Get("endod")
	removeDeleted := r.URL.Query().Get("removedeleted")

	results, err := app.contract.SearchV2(search, state, officer, batch, npl, startOd, endOd, removeDeleted)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (app *application) receiptSearch(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("startdate")
	_, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	endDate := r.URL.Query().Get("enddate")
	_, err = time.Parse("2006-01-02", endDate)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	officer := r.URL.Query().Get("officer")

	results, err := app.reporting.ReceiptSearch(startDate, endDate, officer)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (app *application) performanceReview(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("startdate")
	_, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	endDate := r.URL.Query().Get("enddate")
	_, err = time.Parse("2006-01-02", endDate)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	officer := r.URL.Query().Get("officer")
	batch := r.URL.Query().Get("batch")
	npl := r.URL.Query().Get("npl")

	results, err := app.contract.PerformanceReview(startDate, endDate, state, officer, batch, npl)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (app *application) searchContract(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	state := r.URL.Query().Get("state")
	officer := r.URL.Query().Get("officer")
	batch := r.URL.Query().Get("batch")

	results, err := app.contract.Search(search, state, officer, batch)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (app *application) csqaSearchContract(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	question := r.URL.Query().Get("question")
	empty := r.URL.Query().Get("empty")

	results, err := app.contract.CSQASearch(search, question, empty)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (app *application) accountTransaction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tid, err := strconv.Atoi(vars["tid"])
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	ledger, err := app.account.Transaction(tid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ledger)
}

func (app *application) accountLedger(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	aid, err := strconv.Atoi(vars["aid"])
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	ledger, err := app.account.Ledger(aid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ledger)
}

func (app *application) workDocuments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid, err := strconv.Atoi(vars["cid"])
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	wds, err := app.contract.WorkDocuments(cid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wds)
}

func (app *application) workQuestions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid, err := strconv.Atoi(vars["cid"])
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	wqs, err := app.contract.WorkQuestions(cid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(wqs)
}

func (app *application) contractQuestions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid, err := strconv.Atoi(vars["cid"])
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	qs, err := app.contract.Questions(cid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(qs)
}

func (app *application) contractDocuments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid, err := strconv.Atoi(vars["cid"])
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	ds, err := app.contract.Documents(cid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ds)
}

func (app *application) contractHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid, err := strconv.Atoi(vars["cid"])
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	h, err := app.contract.History(cid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h)
}

func (app *application) contractAnswer(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	requiredParams := []string{"contract_state_id", "question_id", "user_id", "answer"}
	for _, param := range requiredParams {
		if v := r.PostForm.Get(param); v == "" {
			fmt.Println(param)
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	t := time.Now()
	r.PostForm.Set("created", t.Format("2006-01-02 15:04:05"))
	id, err := app.contract.StateAnswer([]string{"contract_state_id", "question_id", "user_id", "created", "answer"}, []string{}, r.PostForm)
	if err != nil {
		app.serverError(w, err)
		return
	}

	fmt.Fprintf(w, "%d", id)
}

func (app *application) contractDocument(w http.ResponseWriter, r *http.Request) {
	maxSize := int64(5120000)
	err := r.ParseMultipartForm(maxSize)
	if err != nil {
		app.serverError(w, err)
		return
	}

	requiredParams := []string{"contract_state_id", "document_id", "user_id"}
	for _, param := range requiredParams {
		if v := r.FormValue(param); v == "" {
			fmt.Println(param)
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	file, fileHeader, err := r.FormFile("source")
	if err != nil {
		app.serverError(w, err)
		return
	}
	defer file.Close()

	s, err := app.getS3Session(app.s3endpoint, app.s3region)
	if err != nil {
		app.serverError(w, err)
		return
	}

	fileName, err := app.uploadFileToS3(s, file, fileHeader)
	if err != nil {
		app.serverError(w, err)
		return
	}

	t := time.Now()
	r.Form.Set("created", t.Format("2006-01-02 15:04:05"))
	r.Form.Set("s3bucket", app.s3bucket)
	r.Form.Set("s3region", app.s3region)
	r.Form.Set("source", fileName)

	id, err := app.contract.StateDocument([]string{"contract_state_id", "document_id", "user_id", "created", "s3bucket", "s3region", "source"}, []string{}, r.Form)
	if err != nil {
		app.serverError(w, err)
		return
	}

	fmt.Fprintf(w, "%d", id)
}

func (app *application) contractDocumentDownload(w http.ResponseWriter, r *http.Request) {
	bucket := r.URL.Query().Get("bucket")
	region := r.URL.Query().Get("region")
	source := r.URL.Query().Get("source")
	if bucket == "" || region == "" || source == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	sess, err := app.getS3Session(fmt.Sprintf("%s.digitaloceanspaces.com", region), region)
	if err != nil {
		app.serverError(w, err)
		return
	}

	s3c := s3.New(sess)
	output, err := s3c.GetObject(&s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(source)})
	if err != nil {
		app.serverError(w, err)
		return
	}

	buff, err := ioutil.ReadAll(output.Body)
	if err != nil {
		app.serverError(w, err)
		return
	}

	reader := bytes.NewReader(buff)

	http.ServeContent(w, r, source, time.Now(), reader)
}

func (app *application) contractDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid, err := strconv.Atoi(vars["cid"])
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	cds, err := app.contract.Detail(cid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cds)
}

func (app *application) contractInstallments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid, err := strconv.Atoi(vars["cid"])
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	installments, err := app.contract.Installment(cid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(installments)
}

func (app *application) contractReceiptsV2(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid, err := strconv.Atoi(vars["cid"])
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	receipts, err := app.contract.ReceiptsV2(cid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(receipts)
}

func (app *application) contractFloatReceipts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid, err := strconv.Atoi(vars["cid"])
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	receipts, err := app.contract.FloatReceipts(cid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(receipts)
}

func (app *application) contractReceipts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid, err := strconv.Atoi(vars["cid"])
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	receipts, err := app.contract.Receipts(cid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(receipts)
}

func (app *application) contractOfficerReceipts(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	oid, err := strconv.Atoi(vars["officer"])
	date := vars["date"]
	if err != nil || date == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	receipts, err := app.contract.OfficerReceipts(oid, date)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(receipts)
}

func (app *application) contractRequestability(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid, err := strconv.Atoi(vars["cid"])
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Check if a request current exists for the contract current state
	requestExists, err := app.contract.CurrentRequestExists(cid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	if requestExists {
		cr := models.ContractRequestable{
			Requestable:           false,
			NonRequestableMessage: " A request is currently pending for this contract",
			States:                nil,
			RejectedRequests:      nil,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cr)
		return
	}

	wds, err := app.contract.WorkDocuments(cid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	wqs, err := app.contract.WorkQuestions(cid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	requestable := true
	for _, d := range wds {
		if d.Compulsory == 1 && !d.Source.Valid {
			requestable = false
		}
	}

	for _, q := range wqs {
		if q.Compulsory == 1 && !q.Answer.Valid {
			requestable = false
		}
	}

	if requestable == false {
		cr := models.ContractRequestable{
			Requestable:           requestable,
			NonRequestableMessage: "Required answers and/or documents are not complete",
			States:                nil,
			RejectedRequests:      nil,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cr)
		return
	}

	ts, err := app.contract.TransionableStates(cid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	rr, err := app.contract.RejectedRequests(cid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	cr := models.ContractRequestable{
		Requestable:           requestable,
		NonRequestableMessage: "",
		States:                ts,
		RejectedRequests:      rr,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cr)
}

func (app *application) contractRequest(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	requiredParams := []string{"contract_id", "state_id", "user_id"}
	for _, param := range requiredParams {
		if v := r.PostForm.Get(param); v == "" {
			fmt.Println(param)
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	rid, err := app.contract.Request([]string{"contract_id", "state_id"}, []string{}, r.PostForm)
	if err != nil {
		app.serverError(w, err)
		return
	}

	fmt.Fprintf(w, "%v", rid)
}

func (app *application) contractSeasonalIncentive(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uid := vars["uid"]
	user, err := strconv.Atoi(uid)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	amt, err := app.contract.SeasonalIncentive(user)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(amt)
}

func (app *application) contractRequests(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uid := vars["uid"]
	user, err := strconv.Atoi(uid)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	rs, err := app.contract.Requests(user)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rs)
}

func (app *application) contractRequestAction(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	requiredParams := []string{"request", "user", "action"}
	for _, param := range requiredParams {
		if v := r.PostForm.Get(param); v == "" {
			fmt.Println(param)
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	user, err := strconv.Atoi(r.PostForm.Get("user"))
	request, err := strconv.Atoi(r.PostForm.Get("request"))
	action := r.PostForm.Get("action")
	note := r.PostForm.Get("note")
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	if action == "1" {
		name, err := app.contract.RequestName(request)
		if err != nil {
			app.serverError(w, err)
			return
		}
		if name == "Contract Initiated" {
			err := app.contract.InitiateContract(user, request)
			if err != nil {
				app.serverError(w, err)
				return
			}
		}
		if name == "Credit Worthiness Approved" {
			err := app.contract.CreditWorthinessApproved(user, request, app.aAPIKey)
			if err != nil {
				app.serverError(w, err)
				return
			}
		}
	}

	c, err := app.contract.RequestAction(user, request, action, note)
	if err != nil {
		app.serverError(w, err)
		return
	}

	fmt.Fprintf(w, "%v", c)
}

func (app *application) contractCommitmentAction(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	requiredParams := []string{"id", "fulfilled", "user"}
	for _, param := range requiredParams {
		if v := r.PostForm.Get(param); v == "" {
			fmt.Println(param)
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	id, err := strconv.Atoi(r.PostForm.Get("id"))
	fulfilled, err := strconv.Atoi(r.PostForm.Get("fulfilled"))
	user, err := strconv.Atoi(r.PostForm.Get("user"))
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	ca, err := app.contract.CommitmentAction(id, fulfilled, user)
	if err != nil {
		app.serverError(w, err)
		return
	}

	fmt.Fprintf(w, "%v", ca)
}

func (app *application) deleteAnswer(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	requiredParams := []string{"id", "table"}
	for _, param := range requiredParams {
		if v := r.PostForm.Get(param); v == "" {
			fmt.Println(param)
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	d, err := app.contract.DeleteStateInfo(r.PostForm)
	if err != nil {
		app.serverError(w, err)
		return
	}
	fmt.Fprintf(w, "%v", d)
}

func (app *application) contractCalculation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	capital, err := strconv.ParseFloat(vars["capital"], 32)
	rate, err := strconv.ParseFloat(vars["rate"], 32)
	installments, err := strconv.Atoi(vars["installments"])
	installmentInterval, err := strconv.Atoi(vars["installmentInterval"])
	initiationDate := vars["initiationDate"]
	method := vars["method"]
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	schedule, err := models.Create(capital, rate, installments, installmentInterval, initiationDate, method)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedule)
}

func (app *application) contractLegacyRebate(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	requiredParams := []string{"user_id", "cid", "amount"}
	for _, param := range requiredParams {
		if v := r.PostForm.Get(param); v == "" {
			fmt.Println(param)
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	user_id, err := strconv.Atoi(r.PostForm.Get("user_id"))
	cid, err := strconv.Atoi(r.PostForm.Get("cid"))
	amount, err := strconv.ParseFloat(r.PostForm.Get("amount"), 32)
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	rid, err := app.contract.LegacyRebate(user_id, cid, amount)
	if err != nil {
		app.serverError(w, err)
		return
	}

	fmt.Fprintf(w, "%v", rid)
}

func (app *application) contractReceipt(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	requiredParams := []string{"user_id", "cid", "amount"}
	for _, param := range requiredParams {
		if v := r.PostForm.Get(param); v == "" {
			fmt.Println(param)
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	user_id, err := strconv.Atoi(r.PostForm.Get("user_id"))
	cid, err := strconv.Atoi(r.PostForm.Get("cid"))
	amount, err := strconv.ParseFloat(r.PostForm.Get("amount"), 32)
	notes := r.PostForm.Get("notes")
	due_date := r.PostForm.Get("due_date")
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	rid, err := app.contract.Receipt(user_id, cid, amount, notes, due_date, app.rAPIKey, app.aAPIKey, app.runtimeEnv)
	if err != nil {
		app.serverError(w, err)
		return
	}

	fmt.Fprintf(w, "%v", rid)
}

func (app *application) contractDebitNote(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	requiredParams := []string{"contract_id", "contract_installment_type_id", "capital"}
	for _, param := range requiredParams {
		if v := r.PostForm.Get(param); v == "" {
			fmt.Println(param)
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	r.PostForm.Set("due_date", time.Now().Format("2006-01-02 15:04:05"))
	dnid, err := app.contract.DebitNote(requiredParams, []string{"due_date"}, r.PostForm)
	if err != nil {
		app.serverError(w, err)
		return
	}

	fmt.Fprintf(w, "%v", dnid)
}

func (app *application) contractCommitment(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	requiredParams := []string{"user_id", "contract_id", "text"}
	optionalParams := []string{"due_date", "created", "commitment"}
	for _, param := range requiredParams {
		if v := r.PostForm.Get(param); v == "" {
			fmt.Println(param)
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	t := time.Now()
	r.PostForm.Set("created", t.Format("2006-01-02 15:04:05"))
	if r.PostForm.Get("due_date") == "" {
		r.PostForm.Set("commitment", "0")
	} else {
		r.PostForm.Set("commitment", "1")
	}

	specialMessage := "0"
	if r.PostForm.Get("special_message") == "1" {
		specialMessage = "1"
	}

	comid, err := app.contract.Commitment(requiredParams, optionalParams, r.PostForm, specialMessage, app.aAPIKey)
	if err != nil {
		app.serverError(w, err)
		return
	}

	fmt.Fprintf(w, "%v", comid)
}

func (app *application) contractCommitments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid, err := strconv.Atoi(vars["cid"])
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	commitments, err := app.contract.Commitments(cid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(commitments)
}

func (app *application) accountTrialBalance(w http.ResponseWriter, r *http.Request) {
	accounts, err := app.account.TrialBalance()
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accounts)
}

func (app *application) accountChart(w http.ResponseWriter, r *http.Request) {
	accounts, err := app.account.ChartOfAccounts()
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accounts)
}

func (app *application) accountPaymentVoucher(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	requiredParams := []string{"user_id", "posting_date", "from_account_id", "amount", "entries", "remark"}
	for _, param := range requiredParams {
		if v := r.PostForm.Get(param); v == "" {
			fmt.Println(param)
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	tid, err := app.account.PaymentVoucher(r.PostForm.Get("user_id"), r.PostForm.Get("posting_date"), r.PostForm.Get("from_account_id"), r.PostForm.Get("amount"), r.PostForm.Get("entries"), r.PostForm.Get("remark"), r.PostForm.Get("due_date"), r.PostForm.Get("check_number"), r.PostForm.Get("payee"))
	if err != nil {
		app.serverError(w, err)
		return
	}

	fmt.Fprintf(w, "%v", tid)
}

func (app *application) accountDeposit(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	requiredParams := []string{"user_id", "posting_date", "to_account_id", "amount", "entries", "remark"}
	for _, param := range requiredParams {
		if v := r.PostForm.Get(param); v == "" {
			fmt.Println(param)
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	tid, err := app.account.Deposit(r.PostForm.Get("user_id"), r.PostForm.Get("posting_date"), r.PostForm.Get("to_account_id"), r.PostForm.Get("amount"), r.PostForm.Get("entries"), r.PostForm.Get("remark"))
	if err != nil {
		app.serverError(w, err)
		return
	}

	fmt.Fprintf(w, "%v", tid)
}

func (app *application) accountJournalEntry(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	requiredParams := []string{"user_id", "posting_date", "remark", "entries"}
	for _, param := range requiredParams {
		if v := r.PostForm.Get(param); v == "" {
			fmt.Println(param)
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	tid, err := app.account.JournalEntry(r.PostForm.Get("user_id"), r.PostForm.Get("posting_date"), r.PostForm.Get("remark"), r.PostForm.Get("entries"))
	if err != nil {
		app.serverError(w, err)
		return
	}

	fmt.Fprintf(w, "%v", tid)
}

func (app *application) dashboardCommitmentsByOfficer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ctype := vars["type"]
	officer := vars["officer"]
	if ctype == "" || officer == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	commitments, err := app.contract.DashboardCommitmentsByOfficer(ctype, officer)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(commitments)
}

func (app *application) dashboardCommitments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ctype := vars["type"]
	if ctype == "" {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	commitments, err := app.contract.DashboardCommitments(ctype)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(commitments)
}

func (app *application) contractReceiptLegacy(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	requiredParams := []string{"user_id", "cid", "amount"}
	for _, param := range requiredParams {
		if v := r.PostForm.Get(param); v == "" {
			fmt.Println(param)
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	user_id, err := strconv.Atoi(r.PostForm.Get("user_id"))
	cid, err := strconv.Atoi(r.PostForm.Get("cid"))
	amount, err := strconv.ParseFloat(r.PostForm.Get("amount"), 32)
	notes := r.PostForm.Get("notes")
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	rid, err := app.contract.LegacyReceipt(user_id, cid, amount, notes)
	if err != nil {
		app.serverError(w, err)
		return
	}

	fmt.Fprintf(w, "%v", rid)
}
