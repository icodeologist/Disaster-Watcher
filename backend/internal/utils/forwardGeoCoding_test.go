package utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLATLONGfromUserLocation(t *testing.T) {
	tests := []struct {
		name        string
		status      int
		body        string
		expectError bool
	}{
		{
			name:   "valid coordinates",
			status: http.StatusOK,
			body:   `[{"lat":"35.6762","lon":"139.6503"}]`,
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
			name:        "empty result",
			status:      http.StatusOK,
			body:        `[]`,
			expectError: true,
		},
		{
			name:        "invalid coordinate",
			status:      http.StatusOK,
			body:        `[{"lat":"north","lon":"139.6503"}]`,
			expectError: true,
		},
		{
			name:        "coordinate outside valid range",
			status:      http.StatusOK,
			body:        `[{"lat":"91","lon":"139.6503"}]`,
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Tokyo", r.URL.Query().Get("q"))
				assert.Equal(t, "json", r.URL.Query().Get("format"))
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()

			location, err := getLATLONGfromUserLocation(context.Background(), server.Client(), server.URL, "Tokyo")
			if test.expectError {
				assert.Error(t, err)
				assert.Nil(t, location)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, location)
			assert.InDelta(t, 35.6762, location.Lat, 0.0001)
			assert.InDelta(t, 139.6503, location.Long, 0.0001)
		})
	}
}

func TestGetLATLONGfromUserLocationRejectsEmptyLocation(t *testing.T) {
	location, err := getLATLONGfromUserLocation(context.Background(), http.DefaultClient, "http://unused.test", "")

	assert.Error(t, err)
	assert.Nil(t, location)
}

func TestGetLATLONGfromUserLocationTimesOut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(time.Second):
		case <-r.Context().Done():
		}
	}))
	defer server.Close()

	client := &http.Client{Timeout: 20 * time.Millisecond}
	location, err := getLATLONGfromUserLocation(context.Background(), client, server.URL, "Tokyo")

	assert.Error(t, err)
	assert.Nil(t, location)
}

func TestGetLATLONGfromUserLocationLimitsResponseSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", maxGeocodingResponseBytes+1)))
	}))
	defer server.Close()

	location, err := getLATLONGfromUserLocation(context.Background(), server.Client(), server.URL, "Tokyo")

	assert.Error(t, err)
	assert.Nil(t, location)
}
