package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/icodeologist/disasterwatch/internal/db"
	emailservice "github.com/icodeologist/disasterwatch/internal/email_service"
	"github.com/icodeologist/disasterwatch/internal/models"
	"golang.org/x/time/rate"
)

type EmailrateLimiter struct {
	limiter *rate.Limiter
}

func loadNotificationDelivery(deliveryID uint) (models.NotificationDelivery, error) {
	var delivery models.NotificationDelivery
	if err := db.DB.First(&delivery, deliveryID).Error; err != nil {
		return models.NotificationDelivery{}, fmt.Errorf("load notification delivery %d: %w", deliveryID, err)
	}
	return delivery, nil
}

func sendRequestFromDelivery(delivery models.NotificationDelivery) emailservice.SendRequest {
	return emailservice.SendRequest{
		IdempotencyKey: delivery.IdempotencyKey,
		To:             delivery.RecipientEmail,
		Title:          delivery.EmailBody.Title,
		Location:       delivery.EmailBody.Location,
		Precaution:     delivery.EmailBody.Precaution,
	}
}

func StartNotificationWorker(rootContext context.Context, wg *sync.WaitGroup, workerCount int, deliveryChannel <-chan models.NotificationDeliveryMessage, retryChannel chan<- models.NotificationDeliveryMessage) {
	slog.Info("NOTIFICATION WORKERS STARTED", "COUNT", workerCount)
	emailRateLimiter := &EmailrateLimiter{limiter: rate.NewLimiter(2, 3)}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer func() {
				if recovered := recover(); recovered != nil {
					slog.Error("Recovered panic in notification worker", "worker_id", id, "panic", recovered)
				}
			}()

			processDelivery := func(message models.NotificationDeliveryMessage) {
				delivery, err := loadNotificationDelivery(message.DeliveryID)
				if err != nil {
					slog.Error("Failed to load notification delivery", "delivery_id", message.DeliveryID, "error", err)
					return
				}

				// Waiting before the claim means shutdown cannot strand the row in processing.
				if err := emailRateLimiter.limiter.Wait(rootContext); err != nil {
					slog.Warn("Email rate limiter wait interrupted", "delivery_id", delivery.ID, "error", err)
					return
				}

				now := time.Now()
				claim := db.DB.Model(&models.NotificationDelivery{}).
					Where("id = ? AND status = ?", delivery.ID, models.DeliveryStatusPending).
					Updates(map[string]any{
						"status":                models.DeliveryStatusProcessing,
						"attempts":              1,
						"processing_started_at": &now,
						"next_attempt_at":       nil,
					})
				if claim.Error != nil {
					slog.Error("Failed to claim notification delivery", "delivery_id", delivery.ID, "error", claim.Error)
					return
				}
				if claim.RowsAffected == 0 {
					slog.Info("Notification delivery already claimed", "delivery_id", delivery.ID)
					return
				}

				slog.Info("Sending email", "delivery_id", delivery.ID, "user_id", delivery.UserID, "attempt", 1)
				sendErr := emailservice.SendEmail(rootContext, sendRequestFromDelivery(delivery))
				if sendErr != nil {
					nextAttempt := time.Now().Add(retryDelay(1))
					result := db.DB.Model(&models.NotificationDelivery{}).
						Where("id = ? AND status = ?", delivery.ID, models.DeliveryStatusProcessing).
						Updates(map[string]any{
							"status":                models.DeliveryStatusRetrying,
							"last_error":            sendErr.Error(),
							"next_attempt_at":       &nextAttempt,
							"processing_started_at": nil,
						})
					if result.Error != nil {
						slog.Error("Failed to record email failure", "delivery_id", delivery.ID, "error", result.Error)
						return
					}
					if result.RowsAffected == 0 {
						slog.Warn("Delivery state changed before failure was recorded", "delivery_id", delivery.ID)
						return
					}

					select {
					case retryChannel <- models.NotificationDeliveryMessage{DeliveryID: delivery.ID}:
					case <-rootContext.Done():
					}
					return
				}

				sentAt := time.Now()
				result := db.DB.Model(&models.NotificationDelivery{}).
					Where("id = ? AND status = ?", delivery.ID, models.DeliveryStatusProcessing).
					Updates(map[string]any{
						"status":                models.DeliveryStatusSuccess,
						"sent_at":               &sentAt,
						"last_error":            "",
						"processing_started_at": nil,
					})
				if result.Error != nil {
					slog.Error("Failed to mark notification delivery successful", "delivery_id", delivery.ID, "error", result.Error)
					return
				}
				if result.RowsAffected == 0 {
					slog.Warn("Delivery state changed before success was recorded", "delivery_id", delivery.ID)
					return
				}
				if err := finalizeNotificationJob(delivery.JobID); err != nil {
					slog.Error("Failed to finalize notification job", "job_id", delivery.JobID, "error", err)
				}
			}

			for {
				select {
				case <-rootContext.Done():
					return
				case message, ok := <-deliveryChannel:
					if !ok {
						return
					}
					processDelivery(message)
				}
			}
		}(i)
	}
}
