package main

import "net/http"

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	groupID := 1
	firstName := "shamal"
	middleName := "damon"
	lastName := "sandeep"
	commonName := "shamal"
	password := "password@123"

	_, err := app.user.Insert(groupID, firstName, middleName, lastName, commonName, password)

	if err != nil {
		app.serverError(w, err)
	}

	w.Write([]byte("Hello World"))
}
