package utils

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/FiraBro/local-go/internal/config"
)

const (
	smtpDialTimeout    = 10 * time.Second
	smtpOperationLimit = 25 * time.Second
)

var sendSMTPMail = sendSMTPMailWithTimeout

// sendSMTPMailWithTimeout provides bounded connection and SMTP operation times.
// It requires STARTTLS before authenticating with SMTP credentials.
func sendSMTPMailWithTimeout(
	addr string,
	auth smtp.Auth,
	from string,
	to []string,
	message []byte,
) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid SMTP address")
	}

	dialer := net.Dialer{Timeout: smtpDialTimeout}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("SMTP connection failed")
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(smtpOperationLimit)); err != nil {
		return fmt.Errorf("SMTP deadline setup failed")
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP client initialization failed")
	}
	defer client.Close()

	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("SMTP greeting failed")
	}

	if ok, _ := client.Extension("STARTTLS"); !ok {
		return fmt.Errorf("SMTP server does not support STARTTLS")
	}

	if err := client.StartTLS(&tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	}); err != nil {
		return fmt.Errorf("SMTP TLS negotiation failed")
	}

	if auth != nil {
		if ok, _ := client.Extension("AUTH"); !ok {
			return fmt.Errorf("SMTP server does not support authentication")
		}
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP authentication failed")
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("SMTP sender rejected")
	}

	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("SMTP recipient rejected")
		}
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP message submission failed")
	}

	if _, err := io.Copy(writer, strings.NewReader(string(message))); err != nil {
		_ = writer.Close()
		return fmt.Errorf("SMTP message write failed")
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("SMTP message submission failed")
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("SMTP session close failed")
	}

	return nil
}

func SendOTPEmail(to, otp string) error {
	recipient, err := mail.ParseAddress(to)
	if err != nil || recipient.Address != to || strings.ContainsAny(to, "\r\n") {
		return fmt.Errorf("invalid password reset recipient")
	}

	if config.SMTPHost == "" || config.SMTPPort == "" ||
		config.SMTPUser == "" || config.SMTPPass == "" {
		return fmt.Errorf("SMTP is not configured")
	}

	msg := fmt.Sprintf(
		"From: Eventify <%s>\r\n"+
			"To: %s\r\n"+
			"Subject: Password Reset OTP\r\n\r\n"+
			"Your OTP is: %s\r\nIt expires in 15 minutes.",
		config.SMTPUser,
		recipient.Address,
		otp,
	)

	auth := smtp.PlainAuth("", config.SMTPUser, config.SMTPPass, config.SMTPHost)

	if err := sendSMTPMail(
		net.JoinHostPort(config.SMTPHost, config.SMTPPort),
		auth,
		config.SMTPUser,
		[]string{recipient.Address},
		[]byte(msg),
	); err != nil {
		// Never log SMTP response details or OTP-related data.
		log.Println("Failed to send OTP email")
		return fmt.Errorf("SMTP delivery failed")
	}

	return nil
}

// SMTPMailer sends plain text staff invitations through the configured SMTP
// server. The caller supplies an already constructed, URL-encoded acceptance
// link; this function never logs it or includes it in returned errors.
type SMTPMailer struct{}

func (SMTPMailer) SendStaffInvitation(
	to, staffName, businessName, acceptanceURL string,
	expiryHours int,
) error {
	recipient, err := mail.ParseAddress(to)
	if err != nil || recipient.Address != to || strings.ContainsAny(to, "\r\n") {
		return fmt.Errorf("invalid invitation recipient")
	}

	if config.SMTPHost == "" || config.SMTPPort == "" ||
		config.SMTPUser == "" || config.SMTPPass == "" {
		return fmt.Errorf("SMTP is not configured")
	}

	body := fmt.Sprintf(
		"Hello %s,\r\n\r\nYou have been invited to join %s on Eventify as a staff member.\r\n\r\nAccept your invitation and create your password here:\r\n%s\r\n\r\nThis invitation expires in %d hours. If you were not expecting this invitation, please contact your administrator.\r\n",
		staffName,
		businessName,
		acceptanceURL,
		expiryHours,
	)

	message := fmt.Sprintf(
		"From: Eventify <%s>\r\nTo: %s\r\nSubject: Your Eventify staff invitation\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		config.SMTPUser,
		recipient.Address,
		body,
	)

	auth := smtp.PlainAuth("", config.SMTPUser, config.SMTPPass, config.SMTPHost)

	if err := sendSMTPMail(
		net.JoinHostPort(config.SMTPHost, config.SMTPPort),
		auth,
		config.SMTPUser,
		[]string{recipient.Address},
		[]byte(message),
	); err != nil {
		return fmt.Errorf("SMTP delivery failed")
	}

	return nil
}
