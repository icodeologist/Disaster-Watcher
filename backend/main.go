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
	"github.com/icodeologist/disasterwatch/internal/utils"
	"github.com/icodeologist/disasterwatch/internal/worker"
	"github.com/joho/godotenv"

	"github.com/icodeologist/disasterwatch/internal/db"

	"github.com/icodeologist/disasterwatch/internal/models"

	"github.com/icodeologist/disasterwatch/internal/routes"
)

func main() {
	// Logging with slog
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	//GRACEful shutdown
	// Creating a internal channel with signalNOtifiy and ctx to kill children processes
	rootCtx := context.Background()
	// each worker gets this context
	// cancel() is calle for graceful shutdown flow and workers can listen and get they have signalled to finish up and exit
	workContext, cancel := context.WithCancel(rootCtx)
	signChannel := make(chan os.Signal, 1)
	signal.Notify(signChannel, syscall.SIGINT, syscall.SIGTERM)

	// Creating buffered channels for my worker pool
	reportsChannel := make(chan models.ReportMessage, 10)
	failedEmailsChan := make(chan models.FailedEmailMessage, 10)
	// If some message failed after retry put them here and later let admin check through it
	deadLetterChannel := make(chan models.DLQJob, 100)
	affectedUsersIDChannel := make(chan models.AffectedUsersMessage, 10)
	verficationMessageChannel := make(chan models.VerificationMessage, 10)
	//retry for failed message
	maxRetries := 5

	//loading the .env file
	err := godotenv.Load(".env")
	if err != nil {
		slog.Error("Loading the .env file", "err", err)
	}

	// Connection to DB
	if err := db.Connect(); err != nil {
		log.Fatal(err)
	}
	slog.Info("Getting all jobs that are still processing")
	utils.RecoverMidProcessingJobs(verficationMessageChannel)
	// utils.GetAllDONEJOBS()
	// utils.GetAllDONEJOBS()

	// Putting all dependencies required for workers in workerServer
	workerServer := &handler.Server{
		ReportChannel:          reportsChannel,
		AffectedUsersIdChannel: affectedUsersIDChannel,
		VerificationChannel:    verficationMessageChannel,
	}

	// IN shutdown flow waiting all go routines to finish up
	var wg sync.WaitGroup

	// starting all workers
	worker.StartVerificationWorkers(workContext, &wg, 5, verficationMessageChannel, reportsChannel)
	worker.StartExtractWorkers(workContext, &wg, 5, reportsChannel, affectedUsersIDChannel)
	worker.StartNotificationWorker(workContext, &wg, 5, affectedUsersIDChannel, failedEmailsChan)
	worker.StartFailedEmailSendingWorker(workContext, &wg, 5, maxRetries, failedEmailsChan, deadLetterChannel)

	// rate limiting middleware
	ratelimitMiddleware := auth.NewRateLimiterMiddleware(1, 10)
	// gin router engine
	r := gin.Default()
	// setting up routes
	routes.SetUpRoutes(r, workerServer, ratelimitMiddleware)

	server := &http.Server{
		Addr:    ":3000",
		Handler: r.Handler(),
	}

	// starting the server in goroutine so it wont block anything
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen : %s\n", err)
		}
	}()
	// Graceful shutdown flow
	// If we reciece any interruptiong or termination
	<-signChannel
	slog.Info("recieved sigint")
	shutDownCtx, shtdownCanc := context.WithTimeout(context.Background(), 5*time.Second)
	defer shtdownCanc()
	if err := server.Shutdown(shutDownCtx); err != nil {
		slog.Info("Server shutdown : ", "error", err)
	}
	// close the root context at the end
	// Each worker sees this for {select Case rootconetxt closed?}
	cancel()
	// waiting for all the go routines to finish
	// until that this blocks
	wg.Wait()
	slog.Info("server stopped")
}
