package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	database "github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/icodeologist/disasterwatch/internal/utils"
	"golang.org/x/crypto/bcrypt"
)

// User registration with passworkd hashing and location caching
// location caching	-> Mapping user location to lat long and storing it in db
func UserRegistration(c *gin.Context) {
	var userInput models.AuthInput
	var userFound models.User

	if err := c.ShouldBindJSON(&userInput); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid or emtpy user data."})
		return
	}
	// checking if the user has already registered
	database.DB.Where("user_name = ?", userInput.Username).First(&userFound)
	if userFound.ID != 0 {
		fmt.Println("userid", userFound.ID)
		c.JSON(http.StatusAlreadyReported, gin.H{"error": "user already exists"})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(userInput.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password could not be hashed"})
		return
	}
	user := models.User{
		UserName: userInput.Username,
		Password: string(passwordHash),
		Email:    userInput.Email,
		Location: userInput.Location,
	}

	res := database.DB.Create(&user)
	if res.Error != nil {
		fmt.Printf("error : %v", res.Error.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while creating a user"})
		return
	}
	er := utils.CachedUserCords(&user)
	if er != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"data": err.Error(),
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}

// UserLogin handler with JWT token generation
// TODO: replace manual copy paste token to more secure cookie set up or local storage then access in clientsidle?
func UserLogin(c *gin.Context) {
	var userInput models.AuthInput
	if err := c.ShouldBindJSON(&userInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}
	var userFound models.User
	database.DB.Where("email=?", userInput.Email).First(&userFound)
	if userFound.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email adress does not exists"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(userFound.Password), []byte(userInput.Password)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid password"})
		return
	}
	generateToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  userFound.ID,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})
	token, err := generateToken.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// GenerateToken remains the same…
func GenerateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func GetUserProfileInfo(c *gin.Context) {
	currentUser, _ := c.Get("currentUser")
	c.JSON(200, gin.H{
		"Current User": currentUser,
	})

}
