package worker

import (
	"context"
	"fmt"
	"github.com/icodeologist/disasterwatch/internal/db"
	emailservice "github.com/icodeologist/disasterwatch/internal/email_service"
	"github.com/icodeologist/disasterwatch/internal/models"
	"log/slog"
	"math"
	"sync"
	"time"
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
					JobID:        failedUserID.JobID,
					User:         failedUserID.User,
					Report:       failedUserID.Report,
					ErrorMessage: failedUserID.ErrorMessage,
					RetryAttempt: failedUserID.RetryAttempt + 1,
					RetryDelay:   time.Duration(timeTOwait) * time.Millisecond,
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
					err := emailservice.SendEmail(emailHelper)
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
