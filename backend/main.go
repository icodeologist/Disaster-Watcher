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
// set up and calling all the essentials

func main() {
	reportsChannel := make(chan models.Report, 10)
	affectedUsersIDChannel := make(chan uint, 10)
	if err := db.Connect(); err != nil {
		log.Fatal(err)
	}
	server := &handler.Server{
		ReportChannel:          reportsChannel,
		AffectedUsersIdChannel: affectedUsersIDChannel,
	}
	worker.StartExtractWorkers(5, reportsChannel, affectedUsersIDChannel)
	worker.StartNotificationWorker(5, affectedUsersIDChannel)

	r := gin.Default()
	routes.SetUpRoutes(r, server)
	r.Run(":3000")

}
