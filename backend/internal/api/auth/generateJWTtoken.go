package auth

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/icodeologist/disasterwatch/internal/models"
)

func GenerateAndSignJwtToken(user models.User) (string, error) {
	// Generate token with Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       user.ID,
		"username": user.UserName,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})
	secret := os.Getenv("SECRET")
	if secret == "" {
		return "", fmt.Errorf("JWT secret cannot be empty")
	}
	jwtToken, err := token.SignedString([]byte(secret))
	if err != nil {
		slog.Error("Failed to sign the jwt token", "error", err)
		return "", err
	}
	return jwtToken, nil
}
