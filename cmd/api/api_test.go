package main

import (
	"bytes"
	"context"
	"github.com/rs/zerolog"
	"io"
	"net/http"
	"os"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	"github.com/mannulus-immortalis/xmtask/internal/api"
	"github.com/mannulus-immortalis/xmtask/internal/api/auth"
	"github.com/mannulus-immortalis/xmtask/internal/db"
	"github.com/mannulus-immortalis/xmtask/internal/models"
	"github.com/mannulus-immortalis/xmtask/internal/models/mocks"
)

const (
	jwtKey = "dGVzdAo="
)

var (
	errDuplicate = &pq.Error{Code: "23505"}
)

func TestCreateItem(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := context.Background()
		log := zerolog.New(os.Stdout).With().Timestamp().Logger()
		id := uuid.MustParse("3f00e7f6-f9c8-4a3a-8a27-2bd529b161e4")

		// mock db
		conn, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer func() {
			_ = conn.Close()
		}()
		dbConn := db.NewFromConn(conn)
		rows := sqlmock.NewRows([]string{"id"})
		rows.AddRow(id.String())
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO companies (name, description, employee_count, is_registered, legal_type) VALUES ($1, $2, $3, $4, $5) RETURNING id`)).
			WithArgs("newcompany", "", 15, false, "Corporations").WillReturnRows(rows)

		// mock kafka
		kafkaMock := mocks.NewNotifyInt(t)
		kafkaMock.On("Send", models.EventNotifications{ID: id, Event: models.EventTypeCreated}).Return(nil)

		// real jwt
		jwtAuth, err := auth.New(jwtKey)
		assert.NoError(t, err)
		token, err := jwtAuth.Generate([]string{"reader", "writer"})
		assert.NoError(t, err)

		// start server
		api := api.New(&log, dbConn, jwtAuth, kafkaMock)
		go func() { _ = api.Run(":9081") }()
		defer api.Close()

		// make request
		reqBody := []byte(`{"name":"newcompany", "employee_count":15, "type":"Corporations"}`)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost:9081/api/v1/company", bytes.NewBuffer(reqBody))
		assert.NoError(t, err)
		req.Header.Add("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer func() {
			_ = resp.Body.Close()
		}()
		respBody, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)

		// test result
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.Equal(t, `{"id":"`+id.String()+`","name":"newcompany","employee_count":15,"is_registered":false,"type":"Corporations"}`, string(respBody))
	})

	t.Run("error_duplicate", func(t *testing.T) {
		ctx := context.Background()
		log := zerolog.New(os.Stdout).With().Timestamp().Logger()

		// mock db
		conn, mock, err := sqlmock.New()
		assert.NoError(t, err)
		defer func() {
			_ = conn.Close()
		}()
		dbConn := db.NewFromConn(conn)
		mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO companies (name, description, employee_count, is_registered, legal_type) VALUES ($1, $2, $3, $4, $5) RETURNING id`)).
			WithArgs("newcompany", "", 15, false, "Corporations").WillReturnError(errDuplicate)

		// mock kafka
		kafkaMock := mocks.NewNotifyInt(t)

		// real jwt
		jwtAuth, err := auth.New(jwtKey)
		assert.NoError(t, err)
		token, err := jwtAuth.Generate([]string{"reader", "writer"})
		assert.NoError(t, err)

		// start server
		api := api.New(&log, dbConn, jwtAuth, kafkaMock)
		go func() { _ = api.Run(":9082") }()
		defer api.Close()

		// make request
		reqBody := []byte(`{"name":"newcompany", "employee_count":15, "type":"Corporations"}`)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost:9082/api/v1/company", bytes.NewBuffer(reqBody))
		assert.NoError(t, err)
		req.Header.Add("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer func() {
			_ = resp.Body.Close()
		}()
		respBody, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)

		// test result
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Equal(t, `{"error":"Duplicate item name"}`, string(respBody))
	})

	t.Run("error_validation", func(t *testing.T) {
		ctx := context.Background()
		log := zerolog.New(os.Stdout).With().Timestamp().Logger()

		// mock db
		dbConn := mocks.NewStorageInt(t)

		// mock kafka
		kafkaMock := mocks.NewNotifyInt(t)

		// real jwt
		jwtAuth, err := auth.New(jwtKey)
		assert.NoError(t, err)
		token, err := jwtAuth.Generate([]string{"reader", "writer"})
		assert.NoError(t, err)

		// start server
		api := api.New(&log, dbConn, jwtAuth, kafkaMock)
		go func() { _ = api.Run(":9083") }()
		defer api.Close()

		// make request
		reqBody := []byte(`{"name":"newcompanydfhsdfjakhdflakjdfhaldkjfhaldfjkh", "employee_count":15, "type":"Corporations"}`)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost:9083/api/v1/company", bytes.NewBuffer(reqBody))
		assert.NoError(t, err)
		req.Header.Add("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer func() {
			_ = resp.Body.Close()
		}()
		respBody, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)

		// test result
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Equal(t, `{"error":"Invalid name"}`, string(respBody))
	})

	t.Run("error_auth", func(t *testing.T) {
		ctx := context.Background()
		log := zerolog.New(os.Stdout).With().Timestamp().Logger()

		// mock db
		dbConn := mocks.NewStorageInt(t)

		// mock kafka
		kafkaMock := mocks.NewNotifyInt(t)

		// real jwt
		jwtAuth, err := auth.New(jwtKey)
		assert.NoError(t, err)
		token, err := jwtAuth.Generate([]string{"reader"})
		assert.NoError(t, err)

		// start server
		api := api.New(&log, dbConn, jwtAuth, kafkaMock)
		go func() { _ = api.Run(":9084") }()
		defer api.Close()

		// make request
		reqBody := []byte(`{"name":"newcompanydfhsdfjakhdflakjdfhaldkjfhaldfjkh", "employee_count":15, "type":"Corporations"}`)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost:9084/api/v1/company", bytes.NewBuffer(reqBody))
		assert.NoError(t, err)
		req.Header.Add("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer func() {
			_ = resp.Body.Close()
		}()
		respBody, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)

		// test result
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		assert.Equal(t, `{"error":"Access denied"}`, string(respBody))
	})
}
