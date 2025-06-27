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

func FindUserWithRadius(report *models.Report, users []models.User, radiusinKm float64) error {
	var processedUsers, failedUsers int

	for _, user := range users {
		if err := CachedUserCords(&user); err != nil {
			fmt.Printf("Skipping user ID %d (location: '%s'): %v\n", user.ID, user.Location, err)
			failedUsers++
			continue // Skip this user, process the rest
		}

		if user.CachedLat == nil || user.CachedLong == nil {
			fmt.Printf("Skipping user ID %d: missing cached coordinates\n", user.ID)
			failedUsers++
			continue
		}

		// Fix the parameter order here too
		distance := Haversine(*user.CachedLat, *user.CachedLong, report.Latitude, report.Longitude)

		if distance <= radiusinKm {
			_ = fmt.Sprintf(
				"Disaster Alert: %s reported nearby %s Distance: %.2f km",
				report.Type, user.Location, distance)

		}
		processedUsers++
	}

	fmt.Printf("Processed %d users successfully, %d failed\n", processedUsers, failedUsers)
	return nil
}

func ProcessDisasterReport(report *models.Report, allUsers []models.User) error {
	radius := 10.0
	err := FindUserWithRadius(report, allUsers, radius)
	if err != nil {
		return fmt.Errorf("Failed to get the notifications. %v. ", err)
	}

	fmt.Printf("Found %d users within the distance.", 10)

	if 10 > 0 {
		// fix this later
		// TODO
		for i := 0; i < 10; i++ {
			fmt.Printf("sending notification to user %v", i)
		}
	} else {
		return fmt.Errorf("There were no nearby users.")
	}
	return nil
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
