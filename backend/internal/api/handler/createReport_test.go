package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	database "github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestCreateReportRollsBackWhenJobCreationFails(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer sqlDB.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open mocked gorm database: %v", err)
	}
	previousDB := database.DB
	database.DB = gormDB
	t.Cleanup(func() { database.DB = previousDB })

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "reports"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(41))
	mock.ExpectQuery(`INSERT INTO "jobs"`).
		WillReturnError(errors.New("job insert failed"))
	mock.ExpectRollback()

	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodPost, "/reports", strings.NewReader(`{
		"title":"Flood warning",
		"description":"Water is entering homes",
		"location":"Central Station",
		"category":"flood",
		"priority":"high",
		"is-cached":true,
		"latitude":12.9,
		"longitude":77.6
	}`))
	context.Set("userId", uint(7))
	server := &Server{
		VerificationChannel: make(chan models.VerificationMessage, 1),
	}

	server.CreateReport(context)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	select {
	case message := <-server.VerificationChannel:
		t.Fatalf("verification message sent after rollback: %+v", message)
	default:
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}

func TestCreateReportRollsBackWhenReportCreationFails(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer sqlDB.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open mocked gorm database: %v", err)
	}
	previousDB := database.DB
	database.DB = gormDB
	t.Cleanup(func() { database.DB = previousDB })

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "reports"`).
		WillReturnError(errors.New("report insert failed"))
	mock.ExpectRollback()

	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(response)
	context.Request = httptest.NewRequest(http.MethodPost, "/reports", strings.NewReader(`{
		"title":"Flood warning",
		"description":"Water is entering homes",
		"location":"Central Station",
		"category":"flood",
		"priority":"high",
		"is-cached":true,
		"latitude":12.9,
		"longitude":77.6
	}`))
	context.Set("userId", uint(7))
	server := &Server{
		VerificationChannel: make(chan models.VerificationMessage, 1),
	}

	server.CreateReport(context)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	select {
	case message := <-server.VerificationChannel:
		t.Fatalf("verification message sent after rollback: %+v", message)
	default:
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}
