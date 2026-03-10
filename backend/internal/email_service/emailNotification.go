package emailservice

import (
	"fmt"
	// "net/smtp"
)

func SendEmail(to string) {
	// smtpHost := "sandbox.smtp.mailtrap.io"
	// smtpPort := "587"
	// username := "80a92df47b164b"
	// password := "f90eac08470a7a" // replace with your full password
	// sender := "DisasterNotifierTeam@example.com"
	// rc := []string{to}
	//
	// subject := "Subject: Disaster happened nearby!\r\n"
	// body := "Please be careful and take care of the precuations measures untilt the !\r\n"
	// message := []byte(subject + "\r\n" + body)
	//
	// auth := smtp.PlainAuth("", username, password, smtpHost)
	//
	// err := smtp.SendMail(smtpHost+":"+smtpPort, auth, sender, rc, message)
	// if err != nil {
	// 	fmt.Println("Error:", err)
	// 	return
	// }
	fmt.Println("Email sent to ", to)
}
