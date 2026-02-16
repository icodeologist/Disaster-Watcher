// get the user account info
package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
)

func GetUserInfo(c *gin.Context) {
	id := c.Param("id")
	fmt.Println("ID :", id)
	var user models.User
	if err := db.DB.Where("id=?", id).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode:    "DATABASE_ERR",
				ErrorDetails: err.Error(),
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
