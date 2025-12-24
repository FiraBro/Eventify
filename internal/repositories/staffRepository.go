package repositories

import (
	"database/sql"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/google/uuid"
)

type StaffRepository struct {
	db *sql.DB
}

func NewStaffRepository(db *sql.DB) *StaffRepository {
	return &StaffRepository{db: db}
}

//
// ------------------- STAFF CRUD -------------------
//

func (r *StaffRepository) GetAll() ([]models.Staff, error) {
	rows, err := r.db.Query(`
		SELECT id, name, email, phone
		FROM staff
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var staffList []models.Staff
	for rows.Next() {
		var s models.Staff
		if err := rows.Scan(&s.ID, &s.Name, &s.Email, &s.Phone); err != nil {
			return nil, err
		}
		staffList = append(staffList, s)
	}
	return staffList, nil
}

func (r *StaffRepository) Create(staff *models.Staff) error {
	if staff.ID == "" {
		staff.ID = uuid.New().String()
	}

	_, err := r.db.Exec(`
		INSERT INTO staff (id, name, email, phone, role)
		VALUES ($1, $2, $3, $4, $5)
	`,
		staff.ID,
		staff.Name,
		staff.Email,
		staff.Phone,
		staff.Role,
	)

	return err
}

func (r *StaffRepository) GetByID(id string) (*models.Staff, error) {
	row := r.db.QueryRow(`
		SELECT id, name, email, phone
		FROM staff
		WHERE id = $1
	`, id)

	var s models.Staff
	if err := row.Scan(&s.ID, &s.Name, &s.Email, &s.Phone); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *StaffRepository) Update(id string, staff *models.Staff) error {
	_, err := r.db.Exec(`
		UPDATE staff
		SET name = $1,
		    email = $2,
		    phone = $3
		WHERE id = $4
	`,
		staff.Name,
		staff.Email,
		staff.Phone,
		id,
	)

	return err
}

func (r *StaffRepository) Delete(id string) error {
	_, err := r.db.Exec(`
		DELETE FROM staff
		WHERE id = $1
	`, id)

	return err
}

//
// ------------------- SERVICES -------------------
//

func (r *StaffRepository) GetServices(staffID string) ([]string, error) {
	rows, err := r.db.Query(`
		SELECT service_name
		FROM staff_services
		WHERE staff_id = $1
	`, staffID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		services = append(services, s)
	}
	return services, nil
}

func (r *StaffRepository) AssignServices(staffID string, services []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, svc := range services {
		id := uuid.New().String()
		_, err := tx.Exec(`
			INSERT INTO staff_services (id, staff_id, service_name)
			VALUES ($1, $2, $3)
		`,
			id,
			staffID,
			svc,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

//
// ------------------- SCHEDULE -------------------
//

func (r *StaffRepository) GetSchedule(staffID string) ([]map[string]string, error) {
	rows, err := r.db.Query(`
		SELECT day_of_week, start_time, end_time
		FROM staff_schedule
		WHERE staff_id = $1
	`, staffID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedule []map[string]string
	for rows.Next() {
		var day, start, end string
		if err := rows.Scan(&day, &start, &end); err != nil {
			return nil, err
		}

		schedule = append(schedule, map[string]string{
			"day":   day,
			"start": start,
			"end":   end,
		})
	}
	return schedule, nil
}

func (r *StaffRepository) SetSchedule(staffID string, entries []map[string]string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear old schedule
	_, err = tx.Exec(`
		DELETE FROM staff_schedule
		WHERE staff_id = $1
	`, staffID)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		id := uuid.New().String()
		_, err := tx.Exec(`
			INSERT INTO staff_schedule
			(id, staff_id, day_of_week, start_time, end_time)
			VALUES ($1, $2, $3, $4, $5)
		`,
			id,
			staffID,
			entry["day"],
			entry["start"],
			entry["end"],
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

//
// ------------------- HOLIDAYS -------------------
//

func (r *StaffRepository) AddHoliday(staffID, date, reason string) error {
	id := uuid.New().String()
	_, err := r.db.Exec(`
		INSERT INTO staff_holidays (id, staff_id, date, reason)
		VALUES ($1, $2, $3, $4)
	`,
		id,
		staffID,
		date,
		reason,
	)

	return err
}
