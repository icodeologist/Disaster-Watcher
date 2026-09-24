package models

import "testing"

func TestCreateReportRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		request CreateReportRequest
		wantErr bool
	}{
		{
			name: "valid",
			request: CreateReportRequest{
				Title: "Flood near station", Description: "Water is entering homes", Location: "Central Station",
				Category: "flood", Priority: "high",
			},
		},
		{
			name: "unknown category",
			request: CreateReportRequest{
				Title: "Flood near station", Description: "Water is entering homes", Location: "Central Station",
				Category: "volcano", Priority: "high",
			},
			wantErr: true,
		},
		{
			name: "unknown priority",
			request: CreateReportRequest{
				Title: "Flood near station", Description: "Water is entering homes", Location: "Central Station",
				Category: "flood", Priority: "urgent",
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.request.Validate(); (err != nil) != test.wantErr {
				t.Fatalf("Validate() error = %v, want error = %v", err, test.wantErr)
			}
		})
	}
}

func TestUserRequestsRejectBlankRequiredFields(t *testing.T) {
	if err := (UserRegistrationRequest{Username: "   ", Location: "London"}).Validate(); err == nil {
		t.Fatal("blank username should be rejected")
	}
	if err := (UserLoginRequest{Email: "   ", Password: "password"}).Validate(); err == nil {
		t.Fatal("blank email should be rejected")
	}
}

func TestUserRequestValidationRejectsBoundaryViolations(t *testing.T) {
	validRegistration := UserRegistrationRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "correct horse",
		Location: "London",
	}
	registrationCases := []struct {
		name   string
		mutate func(*UserRegistrationRequest)
	}{
		{name: "invalid email", mutate: func(request *UserRegistrationRequest) { request.Email = "not-an-email" }},
		{name: "long email", mutate: func(request *UserRegistrationRequest) { request.Email = string(make([]byte, 255)) + "@example.com" }},
		{name: "short password", mutate: func(request *UserRegistrationRequest) { request.Password = "short" }},
		{name: "long password", mutate: func(request *UserRegistrationRequest) {
			request.Password = "1234567890123456789012345678901234567890123456789012345678901234567890123"
		}},
		{name: "blank location", mutate: func(request *UserRegistrationRequest) { request.Location = "   " }},
	}
	for _, test := range registrationCases {
		t.Run(test.name, func(t *testing.T) {
			request := validRegistration
			test.mutate(&request)
			if err := request.Validate(); err == nil {
				t.Fatal("Validate() accepted an invalid registration request")
			}
		})
	}

	if err := (UserLoginRequest{Email: "alice@example.com", Password: "short"}).Validate(); err == nil {
		t.Fatal("Validate() accepted a short login password")
	}
}

func TestCreateReportRequestValidationRejectsLengthBoundaries(t *testing.T) {
	valid := CreateReportRequest{
		Title:       "Flood near station",
		Description: "Water is entering homes",
		Location:    "Central Station",
		Category:    "flood",
		Priority:    "high",
	}
	cases := []struct {
		name   string
		mutate func(*CreateReportRequest)
	}{
		{name: "short title", mutate: func(request *CreateReportRequest) { request.Title = "short" }},
		{name: "short description", mutate: func(request *CreateReportRequest) { request.Description = "short" }},
		{name: "blank location", mutate: func(request *CreateReportRequest) { request.Location = "   " }},
		{name: "long title", mutate: func(request *CreateReportRequest) { request.Title = string(make([]byte, 201)) }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			request := valid
			test.mutate(&request)
			if err := request.Validate(); err == nil {
				t.Fatal("Validate() accepted an invalid report request")
			}
		})
	}
}
