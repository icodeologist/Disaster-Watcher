package controllers

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/icodeologist/disasterwatch/database"
	"github.com/icodeologist/disasterwatch/models"
	pb "github.com/icodeologist/grpc-proto"
)

const (
	DefaultAction  = "Report Posted"
	DefaultMessage = "There was a disaster reported nearby. Please take the precautions."
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

func customPrecautionMessages(reportType string) string {
	temp := strings.ToLower(strings.TrimSpace(reportType))
	switch temp {
	case "earthquake":
		return fmt.Sprintf("Custom earthquake precaution")
	case "flood":
		return fmt.Sprintf("Custom flood precaution")
	case "electricityoutrage":
		return fmt.Sprintf("Custom electricity outrage precaution")
	default:
		return fmt.Sprintf("Custom default precaution")
	}
}

func FindUserAffected(report models.Report) ([]models.User, error) {
	var users []models.User
	r := database.DB.Find(&report)
	if r.Error != nil {
		return users, r.Error
	}

	radius := getNotificationRadius(report.Type)
	var affectedUsers []models.User
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
	fmt.Print("USERID         USER Details\n")
	for _, n := range affectedUsers {
		fmt.Printf("%v   \n", n.ID)

	}
	return affectedUsers, nil
}

func MapEachUsersTONotificationEventInstance(users []models.User, report models.Report) ([]models.NotificationEvent, error) { // report type
	var UserNotificationEventRequest []models.NotificationEvent
	if len(users) == 0 {
		return nil, fmt.Errorf("No users found")
	}
	for _, user := range users {
		precaution := customPrecautionMessages(report.Type)
		customMessage := fmt.Sprintf("There is a %v nearby your location. Please take the proper precautions and be safe. %v\n", report.Type, precaution)
		// Make some dummy events here
		userNotifEvent := models.NotificationEvent{
			UserID:           user.ID,
			Action:           DefaultAction,
			Message:          customMessage,
			Timestamp:        time.Now(),
			NotificationType: pb.Notificationtype_email, // let the user opt or send both way
		}
		UserNotificationEventRequest = append(UserNotificationEventRequest, userNotifEvent)
	}
	log.Printf("Notification events %v\n", UserNotificationEventRequest)
	log.Print("USER ID    Action     Notificationtype\n")
	for _, evt := range UserNotificationEventRequest {
		log.Printf("%v    %v        %v", evt.UserID, evt.Action, evt.NotificationType)
	}

	return UserNotificationEventRequest, nil

}
