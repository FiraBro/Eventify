package models

import (
	"time"

	"local-go/db"
)

func CreateEvent(e *Event) error {
	query := `
	INSERT INTO events (id, name, description, location, user_id, date_time)
	VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := db.DB.Exec(
		query,
		e.ID,
		e.Name,
		e.Description,
		e.Location,
		e.UserId,
		e.DateTime.Format(time.RFC3339), // ✅ store as string
	)

	return err
}

func GetAllEvents() ([]Event, error) {
	rows, err := db.DB.Query(`SELECT * FROM events`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []Event{}

	for rows.Next() {
		var e Event
		var dateStr string // 👈 IMPORTANT

		err := rows.Scan(
			&e.ID,
			&e.Name,
			&e.Description,
			&e.Location,
			&e.UserId,
			&dateStr,
		)
		if err != nil {
			return nil, err
		}

		e.DateTime, _ = time.Parse(time.RFC3339, dateStr)
		events = append(events, e)
	}

	return events, nil
}
