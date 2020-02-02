package main

import (
	"database/sql"
	"flag"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ssrdive/cidium/pkg/models/mysql"
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
	user       *mysql.UserModel
	dropdown   *mysql.DropdownModel
	contract   *mysql.ContractModel
	account    *mysql.AccountModel
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
	flag.Parse()

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

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
		user:       &mysql.UserModel{DB: db},
		dropdown:   &mysql.DropdownModel{DB: db},
		contract:   &mysql.ContractModel{DB: db},
		account:    &mysql.AccountModel{DB: db},
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
