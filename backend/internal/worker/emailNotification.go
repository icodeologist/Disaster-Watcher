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

// Third step.
// For each affected users send email using rate limiter
// if failed using the ProcessedNotification col update error and push to failedEmailsChan
// if passed update ProcessedNotification staus and wait for all of them to finish
// once finished items will finalizeNotificationJob to determine the overall status of the job

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
				// so its first time just update the processed notifcation with this affMsg
				claim := db.DB.Model(&models.ProcessedNotification{}).
					Where("id = ? AND job_id = ? AND user_id = ? AND status = ?", affUserMsg.DeliveryID, affUserMsg.JobID, affUserMsg.UserID, "pending").
					Updates(map[string]interface{}{"status": "processing", "attempts": 1})
				if claim.Error != nil {
					slog.Error("Failed to claim notification delivery", "delivery_id", affUserMsg.DeliveryID, "error", claim.Error)
					return
				}
				// if 2 workers comes and worker 1 already claimed using all the infor above and updated it to processing then second worker Worker B gets RowsAffected == 0 and its already claimed message
				if claim.RowsAffected == 0 {
					slog.Info("Notification delivery already claimed", "delivery_id", affUserMsg.DeliveryID)
					return
				}

				var user models.User
				if err := db.DB.Where("id=?", affUserMsg.UserID).First(&user).Error; err != nil {
					slog.Error("failed to fetch user from db", "user_id", affUserMsg.UserID, "error", err)
					now := time.Now()
					db.DB.Model(&models.ProcessedNotification{}).Where("id=?", affUserMsg.DeliveryID).Updates(map[string]any{
						"status":     "failed",
						"failed_at":  &now,
						"last_error": err.Error(),
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
				var pn models.ProcessedNotification
				err := db.DB.Where("id=?", affUserMsg.DeliveryID).First(&pn).Error
				if err != nil {
					slog.Error("processed notification fetch failed", "error", err)
					return
				}

				emailHelper := models.EmailModel{
					IdempotencyKey: pn.IdempotencyKey,
					Email:          user.Email,
					EmailBody:      emailObj,
				}
				saveErr := db.DB.Save(&emailHelper).Error
				if saveErr != nil {
					slog.Error("save email helper failed", "error", saveErr)
					return
				}
				sendEmailErr := emailservice.SendEmail(emailHelper)
				if sendEmailErr != nil {
					slog.Warn("Failed to send Email", "User", affUserMsg.UserID)
					if updateErr := db.DB.Model(&models.ProcessedNotification{}).Where("id = ?", affUserMsg.DeliveryID).Updates(map[string]interface{}{
						"status": "retrying", "last_error": sendEmailErr.Error(),
					}).Error; updateErr != nil {
						slog.Error("Failed to update notification delivery", "delivery_id", affUserMsg.DeliveryID, "error", updateErr)
						return
					}
					failedMessage := &models.FailedEmailMessage{
						DeliveryID:   affUserMsg.DeliveryID,
						JobID:        affUserMsg.JobID,
						User:         user,
						Report:       affUserMsg.Report,
						ErrorMessage: sendEmailErr,
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
