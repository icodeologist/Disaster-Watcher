package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
)

// RecoverNotificationDeliveries rebuilds the in-memory work queues from durable
// delivery state after a process restart. A processing delivery has an uncertain
// provider outcome, so it is retried with the same provider idempotency key. The
// function must run after the delivery workers have started so sending more
// messages than the channel buffer can hold does not block startup.
func RecoverNotificationDeliveries(ctx context.Context, deliveryChannel chan<- models.NotificationDeliveryMessage, retryChannel chan<- models.NotificationDeliveryMessage) error {
	now := time.Now()

	if err := db.DB.Model(&models.NotificationDelivery{}).
		Where("status = ?", models.DeliveryStatusProcessing).
		Updates(map[string]any{
			"status":                models.DeliveryStatusRetrying,
			"next_attempt_at":       &now,
			"processing_started_at": nil,
			"last_error":            "recovered after process stopped during delivery",
		}).Error; err != nil {
		return fmt.Errorf("release interrupted notification deliveries: %w", err)
	}

	var deliveries []models.NotificationDelivery
	if err := db.DB.Where("status IN ?", []string{models.DeliveryStatusPending, models.DeliveryStatusRetrying}).Find(&deliveries).Error; err != nil {
		return fmt.Errorf("load recoverable notification deliveries: %w", err)
	}

	for _, delivery := range deliveries {
		message := models.NotificationDeliveryMessage{DeliveryID: delivery.ID}
		var target chan<- models.NotificationDeliveryMessage
		if delivery.Status == models.DeliveryStatusPending {
			target = deliveryChannel
		} else {
			target = retryChannel
		}

		select {
		case target <- message:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}
