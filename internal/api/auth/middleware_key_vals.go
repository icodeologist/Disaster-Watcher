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
	adminUser, _ := c.Get("admin")
	c.JSON(200, gin.H{
		"current admin": adminUser,
	})
}
func CheckIfuserIsAdmin(userEmail string) (error, bool) {
	var user models.User
	res := db.DB.Where("email=?", userEmail).First(&user)
	if res.Error != nil {
		return res.Error, false
	}
	return nil, true
}
