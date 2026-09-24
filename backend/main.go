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

func waitForWorkers(ctx context.Context, workers *sync.WaitGroup) error {
	done := make(chan struct{})
	go func() {
		workers.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

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

	// Workers keep using this context while the pipeline drains. It is canceled
	// only when the shutdown deadline expires or a second signal is received.
	workContext, cancelWorkers := context.WithCancel(context.Background())
	defer cancelWorkers()
	// Keep room for the first graceful signal and a second force-stop signal.
	signalChannel := make(chan os.Signal, 2)
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

	var verificationWorkers sync.WaitGroup
	var extractionWorkers sync.WaitGroup
	var notificationWorkers sync.WaitGroup
	var retryWorkers sync.WaitGroup

	// Start each stage of the notification pipeline before accepting HTTP traffic.
	worker.StartVerificationWorkers(workContext, &verificationWorkers, 5, verificationChannel, reportsChannel)
	worker.StartExtractWorkers(workContext, &extractionWorkers, 5, reportsChannel, deliveryChannel)
	worker.StartNotificationWorker(workContext, &notificationWorkers, 5, deliveryChannel, retryDeliveryChannel)
	worker.StartFailedEmailSendingWorker(workContext, &retryWorkers, 5, maxRetries, retryDeliveryChannel, deadLetterChannel)

	// Track startup recovery separately because it can send to several worker
	// channels. Shutdown must wait for this sender before closing those channels.
	var recoveryWG sync.WaitGroup
	recoveryWG.Add(1)
	go func() {
		defer recoveryWG.Done()
		if err := utils.RecoverUnfinishedJobs(workContext, verificationChannel); err != nil && workContext.Err() == nil {
			slog.Error("Failed to recover unfinished jobs", "error", err)
		}
		if err := utils.RecoverNotificationDeliveries(workContext, deliveryChannel, retryDeliveryChannel); err != nil && workContext.Err() == nil {
			slog.Error("Failed to recover notification deliveries", "error", err)
		}
	}()

	// Build the API after all worker dependencies are ready.
	ratelimitMiddleware := auth.NewRateLimiterMiddleware(10, 5)
	ratelimitMiddleware.StartCleanup(workContext)
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

	// The first signal starts a bounded drain. The same deadline covers active
	// HTTP handlers and every worker stage.
	firstSignal := <-signalChannel
	slog.Info("Received shutdown signal", "signal", firstSignal)
	const shutdownGracePeriod = 5 * time.Minute
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
	defer shutdownCancel()

	shutdownComplete := make(chan struct{})
	defer close(shutdownComplete)
	go watchForForcedShutdown(signalChannel, shutdownCtx, shutdownComplete, cancelWorkers, shutdownCancel)

	pipeline := shutdownPipeline{
		recoveryWorkers: &recoveryWG,
		stages: []drainStage{
			{name: "verification", input: func() { close(verificationChannel) }, workers: &verificationWorkers},
			{name: "report extraction", input: func() { close(reportsChannel) }, workers: &extractionWorkers},
			{name: "notification delivery", input: func() { close(deliveryChannel) }, workers: &notificationWorkers},
			{name: "notification retry", input: func() { close(retryDeliveryChannel) }, workers: &retryWorkers},
		},
		closeAfterDrain: func() { close(deadLetterChannel) },
	}
	if err := shutdownGracefully(shutdownCtx, server, pipeline); err != nil {
		slog.Warn("Graceful shutdown did not finish; unfinished work will be recovered on restart", "error", err)
		cancelWorkers()
		return
	}

	slog.Info("All accepted channel work finished before shutdown")
	slog.Info("server stopped")
}
