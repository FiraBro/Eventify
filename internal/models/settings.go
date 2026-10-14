package models

import "time"

// Settings is a single row (id = 1) holding the business-wide configuration.
type Settings struct {
	BusinessName            string    `json:"business_name"`
	ContactEmail            string    `json:"contact_email"`
	ContactPhone            string    `json:"contact_phone"`
	Currency                string    `json:"currency"`
	Timezone                string    `json:"timezone"`
	MinBookingNoticeHours   int       `json:"min_booking_notice_hours"`
	MaxBookingAdvanceDays   int       `json:"max_booking_advance_days"`
	CancellationWindowHours int       `json:"cancellation_window_hours"`
	UpdatedBy               string    `json:"updated_by,omitempty"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// DTO for PATCH updates (partial updates)
type UpdateSettingsRequest struct {
	BusinessName            *string `json:"business_name,omitempty"`
	ContactEmail            *string `json:"contact_email,omitempty"`
	ContactPhone            *string `json:"contact_phone,omitempty"`
	Currency                *string `json:"currency,omitempty"`
	Timezone                *string `json:"timezone,omitempty"`
	MinBookingNoticeHours   *int    `json:"min_booking_notice_hours,omitempty"`
	MaxBookingAdvanceDays   *int    `json:"max_booking_advance_days,omitempty"`
	CancellationWindowHours *int    `json:"cancellation_window_hours,omitempty"`
}

// FieldChange / SettingsChanges describe what a PATCH actually changed (written to the activity log).
type FieldChange struct {
	Old any `json:"old"`
	New any `json:"new"`
}

type SettingsChanges map[string]FieldChange