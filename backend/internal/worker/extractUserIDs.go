package worker

import (
	"context"
	"log"
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
	log.Println("Extracting User IDs: ", n, " WORKERS STARTED")

	var allUsers []models.User

	if err := db.DB.Find(&allUsers).Error; err != nil {
		log.Fatalf("Error Fetching all users from db : %v\n", err)
	}
	log.Printf("[EXTRACT] loaded %d users\n", len(allUsers))
	for i := 0; i < n; i++ {
		wg.Add(1)
		go utils.GetUsersAffectedByDisaster(rootContext, wg, allUsers, reportChannel, affUserIDChannel)
	}
}

type EmailrateLimiter struct {
	limiter *rate.Limiter
}

func StartNotificationWorker(rootContext context.Context, wg *sync.WaitGroup, n int, affUsersIdChannel <-chan models.AffectedUsersMessage, failedEmailsChan chan<- models.FailedEmailMessage) {
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
					log.Printf("RECOVERED IN NOTIFICATION WORKER : %v", r)
				}
			}()

			processSendingEmail := func(affUserMsg models.AffectedUsersMessage) {
				var user models.User
				if err := db.DB.Where("id=?", affUserMsg.UserID).First(&user).Error; err != nil {
					log.Printf("DATABASE_ERR : %v\n", err)
					// FIX: better way to deal with db error ?
				}
				if err := emailRateL.limiter.Wait(rootContext); err != nil {
					log.Println("Error : ", err)
					return
				}
				log.Printf("[SENDING TO ID %v] [1st TIME]", affUserMsg.UserID)
				err := emailservice.SendEmail(user.Email)
				if err != nil {
					log.Printf("[FAILED] [ID : %v] [PUSHING TO FAILED MESSAGE CHANNEL]\n", affUserMsg.UserID)
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
					case <-rootContext.Done():
						return
					}
				} else {
					var job models.Jobs
					if err := db.DB.Where("id=?", affUserMsg.JobID).First(&job).Error; err != nil {
						log.Printf("Error after passing [SUCCESS NOTIFICATION SENDING] %v\n", err)
					} else {
						job.Status = "done"
						if err := db.DB.Save(&job).Error; err != nil {
							log.Printf("ERROR [SAVING SUCCEED JOB TO DB] : %v\n", err)
						} else {
							log.Println("SUCCEED SAVING THE JOB TO DB WITH [SUCCEEDED status]")
						}
					}
				}

			}
			for {
				select {
				case <-rootContext.Done():
					return
				case affectedUserMsg, ok := <-affUsersIdChannel:
					if !ok {
						return
					}
					processSendingEmail(affectedUserMsg)
				}
			}
		}(i)
	}
}

func StartFailedEmailSendingWorker(rootContext context.Context, wg *sync.WaitGroup, n int, maxTries int, failedEmailsChan chan models.FailedEmailMessage, deadMessageChannel chan models.FailedEmailMessage) {
	log.Println("STARTED RETRYING FAILED MESSAGES WORKERS")
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			//recover if this go routiner panicks at some point
			defer func() {
				if r := recover(); r != nil {
					log.Printf("RECOVERED IN FAILED NOTIFICATION WORKER : %v", r)
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
						return
					}
					log.Printf("[RETRY TIME : %v] [ID : %v] [DELAY : %v]\n", fMsg.RetryTime, fMsg.UserId, fMsg.RetryDelay)
					err := emailservice.SendEmail(failedUserID.User.Email)
					if err != nil {
						select {
						case failedEmailsChan <- *fMsg:
						case <-rootContext.Done():
							return
						}
					} else {
						log.Printf("[SUCCESS] [ID : %v] [FAILED COUNT : %v]\n", fMsg.UserId, fMsg.RetryTime)
						var job models.Jobs
						if err := db.DB.Where("id=?", failedUserID.JobID).First(&job).Error; err != nil {
							log.Printf("Error after passing [SUCCESS DB FAILED TO FECTH JOB] %v\n", err)
						} else {
							job.Status = "passed"
							if err := db.DB.Save(&job).Error; err != nil {
								log.Printf("ERROR [SAVING SUCCEED JOB TO DB] : %v\n", err)
							} else {
								log.Println("SUCCEED SAVING THE JOB TO DB WITH [SUCCEEDED status]")
							}
						}
					}
				} else {
					log.Printf("[FAILED - MAX TRIES REACHED] [ID : %v] [PUSHED TO DEAD CHANNEL]", fMsg.UserId)
					var job models.Jobs
					log.Println("JOB ID failedUserID.JOBID : ", failedUserID.JobID)
					log.Println("JOB ID fmsg.JOBID : ", fMsg.JobID)
					if err := db.DB.Where("id=?", failedUserID.JobID).First(&job).Error; err != nil {
						log.Printf("Error after passing [SUCCESS NOTIFICATION SENDING] %v\n", err)
					} else {
						job.Status = "failed"
						if err := db.DB.Save(&job).Error; err != nil {
							log.Printf("ERROR [SAVING SUCCEED JOB TO DB] : %v\n", err)
						} else {
							log.Println("JOB FAILED | SAVING THE JOB TO DB WITH [FAILED status]")
						}
					}
					select {
					case deadMessageChannel <- *fMsg:
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
