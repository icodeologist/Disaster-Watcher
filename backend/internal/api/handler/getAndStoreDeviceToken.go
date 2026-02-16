package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
)

func GetAndStoreDeviceToken(c *gin.Context) {
	var tokenReq models.DeviceTokenRequest
	if err := c.ShouldBindJSON(&tokenReq); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode:    "INVALID_INPUT",
				ErrorDetails: err.Error(),
				Message:      "Invalid input or empty json data",
			},
		})
		return
	}
	if tokenReq.Token == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode: "INVALID_EMPTY_TOKEN.",
				Message:   "Device token is missing.",
			},
		})
		return
	}
	var userFound models.User
	if res := db.DB.Where("id=?", uint(tokenReq.UserID)).First(&userFound); res.Error != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode:    "DB_ERROR",
				ErrorDetails: res.Error.Error(),
			},
		})
		return
	}
	if userFound.ID == 0 {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode: "USER_NOT_FOUND",
				Message:   fmt.Sprintf("There was no user with the ID of %v", userFound.ID),
			},
		})
		return
	}
	// userFound.DeviceToken = tokenReq.Token
	// db.DB.Save(&userFound)
	fmt.Println("Device Token ", tokenReq.Token)
	c.JSON(http.StatusAccepted, models.SuccessResponse{
		Success: true,
		Data:    userFound.DeviceToken,
		Message: "successfully stored user device token",
	})
}
