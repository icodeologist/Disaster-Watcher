package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/icodeologist/disasterwatch/internal/utils"
)

// helper functions to hit the DB
func fetchReport(reportID uint) (models.Report, error) {
	var report models.Report
	if err := db.DB.Where("id=?", reportID).First(&report).Error; err != nil {
		slog.Error("Failed to fetch Report", "Report", report, "error", err)
		return models.Report{}, err
	}
	return report, nil
}

func fetchJobById(jobID int64) (models.Jobs, error) {
	var job models.Jobs
	if err := db.DB.Where("id=?", jobID).First(&job).Error; err != nil {
		slog.Error("Failed to fetch Job", "Job", job, "error", err)
		return models.Jobs{}, err
	}
	return job, nil
}

func fetchUserById(userId uint) (models.User, error) {
	var user models.User
	if err := db.DB.Where("id=?", userId).First(&user).Error; err != nil {
		slog.Error("Failed to fetch User", "User", user, "error", err)
		return models.User{}, err
	}
	return user, nil
}

// Start of notification pipeline
func StartVerificationWorkers(rootCtx context.Context, wg *sync.WaitGroup, n int, verificationMsgChannel chan models.VerificationMessage, reportChan chan models.ReportMessage) {
	slog.Info("VERIFICATION WORKERS STARTED", "COUNT", n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// if something unexpected happens recover the panic
			defer func() {
				if r := recover(); r != nil {
					slog.Error("Panic Recoverd [VERFICATION WORKERS]", "panic", r)
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
					slog.Error("", "Error ", err)
					return
				}

				var payloadData models.PayloadData
				if err := json.Unmarshal(currentJob.Payload, &payloadData); err != nil {
					err := db.DB.Exec("UPDATE jobs SET status='failed' WHERE id=$1", currentJob.Id).Error
					if err != nil {
						slog.Warn("Failed to Update the job to DB", "Error", err)
					}
					return
				}
				report, err := fetchReport(payloadData.ReportID)
				if err != nil {
					err := db.DB.Exec("UPDATE jobs SET status= 'pending', attempts = attempts+1 WHERE id = $1", currentJob.Id).Error
					if err != nil {
						slog.Warn("Failed to Update the job to DB", "Error", err)
					}
					return
				}
				user, err := fetchUserById(payloadData.UserID)
				if err != nil {
					err := db.DB.Exec("UPDATE jobs SET status= 'pending', attempts = attempts+1 WHERE id = $1", currentJob.Id).Error
					if err != nil {
						slog.Warn("Failed to Update the job to DB", "Error", err)
					}
					return
				}
				utils.VerifyReportPostedByUser(rootCtx, &report, user, reportChan, verifyMsg.JobID)
			}
			for {
				select {
				// if the root context channel is closed then graceful shutdown flow
				case <-rootCtx.Done():
					return
				case verifyMsg, ok := <-verificationMsgChannel:
					if !ok {
						slog.Warn("No Jobs found", "Channel", "[VERIFICATION CHANNEL]")
						return
					}
					processCurrentJob(verifyMsg)
				}
			}
		}(i)
	}
}
