package utils

import (
	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"log/slog"
)

// having 10minutes to stimulate the possible delays from retry worker
// could be reduced to even 5 minutes
func RecoverStalledJobs(verificationChannel chan<- models.VerificationMessage) {
	var stalledJobs []models.Jobs
	if err := db.DB.Where("status=? AND NOW() - started_at > INTERVAL '10 minutes'", "processing").Find(&stalledJobs).Error; err != nil {
		slog.Error("Failed to fetch stalled jobs", "error", err)
	}

	for _, job := range stalledJobs {
		job.Status = "pending"
		db.DB.Save(&job)

		vmsg := models.VerificationMessage{
			JobID: job.Id,
		}
		verificationChannel <- vmsg
	}
}
