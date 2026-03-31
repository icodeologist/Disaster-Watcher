package utils

import (
	"context"
	"time"

	"log"

	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
)

func UserTrustScore(user models.User) int {
	return -1
}

//TODO: Refactor this shit code

func Helper(ctx context.Context, user models.User, report *models.Report) (string, int, error) {
	// FIX: Always start with 0
	score := 10 // For testing purpose
	radius := Haversine(*user.CachedLat, *user.CachedLong, *report.CachedLat, *report.CachedLong)
	if radius <= 20 {
		score += 3
	} // TODO : FIX THIS scoring system. Its too harsh on new user
	//check user reputation

	switch {
	case user.TotalReportPosted >= 10:
		score += 3
	case user.TotalReportPosted >= 3:
		score += 1
	case user.TotalReportPosted == 0:
		score -= 1
	}

	// check if similar report are posted near the user posted in a given time
	var similarReportPosted []models.Report
	err := db.DB.Where("category=? AND id != ? AND created_at BETWEEN ? AND ?",
		report.Category,
		report.ID,
		report.CreatedAt.Add(-24*time.Hour),
		report.CreatedAt,
	).Find(&similarReportPosted).Error
	if err != nil {
		return "", 0, err
	}
	switch {
	case len(similarReportPosted) >= 5:
		score += 3
	case len(similarReportPosted) >= 2:
		score += 1
	case len(similarReportPosted) == 0:
		score -= 1
	}

	if score >= 2 {
		report.Status = "Verified"
	} else {
		report.Status = "Unverified"
	}
	err = db.DB.Save(report).Error
	if err != nil {
		return "", 0, err
	}
	return report.Status, score, nil
}

// report trust score
func VerifyReportPostedByUser(ctx context.Context, report *models.Report, user models.User, reportChannel chan models.Report) {
	log.Println("Started verify report")
	if !user.LocationCached || !report.ISLocationCached {
		log.Printf("[VERIFY] report %v or user %v has no cached location, marking unverified\n", report.ID, user.ID)
		report.Status = "Unverified"
		err := db.DB.Save(report).Error
		if err != nil {
			log.Printf("Error Saving Report : %v\n", err)
		}
		return
	}
	reportStatus, score, err := Helper(ctx, user, report)
	if err != nil {
		log.Printf("Error in score algorithm : %v\n", err)
	}
	if reportStatus == "Verified" {
		log.Printf("[VERIFICATION RESULT] [ID : %v] [STATUS : %v] [SCORE : %v]", report.ID, report.Status, score)
		// if shutdown fired and waiting to send drop the report
		select {
		case reportChannel <- *report:
			log.Println("Pushed to report channel")
		case <-ctx.Done():
			log.Printf("Verification worker cancelled, exiting\n")
			return
		}
	} else {
		err := db.DB.Delete(&report).Error
		if err != nil {
			log.Printf("Error While deleting report : %v\n", err)
		}
		log.Println("Else case ran so no reports were verified")
	}
}
