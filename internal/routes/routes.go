package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/internal/api/auth"
	"github.com/icodeologist/disasterwatch/internal/api/handler"
)

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
	authMiddlewareRouter := router.Group("/api")
	authMiddlewareRouter.Use(auth.AuthCheckingMiddleware)
	{
		// authMiddlewareRouter.POST("/create-report", handler.CreateReport)
		// authMiddlewareRouter.GET("/profile", auth.GetUserProfile)
		authMiddlewareRouter.GET("/reports", handler.GetAllReportsByUserID)
		authMiddlewareRouter.GET("/get_current_user", auth.GetUserProfileInfo)
		authMiddlewareRouter.GET("/dummy", auth.AdminMiddleware)
	}
	// router.POST("/isadmin", auth.CheckIfuserIsAdmin)

	// admin middleware
	adminMiddlewareRouter := router.Group("/admin")
	adminMiddlewareRouter.Use(auth.AdminMiddleware)
	adminMiddlewareRouter.GET("/user", auth.GetAdminUserInfo)

}
