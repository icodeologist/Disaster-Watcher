package auth

import (
	"fmt"
	"log"
	"log/slog"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	database "github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

// CheckAuth is a middleware function that checks for a valid JWT token in the Authorization header
func AuthCheckingMiddleware(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Authorization header is required",
		})
		return
	}

	authToken := strings.Split(authHeader, " ")

	if len(authToken) != 2 || authToken[0] != "Bearer" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Authorization header is missing",
		})
		return
	}

	token, err := jwt.Parse(authToken[1], func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method :%v", token.Header["alg"])
		}
		secret := os.Getenv("SECRET")
		if secret == "" {
			return nil, fmt.Errorf("Empty Secret :%v", secret)
		}
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		c.Abort()
		return
	}
	userID, ok := claims["id"].(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		c.Abort()
		return
	}
	expirationTime, ok := claims["exp"].(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		c.Abort()
		return
	}

	if float64(time.Now().Unix()) >= expirationTime {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var currentUser models.User
	log.Printf("USerID : %v\n", userID)
	if userID < 1 || userID != math.Trunc(userID) || userID > float64(^uint(0)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		c.Abort()
		return
	}
	userIDValue := uint(userID)
	fetchErr := database.DB.Where("ID=?", userIDValue).Find(&currentUser).Error
	if fetchErr != nil {
		slog.Error("failed to load authenticated user", "error", fetchErr)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode: "INTERNAL_ERROR",
				Message:   "unable to authenticate request",
			},
		})
		c.Abort()
		return
	}
	if currentUser.ID == 0 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	c.Set("userId", userIDValue)
	c.Set("currentUserEmail", currentUser.Email)
	c.Set("currentUser", currentUser)
	c.Next()
}

func AdminMiddleware(c *gin.Context) {
	currentUserID, ok := c.Get("userId")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"message": "no authenticated user id found."})
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	id := currentUserID
	var currentAuthenticatedUser models.User
	res := database.DB.Where("id=?", id).Find(&currentAuthenticatedUser)
	if res.Error != nil {
		log.Printf("Database error :%v", res.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database query error. Check logs."})
		return
	}

	if !currentAuthenticatedUser.IsAdmin {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "you need to be admin to visit this endpoin"})
		return
	}
	c.Set("AdminUser", currentAuthenticatedUser)
	c.Next()
}

// var Emaillimiter = rate.NewLimiter(2, 5)
//
// func EmailRateLimitMiddleware() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		// check if user logged in
// 		userID, ok := c.Get("userId")
// 		if !ok {
// 			c.AbortWithStatus(http.StatusUnauthorized)
// 			return
// 		}
// 		userIdUint, ok := userID.(uint)
// 		limiter := getUserEmailLimiter(userIdUint)
// 		if !limiter.Allow() {
// 			c.JSON(http.StatusTooManyRequests, models.ErrorResponse{
// 				Success: false,
// 				Error: models.Error{
// 					ErrorCode: "TOO_MANY_REQS",
// 					Message:   "Too many requests",
// 				},
// 			})
// 			c.Abort()
// 			return
// 		}
// 		c.Next()
// 	}
// }
