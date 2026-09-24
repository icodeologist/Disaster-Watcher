package handler

import (
	"context"
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
	ginContext, _ := gin.CreateTestContext(response)
	ginContext.Request = httptest.NewRequest(http.MethodPost, "/reports", strings.NewReader(`{
		"title":"Flood warning",
		"description":"Water is entering homes",
		"location":"Central Station",
		"category":"flood",
		"priority":"high"
	}`))
	ginContext.Set("userId", uint(7))
	server := &Server{
		VerificationChannel: make(chan models.VerificationMessage, 1),
		GeocodeLocation: func(context.Context, string) (*models.Location, error) {
			return &models.Location{Lat: 12.9, Long: 77.6}, nil
		},
	}

	server.CreateReport(ginContext)

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
	ginContext, _ := gin.CreateTestContext(response)
	ginContext.Request = httptest.NewRequest(http.MethodPost, "/reports", strings.NewReader(`{
		"title":"Flood warning",
		"description":"Water is entering homes",
		"location":"Central Station",
		"category":"flood",
		"priority":"high"
	}`))
	ginContext.Set("userId", uint(7))
	server := &Server{
		VerificationChannel: make(chan models.VerificationMessage, 1),
		GeocodeLocation: func(context.Context, string) (*models.Location, error) {
			return &models.Location{Lat: 12.9, Long: 77.6}, nil
		},
	}

	server.CreateReport(ginContext)

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

func TestCreateReportRejectsInvalidInputBeforeDatabaseAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(response)
	ginContext.Request = httptest.NewRequest(http.MethodPost, "/reports", strings.NewReader(`{
		"title":"short",
		"description":"too short",
		"location":"London",
		"category":"volcano",
		"priority":"urgent"
	}`))

	server := &Server{VerificationChannel: make(chan models.VerificationMessage, 1)}
	server.CreateReport(ginContext)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if strings.Contains(response.Body.String(), "volcano") || strings.Contains(response.Body.String(), "sql") {
		t.Fatalf("response leaked request or infrastructure details: %s", response.Body.String())
	}
}

func TestCreateReportRejectsServerOwnedFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(response)
	ginContext.Request = httptest.NewRequest(http.MethodPost, "/reports", strings.NewReader(`{
		"title":"Flood warning",
		"description":"Water is entering homes",
		"location":"Central Station",
		"category":"flood",
		"priority":"high",
		"status":"verified",
		"userid":999
	}`))

	server := &Server{VerificationChannel: make(chan models.VerificationMessage, 1)}
	server.CreateReport(ginContext)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if strings.Contains(response.Body.String(), "status") || strings.Contains(response.Body.String(), "userid") {
		t.Fatalf("response exposed server-owned fields: %s", response.Body.String())
	}
}
