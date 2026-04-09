package utils

import (
	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"log/slog"
)

func GetAllInfoFromDeadletterQueue() {
	var job []models.DLQJob
	if err := db.DB.Find(&job).Error; err != nil {
		slog.Error("DB error In GetAllInfoFromDeadletterQueue", "error", err)
		panic(err)
	}
	if len(job) == 0 {
		slog.Info("No failed jobs in DLQ")
	} else {
		slog.Info("Jobs in DLQ", "Length", len(job))
		for _, j := range job {
			slog.Info("Jon in DLQ", "failed-job-id", j.FailedMsgJOBID, "why it failed", j.ErrorInfo)
		}
	}
}
