package utils

import (
	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"time"
)

func UserTrustScore(user models.User) int {
	return -1
}

// report trust score
func VerifyReportPostedByUser(report *models.Report, user models.User) error {
	if !user.LocationCached || !report.ISLocationCached {
		report.Status = "Unverified"
		return db.DB.Save(&report).Error
	}
	score := 0
	radius := Haversine(*user.CachedLat, *user.CachedLong, *report.CachedLat, *report.CachedLong)
	if radius <= 20 {
		score += 3
	} else {
		score -= 2
	}
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
	err := db.DB.Where("category=? AND id != ? AND create_at BETWEEN ? AND ?",
		report.Category,
		report.ID,
		report.CreatedAt.Add(-24*time.Hour),
		report.CreatedAt,
	).Find(&similarReportPosted).Error
	if err != nil {
		return err
	}
	switch {
	case len(similarReportPosted) >= 5:
		score += 3
	case len(similarReportPosted) >= 2:
		score += 1
	case len(similarReportPosted) == 0:
		score -= 1
	}

	if score >= 3 {
		report.Status = "Verified"
	} else {
		report.Status = "Unverified"
	}
	return db.DB.Save(&report).Error

}
