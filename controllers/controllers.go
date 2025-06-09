package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/database"
	"github.com/icodeologist/disasterwatch/models"
	"github.com/icodeologist/disasterwatch/utils"
)

func CreateReport(c *gin.Context) {
	var report models.Report

	//get the incoming json and parse it
	if err := c.ShouldBindJSON(&report); err != nil {
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

	report.UserId = userId.(uint)

	report.Created_time = time.Now()
	report.Updated_time = time.Now()

	if err := database.DB.Create(&report).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	var allUsers []models.User
	//Process the reports and send notificatino simultaneously
	err := database.DB.Find(&allUsers).Error
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not fetch the users from the DB."})
		return
	} else {
		// TODO : talk with notification microservices
	}

	c.JSON(http.StatusOK, gin.H{"Message": "Notification is being sent asynchronously"})
}

// everybody should see this
// Kind a like insta posts but they can edit it
func GetAllReportsByUserID(c *gin.Context) {
	var reports []models.Report

	userId, exists := c.Get("userId")
	fmt.Println("userid", userId)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user is not authenticated"})
		return
	}
	if err := database.DB.Preload("User").Where("user_id=?", userId).Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	fmt.Println("reports", reports)

	c.JSON(http.StatusOK, reports)
}

func GetReportById(c *gin.Context) {
	id := c.Param("id")
	var report models.Report
	if err := database.DB.First(&report, id).Error; err != nil {
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

	if err := database.DB.Delete(&report, id).Error; err != nil {
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

	if err := database.DB.Find(&reports).Error; err != nil {
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
		distance := utils.Haversine(report.Latitude, lat, report.Longitude, long)
		if radiusDistance <= distance {
			nearByReports = append(nearByReports, report)
		}
	}
	fmt.Println("Nearbyreports", nearByReports)

	c.JSON(200, nearByReports)

}

func ProcessDisasterReport(c *gin.Context) {

}
