package models

import (
	"fmt"
	"strings"
)

// May be we can get and generate the device token with expo thing
type DeviceTokenRequest struct {
	UserID uint   `json:"userId"`
	Token  string `json:"token"`
}

type UserRegistrationRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Location string `json:"location" binding:"required,min=2,max=255"`
}

type UserLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

// Validate checks the required registration values that tags cannot express.
func (request UserRegistrationRequest) Validate() error {
	if strings.TrimSpace(request.Username) == "" || strings.TrimSpace(request.Location) == "" {
		return fmt.Errorf("username and location are required")
	}
	return nil
}

// Validate checks that login has both credentials before touching the database.
func (request UserLoginRequest) Validate() error {
	if strings.TrimSpace(request.Email) == "" || request.Password == "" {
		return fmt.Errorf("email and password are required")
	}
	return nil
}

type CreateReportRequest struct {
	Title       string `json:"title" binding:"required,min=10,max=200"`
	Description string `json:"description" binding:"required,min=10,max=5000"`
	Location    string `json:"location" binding:"required,min=2,max=255"`
	Category    string `json:"category" binding:"required"`
	Priority    string `json:"priority" binding:"required"`
}

// Validate checks that a report uses one of the supported categories and priorities.
func (request CreateReportRequest) Validate() error {
	if strings.TrimSpace(request.Title) == "" ||
		strings.TrimSpace(request.Description) == "" ||
		strings.TrimSpace(request.Location) == "" {
		return fmt.Errorf("title, description, and location are required")
	}
	validCategories := map[string]struct{}{
		"flood": {}, "animals": {}, "landslide": {}, "rain": {}, "electricity outage": {},
	}
	validPriorities := map[string]struct{}{
		"critical": {}, "high": {}, "medium": {}, "low": {},
	}
	if _, ok := validCategories[strings.ToLower(strings.TrimSpace(request.Category))]; !ok {
		return fmt.Errorf("category is not supported")
	}
	if _, ok := validPriorities[strings.ToLower(strings.TrimSpace(request.Priority))]; !ok {
		return fmt.Errorf("priority is not supported")
	}
	return nil
}
