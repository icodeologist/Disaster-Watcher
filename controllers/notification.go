package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"

	"github.com/icodeologist/disasterwatch/database"
	"github.com/icodeologist/disasterwatch/models"
)

type DisasterAlert struct {
	Type        string
	Severity    string
	Location    string
	Description string
	Latitude    float64
	Longitude   float64
	Distance    float64
	ReportID    uint
}

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

// checking for cached location to avoid repeated reqeusts

func CachedUserCords(user *models.User) error {
	//user already has cached results
	if user.LocationCached && user.CachedLat != nil && user.CachedLong != nil {
		return nil
	}
	location, err := ForwardGeoCoding(user.Location)
	if err != nil {
		return fmt.Errorf(" ERROR : %v", err)
	}
	// map the location to user lat longs

	user.CachedLat = &location.Lat
	user.CachedLong = &location.Long
	user.LocationCached = true

	database.DB.Save(&user)
	return nil

}

func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth's radius in kilometers

	// Convert degrees to radians
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180

	lat1 = lat1 * math.Pi / 180
	lat2 = lat2 * math.Pi / 180

	// Haversine formula
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1)*math.Cos(lat2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}
