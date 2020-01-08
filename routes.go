package main

import (
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
)

func (app *application) routes() http.Handler {
	standardMiddleware := alice.New(app.recoverPanic, app.logRequest, secureHeaders)

	r := mux.NewRouter()
	r.Handle("/", app.validateToken(http.HandlerFunc(app.home))).Methods("GET")
	r.HandleFunc("/dropdown/{name}", http.HandlerFunc(app.dropdownHandler)).Methods("GET")
	r.Handle("/contract/search/{search}/{stateid}/{rofficer}/{batchid}", app.validateToken(http.HandlerFunc(app.searchContract))).Methods("GET")
	r.HandleFunc("/authenticate", http.HandlerFunc(app.authenticate)).Methods("POST")
	r.HandleFunc("/contract/new", http.HandlerFunc(app.newContract)).Methods("POST")
	r.Handle("/contract/work/documents/{cid}", app.validateToken(http.HandlerFunc(app.workDocuments))).Methods("GET")
	r.Handle("/contract/work/questions/{cid}", app.validateToken(http.HandlerFunc(app.workQuestions))).Methods("GET")
	r.Handle("/contract/answer", app.validateToken(http.HandlerFunc(app.contractAnswer))).Methods("POST")
	r.Handle("/contract/document", app.validateToken(http.HandlerFunc(app.contractDocument))).Methods("POST")
	r.Handle("/contract/document/download", app.validateToken(http.HandlerFunc(app.contractDocumentDownload))).Methods("GET")
	r.Handle("/contract/state/delete", app.validateToken(http.HandlerFunc(app.deleteAnswer))).Methods("POST")
	r.Handle("/contract/details/{cid}", app.validateToken(http.HandlerFunc(app.contractDetails))).Methods("GET")
	r.Handle("/contract/requestability/{cid}", app.validateToken(http.HandlerFunc(app.contractRequestability))).Methods("GET")
	r.Handle("/contract/request", app.validateToken(http.HandlerFunc(app.contractRequest))).Methods("POST")
	r.Handle("/contract/requests/{uid}", app.validateToken(http.HandlerFunc(app.contractRequests))).Methods("GET")
	r.Handle("/contract/request/action", app.validateToken(http.HandlerFunc(app.contractRequestAction))).Methods("POST")
	r.Handle("/contract/calculation/{capital}/{rate}/{installments}/{installmentInterval}/{initiationDate}/{method}", app.validateToken(http.HandlerFunc(app.contractCalculation))).Methods("GET")
	r.Handle("/contract/receipt", app.validateToken(http.HandlerFunc(app.contractReceipt))).Methods("POST")

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	r.Handle("/static/", http.StripPrefix("/static", fileServer))

	return standardMiddleware.Then(handlers.CORS(handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}), handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS"}), handlers.AllowedOrigins([]string{"*"}))(r))
}
