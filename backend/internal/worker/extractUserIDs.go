package worker

import (
	"fmt"
	"log"

	"github.com/icodeologist/disasterwatch/internal/db"
	emailservice "github.com/icodeologist/disasterwatch/internal/email_service"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/icodeologist/disasterwatch/internal/utils"
)

func CallExtractUserIDWorker(reportChannel chan models.Report, affUsersIdChannel chan uint) {
	for report := range reportChannel {
		go ExtractUserIDs(report, affUsersIdChannel)
	}
}

func ExtractUserIDs(report models.Report, affUsersIDsChannle chan uint) {
	affUsersIDsChannle <- uint(report.UserId)
}

func PrintInfo(affUsersIdChannel chan uint) {
	println("Len of affected users CHANNEL :", len(affUsersIdChannel))
	for id := range affUsersIdChannel {
		fmt.Println("USERID :", id)
	}
}

func StartExtractWorkers(n int, reportChannel <-chan models.Report, affUserIDChannel chan<- uint) {
	// start n of workers
	// TODO: find real affected users
	for i := 0; i < n; i++ {
		go utils.GetUsersAffectedByDisaster(reportChannel, affUserIDChannel)
	}
}

func StartNotificationWorker(n int, affUsersIdChannel <-chan uint) {
	fmt.Println("Started Notification Workers")
	for i := 0; i < n; i++ {
		go func(workerID int) {
			var user models.User
			for userID := range affUsersIdChannel {
				if err := db.DB.Where("id=?", userID).First(&user).Error; err != nil {
					log.Printf("DATABASE_ERR : %v\n", err)
					continue
				}
				go emailservice.SendEmail(user.Email)
				fmt.Println("Email send to USER : ", user.UserName, "USER EMAIL :", user.Email)
				fmt.Println("WorkerID with :", workerID, "Sent Email to user :", user.UserName)
			}
		}(i)
	}
}
