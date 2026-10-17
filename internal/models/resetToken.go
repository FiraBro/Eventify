package models

import "time"

type ResetToken struct {
	Email     string
	OTP       string
	ExpiresAt time.Time
	IsUsed    bool   `json:"is_used"`
	TokenHash string `json:"token_hash"`
}
