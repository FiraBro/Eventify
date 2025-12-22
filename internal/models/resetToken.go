package models

import "time"

type ResetToken struct {
	Email     string
	OTP       string
	ExpiresAt time.Time
}
