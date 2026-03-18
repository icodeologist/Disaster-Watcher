package worker

import (
	"github.com/icodeologist/disasterwatch/internal/api/handler"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/icodeologist/disasterwatch/internal/utils"
	"log"
)

func StartVerificationWorkers(n int, verificationMsgChannel chan handler.VerificationMessage, reportChan chan models.Report) {
	log.Println("STARTED [VERIFICATION PROCESS WORKERS]")
	for i := 0; i < n; i++ {
		go func(id int) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("RECOVERED IN VERIFICATION WORKERS : %v\n", r)
				}
			}()
			for verifyMsg := range verificationMsgChannel {
				report := verifyMsg.Report
				user := verifyMsg.User
				utils.VerifyReportPostedByUser(&report, user, reportChan)
			}
		}(i)
		log.Println("STARTED WOKRKER ID ", i, "for VERIFICATION PROCESS.")
	}
}
