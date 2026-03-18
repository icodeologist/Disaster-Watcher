package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/internal/api/handler"
	"github.com/icodeologist/disasterwatch/internal/worker"

	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/routes"

	"github.com/icodeologist/disasterwatch/internal/models"
	//
	//
	// "github.com/icodeologist/disasterwatch/internal/routes"
)

// everything about main here
//
// set up and calling all the essentials

func main() {
	reportsChannel := make(chan models.Report, 10)
	failedEmailsChan := make(chan worker.FailedEmailMessage, 10)
	deadLetterChannel := make(chan worker.FailedEmailMessage, 10)
	affectedUsersIDChannel := make(chan uint, 10)
	maxRetries := 5
	if err := db.Connect(); err != nil {
		log.Fatal(err)
	}
	server := &handler.Server{
		ReportChannel:          reportsChannel,
		AffectedUsersIdChannel: affectedUsersIDChannel,
	}
	worker.StartExtractWorkers(5, reportsChannel, affectedUsersIDChannel)
	worker.StartNotificationWorker(5, affectedUsersIDChannel, failedEmailsChan)
	worker.StartFailedEmailSendingWorker(5, maxRetries, failedEmailsChan, deadLetterChannel)

	r := gin.Default()
	routes.SetUpRoutes(r, server)
	r.Run(":3000")

}
