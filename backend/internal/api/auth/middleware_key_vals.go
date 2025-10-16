package auth

// helpers functions to get userinfo
// middleware helpers routes
import (
	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
)

func GetUserProfileInfo(c *gin.Context) {
	currentUser, _ := c.Get("currentUser")
	c.JSON(200, gin.H{
		"current user": currentUser,
	})

}

func GetAdminUserInfo(c *gin.Context) {
	adminUser, _ := c.Get("AdminUser")
	c.JSON(200, gin.H{
		"current admin": adminUser.(models.User).Email,
	})
}
func CheckIfuserIsAdmin(userEmail string) bool {
	var user models.User
	db.DB.Where("email=?", userEmail).First(&user)
	if user.IsAdmin {
		return true
	}
	return false
}
