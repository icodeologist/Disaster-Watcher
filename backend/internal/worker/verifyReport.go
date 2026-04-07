package worker

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/icodeologist/disasterwatch/internal/utils"
)

// helper functions to hit the DB
func fetchReport(reportID uint) (models.Report, error) {
	var report models.Report
	if err := db.DB.Where("id=?", reportID).First(&report).Error; err != nil {
		log.Println("Failed report")
		return models.Report{}, err
	}
	return report, nil
}

func fetchJobById(jobID int64) (models.Jobs, error) {
	var job models.Jobs
	if err := db.DB.Where("id=?", jobID).First(&job).Error; err != nil {
		log.Println("Failed job")
		return models.Jobs{}, err
	}
	return job, nil
}

func fetchUserById(userId uint) (models.User, error) {
	var user models.User
	if err := db.DB.Where("id=?", userId).First(&user).Error; err != nil {
		log.Println("Failed user")
		return models.User{}, err
	}
	return user, nil
}

// Start of notification pipeline
func StartVerificationWorkers(rootCtx context.Context, wg *sync.WaitGroup, n int, verificationMsgChannel chan models.VerificationMessage, reportChan chan models.ReportMessage) {
	log.Println("STARTED [VERIFICATION PROCESS WORKERS]")
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// if something unexpected happens recover the panic
			defer func() {
				if r := recover(); r != nil {
					log.Printf("RECOVERED IN VERIFICATION WORKERS : %v\n ", r)
					return
				}
			}()
			processCurrentJob := func(verifyMsg models.VerificationMessage) {
				currentJob, err := fetchJobById(verifyMsg.JobID)
				if err != nil {
					// not stopping here just keep the pending status and increment attempts
					// next cycle process
					// if max attempts reached may be quit
					// TODO:
					log.Printf("Failed to fetch job %d : %v\n", verifyMsg.JobID, err)
					return
				}

				var payloadData models.PayloadData
				if err := json.Unmarshal(currentJob.Payload, &payloadData); err != nil {
					err := db.DB.Exec("UPDATE jobs SET status='failed' WHERE id=$1", currentJob.Id).Error
					if err != nil {
						log.Printf("Failed to update to database : %v\n", err)
					}
					return
				}
				report, err := fetchReport(payloadData.ReportID)
				if err != nil {
					err := db.DB.Exec("UPDATE jobs SET status= 'pending', attempts = attempts+1 WHERE id = $1", currentJob.Id).Error
					if err != nil {
						log.Printf("Failed to update to database : %v\n", err)
					}
					return
				}
				user, err := fetchUserById(payloadData.UserID)
				if err != nil {
					err := db.DB.Exec("UPDATE jobs SET status= 'pending', attempts = attempts+1 WHERE id = $1", currentJob.Id).Error
					if err != nil {
						log.Printf("Failed to update to database : %v\n", err)
					}
					return
				}
				println("Done user fetch")
				utils.VerifyReportPostedByUser(rootCtx, &report, user, reportChan, verifyMsg.JobID)
				log.Println("process current started")
				log.Println("Started the Verification process")
			}
			for {
				select {
				// if the root context channel is closed then graceful shutdown flow
				case <-rootCtx.Done():
					return
				case verifyMsg, ok := <-verificationMsgChannel:
					if !ok {
						return
					}
					log.Printf("VERIFICATION WORKER %d [STARTED VERIFYING REPORTS]\n", id)
					processCurrentJob(verifyMsg)
					log.Println("come out of it")
				}
			}
		}(i)
	}
}
