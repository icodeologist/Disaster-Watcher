package worker

import (
	"context"
	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/icodeologist/disasterwatch/internal/utils"
	"log/slog"
	"sync"
)

// This is step 2:
// once verified report comes its pushed to report channel with job model attached to each report
// This job tracks the whole flow
// THe goal here is the extract all the affected users/report and add a seperate row in db for each affected users
// The reason is to track what happend to the whole job with the help of what happend to all the affected users for this job
// Once we get all affected users job status is updated from pending to processing

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
