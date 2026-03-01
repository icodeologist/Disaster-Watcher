package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/internal/api/auth"
	"github.com/icodeologist/disasterwatch/internal/api/handler"
)

func SetUpRoutes(router *gin.Engine, server *handler.WorkerServer) {

	//auth and middleware routes
	router.POST("user/register", auth.UserRegistration)
	router.POST("user/login", auth.UserLogin)
	router.GET("api/user/:id", handler.GetUserInfo) // for account page (User Account)

	//applying middlewares
	authMiddlewareRouter := router.Group("/api")
	authMiddlewareRouter.Use(auth.AuthCheckingMiddleware)
	{
		authMiddlewareRouter.POST("/create", server.CreateReport)
		// authMiddlewareRouter.GET("/profile", auth.GetUserProfile)
		authMiddlewareRouter.GET("/reports", handler.GetAllReportsByUserID)
		authMiddlewareRouter.GET("/get_current_user", auth.GetUserProfileInfo)
		authMiddlewareRouter.POST("/get_device_token", handler.GetAndStoreDeviceToken)
		authMiddlewareRouter.GET("report/:id", handler.GetReportById)
	}
}
