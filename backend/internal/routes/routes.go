package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/internal/api/auth"
	"github.com/icodeologist/disasterwatch/internal/api/handler"
)

func SetUpRoutes(router *gin.Engine, server *handler.Server, rateLimiter *auth.RateLimitMiddleware) {
	// router.Use(rateLimiter.RateLimitingMiddelware)

	router.POST("user/register", auth.UserRegistration)
	router.POST("user/login", auth.UserLogin)
	router.GET("api/user/:id", handler.GetUserInfo) // for account page (User Account)

	// ratelimiter middelware for all routes

	authMiddlewareRouter := router.Group("/api")
	authMiddlewareRouter.Use(auth.AuthCheckingMiddleware)
	// authMiddlewareRouter.Use(rateLimiter.RateLimitingMiddelware)
	{
		// authMiddlewareRouter.GET("/profile", auth.GetUserProfile)
		authMiddlewareRouter.POST("/reports", server.CreateReport)
		authMiddlewareRouter.GET("/reports", handler.GetAllReportsByUserID)
		authMiddlewareRouter.GET("/get_current_user", auth.GetUserProfileInfo)
		authMiddlewareRouter.GET("report/:id", handler.GetReportById)
	}

}
