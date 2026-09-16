// get the user account info
package handler

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
)

func GetUserInfo(c *gin.Context) {
	id, ok := c.Get("userId")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "StatusUnauthorized",
		})
		return
	}
	var user models.User
	if err := db.DB.Where("id=?", id).First(&user).Error; err != nil {
		slog.Error("failed to load current user", "error", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode: "DATABASE_ERR",
				Message:   "unable to load current user",
			},
		})
		return
	}
	userInfo := models.UserAccountInfoResponse{
		UserName:     user.UserName,
		UserEmail:    user.Email,
		UserLocation: user.Location,
	}

	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Data:    userInfo,
	})
}
