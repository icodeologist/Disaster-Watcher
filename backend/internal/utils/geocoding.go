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
	"strings"
	"time"
	"unicode"

	"github.com/icodeologist/disasterwatch/internal/models"
)

const (
	nominatimSearchEndpoint     = "https://nominatim.openstreetmap.org/search"
	nominatimReverseEndpoint    = "https://nominatim.openstreetmap.org/reverse"
	maxGeocodingResponseBytes   = 1 << 20
	defaultGeocodingHTTPTimeout = 5 * time.Second
	geocodingCandidateLimit     = 5
	minimumLocationMatchScore   = 0.80
	minimumMatchScoreDifference = 0.10
)

var defaultGeocodingClient = &http.Client{Timeout: defaultGeocodingHTTPTimeout}

func GetLATLONGfromUserLocation(ctx context.Context, location string) (*models.Location, error) {
	return getLATLONGfromUserLocation(ctx, defaultGeocodingClient, nominatimSearchEndpoint, location)
}

func getLATLONGfromUserLocation(ctx context.Context, client *http.Client, endpoint string, location string) (*models.Location, error) {
	normalizedLocation := normalizeLocation(location)
	if normalizedLocation == "" {
		return nil, fmt.Errorf("location cannot be empty")
	}

	requestURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse geocoding endpoint: %w", err)
	}
	query := requestURL.Query()
	query.Set("q", normalizedLocation)
	query.Set("format", "json")
	query.Set("limit", strconv.Itoa(geocodingCandidateLimit))
	query.Set("addressdetails", "1")
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

	type scoredCandidate struct {
		location *models.Location
		score    float64
	}

	candidates := make([]scoredCandidate, 0, len(geocodingresults))
	for _, result := range geocodingresults {
		lat, long, err := parseGeocodingCoordinates(result)
		if err != nil {
			continue
		}

		score := 0.0
		for _, label := range []string{result.Name, result.DisplayName, result.LocationName} {
			if label != "" {
				score = math.Max(score, locationMatchScore(normalizedLocation, label))
			}
		}
		if score == 0 {
			continue
		}

		candidates = append(candidates, scoredCandidate{
			location: &models.Location{Lat: lat, Long: long},
			score:    score,
		})
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no valid geocoding result")
	}

	best := candidates[0]
	secondBestScore := 0.0
	for _, candidate := range candidates[1:] {
		if candidate.score > best.score {
			secondBestScore = best.score
			best = candidate
			continue
		}
		if candidate.score > secondBestScore {
			secondBestScore = candidate.score
		}
	}

	if best.score < minimumLocationMatchScore {
		return nil, fmt.Errorf("no sufficiently similar geocoding result")
	}
	if secondBestScore >= minimumLocationMatchScore && best.score-secondBestScore < minimumMatchScoreDifference {
		return nil, fmt.Errorf("ambiguous geocoding result")
	}

	return best.location, nil
}

func parseGeocodingCoordinates(result models.GeocodingResult) (float64, float64, error) {
	lat, err := strconv.ParseFloat(result.Latitude, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse latitude: %w", err)
	}
	long, err := strconv.ParseFloat(result.Longitude, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse longitude: %w", err)
	}
	if math.IsNaN(lat) || math.IsInf(lat, 0) || lat < -90 || lat > 90 {
		return 0, 0, fmt.Errorf("latitude is outside the valid range")
	}
	if math.IsNaN(long) || math.IsInf(long, 0) || long < -180 || long > 180 {
		return 0, 0, fmt.Errorf("longitude is outside the valid range")
	}
	return lat, long, nil
}

func normalizeLocation(location string) string {
	var normalized strings.Builder
	spacePending := false

	for _, char := range strings.ToLower(strings.TrimSpace(location)) {
		switch {
		case unicode.IsLetter(char) || unicode.IsDigit(char):
			if spacePending && normalized.Len() > 0 {
				normalized.WriteByte(' ')
			}
			normalized.WriteRune(char)
			spacePending = false
		case !spacePending:
			spacePending = true
		}
	}

	return normalized.String()
}

func locationMatchScore(query string, candidate string) float64 {
	query = normalizeLocation(query)
	candidate = normalizeLocation(candidate)
	if query == "" || candidate == "" {
		return 0
	}
	if query == candidate {
		return 1
	}

	queryTokens := strings.Fields(query)
	candidateTokens := strings.Fields(candidate)
	tokenScore := 0.0
	for _, queryToken := range queryTokens {
		bestTokenScore := 0.0
		for _, candidateToken := range candidateTokens {
			bestTokenScore = math.Max(bestTokenScore, stringSimilarity(queryToken, candidateToken))
		}
		tokenScore += bestTokenScore
	}
	tokenScore /= float64(len(queryTokens))

	return math.Max(tokenScore, stringSimilarity(query, candidate))
}

func stringSimilarity(first string, second string) float64 {
	firstRunes := []rune(first)
	secondRunes := []rune(second)
	if len(firstRunes) == 0 && len(secondRunes) == 0 {
		return 1
	}
	if len(firstRunes) == 0 || len(secondRunes) == 0 {
		return 0
	}

	previous := make([]int, len(secondRunes)+1)
	for column := range previous {
		previous[column] = column
	}

	for row, firstRune := range firstRunes {
		current := make([]int, len(secondRunes)+1)
		current[0] = row + 1
		for column, secondRune := range secondRunes {
			cost := 0
			if firstRune != secondRune {
				cost = 1
			}
			current[column+1] = minInt(
				current[column]+1,
				previous[column+1]+1,
				previous[column]+cost,
			)
		}
		previous = current
	}

	distance := previous[len(secondRunes)]
	longest := len(firstRunes)
	if len(secondRunes) > longest {
		longest = len(secondRunes)
	}
	return 1 - float64(distance)/float64(longest)
}

func minInt(values ...int) int {
	minimum := values[0]
	for _, value := range values[1:] {
		if value < minimum {
			minimum = value
		}
	}
	return minimum
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
