package utils

import (
	"log"

	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
)

func RecoverPendingJobsFromDBOnStarting(verifcationChannel chan<- models.VerificationMessage) {
	var pendingJobs []models.Jobs
	if err := db.DB.Where("status=?", "pending").Find(&pendingJobs).Error; err != nil {
		log.Println("Err : ", err)
	}
	if len(pendingJobs) > 0 {
		log.Printf("[PENDING JOBS : %d]\n", len(pendingJobs))
		for _, job := range pendingJobs {
			vmsg := models.VerificationMessage{
				JobID: job.Id,
			}
			verifcationChannel <- vmsg
			log.Printf("[JOB ID : %d] [RECOVERED AND PROCESSING]\n", job.Id)
		}
	} else {
		log.Println("[NO PENDING JOBS]")
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
		for j := range job {
			log.Println("Job DONE : ", j)
		}
	}
}
