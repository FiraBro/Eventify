package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"local-go/db"
	"local-go/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CREATE EVENT
func CreateEvent(c *gin.Context) {
	var event models.Event

	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	event.ID = uuid.New().String()

	if event.DateTime.IsZero() {
		event.DateTime = time.Now()
	}

	_, err := db.DB.Exec(
		`INSERT INTO events (id, name, description, location, user_id, date_time)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		event.ID,
		event.Name,
		event.Description,
		event.Location,
		event.UserId,
		event.DateTime.Format(time.RFC3339),
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, event)
}


// GET ALL EVENTS
func GetEvents(c *gin.Context) {
	rows, err := db.DB.Query(`SELECT * FROM events`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var events []models.Event

	for rows.Next() {
		var e models.Event
		var dateStr string

		err := rows.Scan(
			&e.ID,
			&e.Name,
			&e.Description,
			&e.Location,
			&e.UserId,
			&dateStr,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		e.DateTime, _ = time.Parse(time.RFC3339, dateStr)
		events = append(events, e)
	}

	c.JSON(http.StatusOK, events)
}


// GET EVENT BY ID ✅
func GetEventByID(c *gin.Context) {
	id := c.Param("id")

	var event models.Event
	var dateStr string

	err := db.DB.QueryRow(
		`SELECT * FROM events WHERE id = ?`,
		id,
	).Scan(
		&event.ID,
		&event.Name,
		&event.Description,
		&event.Location,
		&event.UserId,
		&dateStr,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	event.DateTime, _ = time.Parse(time.RFC3339, dateStr)

	c.JSON(http.StatusOK, event)
}
