package worker

import (
	"context"
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

func StartNotificationWorker(rootContext context.Context, wg *sync.WaitGroup, n int, affUsersIdChannel <-chan models.AffectedUsersMessage, failedEmailsChan chan<- models.FailedEmailMessage) {
	slog.Info("NOTIFICATION WORKERS STARTED", "COUNT", n)
	emailRateL := &EmailrateLimiter{
		limiter: rate.NewLimiter(2, 3),
	}

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			//recover if this go routiner panicks at some point
			defer func() {
				if r := recover(); r != nil {
					slog.Error("Recovered panic in Notification Workers", "panic", r)
				}
			}()
			// first time sending email
			processSendingEmail := func(affUserMsg models.AffectedUsersMessage) {
				claim := db.DB.Model(&models.ProcessedNotification{}).
					Where("id = ? AND job_id = ? AND user_id = ? AND status = ?", affUserMsg.DeliveryID, affUserMsg.JobID, affUserMsg.UserID, "pending").
					Updates(map[string]interface{}{"status": "processing", "attempts": 1})
				if claim.Error != nil {
					slog.Error("Failed to claim notification delivery", "delivery_id", affUserMsg.DeliveryID, "error", claim.Error)
					return
				}
				if claim.RowsAffected == 0 {
					slog.Info("Notification delivery already claimed", "delivery_id", affUserMsg.DeliveryID)
					return
				}

				var user models.User
				if err := db.DB.Where("id=?", affUserMsg.UserID).First(&user).Error; err != nil {
					slog.Error("failed to fetch user from db", "user_id", affUserMsg.UserID, "error", err)
					now := time.Now()
					db.DB.Model(&models.ProcessedNotification{}).Where("id = ?", affUserMsg.DeliveryID).Updates(map[string]interface{}{
						"status": "failed", "failed_at": &now, "last_error": err.Error(),
					})
					if finalizeErr := finalizeNotificationJob(affUserMsg.JobID); finalizeErr != nil {
						slog.Error("Failed to finalize notification job", "job_id", affUserMsg.JobID, "error", finalizeErr)
					}
					return
				}
				if err := emailRateL.limiter.Wait(rootContext); err != nil {
					slog.Warn("email rate limiter wait interrupted", "error", err)
					db.DB.Model(&models.ProcessedNotification{}).Where("id = ?", affUserMsg.DeliveryID).Update("status", "pending")
					return
				}
				slog.Info("Email Sending", "User", affUserMsg.UserID, "Tries", "First Time")
				emailObj := models.EmailBody{
					Title:      affUserMsg.Report.Title,
					Location:   affUserMsg.Report.Location,
					Precaution: "Please take care of you and watch out. Call this help line 3939393.",
				}
				emailHelper := models.EmailModel{
					Email:     user.Email,
					EmailBody: emailObj,
				}
				err := emailservice.SendEmail(emailHelper)
				if err != nil {
					slog.Warn("Failed to send Email", "User", affUserMsg.UserID)
					if updateErr := db.DB.Model(&models.ProcessedNotification{}).Where("id = ?", affUserMsg.DeliveryID).Updates(map[string]interface{}{
						"status": "retrying", "last_error": err.Error(),
					}).Error; updateErr != nil {
						slog.Error("Failed to update notification delivery", "delivery_id", affUserMsg.DeliveryID, "error", updateErr)
						return
					}
					failedMessage := &models.FailedEmailMessage{
						DeliveryID:   affUserMsg.DeliveryID,
						JobID:        affUserMsg.JobID,
						User:         user,
						Report:       affUserMsg.Report,
						ErrorMessage: err,
						RetryAttempt: 0,
					}
					// same logic
					// if shutdown fired and worker wait to send
					// drop it assuming downline workers already left
					select {
					case failedEmailsChan <- *failedMessage:
						slog.Info("Push failed email to FailedMessageChannel", "User", affUserMsg.UserID)
					case <-rootContext.Done():
						slog.Info("shutdown fired, stopping workers")
						return
					}
					return
				}

				now := time.Now()
				if err := db.DB.Model(&models.ProcessedNotification{}).Where("id = ?", affUserMsg.DeliveryID).Updates(map[string]interface{}{
					"status": "success", "sent_at": &now, "last_error": "",
				}).Error; err != nil {
					slog.Error("Failed to mark notification delivery successful", "delivery_id", affUserMsg.DeliveryID, "error", err)
					return
				}
				if err := finalizeNotificationJob(affUserMsg.JobID); err != nil {
					slog.Error("Failed to finalize notification job", "job_id", affUserMsg.JobID, "error", err)
				}
			}
			for {
				select {
				case <-rootContext.Done():
					slog.Info("shutdown fired, stopping workers")
					return
				case affectedUserMsg, ok := <-affUsersIdChannel:
					slog.Info("Affected User INFO", "affectedUserMsg", affectedUserMsg)
					if !ok {
						slog.Info("affected users channel closed, worker exiting")
						return
					}
					processSendingEmail(affectedUserMsg)
				}
			}
		}(i)
	}
}
