package utils

import (
	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"log/slog"
)

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
