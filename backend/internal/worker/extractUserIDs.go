package worker

import (
	"context"
	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/icodeologist/disasterwatch/internal/utils"
	"log/slog"
	"sync"
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
