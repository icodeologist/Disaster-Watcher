// user_api_endpoints.go contains all the CRUD endpoint functions.
package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	database "github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/icodeologist/disasterwatch/internal/utils"
	"gorm.io/gorm"
)

func (s *Server) CreateReport(c *gin.Context) {
	var report models.Report
	if err := c.ShouldBindJSON(&report); err != nil {
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

	userIDValue, exists := c.Get("userId")
	userID, validUserID := userIDValue.(uint)
	if !exists || !validUserID {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode: "UNAUTHORIZED",
				Message:   "authenticated user is required",
			},
		})
		return
	}
	report.UserId = userID

	// Geocoding is an external call. Do it before opening the transaction so
	// the transaction stays short and never holds a database connection while
	// waiting on the network.
	if !report.ISLocationCached || report.CachedLat == nil || report.CachedLong == nil {
		location, err := utils.GetLATLONGfromUserLocation(c.Request.Context(), report.Location)
		if err != nil {
			slog.Error("failed to geocode report location", "error", err)
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Success: false,
				Error: models.Error{
					ErrorCode: "CACHING_COORDINATES_ERR",
					Message:   "unable to resolve report location",
				},
			})
			return
		}
		report.CachedLat = &location.Lat
		report.CachedLong = &location.Long
		report.ISLocationCached = true
	}

	var job models.Jobs
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&report).Error; err != nil {
			return fmt.Errorf("create report: %w", err)
		}

		payload, err := json.Marshal(models.PayloadData{
			ReportID: report.ID,
			UserID:   report.UserId,
		})
		if err != nil {
			return fmt.Errorf("marshal report job payload: %w", err)
		}

		now := time.Now()
		job = models.Jobs{
			Status:     "processing",
			Payload:    payload,
			Created_at: now,
			Started_at: now,
		}
		if err := tx.Create(&job).Error; err != nil {
			return fmt.Errorf("create report job: %w", err)
		}
		return nil
	})
	if err != nil {
		slog.Error("failed to persist report and job", "error", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode:    "DATABASE_ERR",
				Message:      "report was not accepted",
				ErrorDetails: err.Error(),
			},
		})
		return
	}

	// Publish only after the transaction commits. If the queue is full, the
	// durable processing job remains available to startup recovery.
	select {
	case s.VerificationChannel <- models.VerificationMessage{JobID: job.Id}:
		slog.Info("report accepted", "report_id", report.ID, "job_id", job.Id)
		c.JSON(http.StatusAccepted, models.SuccessResponse{
			Success: true,
			Data:    report,
			Message: "Successfully created the report. Wait for the verification process.",
		})
	default:
		c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode: "QUEUE_FULL",
				Message:   "report was stored but could not be queued immediately; it will be recovered",
			},
		})
	}
}
