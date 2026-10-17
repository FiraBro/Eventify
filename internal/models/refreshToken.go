package models

import "time"

// RefreshToken represents a persisted tracking row in your repository layer.
type RefreshToken struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	IsUsed    bool      `json:"is_used"`
	ParentTag string    `json:"parent_tag"` // Tracks parent token linkage to spot breach replays
}
