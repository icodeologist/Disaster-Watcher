package models

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"
)

// May be we can get and generate the device token with expo thing
type DeviceTokenRequest struct {
	UserID uint   `json:"userId"`
	Token  string `json:"token"`
}

type UserRegistrationRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Location string `json:"location" binding:"required,min=2,max=255"`
}

type UserLoginRequest struct {
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

// Validate checks the required registration values that tags cannot express.
func (request UserRegistrationRequest) Validate() error {
	if !validText(request.Username, 3, 50) {
		return fmt.Errorf("username must be between 3 and 50 characters")
	}
	if !validEmail(request.Email) {
		return fmt.Errorf("email is invalid")
	}
	if !validPassword(request.Password) {
		return fmt.Errorf("password must be between 8 and 72 bytes")
	}
	if !validText(request.Location, 2, 255) {
		return fmt.Errorf("location must be between 2 and 255 characters")
	}
	return nil
}

// Validate checks that login has both credentials before touching the database.
func (request UserLoginRequest) Validate() error {
	if !validEmail(request.Email) {
		return fmt.Errorf("email is invalid")
	}
	if !validPassword(request.Password) {
		return fmt.Errorf("password must be between 8 and 72 bytes")
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
	if !validText(request.Title, 10, 200) {
		return fmt.Errorf("title must be between 10 and 200 characters")
	}
	if !validText(request.Description, 10, 5000) {
		return fmt.Errorf("description must be between 10 and 5000 characters")
	}
	if !validText(request.Location, 2, 255) {
		return fmt.Errorf("location must be between 2 and 255 characters")
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

func validText(value string, minimum int, maximum int) bool {
	value = strings.TrimSpace(value)
	length := utf8.RuneCountInString(value)
	return length >= minimum && length <= maximum
}

func validEmail(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) > 254 {
		return false
	}
	parsed, err := mail.ParseAddress(value)
	return err == nil && parsed.Address == value && strings.Contains(value, "@")
}

func validPassword(value string) bool {
	return len(value) >= 8 && len(value) <= 72
}
