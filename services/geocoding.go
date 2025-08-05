package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/icodeologist/disasterwatch/models"
)

func ForwardGeoCoding(location string) (*models.Location, error) {

	//
	//check if the location is empty
	if location == "" {
		err := fmt.Errorf("Cannot geocode the empty location.")
		return nil, err
	}

	// remove space and other chars from location
	parsedLocation := url.QueryEscape(location)
	url := fmt.Sprintf("https://nominatim.openstreetmap.org/search?q=%v&format=json", parsedLocation)
	fmt.Println("url\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to send the request %v", req)
	}
	req.Header.Set("User-Agent", "DisasterNotifierapp/v1")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Exited while sending the request with the error %v", err)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Failed witht the code %v", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("Failed to read the response body.")
	}

	var geocodingresults []models.GeocodingResult

	err = json.Unmarshal(body, &geocodingresults)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse the incoming request: %v", err)
	}
	fmt.Println("Len of GeocodingResult ", geocodingresults)
	if len(geocodingresults) == 0 {
		return nil, fmt.Errorf("No matching results.")
	}
	// sscanf convertes strings (lat long) to float64
	var lat, long float64

	fmt.Sscanf(geocodingresults[0].Latitude, "%f", &lat)
	fmt.Sscanf(geocodingresults[0].Longitude, "%f", &long)

	return &models.Location{
		Lat:  lat,
		Long: long,
	}, nil

}

func ReverseGeocoding(lat float64, long float64) (string, error) {
	if lat == 0.0 || long == 0.0 {
		return "", fmt.Errorf("Fields cannot be empty")
	}

	url := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?lat=%v&lon=%v&format=json", lat, long)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("Failed to send the request %v", req)
	}
	req.Header.Set("User-Agent", "DisasterNotifierapp/v1")
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Exited while sending the request with the error %v", err)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Failed witht the code %v", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("Failed to read the response body.")
	}

	var result struct {
		LocationName string
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", fmt.Errorf("Cannot parse the incoming response")
	}

	if result.LocationName == "" {
		return "", fmt.Errorf("Village area OR Off limit area")
	}
	return result.LocationName, nil
}
