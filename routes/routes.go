package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/controllers"
	"github.com/icodeologist/disasterwatch/handlers"
	"github.com/icodeologist/disasterwatch/middlewares"
)

var c *gin.Context

func SetUpRoutes(router *gin.Engine) {
	router.Static("/static", "./static")

	// Load HTML templates from templates folder
	router.LoadHTMLGlob("templates/*")
	router.GET("/report/:id", controllers.GetReportById)
	router.GET("/", controllers.DisplayMap)
	router.DELETE("/delete/:id", controllers.DeleteReportById)

	// get all reports
	router.GET("/reports/all", controllers.GetAllReports)
	//get nearby reports
	router.GET("/reports/nearby", controllers.GetNearByReports)

	//auth section
	router.POST("auth/register", handlers.Register)
	router.POST("auth/login", handlers.Login)
	router.GET("auth/reset/password", handlers.ResetPassword)
	router.POST("auth/reset/passsword", handlers.HandleResetPassword)

	//applying middlewares
	authRoutes := router.Group("/user")
	authRoutes.Use(middlewares.CheckAuth)
	ns := controllers.NewGeoService()
	{
		authRoutes.POST("/api/report", ns.CreateReport)
		authRoutes.GET("/profile", controllers.GetUserProfile)
		authRoutes.GET("/reports", controllers.GetAllReportsByUserID)
	}

}
