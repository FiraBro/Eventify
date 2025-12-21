package models

import "time"

type Event struct {
	ID          string    `json:"id"`
	Name        string    `json:"name" binding:"required"`
	Description string    `json:"description" binding:"required"`
	Location    string    `json:"location" binding:"required"`
	UserId      string    `json:"user_id"`
	DateTime    time.Time `json:"date_time"`
}
