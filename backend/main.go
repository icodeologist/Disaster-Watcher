package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/internal/api/auth"
	"github.com/icodeologist/disasterwatch/internal/api/handler"
	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/icodeologist/disasterwatch/internal/routes"
	"github.com/icodeologist/disasterwatch/internal/utils"
	"github.com/icodeologist/disasterwatch/internal/worker"
	"github.com/joho/godotenv"
)

func main() {
	// Use one logger for the API and every background worker.
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Local development uses .env. Deployed environments can provide variables directly.
	if err := godotenv.Load(".env"); err != nil {
		slog.Error("Loading the .env file", "err", err)
	}
	if err := db.Connect(); err != nil {
		log.Fatal(err)
	}

	// This context tells every worker when graceful shutdown begins.
	workContext, cancelWorkers := context.WithCancel(context.Background())
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalChannel)

	// Channels only wake workers. Durable job and delivery data stays in PostgreSQL.
	verificationChannel := make(chan models.VerificationMessage, 10)
	reportsChannel := make(chan models.ReportMessage, 10)
	deliveryChannel := make(chan models.NotificationDeliveryMessage, 10)
	retryDeliveryChannel := make(chan models.NotificationDeliveryMessage, 10)
	deadLetterChannel := make(chan models.DLQJob, 1000)
	const maxRetries = 5

	// Log existing dead letters so failed work is visible when the service starts.
	utils.GetAllInfoFromDeadletterQueue()

	// The HTTP handlers only receive the channels they need to start background work.
	workerServer := &handler.Server{
		ReportChannel:          reportsChannel,
		AffectedUsersIdChannel: deliveryChannel,
		VerificationChannel:    verificationChannel,
	}

	var wg sync.WaitGroup

	// Start each stage of the notification pipeline before accepting HTTP traffic.
	worker.StartVerificationWorkers(workContext, &wg, 5, verificationChannel, reportsChannel)
	worker.StartExtractWorkers(workContext, &wg, 5, reportsChannel, deliveryChannel)
	worker.StartNotificationWorker(workContext, &wg, 5, deliveryChannel, retryDeliveryChannel)
	worker.StartFailedEmailSendingWorker(workContext, &wg, 5, maxRetries, retryDeliveryChannel, deadLetterChannel)

	// Rebuild channel messages for unfinished work stored before a restart.
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := utils.RecoverUnfinishedJobs(workContext, verificationChannel); err != nil && workContext.Err() == nil {
			slog.Error("Failed to recover unfinished jobs", "error", err)
		}
		if err := utils.RecoverNotificationDeliveries(workContext, deliveryChannel, retryDeliveryChannel, 10*time.Minute); err != nil && workContext.Err() == nil {
			slog.Error("Failed to recover notification deliveries", "error", err)
		}
	}()

	// Build the API after all worker dependencies are ready.
	ratelimitMiddleware := auth.NewRateLimiterMiddleware(10, 5)
	router := gin.Default()
	routes.SetUpRoutes(router, workerServer, ratelimitMiddleware)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router.Handler(),
	}

	// Run HTTP separately so main can wait for a shutdown signal.
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen : %s\n", err)
		}
	}()

	// Stop accepting requests, notify workers, and wait for their current work.
	<-signalChannel
	slog.Info("Received shutdown signal")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Info("Server shutdown : ", "error", err)
	}
	cancelWorkers()
	wg.Wait()
	slog.Info("server stopped")
}
