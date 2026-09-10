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
)

func StartFailedEmailSendingWorker(rootContext context.Context, wg *sync.WaitGroup, n int, maxTries int, failedEmailsChan chan models.FailedEmailMessage, deadMessageChannel chan models.DLQJob) {
	slog.Info("FAILED EMAIL WORKERS STARTED", "COUNT", n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			//recover if this go routiner panicks at some point
			defer func() {
				if r := recover(); r != nil {
					slog.Error("Recoverd panic in StartFailedEmailSendingWorker", "Panic", r)
				}
			}()
			processFailedEmailSending := func(failedUserID models.FailedEmailMessage) {
				// 2 4 8 16 64
				attempt := failedUserID.RetryAttempt
				if attempt == 0 {
					attempt = 1
				}
				timeTOwait := math.Pow(2, float64(attempt))
				fMsg := &models.FailedEmailMessage{
					DeliveryID:   failedUserID.DeliveryID,
					JobID:        failedUserID.JobID,
					User:         failedUserID.User,
					Report:       failedUserID.Report,
					ErrorMessage: failedUserID.ErrorMessage,
					RetryAttempt: failedUserID.RetryAttempt + 1,
					RetryDelay:   time.Duration(timeTOwait) * time.Second,
				}
				if fMsg.RetryAttempt <= maxTries {
					select {
					case <-time.After(time.Duration(timeTOwait) * time.Second):
					case <-rootContext.Done():
						slog.Info("shutdown fired, stopping workers")
						return
					}
					slog.Info("Retrying", "User", fMsg.User.ID, "Retry time", fMsg.RetryAttempt, "Retrying again in", fMsg.RetryDelay)
					emailObj := models.EmailBody{
						Title:      fMsg.Report.Title,
						Location:   fMsg.Report.Location,
						Precaution: "Please take care of you and watch out. Call this help line 3939393.",
					}
					emailHelper := models.EmailModel{
						Email:     fMsg.User.Email,
						EmailBody: emailObj,
					}

					claim := db.DB.Model(&models.ProcessedNotification{}).
						Where("id = ? AND job_id = ? AND user_id = ? AND status = ?", fMsg.DeliveryID, fMsg.JobID, fMsg.User.ID, "retrying").
						Updates(map[string]interface{}{"status": "processing", "attempts": fMsg.RetryAttempt + 1})
					if claim.Error != nil {
						slog.Error("Failed to claim notification retry", "delivery_id", fMsg.DeliveryID, "error", claim.Error)
						return
					}
					if claim.RowsAffected == 0 {
						slog.Info("Notification retry already claimed or completed", "delivery_id", fMsg.DeliveryID)
						return
					}

					err := emailservice.SendEmail(emailHelper)
					if err != nil {
						if updateErr := db.DB.Model(&models.ProcessedNotification{}).Where("id = ?", fMsg.DeliveryID).Updates(map[string]interface{}{
							"status": "retrying", "last_error": err.Error(),
						}).Error; updateErr != nil {
							slog.Error("Failed to update notification retry", "delivery_id", fMsg.DeliveryID, "error", updateErr)
							return
						}
						select {
						case failedEmailsChan <- *fMsg:
						case <-rootContext.Done():
							return
						}
					} else {
						slog.Info("email sent successfully", "User", fMsg.User.ID, "attempt", fMsg.RetryAttempt)
						now := time.Now()
						if err := db.DB.Model(&models.ProcessedNotification{}).Where("id = ?", fMsg.DeliveryID).Updates(map[string]interface{}{
							"status": "success", "sent_at": &now, "last_error": "",
						}).Error; err != nil {
							slog.Error("Failed to mark notification retry successful", "delivery_id", fMsg.DeliveryID, "error", err)
							return
						}
						if err := finalizeNotificationJob(fMsg.JobID); err != nil {
							slog.Error("Failed to finalize notification job", "job_id", fMsg.JobID, "error", err)
						}
					}
				} else {
					now := time.Now()
					if err := db.DB.Model(&models.ProcessedNotification{}).Where("id = ? AND status <> ?", fMsg.DeliveryID, "success").Updates(map[string]interface{}{
						"status": "failed", "failed_at": &now, "last_error": fmt.Sprint(fMsg.ErrorMessage),
					}).Error; err != nil {
						slog.Error("Failed to mark notification delivery failed", "delivery_id", fMsg.DeliveryID, "error", err)
						return
					}
					if err := finalizeNotificationJob(fMsg.JobID); err != nil {
						slog.Error("Failed to finalize notification job", "job_id", fMsg.JobID, "error", err)
					}
					dlqJob := models.DLQJob{
						ErrorMessage:   fmt.Sprintf("ERR_MAX_RETRY_EXHAUSTER : %v", fMsg.ErrorMessage),
						FailedMsgJOBID: fMsg.JobID,
						CreatedAt:      time.Now(),
						WhereFailed:    "FailedEmailSendingWorker",
					}
					if err := db.DB.Save(&dlqJob).Error; err != nil {
						slog.Error("Error saving dlqjob to DB", "err", err)
						return
					}

					select {
					// if faile after maxtries  == retry attempts
					// push to DL
					// and save the job
					case deadMessageChannel <- dlqJob:
						slog.Warn("Notification Failed, Sending to DeadLetterChannel", "User", fMsg.User.ID, "RetryLeft", maxTries-fMsg.RetryAttempt)
					case <-rootContext.Done():
						return
					}
				}
			}
			for {
				select {
				case <-rootContext.Done():
					return
				case failedUserID, ok := <-failedEmailsChan:
					if !ok {
						return
					}
					processFailedEmailSending(failedUserID)
				}
			}
		}(i)
	}
}
