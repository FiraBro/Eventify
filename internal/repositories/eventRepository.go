package repositories

import (
	"database/sql"
	"time"

	"github.com/FiraBro/local-go/internal/models"
)

type EventRepository struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
}

// ----------------------------
// CREATE EVENT
// ----------------------------
func (r *EventRepository) Create(event *models.Event) error {
	if event.DateTime.IsZero() {
		event.DateTime = time.Now()
	}

	query := `
		INSERT INTO events (id, name, description, location, user_id, date_time)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(
		query,
		event.ID,
		event.Name,
		event.Description,
		event.Location,
		event.UserId,
		event.DateTime,
	)
	return err
}

// ----------------------------
// GET ALL EVENTS
// ----------------------------
func (r *EventRepository) GetAll() ([]models.Event, error) {
	rows, err := r.db.Query(`
		SELECT id, name, description, location, user_id, date_time
		FROM events
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var e models.Event
		if err := rows.Scan(
			&e.ID,
			&e.Name,
			&e.Description,
			&e.Location,
			&e.UserId,
			&e.DateTime,
		); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

// ----------------------------
// GET EVENT BY ID
// ----------------------------
func (r *EventRepository) GetByID(id string) (*models.Event, error) {
	row := r.db.QueryRow(`
		SELECT id, name, description, location, user_id, date_time
		FROM events
		WHERE id = $1
	`, id)

	var e models.Event
	if err := row.Scan(
		&e.ID,
		&e.Name,
		&e.Description,
		&e.Location,
		&e.UserId,
		&e.DateTime,
	); err != nil {
		return nil, err
	}
	return &e, nil
}

// ----------------------------
// UPDATE EVENT
// ----------------------------
func (r *EventRepository) Update(event *models.Event) error {
	query := `
		UPDATE events
		SET name = $1,
		    description = $2,
		    location = $3,
		    user_id = $4,
		    date_time = $5
		WHERE id = $6
	`
	_, err := r.db.Exec(
		query,
		event.Name,
		event.Description,
		event.Location,
		event.UserId,
		event.DateTime,
		event.ID,
	)
	return err
}

// ----------------------------
// DELETE EVENT
// ----------------------------
func (r *EventRepository) Delete(id string) error {
	_, err := r.db.Exec(`DELETE FROM events WHERE id = $1`, id)
	return err
}
