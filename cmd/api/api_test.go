package main

import (
	"bytes"
	"context"
	"database/sql"
	"github.com/rs/zerolog"
	"io"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"

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
		time.Sleep(10 * time.Millisecond)
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
		time.Sleep(10 * time.Millisecond)
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
		time.Sleep(10 * time.Millisecond)
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
		time.Sleep(10 * time.Millisecond)
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

func TestUpdateItem(t *testing.T) {
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
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE companies 
	SET
		name=COALESCE($2, name), 
		description=COALESCE($3, description), 
		employee_count=COALESCE($4, employee_count), 
		is_registered=COALESCE($5, is_registered), 
		legal_type=COALESCE($6, legal_type)
	WHERE id = $1`)).
			WithArgs(id, nil, nil, 1, nil, "Sole Proprietorship").WillReturnResult(sqlmock.NewResult(0, 1))

		// mock kafka
		kafkaMock := mocks.NewNotifyInt(t)
		kafkaMock.On("Send", models.EventNotifications{ID: id, Event: models.EventTypeUpdated}).Return(nil)

		// real jwt
		jwtAuth, err := auth.New(jwtKey)
		assert.NoError(t, err)
		token, err := jwtAuth.Generate([]string{"reader", "writer"})
		assert.NoError(t, err)

		// start server
		api := api.New(&log, dbConn, jwtAuth, kafkaMock)
		go func() { _ = api.Run(":9081") }()
		time.Sleep(10 * time.Millisecond)
		defer api.Close()

		// make request
		reqBody := []byte(`{"employee_count":1, "type":"Sole Proprietorship"}`)
		req, err := http.NewRequestWithContext(ctx, http.MethodPatch, "http://localhost:9081/api/v1/company/"+id.String(), bytes.NewBuffer(reqBody))
		assert.NoError(t, err)
		req.Header.Add("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		_ = resp.Body.Close()

		// test result
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("error_duplicate", func(t *testing.T) {
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
		mock.ExpectExec(regexp.QuoteMeta(`UPDATE companies 
	SET
		name=COALESCE($2, name), 
		description=COALESCE($3, description), 
		employee_count=COALESCE($4, employee_count), 
		is_registered=COALESCE($5, is_registered), 
		legal_type=COALESCE($6, legal_type)
	WHERE id = $1`)).
			WithArgs(id, nil, nil, 1, nil, "Sole Proprietorship").WillReturnError(errDuplicate)

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
		time.Sleep(10 * time.Millisecond)
		defer api.Close()

		// make request
		reqBody := []byte(`{"employee_count":1, "type":"Sole Proprietorship"}`)
		req, err := http.NewRequestWithContext(ctx, http.MethodPatch, "http://localhost:9082/api/v1/company/"+id.String(), bytes.NewBuffer(reqBody))
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
		id := uuid.MustParse("3f00e7f6-f9c8-4a3a-8a27-2bd529b161e4")

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
		time.Sleep(10 * time.Millisecond)
		defer api.Close()

		// make request
		reqBody := []byte(`{"type":"LTD"}`)
		req, err := http.NewRequestWithContext(ctx, http.MethodPatch, "http://localhost:9083/api/v1/company/"+id.String(), bytes.NewBuffer(reqBody))
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
		assert.Equal(t, `{"error":"Invalid type"}`, string(respBody))
	})
}

func TestDeleteItem(t *testing.T) {
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
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM companies WHERE id = $1`)).
			WithArgs(id).WillReturnResult(sqlmock.NewResult(0, 1))

		// mock kafka
		kafkaMock := mocks.NewNotifyInt(t)
		kafkaMock.On("Send", models.EventNotifications{ID: id, Event: models.EventTypeDeleted}).Return(nil)

		// real jwt
		jwtAuth, err := auth.New(jwtKey)
		assert.NoError(t, err)
		token, err := jwtAuth.Generate([]string{"reader", "writer"})
		assert.NoError(t, err)

		// start server
		api := api.New(&log, dbConn, jwtAuth, kafkaMock)
		go func() { _ = api.Run(":9081") }()
		time.Sleep(10 * time.Millisecond)
		defer api.Close()

		// make request
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, "http://localhost:9081/api/v1/company/"+id.String(), http.NoBody)
		assert.NoError(t, err)
		req.Header.Add("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		_ = resp.Body.Close()

		// test result
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("error_not_found", func(t *testing.T) {
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
		mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM companies WHERE id = $1`)).
			WithArgs(id).WillReturnResult(sqlmock.NewResult(0, 1)).WillReturnResult(sqlmock.NewResult(0, 0))

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
		time.Sleep(10 * time.Millisecond)
		defer api.Close()

		// make request
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, "http://localhost:9082/api/v1/company/"+id.String(), http.NoBody)
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
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		assert.Equal(t, `{"error":"Item not found"}`, string(respBody))
	})

}

func TestGetItem(t *testing.T) {
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
		rows := sqlmock.NewRows([]string{"id", "name", "description", "employee_count", "is_registered", "legal_type"})
		rows.AddRow(id.String(), "name", "description", 3, true, "Corporations")
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, description, employee_count, is_registered, legal_type FROM companies WHERE id = $1`)).
			WithArgs(id).WillReturnRows(rows)

		// mock kafka
		kafkaMock := mocks.NewNotifyInt(t)

		// real jwt
		jwtAuth, err := auth.New(jwtKey)
		assert.NoError(t, err)
		token, err := jwtAuth.Generate([]string{"reader", "writer"})
		assert.NoError(t, err)

		// start server
		api := api.New(&log, dbConn, jwtAuth, kafkaMock)
		go func() { _ = api.Run(":9081") }()
		time.Sleep(10 * time.Millisecond)
		defer api.Close()

		// make request
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:9081/api/v1/company/"+id.String(), http.NoBody)
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
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, `{"id":"`+id.String()+`","name":"name","description":"description","employee_count":3,"is_registered":true,"type":"Corporations"}`, string(respBody))
	})

	t.Run("error_not_found", func(t *testing.T) {
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
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, description, employee_count, is_registered, legal_type FROM companies WHERE id = $1`)).
			WithArgs(id).WillReturnError(sql.ErrNoRows)

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
		time.Sleep(10 * time.Millisecond)
		defer api.Close()

		// make request
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:9082/api/v1/company/"+id.String(), http.NoBody)
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
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		assert.Equal(t, `{"error":"Item not found"}`, string(respBody))
	})

}
