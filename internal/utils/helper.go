package utils

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/redis/go-redis/v9"
)

// CachedUserCords checks if the user location is already cached
// if not it calls the ForwardGeoCoding function to get the lat long from the user location
// and updates the user model with the cached lat long and sets the LocationCached to true
// TODO: add redis cashing for already updated locaitons
func CachedUserCords(user *models.User) error {
	if user.LocationCached && user.CachedLat != nil && user.CachedLong != nil {
		return nil
	}
	// calling notinatim api
	location, err := GetLATLONGfromUserLocation(user.Location)
	if err != nil {
		return fmt.Errorf(" ERROR : %v", err)
	}
	user.CachedLat = &location.Lat
	user.CachedLong = &location.Long
	user.LocationCached = true
	db.DB.Save(&user)
	return nil
}

// Haversine formula to calculate the distance between two lat long points
// returns distance in kilometers
func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth's radius in kilometers
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	lat1 = lat1 * math.Pi / 180
	lat2 = lat2 * math.Pi / 180
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

// redis helper func setup
func Redishelper(client *redis.Client, reportId uint) error {
	ctx := context.Background()
	err := client.Set(ctx, "report_id", 1, 10*time.Minute)
	if err != nil {
		return nil
	}
	fmt.Printf("Report[%v]added to redis", reportId)
	return nil
}
