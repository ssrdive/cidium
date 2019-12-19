package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/ssrdive/cidium/pkg/models"
	"golang.org/x/crypto/bcrypt"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := ctx.Value("user").(string)

	w.Write([]byte(user))
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

	fmt.Fprintf(w, "%v", u)
}
