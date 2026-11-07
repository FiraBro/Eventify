package utils

import (
	"errors"
	"net/smtp"
	"strings"
	"testing"

	"github.com/FiraBro/local-go/internal/config"
)

func TestSendStaffInvitationEmail(t *testing.T) {
	oldHost, oldPort, oldUser, oldPass := config.SMTPHost, config.SMTPPort, config.SMTPUser, config.SMTPPass
	config.SMTPHost, config.SMTPPort, config.SMTPUser, config.SMTPPass = "smtp.example.test", "587", "sender@example.test", "secret"
	t.Cleanup(func() {
		config.SMTPHost, config.SMTPPort, config.SMTPUser, config.SMTPPass = oldHost, oldPort, oldUser, oldPass
	})

	oldSend := sendSMTPMail
	t.Cleanup(func() { sendSMTPMail = oldSend })
	acceptanceURL := "https://app.example.test/accept?token=secret-invite-token"
	var capturedMessage string
	sendSMTPMail = func(addr string, _ smtp.Auth, from string, to []string, message []byte) error {
		if addr != "smtp.example.test:587" || from != "sender@example.test" || len(to) != 1 || to[0] != "staff@example.test" {
			t.Fatalf("unexpected SMTP envelope: %s, %s, %v", addr, from, to)
		}
		capturedMessage = string(message)
		return nil
	}

	if err := (SMTPMailer{}).SendStaffInvitation("staff@example.test", "Taylor Staff", "Northside Studio", acceptanceURL, 24); err != nil {
		t.Fatalf("send staff invitation: %v", err)
	}
	for _, expected := range []string{
		"Subject: Your Eventify staff invitation",
		"Taylor Staff",
		"Northside Studio",
		acceptanceURL,
		"expires in 24 hours",
		"contact your administrator",
	} {
		if !strings.Contains(capturedMessage, expected) {
			t.Errorf("invitation email is missing %q", expected)
		}
	}
	if strings.Contains(strings.ToLower(capturedMessage), "password:") {
		t.Fatal("invitation email must not include a password")
	}

	sendSMTPMail = func(string, smtp.Auth, string, []string, []byte) error {
		return errors.New("remote server rejected delivery")
	}
	err := (SMTPMailer{}).SendStaffInvitation("staff@example.test", "Taylor Staff", "Northside Studio", acceptanceURL, 24)
	if err == nil || strings.Contains(err.Error(), "secret-invite-token") {
		t.Fatalf("expected sanitized SMTP failure, got %v", err)
	}
}
