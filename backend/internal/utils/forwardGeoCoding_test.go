package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLATLONGfromUserLocation(t *testing.T) {
	// This is the "Table". Each row is a different scenario.
	tests := []struct {
		name        string
		location    string
		expectError bool
	}{
		{
			name:        "Happy Path: Valid City",
			location:    "Tokyo", // A real city that will definitely return results
			expectError: false,
		},
		{
			name:        "Edge Case: Empty String",
			location:    "",
			expectError: true, // Your code explicitly checks for this
		},
		{
			name:        "Edge Case: Gibberish (No matching results)",
			location:    "xyzabc123nonexistentcity987qwerty",
			expectError: true, // The API will return an empty array, triggering your len == 0 check
		},
	}

	// This loop runs every scenario in the table
	for _, tt := range tests {
		// t.Run creates a named subtest. If one fails, you'll know exactly which one!
		t.Run(tt.name, func(t *testing.T) {

			// Call your actual function
			loc, err := GetLATLONGfromUserLocation(tt.location)

			// Assert the results based on what we expected
			if tt.expectError {
				assert.Error(t, err, "Expected an error but got nil")
				assert.Nil(t, loc, "Expected nil location when there is an error")
			} else {
				assert.NoError(t, err, "Expected no error but got: %v", err)
				assert.NotNil(t, loc, "Expected a location object but got nil")

				// Just a sanity check to ensure it actually parsed some numbers
				assert.NotZero(t, loc.Lat, "Latitude should not be zero")
				assert.NotZero(t, loc.Long, "Longitude should not be zero")
			}
		})
	}
}
