package worker

import (
	"context"
	"github.com/icodeologist/disasterwatch/internal/db"
	emailservice "github.com/icodeologist/disasterwatch/internal/email_service"
	"github.com/icodeologist/disasterwatch/internal/models"
	"golang.org/x/time/rate"
	"log/slog"
	"sync"
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

			processSendingEmail := func(affUserMsg models.AffectedUsersMessage) {
				var user models.User
				if err := db.DB.Where("id=?", affUserMsg.UserID).First(&user).Error; err != nil {
					slog.Error("failed to fetch user from db", "user_id", affUserMsg.UserID, "error", err)
					return
					// FIX: better way to deal with db error ?
				}
				if err := emailRateL.limiter.Wait(rootContext); err != nil {
					slog.Warn("email rate limiter wait interrupted", "error", err)
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
					failedMessage := &models.FailedEmailMessage{
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
				} else {
					var job models.Jobs
					if err := db.DB.Where("id=?", affUserMsg.JobID).First(&job).Error; err != nil {
						slog.Error("Failed to fetch the job from DB", "Occured in", "After sending email")
					} else {
						job.Status = "done"
						if err := db.DB.Save(&job).Error; err != nil {
							slog.Error("Failed to update job", "error", err)
						}
					}
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
