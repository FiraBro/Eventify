package utils

import (
	"fmt"
	"log"
	"net/smtp"

	"github.com/FiraBro/local-go/internal/config"
)

func SendOTPEmail(to, otp string) error {
	msg := fmt.Sprintf(
		"From: Event Booking <no-reply@yourapp.com>\r\n"+
			"To: %s\r\n"+
			"Subject: Password Reset OTP\r\n\r\n"+
			"Your OTP is: %s\nIt expires in 5 minutes.",
		to,
		otp,
	)

	auth := smtp.PlainAuth(
		"",
		config.SMTPUser,
		config.SMTPPass,
		config.SMTPHost,
	)

	err := smtp.SendMail(
		config.SMTPHost+":"+config.SMTPPort,
		auth,
		config.SMTPUser,
		[]string{to},
		[]byte(msg),
	)
	if err != nil {
		log.Println("Failed to send OTP email:", err)
	}
	return err
}
