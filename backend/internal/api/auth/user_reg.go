package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	database "github.com/icodeologist/disasterwatch/internal/db"
	"github.com/icodeologist/disasterwatch/internal/models"
	"github.com/icodeologist/disasterwatch/internal/utils"
	"gorm.io/gorm"
)

// Validate the request, cache the location, and store a new user.
func UserRegistration(c *gin.Context) {
	var input models.UserRegistrationRequest
	if err := c.ShouldBindJSON(&input); err != nil || input.Validate() != nil {
		writeInvalidInput(c)
		return
	}

	var existing models.User
	lookup := database.DB.Where("user_name = ?", input.Username).First(&existing)
	if lookup.Error != nil && !errors.Is(lookup.Error, gorm.ErrRecordNotFound) {
		slog.Error("failed to check username", "error", lookup.Error)
		writeServerError(c, "unable to create user")
		return
	}
	if existing.ID != 0 {
		c.JSON(http.StatusConflict, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode: "DUPLICATE_USER",
				Message:   "Username already exists. Try to login.",
			},
		})
		return
	}

	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		slog.Error("failed to hash password", "error", err)
		writeServerError(c, "unable to create user")
		return
	}

	user := models.User{
		UserName: input.Username,
		Password: hashedPassword,
		Email:    input.Email,
		Location: input.Location,
	}
	if err := utils.CachedUserCords(c.Request.Context(), &user); err != nil {
		slog.Error("failed to cache user location", "error", err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode: "CACHED_COORDINATES_ERROR",
				Message:   "Please enter a valid location.",
			},
		})
		return
	}

	if err := database.DB.Create(&user).Error; err != nil {
		slog.Error("failed to create user", "error", err)
		writeServerError(c, "unable to create user")
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

// Check the credentials and return a signed token.
func UserLogin(c *gin.Context) {
	var input models.UserLoginRequest
	if err := c.ShouldBindJSON(&input); err != nil || input.Validate() != nil {
		writeInvalidInput(c)
		return
	}

	var user models.User
	lookup := database.DB.Where("email = ?", input.Email).First(&user)
	if errors.Is(lookup.Error, gorm.ErrRecordNotFound) || user.ID == 0 {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode: "USER_NOT_FOUND",
				Message:   "User email does not exist.",
			},
		})
		return
	}
	if lookup.Error != nil {
		slog.Error("failed to find user for login", "error", lookup.Error)
		writeServerError(c, "unable to login")
		return
	}

	if err := utils.CheckHashPasswords(input.Password, user.Password); err != nil {
		slog.Warn("password verification failed", "error", err)
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Success: false,
			Error: models.Error{
				ErrorCode: "INVALID_CREDENTIALS",
				Message:   "Invalid email or password.",
			},
		})
		return
	}

	token, err := GenerateAndSignJwtToken(user)
	if err != nil {
		slog.Error("failed to sign login token", "error", err)
		writeServerError(c, "unable to login")
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{
		Success: true,
		Data:    token,
		Message: "You logged in. Use this token to access authorized endpoints.",
	})
}

func writeInvalidInput(c *gin.Context) {
	c.JSON(http.StatusBadRequest, models.ErrorResponse{
		Success: false,
		Error: models.Error{
			ErrorCode: "INVALID_INPUT",
			Message:   "Please provide valid required fields.",
		},
	})
}

func writeServerError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, models.ErrorResponse{
		Success: false,
		Error: models.Error{
			ErrorCode: "INTERNAL_ERROR",
			Message:   message,
		},
	})
}

// GenerateToken returns a cryptographically random hexadecimal token.
func GenerateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
