package handler

import (
	"time"

	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/icodeologist/disasterwatch/internal/utils"
)

func GetReportNearbyPostedLastHours(c *gin.Context) {
	hours := c.DefaultQuery("hours", "24")
	hoursInt, err := strconv.Atoi(hours)
	if err != nil {
		slog.Error("converting string to int", "err", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode:    "STR_CONV_ERR",
				ErrorDetails: err.Error(),
			},
		})
		return
	}
	cutoffTime := time.Now().Add(-time.Duration(hoursInt) * time.Hour)
	var reports []models.Report

	err = db.DB.Where("created_at >?", cutoffTime).Order("created_at DESC").Find(&reports).Error
	if err != nil {
		slog.Error("failed to fetch reports from db to get nearby reports", "err", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode:    "DB_FETCH_ERR",
				ErrorDetails: err.Error(),
			},
		})
		return
	}
	//get user id from gin context
	uId, exists := c.Get("userId")
	if !exists {
		slog.Warn("UnAuthorized")
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode:    "UNAUTHORIZED",
				ErrorDetails: "user not authenticated",
			},
		})
		return

	}
	var user models.User
	if err := db.DB.Where("user_id=?", uId).First(&user).Error; err != nil {
		slog.Error("failed to fetch user from db", "err", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode:    "DB_FETCH_ERR",
				ErrorDetails: err.Error(),
			},
		})
		return
	}
	var nearbyReports []models.Report
	for _, r := range reports {
		if !r.ISLocationCached || !user.LocationCached {
			slog.Warn("No location found", "report_location", r.Location, "user_location", user.Location)
			continue
		}
		dis := utils.Haversine(*user.CachedLat, *user.CachedLong, *r.CachedLat, *r.CachedLong)
		if dis <= 50 {
			nearbyReports = append(nearbyReports, r)
		}
	}

	data := struct {
		Count   int
		Reports []models.Report
	}{
		Count:   len(nearbyReports),
		Reports: nearbyReports,
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Data:    data,
	})

}
