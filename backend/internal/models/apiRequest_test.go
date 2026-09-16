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
