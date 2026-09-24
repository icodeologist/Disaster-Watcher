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
			body:   `[{"lat":"35.6762","lon":"139.6503","name":"Tokyo","display_name":"Tokyo, Japan"}]`,
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
				assert.Equal(t, "tokyo", r.URL.Query().Get("q"))
				assert.Equal(t, "json", r.URL.Query().Get("format"))
				assert.Equal(t, "5", r.URL.Query().Get("limit"))
				assert.Equal(t, "1", r.URL.Query().Get("addressdetails"))
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

func TestGetLATLONGfromUserLocationFuzzyMatchesBestCandidate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "bangalor", r.URL.Query().Get("q"))
		_, _ = w.Write([]byte(`[
			{"lat":"12.9716","lon":"77.5946","name":"Bangalore","display_name":"Bangalore, Karnataka, India"},
			{"lat":"44.0462","lon":"123.0220","name":"Bangor","display_name":"Bangor, Oregon, United States"}
		]`))
	}))
	defer server.Close()

	location, err := getLATLONGfromUserLocation(context.Background(), server.Client(), server.URL, "  Bangalor! ")

	require.NoError(t, err)
	require.NotNil(t, location)
	assert.InDelta(t, 12.9716, location.Lat, 0.0001)
	assert.InDelta(t, 77.5946, location.Long, 0.0001)
}

func TestGetLATLONGfromUserLocationRejectsAmbiguousCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{"lat":"39.7817","lon":"-89.6501","name":"Springfield","display_name":"Springfield, Illinois, United States"},
			{"lat":"37.2089","lon":"-93.2923","name":"Springfield","display_name":"Springfield, Missouri, United States"}
		]`))
	}))
	defer server.Close()

	location, err := getLATLONGfromUserLocation(context.Background(), server.Client(), server.URL, "Springfield")

	assert.ErrorContains(t, err, "ambiguous geocoding result")
	assert.Nil(t, location)
}

func TestGetLATLONGfromUserLocationRejectsWeakCandidate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[
			{"lat":"48.8566","lon":"2.3522","name":"Paris","display_name":"Paris, France"}
		]`))
	}))
	defer server.Close()

	location, err := getLATLONGfromUserLocation(context.Background(), server.Client(), server.URL, "Tokio")

	assert.ErrorContains(t, err, "no sufficiently similar geocoding result")
	assert.Nil(t, location)
}

func TestNormalizeLocation(t *testing.T) {
	assert.Equal(t, "new york", normalizeLocation("  New-York,  "))
	assert.Equal(t, "são paulo", normalizeLocation("São Paulo"))
	assert.Empty(t, normalizeLocation("---"))
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
