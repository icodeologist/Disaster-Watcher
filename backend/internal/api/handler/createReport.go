// user_api_endpoints.go contains all the crud endpoint functions
package handler

import (
	// "context"
	"encoding/json"
	"net/http"
	"time"

	// "time"

	// "strconv"
	//
	// "fmt"
	"log"

	"github.com/gin-gonic/gin"
	database "github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/icodeologist/disasterwatch/internal/utils"
)

func (s *Server) CreateReport(c *gin.Context) {
	var userReport models.Report
	if err := c.ShouldBindJSON(&userReport); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode:    "INVALID_JSON",
				Message:      "User request either has empty or invalid json",
				ErrorDetails: err.Error(),
			},
		})
		return
	}
	userId, exists := c.Get("userId")
	log.Printf("UserID :%v\n", userId)
	if !exists {
		log.Fatalf("%v user id does not exist", userId)
	}
	userReport.UserId = userId.(uint)

	if err := database.DB.Create(&userReport).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode:    "DATABASE_ERR",
				ErrorDetails: err.Error(),
			},
		})
		return
	}
	println("Done create")
	if err := database.DB.Preload("User").First(&userReport, userReport.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode:    "DATABASE_ERR",
				ErrorDetails: err.Error(),
			},
		})
		return
	}
	println("done fetch")

	if err := utils.ConvertReportLocationTOLatAndLong(&userReport); err != nil {
		println("err : ", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode: "CACHING_COORDINATES_ERR",
				// FIX: Make something about this
				Message:      "Add the exact land mark not vague location",
				ErrorDetails: err.Error(),
			},
		})
		return
	}
	println("done convert")
	// data that we will send to other worker stream
	payloadData := models.PayloadData{
		ReportID: userReport.ID,
		UserID:   userReport.UserId,
	}
	// payloadData struct to byte
	payLoadBytes, err := json.Marshal(payloadData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode:    "JSON_MARSHAL_ERR",
				Message:      "Converting payload data struct to bytes failed",
				ErrorDetails: err.Error(),
			},
		})
		return
	}
	println("Done paylaod marshal")
	job := models.Jobs{
		Status:     "pending",
		Payload:    payLoadBytes,
		Created_at: time.Now(),
	}
	log.Println("JOB [CREATED] at : ", job.Created_at)

	// making each job persist
	if err := database.DB.Create(&job).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode:    "DATABASE_ERR",
				ErrorDetails: err.Error(),
			},
		})
		return
	}
	println("done create job")
	// push to channel in select block under normal flow
	verificationMsg := models.VerificationMessage{
		JobID: job.Id,
	}
	println("Now select block")
	select {
	case s.VerificationChannel <- verificationMsg:
		job.Started_at = time.Now()
		err := database.DB.Save(&job).Error
		if err != nil {
			log.Println("ERR : ", err)
		}
		log.Println("JOB [STARTED] at : ", job.Started_at)
		c.JSON(http.StatusAccepted, models.SuccessResponse{
			Success: true,
			Data:    userReport,
			Message: "Successfully created the report. Wait for the verfication process.",
		})
	default:
		c.JSON(503, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode: "QUEUE_FULL",
				Message:   "Worker queue is currently full.",
			},
		})
	}
}
