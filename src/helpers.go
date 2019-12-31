package main

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"runtime/debug"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/dgrijalva/jwt-go"
	"github.com/globalsign/mgo/bson"
)

func (app *application) serverError(w http.ResponseWriter, err error) {
	trace := fmt.Sprintf("%s\n%s", err.Error(), debug.Stack())
	app.errorLog.Output(2, trace)

	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func (app *application) notFound(w http.ResponseWriter) {
	app.clientError(w, http.StatusNotFound)
}

func (app *application) extractUser(r *http.Request) jwt.Claims {
	ctx := r.Context()
	return ctx.Value(contextKey("User")).(jwt.MapClaims)
}

func (app *application) getS3Session(endpoint, region string) (*session.Session, error) {
	s, err := session.NewSession(&aws.Config{
		Endpoint: &endpoint,
		Region:   &region,
		Credentials: credentials.NewStaticCredentials(
			app.s3id,
			app.s3secret,
			""),
	})
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (app *application) uploadFileToS3(s *session.Session, file multipart.File, fileHeader *multipart.FileHeader) (string, error) {
	size := fileHeader.Size
	buffer := make([]byte, size)
	file.Read(buffer)

	tempFileName := "documents/" + bson.NewObjectId().Hex() + filepath.Ext(fileHeader.Filename)

	_, err := s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:             aws.String(app.s3bucket),
		Key:                aws.String(tempFileName),
		ACL:                aws.String("private"),
		Body:               bytes.NewReader(buffer),
		ContentLength:      aws.Int64(int64(size)),
		ContentType:        aws.String(http.DetectContentType(buffer)),
		ContentDisposition: aws.String("attachment"),
		StorageClass:       aws.String("INTELLIGENT_TIERING"),
	})
	if err != nil {
		return "", err
	}

	return tempFileName, nil
}
