package models

import "time"

// Event model
type Event struct {
	ID          string    `json:"id"`
	Name        string    `json:"name" binding:"required"`
	Description string    `json:"description" binding:"required"`
	Location    string    `json:"location" binding:"required"`
	UserId      string    `json:"user_id"`
	DateTime    time.Time `json:"date_time"`
}

// Global in-memory event storage
var Events []Event

// AddEvent save a new event to the in-memory storage
func Save(e Event) {
	Events = append(Events, e)
}
