package auth

import (
	"fmt"
	"log"
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
		return []byte(os.Getenv("SECRET")), nil
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
	if float64(time.Now().Unix()) > claims["exp"].(float64) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	var currentUser models.User
	database.DB.Where("ID=?", claims["id"]).Find(&currentUser)
	if currentUser.ID == 0 {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	currentUserIDinFloat, ok := claims["id"].(float64)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	// set the current authenticated user
	c.Set("userId", uint(currentUserIDinFloat))
	c.Set("currentUserEmail", currentUser.Email)
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
