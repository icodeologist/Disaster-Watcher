package utils

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"gorm.io/gorm"
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
func GetUsersAffectedByDisaster(ctx context.Context, wg *sync.WaitGroup, allUsers []models.User, reportChan <-chan models.ReportMessage, deliveryChannel chan<- models.NotificationDeliveryMessage) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Recovered Panic in GetUsersAffectedByDisaster", "Panic", r)
		}
	}()
	// for normal operation flow
	// worker sit in for select watiing for jobs
	// if I interrupt or railway sends sigTERM then its shutdown flow
	// workerContext channel close -> every worker is CTX.DONE() fired
	// idle worker sees ctx.Done() -> finishes wg.DONE()
	// busy workers just finish the process come back and see ctx.DONE()
	// wg.DONE() fires
	// wg.WAIT() in main unblocks and we shutdown
	//
	processReport := func(reportMsg models.ReportMessage) {
		var deliveries []models.NotificationDelivery
		for _, user := range allUsers {
			report := reportMsg.Report
			userLat := user.CachedLat
			userLong := user.CachedLong
			if !user.LocationCached || !report.ISLocationCached {
				slog.Warn("User or Report location not found", "Is User Cached", user.LocationCached, "Is Report Cached", report.ISLocationCached, "user_id", user.ID, "report_id", report.ID)
				continue
			}
			radius := Haversine(*report.CachedLat, *report.CachedLong, *userLat, *userLong)
			if radius <= 20 {
				deliveries = append(deliveries, models.NotificationDelivery{
					IdempotencyKey: fmt.Sprintf("disaster-email/%d/%d", reportMsg.JobID, user.ID),
					JobID:          reportMsg.JobID,
					UserID:         user.ID,
					RecipientEmail: user.Email,
					EmailBody: models.EmailBody{
						Title:      report.Title,
						Location:   report.Location,
						Precaution: "Please take care of yourself and stay alert. Call the emergency helpline if you need assistance.",
					},
					Status: models.DeliveryStatusPending,
				})
			}
		}

		if len(deliveries) == 0 {
			if err := db.DB.Model(&models.Jobs{}).Where("id = ?", reportMsg.JobID).Update("status", "done").Error; err != nil {
				slog.Error("Failed to finish job with no affected users", "job_id", reportMsg.JobID, "error", err)
			}
			return
		}
		// The complete delivery is committed before its ID is published. If the
		// process stops before publishing, startup recovery can find the pending row.
		err := db.DB.Transaction(func(tx *gorm.DB) error {
			for i := range deliveries {
				if err := tx.Create(&deliveries[i]).Error; err != nil {
					return err
				}
			}
			return tx.Model(&models.Jobs{}).Where("id = ?", reportMsg.JobID).Update("status", "processing").Error
		})
		if err != nil {
			slog.Error("Failed to create notification deliveries", "job_id", reportMsg.JobID, "error", err)
			return
		}

		for _, delivery := range deliveries {
			select {
			case deliveryChannel <- models.NotificationDeliveryMessage{DeliveryID: delivery.ID}:
			case <-ctx.Done():
				return
			}
		}
	}
	for {
		select {
		case <-ctx.Done():
			return
		case reportMsg, ok := <-reportChan:
			if !ok {
				return
			}
			processReport(reportMsg)
		}
	}
}
