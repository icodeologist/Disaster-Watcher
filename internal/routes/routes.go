package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/internal/api/auth"
	"github.com/icodeologist/disasterwatch/internal/api/handler"
)

var c *gin.Context

func SetUpRoutes(router *gin.Engine) {
	router.GET("/report/:id", handler.GetReportById)
	// TODO: Make a endpoint that deletes the resolved or not active reports.
	router.DELETE("/delete/:id", handler.DeleteReportById)

	// get all reports that are in database - authentication is not required
	router.GET("/reports/all", handler.GetAllReports)
	// get all the nearby reports to your locaiton
	router.GET("/reports/nearby", handler.GetNearByReports)

	//auth and middleware routes
	router.POST("user/register", auth.UserRegistration)
	router.POST("user/login", auth.UserLogin)
	// TODO : come up with better endpoint design
	router.GET("/hello", handler.CreateReport)

	//applying middlewares
	authRoutes := router.Group("/api")
	authRoutes.Use(auth.AuthCheckingMiddleware)
	{
		// authRoutes.POST("/create-report", handler.CreateReport)
		// authRoutes.GET("/profile", auth.GetUserProfile)
		authRoutes.GET("/reports", handler.GetAllReportsByUserID)
		authRoutes.GET("/get_current_user", auth.GetUserProfileInfo)
		authRoutes.GET("/dummy", auth.AdminMiddleware)
	}

}
