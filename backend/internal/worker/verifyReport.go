package worker

import (
	"context"
	"log"
	"sync"

	"github.com/icodeologist/disasterwatch/internal/api/handler"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/icodeologist/disasterwatch/internal/utils"
)

// Start of notification pipeline
func StartVerificationWorkers(rootCtx context.Context, wg *sync.WaitGroup, n int, verificationMsgChannel chan handler.VerificationMessage, reportChan chan models.Report) {
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
			for {
				select {
				// if the root context channel is closed then graceful shutdown flow
				case <-rootCtx.Done():
					// check if all the ongoing process is finished or not
					if len(verificationMsgChannel) == 0 {
						// all process finished
						log.Printf("VERIFICATION WORKER %d cancelled , exiting\n", id)
						return
					} else if len(verificationMsgChannel) > 0 {
						// just process remaining things
						verifyMsg, ok := <-verificationMsgChannel
						if !ok {
							return
						}
						log.Printf("VERIFICATON WORKER [DRAINING] REMAINING JOBS %d\n", id)
						report := verifyMsg.Report
						user := verifyMsg.User
						log.Print("VERIFICATION STARTED WITH WORKER : ", id)
						utils.VerifyReportPostedByUser(rootCtx, &report, user, reportChan)
					}
				// this is normal flow
				case verifyMsg, ok := <-verificationMsgChannel:
					if !ok {
						return
					}
					report := verifyMsg.Report
					user := verifyMsg.User
					log.Print("VERIFICATION STARTED WITH WORKER : ", id)
					utils.VerifyReportPostedByUser(rootCtx, &report, user, reportChan)
					log.Print("Comes out from here")
				}
			}
		}(i)
	}
}
