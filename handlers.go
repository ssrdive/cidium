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
	user := app.extractUser(r)

	fmt.Fprintf(w, "%v", user)
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

	user := models.UserResponse{u.ID, u.Username, u.Name, "Admin", ts}
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

	cds, err := app.contract.ContractDetail(cid)
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

	installments, err := app.contract.ContractInstallments(cid)
	if err != nil {
		app.serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(installments)
}

func (app *application) contractRequestability(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cid, err := strconv.Atoi(vars["cid"])
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	// Check if a request current exists for the contract current state
	requestExists, err := app.contract.CurrentRequetExists(cid)
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

	ts, err := app.contract.ContractTransionableStates(cid)
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
			err := app.contract.InitiateContract(request)
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
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	rid, err := app.contract.Receipt(user_id, cid, amount, notes)
	if err != nil {
		app.serverError(w, err)
		return
	}

	fmt.Fprintf(w, "%v", rid)
}
