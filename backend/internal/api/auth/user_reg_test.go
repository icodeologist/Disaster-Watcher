package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUserRegistrationRejectsInvalidInputBeforeDatabaseAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(response)
	ginContext.Request = httptest.NewRequest(http.MethodPost, "/user/register", strings.NewReader(`{
		"username":"ab",
		"email":"not-an-email",
		"password":"short",
		"location":""
	}`))

	UserRegistration(ginContext)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if strings.Contains(response.Body.String(), "validator") || strings.Contains(response.Body.String(), "password") {
		t.Fatalf("response leaked validation details: %s", response.Body.String())
	}
}

func TestUserLoginRejectsInvalidInputBeforeDatabaseAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(response)
	ginContext.Request = httptest.NewRequest(http.MethodPost, "/user/login", strings.NewReader(`{
		"email":"not-an-email",
		"password":"short"
	}`))

	UserLogin(ginContext)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if strings.Contains(response.Body.String(), "validator") || strings.Contains(response.Body.String(), "password") {
		t.Fatalf("response leaked validation details: %s", response.Body.String())
	}
}

func TestUserRegistrationRejectsUnknownServerFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	ginContext, _ := gin.CreateTestContext(response)
	ginContext.Request = httptest.NewRequest(http.MethodPost, "/user/register", strings.NewReader(`{
		"username":"alice",
		"email":"alice@example.com",
		"password":"correct horse",
		"location":"London",
		"isAdmin":true
	}`))

	UserRegistration(ginContext)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if strings.Contains(response.Body.String(), "isAdmin") {
		t.Fatalf("response exposed server-owned field: %s", response.Body.String())
	}
}
