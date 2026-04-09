package auth

import (
	"github.com/golang-jwt/jwt/v4"
	"github.com/icodeologist/disasterwatch/internal/models"
	"log/slog"
	"os"
	"time"
)

func GenerateAndSignJwtToken(user models.User) (string, error) {
	// Generate token with Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       user.ID,
		"username": user.UserName,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})
	jwtToken, err := token.SignedString([]byte(os.Getenv("SECRET")))
	if err != nil {
		slog.Error("Failed to sign the jwt token", "error", err)
		return "", err
	}
	return jwtToken, nil

}
