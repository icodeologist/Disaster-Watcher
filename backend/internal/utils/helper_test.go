package utils

import (
	"context"
	"fmt"
	"testing"

	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/stretchr/testify/assert"
)

func floatPtr(f float64) *float64 {
	return &f
}

func TestCachedUserCords(t *testing.T) {
	tests := []struct {
		name         string
		user         *models.User
		geocode      func(context.Context, string) (*models.Location, error)
		expectError  bool
		expectCached bool
		expectCalls  int
	}{
		{
			name: "already cached skips geocoding",
			user: &models.User{
				Location:       "Tokyo",
				LocationCached: true,
				CachedLat:      floatPtr(35.6895),
				CachedLong:     floatPtr(139.6917),
			},
			geocode: func(context.Context, string) (*models.Location, error) {
				return nil, fmt.Errorf("geocoder should not be called")
			},
			expectCached: true,
		},
		{
			name: "successful geocoding caches coordinates",
			user: &models.User{Location: "London"},
			geocode: func(context.Context, string) (*models.Location, error) {
				return &models.Location{Lat: 51.5072, Long: -0.1276}, nil
			},
			expectCached: true,
			expectCalls:  1,
		},
		{
			name: "geocoding failure leaves coordinates uncached",
			user: &models.User{Location: ""},
			geocode: func(context.Context, string) (*models.Location, error) {
				return nil, fmt.Errorf("location cannot be empty")
			},
			expectError: true,
			expectCalls: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			lookup := func(ctx context.Context, location string) (*models.Location, error) {
				calls++
				return test.geocode(ctx, location)
			}

			err := cachedUserCords(context.Background(), test.user, lookup)
			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, test.expectCached, test.user.LocationCached)
			assert.Equal(t, test.expectCalls, calls)

			if test.expectCached {
				assert.NotNil(t, test.user.CachedLat)
				assert.NotNil(t, test.user.CachedLong)
			} else {
				assert.Nil(t, test.user.CachedLat)
				assert.Nil(t, test.user.CachedLong)
			}
		})
	}
}
