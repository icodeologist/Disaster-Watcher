package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/controllers"
)

var c *gin.Context

func SetUpRoutes(router *gin.Engine) {

	router.POST("/api/create", controllers.CreateReport)
	router.GET("/report/:id", controllers.GetReportById)
	router.GET("/report/all", controllers.GetAllReport)
	router.DELETE("/delete/:id", controllers.DeleteReportById)

}
