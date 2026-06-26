package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReversegeocoding(t *testing.T) {
	tests := []struct {
		name          string
		latitude      float64
		longitude     float64
		expectederror bool
	}{
		{
			name:          "happy path",
			latitude:      40.7128,
			longitude:     -74.0060,
			expectederror: false,
		},
		{
			name:          "empty long and lats",
			latitude:      0.0,
			longitude:     0.0,
			expectederror: true,
		}, {
			name:          "gibberish and invalid input",
			latitude:      3433434.34343434,
			longitude:     -123213123123.12321,
			expectederror: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			locationname, err := ReverseGeocoding(tt.latitude, tt.longitude)

			if tt.expectederror {
				assert.Error(t, err, "expected an error but got nil")
				assert.Empty(t, locationname, "expected emtpy location when there was error")
			} else {
				assert.NoError(t, err, "expected no error but got : %v", err)
				assert.NotNil(t, locationname, "expected a location string but got nil")
			}
		})
	}
}
