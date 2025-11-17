package utils

import (
	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"time"
)

func UserTrustScore(user models.User) int {
	return -1
}

// if user is posting somewhere far from his location
// if user has no posts
// if there are no similar posts in that area
// then simply make the post Unverified
func VerifyReportPostedByUser(report *models.Report, user models.User) error {
	// TODO: what if all the posts are unverfied or not near that location
	radius := Haversine(*user.CachedLat, *user.CachedLong, *report.CachedLat, *report.CachedLong)
	badPost := 0
	if radius > 20 {
		badPost += 1
	}
	if user.TotalReportPosted == 0 {
		badPost += 2
	} else if user.TotalReportPosted > 0 {
		if badPost >= 1 {
			badPost -= 1
		}
	}
	var similarReports []models.Report
	if err := db.DB.Where("category=? AND id !=? AND created_at >= ?", report.Category, report.ID, report.CreatedAt.Add(-24*time.Hour)).Find(&similarReports).Error; err != nil {
		return err
	}
	if len(similarReports) > 2 {
		if badPost > 2 {
			badPost -= 2
		}
	} else if len(similarReports) <= 2 {
		badPost += 2
	}

	if badPost <= 2 {
		report.Status = "Verified"
	} else {
		report.Status = "Unverified"
	}
	db.DB.Save(&report)
	return nil
}
