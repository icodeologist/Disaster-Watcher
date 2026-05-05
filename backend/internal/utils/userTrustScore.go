package utils

import (
	"context"

	"log/slog"

	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
)

func UserTrustScore(user models.User) int {
	return -1
}

//TODO: Refactor this shit code

func Helper(ctx context.Context, user models.User, report *models.Report) (string, error) {
	if !user.LocationCached || !report.ISLocationCached {
		return "Unverified", nil
	} else if report.Description == "" || len(report.Description) < 10 ||
		report.Title == "" || len(report.Title) < 10 {
		return "Unverified", nil
	} else {
		return "Verified", nil
	}
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
	reportStatus, err := Helper(ctx, user, report)
	if err != nil {
		slog.Error("Report scoring algorithm returned ", "Error", err)
	}
	if reportStatus == "Verified" {
		slog.Info("Verified", "report_id", report.ID)
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
