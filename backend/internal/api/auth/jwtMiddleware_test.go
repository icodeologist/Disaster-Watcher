package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/icodeologist/disasterwatch/internal/db"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestAuthMiddlewareDoesNotExposeDatabaseErrors(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sql mock: %v", err)
	}
	defer sqlDB.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatalf("open mocked gorm database: %v", err)
	}
	previousDB := db.DB
	db.DB = gormDB
	t.Cleanup(func() { db.DB = previousDB })
	t.Setenv("SECRET", "test-secret")

	mock.ExpectQuery(`SELECT .* FROM "users"`).
		WithArgs(uint(7)).
		WillReturnError(errors.New("database password should not be exposed"))
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  float64(7),
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}

	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(response)
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/api/reports", nil)
	ginContext.Request.Header.Set("Authorization", "Bearer "+token)

	AuthCheckingMiddleware(ginContext)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if strings.Contains(response.Body.String(), "database password") {
		t.Fatalf("response leaked database error: %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "INTERNAL_ERROR") {
		t.Fatalf("response did not contain stable error code: %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("database expectations: %v", err)
	}
}
