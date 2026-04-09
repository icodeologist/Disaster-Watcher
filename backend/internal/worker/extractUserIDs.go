package worker

import (
	"context"
	"log"
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
					log.Printf("DATABASE_ERR : %v\n", err)
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
						JobID:     affUserMsg.JobID,
						UserId:    affUserMsg.UserID,
						User:      user,
						Status:    "not sent",
						Message:   "Message",
						RetryTime: 1,
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
						slog.Error("No jobs found for affUsersIdChannel")
						return
					}
					processSendingEmail(affectedUserMsg)
				}
			}
		}(i)
	}
}

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
				timeTOwait := math.Pow(2, float64(failedUserID.RetryTime))
				fMsg := &models.FailedEmailMessage{
					JobID:      failedUserID.JobID,
					User:       failedUserID.User,
					UserId:     failedUserID.UserId,
					Status:     "Email provider error",
					Message:    "message",
					RetryTime:  failedUserID.RetryTime + 1,
					RetryDelay: time.Duration(timeTOwait),
				}
				if fMsg.RetryTime <= maxTries {
					select {
					case <-time.After(time.Duration(timeTOwait) * time.Second):
					case <-rootContext.Done():
						slog.Info("shutdown fired, stopping workers")
						return
					}
					slog.Info("Retrying", "User", fMsg.UserId, "Retry time", fMsg.RetryTime, "Retrying again in", fMsg.RetryDelay.Minutes())
					err := emailservice.SendEmail(failedUserID.User.Email)
					if err != nil {
						select {
						case failedEmailsChan <- *fMsg:
						case <-rootContext.Done():
							return
						}
					} else {
						slog.Info("Successusfully Send the email", "User", "fmsg.UserId", "Failed", fMsg.RetryTime)
						var job models.Jobs
						if err := db.DB.Where("id=?", failedUserID.JobID).First(&job).Error; err != nil {
							slog.Error("Failed to fetch the job from DB", "User", fMsg.UserId, "Error", err)
						} else {
							job.Status = "done"
							if err := db.DB.Save(&job).Error; err != nil {
								slog.Error("Failed to update the job status to DB", "User", fMsg.UserId, "Error", err)
							}
						}
					}
				} else {
					log.Printf("[FAILED - MAX TRIES REACHED] [ID : %v] [PUSHED TO DEAD CHANNEL]", fMsg.UserId)
					slog.Info("Notification Failed, Sending to DeadLetterChannel", "Tries Left", maxTries-fMsg.RetryTime)
					var job models.Jobs
					if err := db.DB.Where("id=?", failedUserID.JobID).First(&job).Error; err != nil {
						slog.Error("Failed to fetch the job from DB", "User", fMsg.UserId, "Error", err)
					} else {
						job.Status = "failed"
						if err := db.DB.Save(&job).Error; err != nil {
							slog.Error("Failed to update the job status to DB", "User", fMsg.UserId, "Error", err)
						}
					}
					fmsgInfo := models.FailedInfo{
						AttemptedTime: fMsg.RetryTime,
					}

					dlqJob := models.DLQJob{
						Error:         "Failed its max attempts of retry",
						FailedMsgInfo: fmsgInfo.AttemptedTime,
					}
					select {
					case deadMessageChannel <- dlqJob:
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
