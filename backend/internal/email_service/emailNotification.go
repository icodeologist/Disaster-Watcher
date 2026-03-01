package emailservice

import (
	"fmt"
	"net/smtp"
)

func SetUpEmailServer() (*smtp.Client, error) {
	c, err := smtp.Dial("mail.example.com:25")
	if err != nil {
		return nil, err
	}
	return c, nil
}

func SendEmail(c *smtp.Client, toEmail, fromEmail string) error {
	if err := c.Mail(toEmail); err != nil {
		return err
	}
	if err := c.Rcpt(fromEmail); err != nil {
		return err
	}

	// send email
	wc, err := c.Data()
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(wc, "THIS IS EMAIL BODY")
	if err != nil {
		return err
	}
	err = wc.Close()
	if err != nil {
		return err
	}
	err = c.Quit()
	if err != nil {
		return err
	}
	fmt.Println("Email sent")
	return nil
}
