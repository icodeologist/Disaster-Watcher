package worker

import (
	"fmt"
	"log/slog"

	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
)

type deliveryStatusCounts struct {
	Total      int64
	Successful int64
	Failed     int64
	Unfinished int64
}

func finalizeNotificationJob(jobID int64) error {
	var counts deliveryStatusCounts
	err := db.DB.Model(&models.ProcessedNotification{}).
		Select(`COUNT(*) AS total,
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) AS successful,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failed,
			SUM(CASE WHEN status IN ('pending', 'processing', 'retrying') THEN 1 ELSE 0 END) AS unfinished`).
		Where("job_id = ?", jobID).
		Scan(&counts).Error
	if err != nil {
		return fmt.Errorf("count notification deliveries: %w", err)
	}
	status, complete := notificationJobStatus(counts)
	if !complete {
		return nil
	}

	result := db.DB.Model(&models.Jobs{}).
		Where("id = ? AND status = ?", jobID, "processing").
		Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("finalize notification job: %w", result.Error)
	}
	if result.RowsAffected > 0 {
		slog.Info("Notification job finalized", "job_id", jobID, "status", status, "total", counts.Total, "successful", counts.Successful, "failed", counts.Failed)
	}
	return nil
}

func notificationJobStatus(counts deliveryStatusCounts) (string, bool) {
	if counts.Total == 0 || counts.Unfinished > 0 {
		return "", false
	}
	if counts.Successful == counts.Total {
		return "done", true
	}
	if counts.Failed == counts.Total {
		return "failed", true
	}
	return "partially_failed", true
}
