package controllers

import (
	"fmt"
	"github.com/icodeologist/disasterwatch/models"
	"strings"
)

func getNotificationRadius(reportType string) float64 {
	temp := strings.ToLower(strings.TrimSpace(reportType))
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

func FindUserAffected(report *models.Report) ([]models.User, error) {
	radius := getNotificationRadius(report.Type)
	var affectedUsers []models.User
	var users []models.User
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
