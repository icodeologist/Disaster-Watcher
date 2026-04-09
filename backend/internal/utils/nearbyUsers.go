package utils

import (
	"context"
	"log/slog"
	"sync"

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

// Users in the distance from the report posted in  radius like 20km?
func GetUsersAffectedByDisaster(ctx context.Context, wg *sync.WaitGroup, allUsers []models.User, reportChan <-chan models.ReportMessage, affectedUserIdsChan chan<- models.AffectedUsersMessage) {
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
		for _, user := range allUsers {
			report := reportMsg.Report
			userLat := user.CachedLat
			userLong := user.CachedLong
			if !user.LocationCached || !report.ISLocationCached {
				slog.Warn("User or Report location not found", "Is User Cached", user.LocationCached, "Is Report Cached", report.ISLocationCached, "user_id", user.ID, "report_id", report.ID)
				continue
			}
			radius := Haversine(*report.CachedLat, *report.CachedLong, *userLat, *userLong)
			// FIX: fix this
			if radius > 20 {
				// incase if the affectedUserIdsChan is full and incase of blocking
				// only if the shutdonw signal is fired instead of waiting just check if fired and return
				// Because if downstream workers are already left this will keep hanging
				affectedUsersMsg := models.AffectedUsersMessage{
					JobID:  reportMsg.JobID,
					UserID: user.ID,
				}
				select {
				case affectedUserIdsChan <- affectedUsersMsg:
					// if this case blocked we just drop the work
				case <-ctx.Done():
					slog.Info("Shutdown Fired", "Idle worker exiting", "Nothing to do")
					return
				}
			} else {
				slog.Warn("Found 0 users nearby", "report_id", report.ID, "report_location", report.Location)
			}
		}
	}
	for {
		select {
		case <-ctx.Done():
			return
		case reportMsg := <-reportChan:
			processReport(reportMsg)
		}
	}
}
