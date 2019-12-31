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
	claims["exp"] = time.Now().Add(time.Minute * 30).Unix()

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

	requiredParams := []string{"user_id", "contract_type_id", "institute_dealer_id", "contract_batch_id", "model_id", "chassis_number", "customer_nic", "customer_name", "customer_address", "customer_contact", "price"}
	optionalParams := []string{"institute_id", "liaison_name", "liaison_contact", "liaison_comment", "downpayment"}

	for _, param := range requiredParams {
		if v := r.PostForm.Get(param); v == "" {
			fmt.Println(param)
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	id, err := app.contract.Insert(requiredParams, optionalParams, r.PostForm)
	if err != nil {
		app.serverError(w, err)
		return
	}

	fmt.Fprintf(w, "%d", id)
}

func (app *application) searchContract(w http.ResponseWriter, r *http.Request) {

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

func (app *application) contractAnswer(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	requiredParams := []string{"contract_state_id", "question_id", "answer"}
	for _, param := range requiredParams {
		if v := r.PostForm.Get(param); v == "" {
			fmt.Println(param)
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

	id, err := app.contract.StateAnswer(requiredParams, []string{}, r.PostForm)
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
