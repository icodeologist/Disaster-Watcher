package handlers

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/icodeologist/disasterwatch/database"
	"github.com/icodeologist/disasterwatch/models"
	"golang.org/x/crypto/bcrypt"
)

func Register(c *gin.Context) {
	var authInput models.AuthInput

	if err := c.ShouldBindJSON(&authInput); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "parsing error"})
		return
	}

	var userFound models.User

	//check if the user account already exists
	database.DB.Where("username = ?", authInput.Username).First(&userFound)
	if userFound.ID != 0 {
		//user already exists
		c.JSON(http.StatusAlreadyReported, gin.H{"error": "user already exists"})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(authInput.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := models.User{
		UserName: authInput.Username,
		Password: string(passwordHash),
		Email:    authInput.Email,
	}

	res := database.DB.Create(&user)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user already exists. please log in."})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": user})
}

func Login(c *gin.Context) {
	var authInput models.AuthInput

	if err := c.ShouldBindJSON(&authInput); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"err": err.Error()})
		return
	}
	//check username and password
	var userCheck models.User

	database.DB.Where("email=?", authInput.Email).First(&userCheck)
	if userCheck.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email adress does not exists"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(userCheck.Password), []byte(authInput.Password)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid password"})
		return
	}

	generateToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  userCheck.ID,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})
	token, err := generateToken.SignedString([]byte(os.Getenv("SECRET")))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
