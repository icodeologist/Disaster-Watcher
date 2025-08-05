package services

import (
	"fmt"
	"math"

	"github.com/icodeologist/disasterwatch/database"
	"github.com/icodeologist/disasterwatch/models"
)

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
