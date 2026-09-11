package utils

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
)

// RecoverUnfinishedJobs restarts parent jobs that never created email
// deliveries. Jobs with delivery rows belong to delivery recovery instead.
func RecoverUnfinishedJobs(ctx context.Context, verificationChannel chan<- models.VerificationMessage) error {
	var unfinishedJobs []models.Jobs
	if err := db.DB.
		Where("status IN ?", []string{"pending", "processing"}).
		Where("NOT EXISTS (?)",
			db.DB.Model(&models.NotificationDelivery{}).
				Select("1").
				Where("notification_deliveries.job_id = jobs.id"),
		).
		Find(&unfinishedJobs).Error; err != nil {
		return fmt.Errorf("load unfinished jobs: %w", err)
	}

	for _, job := range unfinishedJobs {
		if job.Status == "processing" {
			result := db.DB.Model(&models.Jobs{}).
				Where("id = ? AND status = ?", job.Id, "processing").
				Update("status", "pending")
			if result.Error != nil {
				return fmt.Errorf("reset processing job %d: %w", job.Id, result.Error)
			}
			if result.RowsAffected == 0 {
				slog.Info("Unfinished job state changed before recovery", "job_id", job.Id)
				continue
			}
		}

		message := models.VerificationMessage{JobID: job.Id}
		select {
		case verificationChannel <- message:
			slog.Info("Recovered unfinished job", "job_id", job.Id)
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}
