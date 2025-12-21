package main

import (
	"net/http"
	"time"

	"local-go/db"
	"local-go/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// In-memory event storage
var events []models.Event

func main() {
	db.InitDb()
	r := gin.Default()

	// Health / test route
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Event Booking API is running 🚀",
		})
	})

	// Event routes
	r.GET("/events", GetEvent)
	r.POST("/events", CreateEvent)

	r.Run(":8080") // http://localhost:8080
}

// Get all events
func GetEvent(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"events": events,
	})
}

// Create new event
func CreateEvent(c *gin.Context) {
	var newEvent models.Event

	// Bind JSON and validate required fields
	if err := c.ShouldBindJSON(&newEvent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Assign unique ID and set default DateTime if empty
	newEvent.ID = uuid.New().String()
	if newEvent.DateTime.IsZero() {
		newEvent.DateTime = time.Now()
	}

	// Save event in memory
	events = append(events, newEvent)

	// Return success response
	c.JSON(http.StatusCreated, gin.H{
		"message": "Event created successfully",
		"event":   newEvent,
	})
}
