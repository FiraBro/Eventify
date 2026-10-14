package models

import (
	"encoding/json"
	"time"
)

// ActivityLog is one recorded write request (POST / PUT / PATCH / DELETE).
// Request bodies, headers and query strings are NEVER stored.
type ActivityLog struct {
	ID         string          `json:"id"`
	UserID     string          `json:"user_id,omitempty"`
	Role       string          `json:"role,omitempty"`
	Method     string          `json:"method"`
	Path       string          `json:"path"`
	Route      string          `json:"route,omitempty"` // pattern, e.g. /api/v1/bookings/:id
	StatusCode int             `json:"status_code"`
	IP         string          `json:"ip,omitempty"`
	DurationMs int             `json:"duration_ms"`
	Details    json.RawMessage `json:"details,omitempty"` // optional extra info set by a handler
	CreatedAt  time.Time       `json:"created_at"`
}

// LogListParams are the raw query parameters of GET /logs.
type LogListParams struct {
	UserID string
	Method string
	Status string
	Q      string
	From   string
	To     string
	Page   int
	Limit  int
}

// LogFilter is the validated form handed to the repository.
type LogFilter struct {
	UserID string
	Method string
	Status int
	Q      string // already escaped for ILIKE
	From   *time.Time
	To     *time.Time // exclusive
	Limit  int
	Offset int
}

type LogPage struct {
	Items []ActivityLog
	Total int
	Page  int
	Limit int
}