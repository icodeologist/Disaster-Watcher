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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is missing"})
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	authToken := strings.Split(authHeader, " ")
	if len(authToken) != 2 || authToken[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token format"})
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	token, err := jwt.Parse(authToken[1], func(token *jwt.Token) (interface{}, error) {
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
	c.Set("userId", uint(currentUserIDinFloat))
	c.Set("currentUser", currentUser)
	c.Next()
}

func AdminMiddleware(c *gin.Context) {
	// get the user id through auth middleware
	// then check the role may be
	currentUser, _ := c.Get("currentUser")
	id := currentUser.(models.User).ID
	var currentAuthenticatedUser models.User
	err := database.DB.Where("id=?", id).First(&currentAuthenticatedUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database query error. Check logs."})
	}

	if !currentAuthenticatedUser.IsAdmin {
		err, val := CheckIfuserIsAdmin(currentAuthenticatedUser.Email)
		if err != nil {
			log.Fatal("error @ admin middleware :", err)
		}
		println("val @middleware admin :", val)
		c.JSON(http.StatusUnauthorized, gin.H{"message": "you need to be admin to visit this endpoin"})
	}

	c.Set("admin", currentUser)
	c.Next()
	c.JSON(200, gin.H{"message": "working"})
}
