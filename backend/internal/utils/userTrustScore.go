package utils

import (
	"context"
	"time"

	"log/slog"

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
func VerifyReportPostedByUser(ctx context.Context, report *models.Report, user models.User, reportChannel chan models.ReportMessage, jobId int64) {
	slog.Info("VERIFICATION PROCESS STARTED", "JOB", jobId)
	if !user.LocationCached || !report.ISLocationCached {
		slog.Warn("User or Report location not found. Marking Report Unverified", "Is User Cached", user.LocationCached, "Is Report Cached", report.ISLocationCached, "user_id", user.ID, "report_id", report.ID)
		report.Status = "Unverified"
		err := db.DB.Save(report).Error
		if err != nil {
			slog.Error("Report save Failed", "report_id", report.ID, "Error", err)
		}
		return
	}
	reportStatus, score, err := Helper(ctx, user, report)
	if err != nil {
		slog.Error("Report scoring algorithm returned ", "Error", err)
	}
	if reportStatus == "Verified" {
		slog.Info("Verified", "report_id", report.ID, "report_score", score)
		// if shutdown fired and waiting to send drop the report
		reportMsg := models.ReportMessage{
			JobID:  jobId,
			Report: *report,
		}

		select {
		case reportChannel <- reportMsg:
		case <-ctx.Done():
			slog.Info("Shutdown Fired", "Idle worker exiting", "Nothing to do")
			return
		}
	} else {
		slog.Warn("Unverified", "report_id", report.ID)
		err := db.DB.Delete(&report).Error
		if err != nil {
			slog.Error("Delete Unverified report Failed", "report_id", report.ID, "DB.Error", err)
		}
	}
}
