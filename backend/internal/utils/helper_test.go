// test for cachedCoora function
package utils

import (
	"testing"

	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/stretchr/testify/assert"
)

// Helper function to easily create pointers to floats for our test table
func floatPtr(f float64) *float64 {
	return &f
}

func TestCachedUserCords(t *testing.T) {
	tests := []struct {
		name         string
		user         *models.User
		expectError  bool
		expectCached bool // What we expect LocationCached to be AFTER the function runs
	}{
		{
			name: "Already cached (Early Exit)",
			user: &models.User{
				Location:       "Tokyo",
				LocationCached: true,
				CachedLat:      floatPtr(35.6895),
				CachedLong:     floatPtr(139.6917),
			},
			expectError:  false,
			expectCached: true,
		},
		{
			name: "Not cached, Valid Location",
			user: &models.User{
				Location: "London", // We know this works from our last test!
			},
			expectError:  false,
			expectCached: true, // The function should set this to true
		},
		{
			name: "Not cached, Invalid Location",
			user: &models.User{
				Location: "", // We know this triggers an error
			},
			expectError:  true,
			expectCached: false, // Should remain false because it failed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Call the function
			err := CachedUserCords(tt.user)

			// 1. Check the error
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// 2. Check if the caching state was updated correctly
			assert.Equal(t, tt.expectCached, tt.user.LocationCached, "LocationCached boolean mismatch")

			// 3. If it was supposed to be cached, ensure the pointers aren't nil
			if tt.expectCached {
				assert.NotNil(t, tt.user.CachedLat, "CachedLat should not be nil")
				assert.NotNil(t, tt.user.CachedLong, "CachedLong should not be nil")
			}
		})
	}
}
