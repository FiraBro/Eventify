package models

import "time"

// CreateBookingRequest contains fields accepted from the client.
type CreateBookingRequest struct {
	ServiceID   string    `json:"service_id" binding:"required,uuid"`
	StaffID     string    `json:"staff_id,omitempty" binding:"omitempty,uuid"`
	ScheduledAt time.Time `json:"scheduled_at" binding:"required"`
	Notes       string    `json:"notes" binding:"max=1000"`
}

type Booking struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	ServiceID   string    `json:"service_id"`
	StaffID     string    `json:"staff_id,omitempty"`
	ScheduledAt time.Time `json:"scheduled_at"`
	EndAt       time.Time `json:"end_at"` // computed by the server, never read from the client
	Status      string    `json:"status"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DTO for PATCH updates (partial updates)
type UpdateBookingRequest struct {
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
	StaffID     *string    `json:"staff_id,omitempty"` // admin only; "" clears the assignment
}

// DTO for PATCH /bookings/:id/status
type UpdateBookingStatusRequest struct {
	Status string `json:"status"`
}
