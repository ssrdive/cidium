package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

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

	fmt.Println(r.PostForm)

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

	requiredParams := []string{"user_id", "contract_type_id", "contract_state_id", "institute_dealer_id", "contract_batch_id", "model_id", "chassis_number", "customer_nic", "customer_name", "customer_address", "customer_contact", "price"}
	optionalParams := []string{"institute_id", "liaison_name", "liaison_contact", "liaison_comment", "downpayment"}

	allParams := append(requiredParams, optionalParams...)

	fmt.Println(r.PostForm)
	id, err := app.contract.Insert(allParams, r.PostForm)
	if err != nil {
		app.serverError(w, err)
	}

	fmt.Println(id)
	
	for _, param := range requiredParams {
		if v := r.PostForm.Get(param); v == "" {
			app.clientError(w, http.StatusBadRequest)
			return
		}
	}

}