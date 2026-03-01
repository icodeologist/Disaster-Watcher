package main

import (
	"fmt"

	// "hash/adler32"
	// "time"

	"log"

	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/internal/api/handler"

	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/routes"

	// "github.com/icodeologist/disasterwatch/internal/models"
	"github.com/icodeologist/disasterwatch/internal/utils"
	//
	//
	// "github.com/icodeologist/disasterwatch/internal/routes"
)

// everything about main here
// set up and calling all the essentials

func main() {
	if err := db.Connect(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("successfully connected to DB")
	jobs := make(chan uint, 4)
	// create 5 worke
	for i := 0; i < 2; i++ {
		go utils.ExtractReportDetails(jobs)
	}
	server := &handler.WorkerServer{
		Jobs: jobs,
	}
	// r := gin.Default()
	// r.POST("/create", server.CreateReport)
	r := gin.Default()
	routes.SetUpRoutes(r, server)
	r.Run(":3000")
}

type Server struct {
	jobs chan int
}

func (s *Server) GetReport(c *gin.Context) {
	select {
	case s.jobs <- 1:
		c.JSON(200, gin.H{"status": "queued"})
	default:
		c.JSON(503, gin.H{"error": "queue full"})
	}

}
