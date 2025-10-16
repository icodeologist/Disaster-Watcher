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
)

// User registration with password hashing and location caching
// location caching	-> Mapping user location to lat long and storing it in db
func UserRegistration(c *gin.Context) {
	var userInput models.AuthInput
	var userFound models.User

	if err := c.ShouldBindJSON(&userInput); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode:    "INVALID_JSON",
				Message:      "Invalid input or empty json datat",
				ErrorDetails: err.Error(),
			},
		})
	}
	// checking if the user has already registered
	database.DB.Where("user_name = ?", userInput.Username).First(&userFound)
	if userFound.ID != 0 {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode: "DUPLICATE_USER",
				Message:   "Username already exists. Try to login.",
			},
		})
		return
	}
	hashedPassword, err := utils.HashPassword(userInput.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode:    "HASH_PASSWORD_ERROR",
				ErrorDetails: err.Error(),
			},
		})
		return
	}

	var user models.User
	user.UserName = userInput.Username
	user.Password = hashedPassword
	user.Email = userInput.Email
	user.Location = userInput.Location

	err = utils.CachedUserCords(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode:    "CACHED_CORDINATES_ERR",
				Message:      "Please enter the valid location name. If You live in remote area you could try to find the lat and long from the nominatm api",
				ErrorDetails: err.Error(),
			},
		})
		return
	}
	// check if user gives a special key
	res := database.DB.Create(&user)
	fmt.Println("res error :", res)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode:    "DATABASE_ERR",
				Message:      "Error occured while creating a user.",
				ErrorDetails: res.Error.Error(),
			},
		})
		database.DB.Delete(&user)
		return
	}
	c.JSON(http.StatusCreated, models.SuccessResponse{
		Success: true,
		Message: "User has been created. You can login.",
		Data: models.UserREGISTERRDataResponse{
			ID:       user.ID,
			Username: user.UserName,
			Email:    user.Email,
			Location: user.Location,
			LatANDLongs: models.CachedCord{
				Latitude:  user.CachedLat,
				Longitude: user.CachedLong,
			},
		},
	})
}

// UserLogin handler with JWT token generation
func UserLogin(c *gin.Context) {
	var userInput models.AuthInput
	if err := c.ShouldBindJSON(&userInput); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode:    "INVALID_JSON",
				Message:      "Invalid input or empty json datat",
				ErrorDetails: err.Error(),
			},
		})
		return
	}
	var userFound models.User
	database.DB.Where("email=?", userInput.Email).First(&userFound)
	if userFound.ID == 0 {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode: "USER_NOT_FOUND",
				Message:   "User email does not exists",
			},
		})
		return
	}
	if err := utils.CheckHashPasswords(userInput.Password, userFound.Password); err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode:    "INVALID_CREDENTIALS",
				Message:      "Your password does not match.",
				ErrorDetails: err.Error(),
			},
		})
		return
	}
	if userInput.AmdinSecretKey != "" {

		// TODO: change the status code
		if userInput.AmdinSecretKey != os.Getenv("ADMINCODE") {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Success: false,
				Error: models.Error{
					ErrorCode: "INVALID_ADMIN_CREDENTIALS",
					Message:   "Your admin secret key does not match",
				},
			})
			return
		}
		userFound.IsAdmin = true
		database.DB.Save(&userFound)
	}

	generateToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       userFound.ID,
		"username": userFound.UserName,
		"email":    userFound.Email,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})
	jwtToken, err := generateToken.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode:    "JWT_ERROR",
				ErrorDetails: err.Error(),
			},
		})
	}
	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Data:    jwtToken,
		Message: "You logged in. Use this token to access authorized endpoints.",
	})
}

// GenerateToken remains the same…
func GenerateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
