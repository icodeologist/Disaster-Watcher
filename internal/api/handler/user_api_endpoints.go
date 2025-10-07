// user_api_endpoints.go contains all the crud endpoint functions
package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	database "github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/icodeologist/disasterwatch/internal/utils"
)

var ctx = context.Background()

/*
 user creates report at /disaster/user/create_report endpoint
 user posts with valid json and jwt auth token
 TODO: this should trigger the notification service once the post has been made
*/

func CreateReport(c *gin.Context) {
	var userReport models.Report
	if err := c.ShouldBindJSON(&userReport); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "could not parse incoming json",
		})
		return
	}
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User is not authenticated"})
		return
	}
	// type conversion
	userReport.UserId = userId.(uint)
	userReport.Created_time = time.Now()

	if err := database.DB.Create(&userReport).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "You posted the report"})
}

func GetAllReportsByUserID(c *gin.Context) {
	var reportsPostedByUser []models.Report

	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user is not authenticated"})
		return
	}
	if err := database.DB.Preload("User").Where("user_id=?", userId).Find(&reportsPostedByUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, reportsPostedByUser)
}

func GetReportById(c *gin.Context) {
	id := c.Param("id")
	var report models.Report
	if err := database.DB.Preload("User").First(&report, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "given id is not found",
		})
		return
	}

	c.JSON(http.StatusOK, report)

}

func DeleteReportById(c *gin.Context) {
	id := c.Param("id")
	var report models.Report

	if err := database.DB.Preload("User").Delete(&report, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "could not delete the given record",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": "successfully deleted the report",
	})
}

// front end map related functions
func DisplayMap(c *gin.Context) {
	c.HTML(200, "home.html", gin.H{})

}

func GetAllReports(c *gin.Context) {
	var reports []models.Report
	if err := database.DB.Preload("User").Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, reports)

}

//
// get near by reports with custom distance applied but set defualt to 10

func GetNearByReports(c *gin.Context) {
	// get the variables from query
	latitude := c.Query("lat")
	longitude := c.Query("long")
	radius := c.DefaultQuery("rad", "10")

	lat, err1 := strconv.ParseFloat(latitude, 64)
	long, err2 := strconv.ParseFloat(longitude, 64)
	radiusDistance, err3 := strconv.ParseFloat(radius, 64)

	if err1 != nil || err2 != nil || err3 != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "some error during parsing the queried variables"})
		return
	}

	var allReports []models.Report

	if err := database.DB.Preload("User").Find(&allReports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var nearByReports []models.Report

	for _, report := range allReports {
		distance := utils.Haversine(lat, long, report.Latitude, report.Longitude)
		if distance <= radiusDistance {
			nearByReports = append(nearByReports, report)
		}
	}

	if len(nearByReports) == 0 {
		c.JSON(400, gin.H{"Message": "There are so reported disasters nearby."})
		return
	}
	// The nearby reports contains the user who posted the reports and the reports itself.
	c.JSON(200, nearByReports)

}
