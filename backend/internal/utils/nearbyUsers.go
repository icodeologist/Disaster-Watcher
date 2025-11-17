package utils

import (
	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
)

func GetNearbyUsers(atLat float64, atLong float64, radiusInKm float64) ([]uint, error) {
	// user lat long should be comapared with all the users in the db
	var nearbyUserIDS []uint
	var allUsers []models.User
	if err := db.DB.Find(&allUsers).Error; err != nil {
		return nil, err
	}
	for _, user := range allUsers {
		userLat := user.CachedLat
		userLong := user.CachedLong
		// User without valid location will be skipped
		if user.LocationCached == false {
			continue
		}
		radius := Haversine(atLat, atLong, *userLat, *userLong)
		if radius <= radiusInKm {
			nearbyUserIDS = append(nearbyUserIDS, user.ID)
		}
	}
	return nearbyUserIDS, nil
}

// Send the report to these users first
func NearByTrustedUsers(nearbyUserIDS []uint) ([]uint, error) {
	var trustedUserIDS []uint
	for _, userID := range nearbyUserIDS {
		//TODO: if the trustscore > 10 then verify that report
		if err := db.DB.Where("id= ? AND trustscore = ?", userID, 0).First(&models.User{}).Error; err == nil {
			trustedUserIDS = append(trustedUserIDS, userID)
		} else {
			return nil, err
		}
	}
	return trustedUserIDS, nil
}
