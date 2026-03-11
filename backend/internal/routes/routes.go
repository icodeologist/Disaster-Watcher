package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/internal/api/auth"
	"github.com/icodeologist/disasterwatch/internal/api/handler"
)

func SetUpRoutes(router *gin.Engine, server *handler.Server) {

	router.POST("user/register", auth.UserRegistration)
	router.POST("user/login", auth.UserLogin)
	router.GET("api/user/:id", handler.GetUserInfo) // for account page (User Account)

	// Rate limit middleware for external email provider

	authMiddlewareRouter := router.Group("/api")
	rateLimiterPlusAuthMiddleware := router.Group("/api")
	rateLimiterPlusAuthMiddleware.Use(auth.AuthCheckingMiddleware)
	rateLimiterPlusAuthMiddleware.Use(auth.EmailRateLimitMiddleware())
	authMiddlewareRouter.Use(auth.AuthCheckingMiddleware)
	{
		// authMiddlewareRouter.GET("/profile", auth.GetUserProfile)
		rateLimiterPlusAuthMiddleware.POST("/reports", server.CreateReport)
		authMiddlewareRouter.GET("/reports", handler.GetAllReportsByUserID)
		authMiddlewareRouter.GET("/get_current_user", auth.GetUserProfileInfo)
		authMiddlewareRouter.GET("report/:id", handler.GetReportById)
	}

}
