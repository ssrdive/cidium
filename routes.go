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
	r.Handle("/", http.HandlerFunc(app.home)).Methods("GET")
	r.Handle("/dropdown/{name}", app.validateToken(http.HandlerFunc(app.dropdownHandler))).Methods("GET")
	r.Handle("/dropdown/condition/{name}/{where}/{value}", app.validateToken(http.HandlerFunc(app.dropdownConditionHandler))).Methods("GET")
	r.Handle("/dropdown/condition/accounts/{name}/{where}/{value}", app.validateToken(http.HandlerFunc(app.dropdownConditionAccountsHandler))).Methods("GET")
	r.Handle("/account/category/new", app.validateToken(http.HandlerFunc(app.newAccountCategory))).Methods("POST")
	r.Handle("/account/new", app.validateToken(http.HandlerFunc(app.newAccount))).Methods("POST")
	r.Handle("/account/chart", app.validateToken(http.HandlerFunc(app.accountChart))).Methods("GET")
	r.Handle("/account/journalentry", app.validateToken(http.HandlerFunc(app.accountJournalEntry))).Methods("POST")
	r.Handle("/account/paymentvoucher", app.validateToken(http.HandlerFunc(app.accountPaymentVoucher))).Methods("POST")
	r.Handle("/account/deposit", app.validateToken(http.HandlerFunc(app.accountDeposit))).Methods("POST")
	r.Handle("/account/trialbalance", app.validateToken(http.HandlerFunc(app.accountTrialBalance))).Methods("GET")
	r.Handle("/account/ledger/{aid}", app.validateToken(http.HandlerFunc(app.accountLedger))).Methods("GET")
	r.Handle("/transaction/{tid}", app.validateToken(http.HandlerFunc(app.accountTransaction))).Methods("GET")
	r.Handle("/contract/search", app.validateToken(http.HandlerFunc(app.searchContractOld))).Methods("GET")
	r.Handle("/contract/searchnew", app.validateToken(http.HandlerFunc(app.searchContract))).Methods("GET")
	r.Handle("/contract/csqasearch", app.validateToken(http.HandlerFunc(app.csqaSearchContract))).Methods("GET")
	r.HandleFunc("/authenticate", http.HandlerFunc(app.authenticate)).Methods("POST")
	r.Handle("/contract/new", app.validateToken(http.HandlerFunc(app.newContract))).Methods("POST")
	r.Handle("/contract/legacy/new", app.validateToken(http.HandlerFunc(app.newLegacyContract))).Methods("POST")
	r.Handle("/contract/work/documents/{cid}", app.validateToken(http.HandlerFunc(app.workDocuments))).Methods("GET")
	r.Handle("/contract/work/questions/{cid}", app.validateToken(http.HandlerFunc(app.workQuestions))).Methods("GET")
	r.Handle("/contract/questions/{cid}", app.validateToken(http.HandlerFunc(app.contractQuestions))).Methods("GET")
	r.Handle("/contract/documents/{cid}", app.validateToken(http.HandlerFunc(app.contractDocuments))).Methods("GET")
	r.Handle("/contract/history/{cid}", app.validateToken(http.HandlerFunc(app.contractHistory))).Methods("GET")
	r.Handle("/contract/answer", app.validateToken(http.HandlerFunc(app.contractAnswer))).Methods("POST")
	r.Handle("/contract/document", app.validateToken(http.HandlerFunc(app.contractDocument))).Methods("POST")
	r.Handle("/contract/document/download", app.validateToken(http.HandlerFunc(app.contractDocumentDownload))).Methods("GET")
	r.Handle("/contract/state/delete", app.validateToken(http.HandlerFunc(app.deleteAnswer))).Methods("POST")
	r.Handle("/contract/details/{cid}", app.validateToken(http.HandlerFunc(app.contractDetails))).Methods("GET")
	r.Handle("/contract/installments/{cid}", app.validateToken(http.HandlerFunc(app.contractInstallments))).Methods("GET")
	r.Handle("/contract/receipts/{cid}", app.validateToken(http.HandlerFunc(app.contractReceipts))).Methods("GET")
	r.Handle("/contract/receipts/officer/{officer}/{date}", app.validateToken(http.HandlerFunc(app.contractOfficerReceipts))).Methods("GET")
	r.Handle("/contract/requestability/{cid}", app.validateToken(http.HandlerFunc(app.contractRequestability))).Methods("GET")
	r.Handle("/contract/request", app.validateToken(http.HandlerFunc(app.contractRequest))).Methods("POST")
	r.Handle("/contract/requests/{uid}", app.validateToken(http.HandlerFunc(app.contractRequests))).Methods("GET")
	r.Handle("/contract/request/action", app.validateToken(http.HandlerFunc(app.contractRequestAction))).Methods("POST")
	r.Handle("/contract/calculation/{capital}/{rate}/{installments}/{installmentInterval}/{initiationDate}/{method}", app.validateToken(http.HandlerFunc(app.contractCalculation))).Methods("GET")
	r.Handle("/contract/receipt", app.validateToken(http.HandlerFunc(app.contractReceipt))).Methods("POST")
	r.Handle("/contract/debitnote", app.validateToken(http.HandlerFunc(app.contractDebitNote))).Methods("POST")
	r.Handle("/contract/commitment", app.validateToken(http.HandlerFunc(app.contractCommitment))).Methods("POST")
	r.Handle("/contract/commitments/{cid}", app.validateToken(http.HandlerFunc(app.contractCommitments))).Methods("GET")
	r.Handle("/dashboard/commitments/{type}", app.validateToken(http.HandlerFunc(app.dashboardCommitments))).Methods("GET")
	r.Handle("/dashboard/commitments/{type}/{officer}", app.validateToken(http.HandlerFunc(app.dashboardCommitmentsByOfficer))).Methods("GET")
	r.Handle("/contract/receipt/legacy", app.validateToken(http.HandlerFunc(app.contractReceiptLegacy))).Methods("POST")
	r.Handle("/contract/commitment/action", app.validateToken(http.HandlerFunc(app.contractCommitmentAction))).Methods("POST")
	r.Handle("/paymentvouchers", app.validateToken(http.HandlerFunc(app.paymentVouchers))).Methods("GET")
	r.Handle("/paymentvoucher/{pid}", app.validateToken(http.HandlerFunc(app.paymentVoucherDetails))).Methods("GET")

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	r.Handle("/static/", http.StripPrefix("/static", fileServer))

	return standardMiddleware.Then(handlers.CORS(handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}), handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS"}), handlers.AllowedOrigins([]string{"*"}))(r))
}
