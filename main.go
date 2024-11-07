package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ssrdive/cidium/pkg/models/mysql"
	"github.com/ssrdive/scribe"
)

type application struct {
	errorLog   *log.Logger
	infoLog    *log.Logger
	secret     []byte
	s3id       string
	s3secret   string
	s3endpoint string
	s3region   string
	s3bucket   string
	rAPIKey    string
	aAPIKey    string
	aAPIPass   string
	runtimeEnv string
	user       *mysql.UserModel
	dropdown   *mysql.DropdownModel
	contract   *mysql.ContractModel
	account    *scribe.AccountModel
	reporting  *mysql.ReportingModel
}

func main() {
	addr := flag.String("addr", ":4000", "HTTP network address")
	dsn := flag.String("dsn", "user:password@tcp(host)/database_name?parseTime=true", "MySQL data source name")
	secret := flag.String("secret", "cidium", "Secret key for generating jwts")
	s3id := flag.String("id", "", "AWS S3 identification")
	s3secret := flag.String("s3secret", "", "AWS S3 secret")
	s3endpoint := flag.String("endpoint", "sgp1.digitaloceanspaces.com", "AWS S3 endpoint")
	s3region := flag.String("region", "sgp1", "AWS S3 region")
	s3bucket := flag.String("bucket", "agrivest", "AWS S3 bucket")
	rAPIKey := flag.String("rAPIKey", "", "Randeepa Text Message API Key")
	aAPIKey := flag.String("aAPIKey", "", "Agrivest Text Message API User")
	aAPIPass := flag.String("aAPIPass", "", "Agrivest Text Message API Password")
	runtimeEnv := flag.String("renv", "prod", "Runtime environment mode")
	logPath := flag.String("logpath", "/var/www/agrivest.app/logs/", "Path to create or alter log files")
	flag.Parse()

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	receiptLogFile, err := openLogFile(*logPath + time.Now().Format("2006-01-02") + "_receipt.log")
	if err != nil {
		fmt.Println("Failed to open receipt log file")
		os.Exit(1)
	}

	receiptLog := log.New(receiptLogFile, "", log.Ldate|log.Ltime)

	db, err := openDB(*dsn)
	if err != nil {
		errorLog.Fatal(err)
	}

	defer db.Close()

	app := &application{
		errorLog:   errorLog,
		infoLog:    infoLog,
		secret:     []byte(*secret),
		s3id:       *s3id,
		s3secret:   *s3secret,
		s3endpoint: *s3endpoint,
		s3region:   *s3region,
		s3bucket:   *s3bucket,
		rAPIKey:    *rAPIKey,
		aAPIKey:    *aAPIKey,
		aAPIPass:   *aAPIPass,
		runtimeEnv: *runtimeEnv,
		user:       &mysql.UserModel{DB: db},
		dropdown:   &mysql.DropdownModel{DB: db},
		contract:   &mysql.ContractModel{DB: db, ReceiptLogger: receiptLog},
		account:    &scribe.AccountModel{DB: db},
		reporting:  &mysql.ReportingModel{DB: db},
	}

	srv := &http.Server{
		Addr:     *addr,
		ErrorLog: errorLog,
		Handler:  app.routes(),
	}

	infoLog.Printf("Starting server on %s", *addr)
	err = srv.ListenAndServe()
	errorLog.Fatal(err)
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, err
}

func openLogFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}
