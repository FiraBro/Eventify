package models

import "time"

type Notification struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Title       string     `json:"title"`
	Message     string     `json:"message"`
	Type        string     `json:"type"`                   // info | booking | payment | system
	ReferenceID string     `json:"reference_id,omitempty"` // e.g. a booking id
	IsRead      bool       `json:"is_read"`
	ReadAt      *time.Time `json:"read_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// POST /notifications/send (admin)
type SendNotificationRequest struct {
	UserID      string `json:"user_id"`
	Title       string `json:"title"`
	Message     string `json:"message"`
	Type        string `json:"type,omitempty"`
	ReferenceID string `json:"reference_id,omitempty"`
}