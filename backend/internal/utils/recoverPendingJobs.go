package utils

import (
	"log"

	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
)

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
