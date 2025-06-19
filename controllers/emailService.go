package controllers

import (
	"fmt"
	"github.com/icodeologist/disasterwatch/database"
	"github.com/icodeologist/disasterwatch/models"
	"log"
	"strings"
)

func getNotificationRadius(reportType string) float64 {
	temp := strings.ToLower(strings.TrimSpace(reportType))
	// there could be other disasters.
	// TODO: Handle this better even for fisrt time disasters
	switch temp {
	case "earthquake":
		return 50.0
	case "flood":
		return 20.0
	case "electricityoutrage":
		return 15.0
	default:
		return 10.0
	}
}

func FindUserAffected(reportID uint) ([]models.User, error) {
	var report models.Report
	err := database.DB.Find(&report)
	if err != nil {
		log.Printf("Could not find the report with given id %v : %v", reportID, err)
	}

	radius := getNotificationRadius(report.Type)
	var affectedUsers []models.User
	var users []models.User
	res := database.DB.Find(&users)
	//there were no users
	if res.RowsAffected == 0 {
		er := fmt.Errorf("There were no registered users.")
		return nil, er
	}
	for _, user := range users {
		if user.LocationCached {
			distance := Haversine(*user.CachedLat, *user.CachedLong, report.Latitude, report.Longitude)
			if distance <= radius {
				affectedUsers = append(affectedUsers, user)
			}
		}
	}
	if len(affectedUsers) == 0 {
		err := fmt.Errorf("No users were found")
		return nil, err
	}
	return affectedUsers, nil
}
