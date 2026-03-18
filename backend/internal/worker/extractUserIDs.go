package worker

import (
	"context"
	"log"
	"math"
	"time"

	"github.com/icodeologist/disasterwatch/internal/db"
	emailservice "github.com/icodeologist/disasterwatch/internal/email_service"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/icodeologist/disasterwatch/internal/utils"
	"golang.org/x/time/rate"
)

// Get the users effected from the disasters and push their id to affuserchannel
func StartExtractWorkers(n int, reportChannel <-chan models.Report, affUserIDChannel chan<- uint) {
	// start n of workers
	// TODO: add a recover defer function for each go routines spawned
	log.Println("Extracting User IDs: ", n, " WORKERS STARTED")
	for i := 0; i < n; i++ {
		go utils.GetUsersAffectedByDisaster(reportChannel, affUserIDChannel)
	}
}

type EmailrateLimiter struct {
	limiter *rate.Limiter
}
type FailedEmailMessage struct {
	UserId     uint
	User       models.User
	Status     string
	Message    string
	RetryTime  int
	RetryDelay time.Duration
}

func StartNotificationWorker(n int, affUsersIdChannel <-chan uint, failedEmailsChan chan<- FailedEmailMessage) {
	emailRateL := &EmailrateLimiter{
		limiter: rate.NewLimiter(2, 5),
	}

	for i := 0; i < n; i++ {
		go func(id int) {
			for userID := range affUsersIdChannel {
				var user models.User
				if err := db.DB.Where("id=?", userID).First(&user).Error; err != nil {
					log.Printf("DATABASE_ERR : %v\n", err)
					continue
				}
				ctx := context.Background()

				if err := emailRateL.limiter.Wait(ctx); err != nil {
					log.Println("Error : ", err)
					return
				}
				log.Printf("[SENDING TO ID %v] [1st TIME]", userID)
				err := emailservice.SendEmail(user.Email)
				if err != nil {
					log.Printf("[FAILED] [ID : %v] [PUSHING TO FAILED MESSAGE CHANNEL]\n", userID)
					failedMessage := &FailedEmailMessage{
						UserId:    userID,
						User:      user,
						Status:    "not sent",
						Message:   "Message",
						RetryTime: 1,
					}
					failedEmailsChan <- *failedMessage
				} else {
					log.Printf("[SUCCESS] [ID : %v] [NOTIFICATION SENT]\n", userID)
				}

			}
		}(i)
	}
}

func StartFailedEmailSendingWorker(n int, maxTries int, failedEmailsChan chan FailedEmailMessage, deadMessageChannel chan FailedEmailMessage) {
	log.Println("STARTED RETRYING FAILED MESSAGES WORKERS")
	for i := 0; i < n; i++ {
		go func(id int) {
			for failedUserIDs := range failedEmailsChan {
				timeTOwait := math.Pow(2, float64(failedUserIDs.RetryTime))
				fMsg := &FailedEmailMessage{
					User:       failedUserIDs.User,
					UserId:     failedUserIDs.UserId,
					Status:     "Email provider error",
					Message:    "message",
					RetryTime:  failedUserIDs.RetryTime + 1,
					RetryDelay: time.Duration(timeTOwait),
				}
				if fMsg.RetryTime <= maxTries {
					time.Sleep(time.Duration(timeTOwait) * time.Second)
					log.Printf("[RETRY TIME : %v] [ID : %v] [DELAY : %v]\n", fMsg.RetryTime, fMsg.UserId, fMsg.RetryDelay)
					err := emailservice.SendEmail(failedUserIDs.User.Email)
					if err != nil {
						failedEmailsChan <- *fMsg
					} else {
						log.Printf("[SUCCESS] [ID : %v] [FAILED COUNT : %v]\n", fMsg.UserId, fMsg.RetryTime)
					}
				} else {
					log.Printf("[FAILED - MAX TRIES REACHED] [ID : %v] [PUSHED TO DEAD CHANNEL]", fMsg.UserId)
					deadMessageChannel <- *fMsg
				}
			}
		}(i)
	}
}
