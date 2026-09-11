package worker

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/icodeologist/disasterwatch/internal/db"
	emailservice "github.com/icodeologist/disasterwatch/internal/email_service"
	"github.com/icodeologist/disasterwatch/internal/models"
	"gorm.io/gorm"
)

func retryDelay(attempts int) time.Duration {
	return time.Duration(math.Pow(2, float64(attempts))) * time.Second
}

func waitUntil(ctx context.Context, when *time.Time) error {
	if when == nil || !when.After(time.Now()) {
		return nil
	}
	timer := time.NewTimer(time.Until(*when))
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func StartFailedEmailSendingWorker(rootContext context.Context, wg *sync.WaitGroup, workerCount int, maxRetries int, retryChannel <-chan models.NotificationDeliveryMessage, deadMessageChannel chan models.DLQJob) {
	slog.Info("FAILED EMAIL WORKERS STARTED", "COUNT", workerCount)
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if recovered := recover(); recovered != nil {
					slog.Error("Recovered panic in failed email worker", "worker_id", id, "panic", recovered)
				}
			}()

			processRetry := func(message models.NotificationDeliveryMessage) {
				for {
					delivery, err := loadNotificationDelivery(message.DeliveryID)
					if err != nil {
						slog.Error("Failed to load notification delivery for retry", "delivery_id", message.DeliveryID, "error", err)
						return
					}

					if delivery.Attempts > maxRetries {
						moveDeliveryToDLQ(rootContext, delivery, deadMessageChannel)
						return
					}

					if err := waitUntil(rootContext, delivery.NextAttemptAt); err != nil {
						return
					}

					nextAttemptNumber := delivery.Attempts + 1
					processingStartedAt := time.Now()
					claim := db.DB.Model(&models.NotificationDelivery{}).
						Where("id = ? AND status = ? AND attempts = ?", delivery.ID, models.DeliveryStatusRetrying, delivery.Attempts).
						Updates(map[string]any{
							"status":                models.DeliveryStatusProcessing,
							"attempts":              nextAttemptNumber,
							"processing_started_at": &processingStartedAt,
							"next_attempt_at":       nil,
						})
					if claim.Error != nil {
						slog.Error("Failed to claim notification retry", "delivery_id", delivery.ID, "error", claim.Error)
						return
					}
					if claim.RowsAffected == 0 {
						slog.Info("Notification retry already claimed or outdated", "delivery_id", delivery.ID)
						return
					}

					slog.Info("Retrying email", "delivery_id", delivery.ID, "user_id", delivery.UserID, "attempt", nextAttemptNumber)
					sendErr := emailservice.SendEmail(rootContext, sendRequestFromDelivery(delivery))
					if sendErr != nil {
						nextAttemptAt := time.Now().Add(retryDelay(nextAttemptNumber))
						result := db.DB.Model(&models.NotificationDelivery{}).
							Where("id = ? AND status = ? AND attempts = ?", delivery.ID, models.DeliveryStatusProcessing, nextAttemptNumber).
							Updates(map[string]any{
								"status":                models.DeliveryStatusRetrying,
								"last_error":            sendErr.Error(),
								"next_attempt_at":       &nextAttemptAt,
								"processing_started_at": nil,
							})
						if result.Error != nil {
							slog.Error("Failed to record notification retry failure", "delivery_id", delivery.ID, "error", result.Error)
							return
						}
						if result.RowsAffected == 0 {
							return
						}
						continue
					}

					sentAt := time.Now()
					result := db.DB.Model(&models.NotificationDelivery{}).
						Where("id = ? AND status = ? AND attempts = ?", delivery.ID, models.DeliveryStatusProcessing, nextAttemptNumber).
						Updates(map[string]any{
							"status":                models.DeliveryStatusSuccess,
							"sent_at":               &sentAt,
							"last_error":            "",
							"processing_started_at": nil,
						})
					if result.Error != nil {
						slog.Error("Failed to mark notification retry successful", "delivery_id", delivery.ID, "error", result.Error)
						return
					}
					if result.RowsAffected == 0 {
						slog.Warn("Delivery state changed before retry success was recorded", "delivery_id", delivery.ID)
						return
					}
					if err := finalizeNotificationJob(delivery.JobID); err != nil {
						slog.Error("Failed to finalize notification job", "job_id", delivery.JobID, "error", err)
					}
					return
				}
			}

			for {
				select {
				case <-rootContext.Done():
					return
				case message, ok := <-retryChannel:
					if !ok {
						return
					}
					processRetry(message)
				}
			}
		}(i)
	}
}

func moveDeliveryToDLQ(ctx context.Context, delivery models.NotificationDelivery, deadMessageChannel chan<- models.DLQJob) {
	failedAt := time.Now()
	dlqJob := models.DLQJob{
		DeliveryID:     delivery.ID,
		FailedMsgJOBID: delivery.JobID,
		ErrorMessage:   fmt.Sprintf("ERR_MAX_RETRY_EXHAUSTED: %s", delivery.LastError),
		CreatedAt:      time.Now(),
		WhereFailed:    "FailedEmailSendingWorker",
	}
	moved := false
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.NotificationDelivery{}).
			Where("id = ? AND status = ? AND attempts = ?", delivery.ID, models.DeliveryStatusRetrying, delivery.Attempts).
			Updates(map[string]any{
				"status":          models.DeliveryStatusFailed,
				"failed_at":       &failedAt,
				"next_attempt_at": nil,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		if err := tx.Create(&dlqJob).Error; err != nil {
			return err
		}
		moved = true
		return nil
	}); err != nil {
		slog.Error("Failed to move notification delivery to DLQ", "delivery_id", delivery.ID, "error", err)
		return
	}
	if !moved {
		return
	}
	if err := finalizeNotificationJob(delivery.JobID); err != nil {
		slog.Error("Failed to finalize notification job", "job_id", delivery.JobID, "error", err)
	}

	select {
	case deadMessageChannel <- dlqJob:
	case <-ctx.Done():
	}
}
