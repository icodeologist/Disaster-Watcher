package utils

import (
	"fmt"
	"time"
	// "github.com/icodeologist/disasterwatch/internal/models"
)

func ExtractReportDetails(jobs chan uint) {
	for job := range jobs {
		fmt.Println("REPORT ID : ", job)
		time.Sleep(2 * time.Second)
	}
}
