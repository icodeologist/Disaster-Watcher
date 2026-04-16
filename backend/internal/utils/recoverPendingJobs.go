package utils

import (
	"log"
	"log/slog"
	"time"

	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
)

func RecoverPendingJobsFromDBOnStarting(verifcationChannel chan<- models.VerificationMessage) {
	var pendingJobs []models.Jobs
	if err := db.DB.Where("status=?", "pending").Find(&pendingJobs).Error; err != nil {
		slog.Error("Db update error in RecoverPendingJobsFromDBOnStarting", "error", err)
	}
	if len(pendingJobs) > 0 {
		for _, job := range pendingJobs {
			vmsg := models.VerificationMessage{
				JobID: job.Id,
			}
			job.Status = "processing"
			err := db.DB.Save(&job).Error
			if err != nil {
				slog.Error("Db save error in RecoverPendingJobsFromDBOnStarting", "error", err)
			}
			verifcationChannel <- vmsg
			slog.Info("Job restarted", "started_time", time.Now())
		}
	} else {
		slog.Info("No pending jobs in DB")
	}
}
func RecoverMidProcessingJobs(verifcationChannel chan<- models.VerificationMessage) {
	var midProcessingJobs []models.Jobs
	if err := db.DB.Where("status=?", "pending").Find(&midProcessingJobs).Error; err != nil {
		slog.Error("failed to fetch the jobs with processing status", "error", err)
	}
	if len(midProcessingJobs) > 0 {
		for _, job := range midProcessingJobs {
			vmsg := models.VerificationMessage{
				JobID: job.Id,
			}
			verifcationChannel <- vmsg
			slog.Info("Job restarted", "started_time", time.Now())
		}
	} else {
		slog.Info("No Processing jobs in DB")
	}
}

func GetAllDONEJOBS() {
	var job []models.Jobs
	if err := db.DB.Where("status=?", "done").Find(&job).Error; err != nil {
		log.Println("ERR [while find jobs wiht 'done' ] : ", err)
	}
	if len(job) == 0 {
		log.Println("NO JOBS FAILED")
	} else {
		log.Println("THE LENGTH OF JOBS WITH PENDING STATUS : ", len(job))
		for _, j := range job {
			log.Println("Job DONE : ", j)
		}
	}
}

func GetAllInfoFromDone() {
	var job []models.Jobs
	if err := db.DB.Find(&job).Error; err != nil {
		log.Printf("ERROR : %v\n", err)
		panic(err)
	}
	if len(job) == 0 {
		log.Println("NO JOBS FAILED")
	} else {
		log.Println("THE LENGTH OF JOBS WITH DONE STATUS : ", len(job))
		for _, j := range job {
			log.Println("Job DONE : ", j)
		}
	}
}
