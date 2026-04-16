package emailservice

import (
	"fmt"
	// "log/slog"
	"math/rand/v2"
	// "net/smtp"

	"github.com/icodeologist/disasterwatch/internal/models"
)

func SendEmail(emailObj models.EmailModel) error {
	// smtpHost := "sandbox.smtp.mailtrap.io"
	// smtpPort := "587"
	// username := "80a92df47b164b"
	// password := "f90eac08470a7a" // replace with your full password
	// sender := "DisasterNotifierTeam@example.com"
	// rc := []string{emailObj.Email}
	//
	// subject := "Subject: Disaster happened nearby!\r\n"
	// body := fmt.Sprintf("Report : %v was posted near Location : %v . Please Follow our precaution : %v", emailObj.EmailBody.Title, emailObj.EmailBody.Location, emailObj.EmailBody.Precaution)
	// message := []byte(subject + "\r\n" + body)
	//
	// auth := smtp.PlainAuth("", username, password, smtpHost)
	//
	// err := smtp.SendMail(smtpHost+":"+smtpPort, auth, sender, rc, message)
	// if err != nil {
	// 	slog.Error("Smtp message SENDEMAIL error", "error", err)
	// 	return err
	// } else {
	// 	slog.Info("Message sent to user @", emailObj.Email, " Successfully.")
	// 	return nil
	// }

	n := rand.IntN(100)
	println("N : ", n)
	res := n % 2
	if res == 0 {
		fmt.Println("Email is sent to ", emailObj.Email)
		return nil
	} else {
		return fmt.Errorf("Failed to send email to ", emailObj.Email)
	}
}
