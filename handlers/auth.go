package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/smtp"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/icodeologist/disasterwatch/controllers"
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
	database.DB.Where("user_name = ?", authInput.Username).First(&userFound)
	if userFound.ID != 0 {
		//user already exists
		fmt.Println("userid", userFound.ID)
		c.JSON(http.StatusAlreadyReported, gin.H{"error": "user already exists"})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(authInput.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password could not be hashed"})
		return
	}
	user := models.User{
		UserName:    authInput.Username,
		Password:    string(passwordHash),
		Email:       authInput.Email,
		Location:    authInput.Location,
		PhoneNumber: authInput.PhoneNumber,
	}
	fmt.Println("authInput.Locatioon", authInput.Location)

	res := database.DB.Create(&user)
	if res.Error != nil {
		fmt.Printf("error : %v", res.Error.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error occured while creating a user"})
		return
	}

	// call ForwardGeoCoding here for one time cost
	notifier := controllers.NewGeoService()
	er := notifier.CachedUserCords(&user)
	if er != nil {
		fmt.Println("Please fix this later ", er)
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

// GenerateToken remains the same…
func GenerateToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ResetPassword handler (GET) — no change here, except Create will write into the table
func ResetPassword(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email cannot be empty"})
		return
	}

	var user models.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		// Don’t reveal whether the email exists
		c.JSON(http.StatusOK, gin.H{
			"message": "If an account with that email exists, you will receive a password reset link shortly.",
		})
		return
	}

	tokenString, err := GenerateToken(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to generate reset token"})
		return
	}

	resetToken := models.PasswordResetToken{
		Token:     tokenString,
		ExpiresAt: time.Now().Add(15 * time.Minute),
		UserID:    user.ID,
		Used:      false,
	}
	fmt.Println("Token : ", resetToken.Token)

	if err := database.DB.Create(&resetToken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to save reset token"})
		return
	}

	baseURL := os.Getenv("FRONTEND_RESET_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3000/reset-password"
	}
	resetLink := fmt.Sprintf("%s?token=%s", baseURL, resetToken.Token)

	if err := SendResetLink(resetLink, user.Email); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "If an account with that email exists, you will receive a password reset link shortly.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "If an account with that email exists, you will receive a password reset link shortly.",
	})
}

// SendResetLink is unchanged
func SendResetLink(link, toEmail string) error {
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	from := os.Getenv("FROMEMAILADD")
	password := os.Getenv("FROMPASSWORD")
	if from == "" || password == "" {
		return fmt.Errorf("email credentials not configured")
	}

	subject := "Subject: Password Reset Instructions\r\n"
	toHeader := fmt.Sprintf("To: %s\r\n", toEmail)
	fromHeader := fmt.Sprintf("From: %s\r\n", from)
	mime := "MIME-Version: 1.0\r\nContent-Type: text/plain; charset=\"utf-8\"\r\n\r\n"

	body := fmt.Sprintf(
		"Hello,\n\nWe received a request to reset your password. "+
			"Click the link below to choose a new password. This link will expire in 15 minutes:\n\n%s\n\n"+
			"If you did not request a password reset, you can safely ignore this email.\n\nThanks,\nSupport Team\n",
		link,
	)

	msg := []byte(fromHeader + toHeader + subject + mime + body)
	address := smtpHost + ":" + smtpPort

	auth := smtp.PlainAuth("", from, password, smtpHost)
	return smtp.SendMail(address, auth, from, []string{toEmail}, msg)
}

// ResetRequestPayload must match your JSON keys exactly
type ResetRequestPayload struct {
	Token             string `json:"token" binding:"required"`
	NewPassword       string `json:"new-password" binding:"required"`
	ReTypeNewPassword string `json:"retype-newpassword" binding:"required"`
}

// HandleResetPassword (POST) now checks “Used” and marks it true on success
func HandleResetPassword(c *gin.Context) {
	var payload ResetRequestPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}

	if payload.Token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token cannot be empty"})
		return
	}

	// Look up a token row that matches AND is not yet used
	var resetToken models.PasswordResetToken
	err := database.DB.
		Where("token = ? AND used = false", payload.Token).
		First(&resetToken).Error
	if err != nil {
		// Could be “record not found” or some DB error; treat both as invalid/expired
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired token"})
		return
	}

	// Check expiration
	if time.Now().After(resetToken.ExpiresAt) {
		// Optionally delete or mark expired; here we mark Used = true so it can’t be reused
		resetToken.Used = true
		_ = database.DB.Save(&resetToken).Error
		c.JSON(http.StatusBadRequest, gin.H{"error": "token has expired"})
		return
	}

	// Ensure passwords match
	if payload.NewPassword != payload.ReTypeNewPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "passwords do not match"})
		return
	}

	// Enforce a minimum length (adjust as you see fit)
	if len(payload.NewPassword) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters long"})
		return
	}

	// Fetch the associated user
	var user models.User
	if err := database.DB.First(&user, resetToken.UserID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to find user"})
		return
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not hash password"})
		return
	}

	// Update user’s password
	user.Password = string(hashedPassword)
	if err := database.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update password"})
		return
	}

	// Mark this token as used so it cannot be reused
	resetToken.Used = true
	if err := database.DB.Save(&resetToken).Error; err != nil {
		// We don’t fail the request just because we couldn’t flip “Used = true”,
		// but we log a warning server‐side.
		fmt.Println("warning: failed to mark reset token as used:", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "password has been reset successfully"})
}
