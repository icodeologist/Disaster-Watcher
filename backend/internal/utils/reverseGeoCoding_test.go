package utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReverseGeocoding(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		body        string
		expectError bool
	}{
		{
			name:   "valid location",
			status: http.StatusOK,
			body:   `{"display_name":"New York, United States"}`,
		},
		{
			name:        "provider error",
			status:      http.StatusServiceUnavailable,
			body:        `service unavailable`,
			expectError: true,
		},
		{
			name:        "malformed payload",
			status:      http.StatusOK,
			body:        `{`,
			expectError: true,
		},
		{
			name:        "missing display name",
			status:      http.StatusOK,
			body:        `{}`,
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "40.7128", r.URL.Query().Get("lat"))
				assert.Equal(t, "-74.006", r.URL.Query().Get("lon"))
				assert.Equal(t, "json", r.URL.Query().Get("format"))
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()

			location, err := reverseGeocoding(context.Background(), server.Client(), server.URL, 40.7128, -74.006)
			if test.expectError {
				assert.Error(t, err)
				assert.Empty(t, location)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, "New York, United States", location)
		})
	}
}

func TestReverseGeocodingRejectsInvalidCoordinates(t *testing.T) {
	location, err := reverseGeocoding(context.Background(), http.DefaultClient, "http://unused.test", 91, -74.006)

	assert.Error(t, err)
	assert.Empty(t, location)
}

func TestReverseGeocodingTimesOut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(time.Second):
		case <-r.Context().Done():
		}
	}))
	defer server.Close()

	client := &http.Client{Timeout: 20 * time.Millisecond}
	location, err := reverseGeocoding(context.Background(), client, server.URL, 40.7128, -74.006)

	assert.Error(t, err)
	assert.Empty(t, location)
}
