// user_api_endpoints.go contains all the crud endpoint functions
package handler

import (
	// "context"
	"net/http"
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
	if !exists {
		log.Fatalf("%v user id does not exist", userId)
	}
	println("userID : ", userId)
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

	if err := utils.ConvertReportLocationTOLatAndLong(&userReport); err != nil {
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
	verificationMsg := VerificationMessage{
		Report: userReport,
		User:   userReport.User,
	}

	select {
	case s.VerificationChannel <- verificationMsg:
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
