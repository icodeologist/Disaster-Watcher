package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/icodeologist/disasterwatch/internal/models"
)

const (
	nominatimSearchEndpoint     = "https://nominatim.openstreetmap.org/search"
	nominatimReverseEndpoint    = "https://nominatim.openstreetmap.org/reverse"
	maxGeocodingResponseBytes   = 1 << 20
	defaultGeocodingHTTPTimeout = 5 * time.Second
)

var defaultGeocodingClient = &http.Client{Timeout: defaultGeocodingHTTPTimeout}

func GetLATLONGfromUserLocation(ctx context.Context, location string) (*models.Location, error) {
	return getLATLONGfromUserLocation(ctx, defaultGeocodingClient, nominatimSearchEndpoint, location)
}

func getLATLONGfromUserLocation(ctx context.Context, client *http.Client, endpoint string, location string) (*models.Location, error) {
	if location == "" {
		return nil, fmt.Errorf("location cannot be empty")
	}

	requestURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse geocoding endpoint: %w", err)
	}
	query := requestURL.Query()
	query.Set("q", location)
	query.Set("format", "json")
	requestURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create geocoding request: %w", err)
	}
	req.Header.Set("User-Agent", "DisasterNotifierapp/v1")

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send geocoding request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("geocoding provider returned status %d", res.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, maxGeocodingResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read geocoding response: %w", err)
	}
	if len(body) > maxGeocodingResponseBytes {
		return nil, fmt.Errorf("geocoding response exceeds %d bytes", maxGeocodingResponseBytes)
	}

	var geocodingresults []models.GeocodingResult
	if err := json.Unmarshal(body, &geocodingresults); err != nil {
		return nil, fmt.Errorf("parse geocoding response: %w", err)
	}
	if len(geocodingresults) == 0 {
		return nil, fmt.Errorf("no matching geocoding result")
	}

	lat, err := strconv.ParseFloat(geocodingresults[0].Latitude, 64)
	if err != nil {
		return nil, fmt.Errorf("parse latitude: %w", err)
	}
	long, err := strconv.ParseFloat(geocodingresults[0].Longitude, 64)
	if err != nil {
		return nil, fmt.Errorf("parse longitude: %w", err)
	}
	if math.IsNaN(lat) || math.IsInf(lat, 0) || lat < -90 || lat > 90 {
		return nil, fmt.Errorf("latitude is outside the valid range")
	}
	if math.IsNaN(long) || math.IsInf(long, 0) || long < -180 || long > 180 {
		return nil, fmt.Errorf("longitude is outside the valid range")
	}

	return &models.Location{
		Lat:  lat,
		Long: long,
	}, nil
}

func ReverseGeocoding(ctx context.Context, lat float64, long float64) (string, error) {
	return reverseGeocoding(ctx, defaultGeocodingClient, nominatimReverseEndpoint, lat, long)
}

func reverseGeocoding(ctx context.Context, client *http.Client, endpoint string, lat float64, long float64) (string, error) {
	if math.IsNaN(lat) || math.IsInf(lat, 0) || lat < -90 || lat > 90 {
		return "", fmt.Errorf("latitude is outside the valid range")
	}
	if math.IsNaN(long) || math.IsInf(long, 0) || long < -180 || long > 180 {
		return "", fmt.Errorf("longitude is outside the valid range")
	}

	requestURL, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("parse reverse-geocoding endpoint: %w", err)
	}
	query := requestURL.Query()
	query.Set("lat", strconv.FormatFloat(lat, 'f', -1, 64))
	query.Set("lon", strconv.FormatFloat(long, 'f', -1, 64))
	query.Set("format", "json")
	requestURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create reverse-geocoding request: %w", err)
	}
	req.Header.Set("User-Agent", "DisasterNotifierapp/v1")

	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send reverse-geocoding request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("reverse-geocoding provider returned status %d", res.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, maxGeocodingResponseBytes+1))
	if err != nil {
		return "", fmt.Errorf("read reverse-geocoding response: %w", err)
	}
	if len(body) > maxGeocodingResponseBytes {
		return "", fmt.Errorf("reverse-geocoding response exceeds %d bytes", maxGeocodingResponseBytes)
	}

	var result struct {
		DisplayName string `json:"display_name"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse reverse-geocoding response: %w", err)
	}

	if result.DisplayName == "" {
		return "", fmt.Errorf("reverse-geocoding response has no display name")
	}
	return result.DisplayName, nil
}
