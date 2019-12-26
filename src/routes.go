package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/handlers"
	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	standardMiddleware := alice.New(app.recoverPanic, app.logRequest, secureHeaders)

	r := mux.NewRouter()
	r.Handle("/", app.validateToken(http.HandlerFunc(app.home))).Methods("GET")
	r.HandleFunc("/dropdown/{name}", http.HandlerFunc(app.dropdownHandler)).Methods("GET")
	r.HandleFunc("/authenticate", http.HandlerFunc(app.authenticate)).Methods("POST")
	r.HandleFunc("/contract/new", http.HandlerFunc(app.newContract)).Methods("POST")

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	r.Handle("/static/", http.StripPrefix("/static", fileServer))

	return standardMiddleware.Then(handlers.CORS(handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}), handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS"}), handlers.AllowedOrigins([]string{"*"}))(r))
}
