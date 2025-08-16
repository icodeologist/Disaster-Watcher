package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/controllers"
)

var c *gin.Context

func SetUpRoutes(router *gin.Engine) {
	router.GET("/report/:id", controllers.GetReportById)
	router.GET("/", controllers.DisplayMap)
	router.DELETE("/delete/:id", controllers.DeleteReportById)

	// get all reports
	router.GET("/reports/all", controllers.GetAllReports)
	//get nearby reports
	router.GET("/reports/nearby", controllers.GetNearByReports)

	//auth section
	router.POST("auth/register", controllers.Register)
	router.POST("auth/login", controllers.Login)
	router.GET("auth/reset/password", controllers.ResetPassword)
	router.POST("auth/reset/passsword", controllers.HandleResetPassword)

	//applying middlewares
	authRoutes := router.Group("/user")
	authRoutes.Use(controllers.CheckAuth)
	{
		authRoutes.POST("/api/report", controllers.CreateReport)
		authRoutes.GET("/profile", controllers.GetUserProfile)
		authRoutes.GET("/reports", controllers.GetAllReportsByUserID)
	}

}
