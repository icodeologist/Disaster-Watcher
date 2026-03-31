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
func StartExtractWorkers(rootContext context.Context, wg *sync.WaitGroup, n int, reportChannel <-chan models.Report, affUserIDChannel chan<- uint) {
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
type FailedEmailMessage struct {
	UserId     uint
	User       models.User
	Status     string
	Message    string
	RetryTime  int
	RetryDelay time.Duration
}

func StartNotificationWorker(rootContext context.Context, wg *sync.WaitGroup, n int, affUsersIdChannel <-chan uint, failedEmailsChan chan<- FailedEmailMessage) {
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

			processSendingEmail := func(userID uint) {
				var user models.User
				if err := db.DB.Where("id=?", userID).First(&user).Error; err != nil {
					log.Printf("DATABASE_ERR : %v\n", err)
					// FIX: better way to deal with db error ?
				}
				if err := emailRateL.limiter.Wait(rootContext); err != nil {
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
					// same logic
					// if shutdown fired and worker wait to send
					// drop it assuming downline workers already left
					select {
					case failedEmailsChan <- *failedMessage:
					case <-rootContext.Done():
						return
					}
				} else {
					log.Printf("[SUCCESS] [ID : %v] [NOTIFICATION SENT]\n", userID)
				}

			}
			for {
				select {
				case <-rootContext.Done():
					if len(affUsersIdChannel) == 0 {
						log.Printf("NOTIFICATION worker %d cancelled, exiting", id)
						return
					} else if len(affUsersIdChannel) > 0 {
						userID, ok := <-affUsersIdChannel
						if !ok {
							return
						}
						log.Printf("PUSHING TO AFFECTED CHANNEL [DRAINING] finishing up remaining task\n")
						processSendingEmail(userID)

					}
				case userID, ok := <-affUsersIdChannel:
					if !ok {
						return
					}
					processSendingEmail(userID)
				}
			}
		}(i)
	}
}

func StartFailedEmailSendingWorker(rootContext context.Context, wg *sync.WaitGroup, n int, maxTries int, failedEmailsChan chan FailedEmailMessage, deadMessageChannel chan FailedEmailMessage) {
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
			processFailedEmailSending := func(failedUserID FailedEmailMessage) {
				timeTOwait := math.Pow(2, float64(failedUserID.RetryTime))
				fMsg := &FailedEmailMessage{
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
					}
				} else {
					log.Printf("[FAILED - MAX TRIES REACHED] [ID : %v] [PUSHED TO DEAD CHANNEL]", fMsg.UserId)
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
					if len(failedEmailsChan) == 0 {
						log.Printf("FailedMessage Retry worker %d cancelled, exiting", id)
						return
					} else if len(failedEmailsChan) > 0 {
						failedUserIdmsg, ok := <-failedEmailsChan
						if !ok {
							return
						}
						processFailedEmailSending(failedUserIdmsg)
					}
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
