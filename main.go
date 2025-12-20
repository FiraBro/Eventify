package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// Health / test route
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Event Booking API is running 🚀",
		})
	})

	// Example event route
	r.GET("/events", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"events": []gin.H{
				{
					"id":    1,
					"title": "Tech Conference 2025",
					"city":  "Addis Ababa",
				},
				{
					"id":    2,
					"title": "Music Festival",
					"city":  "Dire Dawa",
				},
			},
		})
	})

	r.Run(":8080") // http://localhost:8080
}
