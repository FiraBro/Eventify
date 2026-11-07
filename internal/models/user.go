package models

import "time"

const (
	AccountStatusActive            = "ACTIVE"
	AccountStatusPendingInvitation = "PENDING_INVITATION"
	AccountStatusSuspended         = "SUSPENDED"
)

type User struct {
	ID             string     `json:"id"`
	Username       string     `json:"username"`
	Email          string     `json:"email"`
	Password       string     `json:"password"` // allow JSON binding
	Role           string     `json:"role"`
	AccountStatus  string     `json:"-"`
	CreatedAt      time.Time  `json:"created_at"`
	DeletedAt      *time.Time `json:"deleted_at"`
	DeleteDeadline *time.Time `json:"delete_deadline"`
}
