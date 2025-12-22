package models

import "time"

type RefreshToken struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
}
