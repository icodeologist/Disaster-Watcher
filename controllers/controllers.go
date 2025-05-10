package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/database"
	"github.com/icodeologist/disasterwatch/models"
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

	if err := database.Db.Create(&report).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "could not create the object",
		})
		return
	}

	c.JSON(http.StatusOK, report)
}

func GetAllReport(c *gin.Context) {
	var reports []models.Report
	if err := database.Db.Find(&reports).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "there were no such records",
		})
		return
	}
	c.JSON(http.StatusOK, reports)
}

func GetReportById(c *gin.Context) {
	id := c.Param("id")
	var report models.Report
	if err := database.Db.First(&report, id).Error; err != nil {
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

	if err := database.Db.Delete(&report, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "could not delete the given record",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": "successfully deleted the report",
	})
}
