package utils

import (
	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"log"
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

// Users in the distance from the report posted in  radius like 20km?
func GetUsersAffectedByDisaster(reportChan <-chan models.Report, affectedUserIdsChan chan<- uint) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("RECOVERED IN STARTING EXTRACTION WORKER : %v", r)
		}
	}()
	var allUsers []models.User
	if err := db.DB.Find(&allUsers).Error; err != nil {
		log.Printf("Error : %v\n", err.Error())
	}
	for report := range reportChan {
		for _, user := range allUsers {
			userLat := user.CachedLat
			userLong := user.CachedLong
			if !user.LocationCached || !report.ISLocationCached {
				continue
			}
			radius := Haversine(*report.CachedLat, *report.CachedLong, *userLat, *userLong)
			if radius <= 20 {
				affectedUserIdsChan <- user.ID
			}
		}
	}
}
