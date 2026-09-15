package auth

// helpers functions to get userinfo
// middleware helpers routes
import (
	"net/http"
	"sync"

	"golang.org/x/time/rate"

	"github.com/gin-gonic/gin"
	"github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
)

func GetUserProfileInfo(c *gin.Context) {
	currentUserValue, exists := c.Get("currentUser")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized user"})
		return
	}

	currentUser, ok := currentUserValue.(models.User)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authenticated user"})
		return
	}

	userInfo := models.UserAccountInfoResponse{
		UserName:     currentUser.UserName,
		UserEmail:    currentUser.Email,
		UserLocation: currentUser.Location,
	}
	c.JSON(http.StatusOK, gin.H{"current user": userInfo})
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

var (
	emailLimiter = make(map[uint]*rate.Limiter)
	mu           sync.Mutex
)

func getUserEmailLimiter(userID uint) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	limiter, exists := emailLimiter[userID]
	if exists {
		return limiter
	}
	// 2 req/sec with 5 burst
	limiter = rate.NewLimiter(2, 5)
	emailLimiter[userID] = limiter
	return limiter
}
