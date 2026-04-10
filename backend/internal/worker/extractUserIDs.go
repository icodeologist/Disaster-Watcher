package worker

import (
	"context"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/icodeologist/disasterwatch/internal/db"
	emailservice "github.com/icodeologist/disasterwatch/internal/email_service"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/icodeologist/disasterwatch/internal/utils"
	"golang.org/x/time/rate"
)

// Get the users effected from the disasters and push their id to affuserchannel
func StartExtractWorkers(rootContext context.Context, wg *sync.WaitGroup, n int, reportChannel <-chan models.ReportMessage, affUserIDChannel chan<- models.AffectedUsersMessage) {
	// start n of workers
	slog.Info("EXTRACTUSERS WORKERS STARTED", "COUNT", n)

	var allUsers []models.User

	if err := db.DB.Find(&allUsers).Error; err != nil {
		slog.Error("Failed to get all users", "Error", err)
	}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go utils.GetUsersAffectedByDisaster(rootContext, wg, allUsers, reportChannel, affUserIDChannel)
	}
}

type EmailrateLimiter struct {
	limiter *rate.Limiter
}

func StartNotificationWorker(rootContext context.Context, wg *sync.WaitGroup, n int, affUsersIdChannel <-chan models.AffectedUsersMessage, failedEmailsChan chan<- models.FailedEmailMessage) {
	slog.Info("NOTIFICATION WORKERS STARTED", "COUNT", n)
	emailRateL := &EmailrateLimiter{
		limiter: rate.NewLimiter(2, 5),
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
				err := emailservice.SendEmail(user.Email)
				if err != nil {
					slog.Warn("Failed to send Email", "User", affUserMsg.UserID)
					failedMessage := &models.FailedEmailMessage{
						JobID:        affUserMsg.JobID,
						User:         user,
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

//FIX: retry for first should be 1s then 2s then 4s 8s 16s

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
					JobID:        failedUserID.JobID,
					User:         failedUserID.User,
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
					err := emailservice.SendEmail(failedUserID.User.Email)
					if err != nil {
						select {
						case failedEmailsChan <- *fMsg:
						case <-rootContext.Done():
							return
						}
					} else {
						slog.Info("email sent successfully", "User", fMsg.User.ID, "attempt", fMsg.RetryAttempt)
						var job models.Jobs
						if err := db.DB.Where("id=?", failedUserID.JobID).First(&job).Error; err != nil {
							slog.Error("Failed to fetch the job from DB", "User", fMsg.User.ID, "Error", err)
						} else {
							job.Status = "done"
							if err := db.DB.Save(&job).Error; err != nil {
								slog.Error("Failed to update the job status to DB", "User", fMsg.User.ID, "Error", err)
							}
						}
					}
				} else {
					var job models.Jobs
					if err := db.DB.Where("id=?", failedUserID.JobID).First(&job).Error; err != nil {
						slog.Error("Failed to fetch the job from DB", "User", fMsg.User.ID, "Error", err)
					} else {
						job.Status = "failed"
						if err := db.DB.Save(&job).Error; err != nil {
							slog.Error("Failed to update the job status to DB", "User", fMsg.User.ID, "Error", err)
						}
					}
					fmsgInfo := models.FailedInfo{
						Error: fMsg.ErrorMessage,
					}

					dlqJob := models.DLQJob{
						ErrorInfo:      fmsgInfo.Error.Error(),
						FailedMsgJOBID: fMsg.JobID,
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
