package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/google/uuid"
)

type StaffInvitation struct {
	ID        uuid.UUID
	ExpiresAt time.Time
}

var ErrInvalidInvitation = errors.New("invalid or expired invitation")

type StaffRepository struct {
	db *sql.DB
}

func NewStaffRepository(db *sql.DB) *StaffRepository {
	return &StaffRepository{db: db}
}

// ------------------- STAFF CRUD -------------------

func (r *StaffRepository) GetAll(ctx context.Context) ([]models.Staff, error) {
	query := `SELECT id, name, email, phone FROM staff ORDER BY name ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	staffList := make([]models.Staff, 0)
	staffIndexes := make(map[string]int)
	for rows.Next() {
		var s models.Staff
		if err := rows.Scan(&s.ID, &s.Name, &s.Email, &s.Phone); err != nil {
			_ = rows.Close()
			return nil, err
		}
		s.Services = make([]string, 0)
		staffIndexes[s.ID] = len(staffList)
		staffList = append(staffList, s)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	serviceRows, err := r.db.QueryContext(ctx, `SELECT staff_id, service_id FROM staff_services`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = serviceRows.Close() }()
	for serviceRows.Next() {
		var staffID, serviceID string
		if err := serviceRows.Scan(&staffID, &serviceID); err != nil {
			return nil, err
		}
		if idx, ok := staffIndexes[staffID]; ok {
			staffList[idx].Services = append(staffList[idx].Services, serviceID)
		}
	}
	if err := serviceRows.Err(); err != nil {
		return nil, err
	}
	return staffList, nil
}

func (r *StaffRepository) Create(ctx context.Context, staff *models.Staff) error {
	if staff.ID == "" {
		staff.ID = uuid.New().String()
	}

	query := `
    INSERT INTO staff (id, name, email, phone)
    VALUES ($1, $2, $3, $4)
`
	_, err := r.db.ExecContext(ctx, query, staff.ID, staff.Name, staff.Email, staff.Phone)
	return err
}

func (r *StaffRepository) GetByID(ctx context.Context, id string) (*models.Staff, error) {
	query := `SELECT id, name, email, phone FROM staff WHERE id = $1`
	var s models.Staff
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.Name, &s.Email, &s.Phone,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *StaffRepository) Update(ctx context.Context, id string, staff *models.Staff) error {
	query := `UPDATE staff SET name = $1, email = $2, phone = $3 WHERE id = $4`
	_, err := r.db.ExecContext(ctx, query, staff.Name, staff.Email, staff.Phone, id)
	return err
}

func (r *StaffRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM staff WHERE id = $1`, id)
	return err
}

// ------------------- SERVICES (MANY-TO-MANY) -------------------

func (r *StaffRepository) GetServiceIDs(ctx context.Context, staffID string) ([]string, error) {
	query := `SELECT service_id FROM staff_services WHERE staff_id = $1`
	rows, err := r.db.QueryContext(ctx, query, staffID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
func (r *StaffRepository) GetStaffByService(ctx context.Context, serviceID string) ([]models.Staff, error) {
	query := `
    SELECT s.id, s.name, s.email, COALESCE(s.phone, '') AS phone
    FROM staff s
    JOIN staff_services ss ON s.id = ss.staff_id
    WHERE ss.service_id = $1
`

	rows, err := r.db.QueryContext(ctx, query, serviceID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var staffList []models.Staff
	for rows.Next() {
		var s models.Staff
		if err := rows.Scan(&s.ID, &s.Name, &s.Email, &s.Phone); err != nil {
			return nil, err
		}
		staffList = append(staffList, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return staffList, nil
}

// AssignServices uses a Sync pattern: Delete existing associations and re-insert
func (r *StaffRepository) AssignServices(ctx context.Context, staffID string, serviceIDs []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	// 1. Clear current assignments
	if _, err := tx.ExecContext(ctx, `DELETE FROM staff_services WHERE staff_id = $1`, staffID); err != nil {
		return err
	}

	// 2. Insert new ones
	for _, svcID := range serviceIDs {
		_, err := tx.ExecContext(ctx, `
            INSERT INTO staff_services (id, staff_id, service_id)
            VALUES ($1, $2, $3)
        `, uuid.New().String(), staffID, svcID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ------------------- SCHEDULE -------------------

func (r *StaffRepository) SetSchedule(ctx context.Context, staffID string, entries []map[string]string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackTx(tx)

	if _, err := tx.ExecContext(ctx, `DELETE FROM staff_schedule WHERE staff_id = $1`, staffID); err != nil {
		return err
	}

	for _, entry := range entries {
		_, err := tx.ExecContext(ctx, `
            INSERT INTO staff_schedule (id, staff_id, day_of_week, start_time, end_time)
            VALUES ($1, $2, $3, $4, $5)
        `, uuid.New().String(), staffID, entry["day"], entry["start"], entry["end"])
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
func (r *StaffRepository) GetSchedule(ctx context.Context, staffID string) ([]map[string]string, error) {
	query := `SELECT day_of_week, start_time, end_time FROM staff_schedule WHERE staff_id = $1`
	rows, err := r.db.QueryContext(ctx, query, staffID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

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

// ------------------- HOLIDAYS -------------------

func (r *StaffRepository) AddHoliday(ctx context.Context, staffID, date, reason string) error {
	query := `INSERT INTO staff_holidays (id, staff_id, date, reason) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query, uuid.New().String(), staffID, date, reason)
	return err
}

// Refactored to return both for the availability service
func (r *StaffRepository) GetAvailabilityData(ctx context.Context, staffID string) ([]map[string]string, []string, error) {
	// 1. Reuse your GetSchedule logic
	schedule, err := r.GetSchedule(ctx, staffID)
	if err != nil {
		return nil, nil, err
	}

	// 2. Get Holiday Dates
	hRows, err := r.db.QueryContext(ctx, `SELECT date FROM staff_holidays WHERE staff_id = $1`, staffID)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		_ = hRows.Close()
	}()

	var holidays []string
	for hRows.Next() {
		var date string
		if err := hRows.Scan(&date); err != nil {
			return nil, nil, err
		}
		holidays = append(holidays, date)
	}

	return schedule, holidays, nil
}

// ------------------- INVITATIONS -------------------

func (r *StaffRepository) GetPendingInvitationByToken(ctx context.Context, token string) (*StaffInvitation, error) {
	query := `
		SELECT id, invitation_expires_at
		FROM staff
		WHERE invitation_token = $1 AND status = 'PENDING_INVITATION'
	`
	var invite StaffInvitation
	err := r.db.QueryRowContext(ctx, query, token).Scan(&invite.ID, &invite.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return &invite, nil
}

func (r *StaffRepository) ActivateStaffPassword(ctx context.Context, id uuid.UUID, token string, passwordHash string) error {
	query := `
		UPDATE staff
		SET password_hash = $1,
		    status = 'ACTIVE',
		    invitation_token = NULL,
		    invitation_expires_at = NULL
		WHERE id = $2
		  AND invitation_token = $3
		  AND status = 'PENDING_INVITATION'
		  AND invitation_expires_at > NOW()
	`
	result, err := r.db.ExecContext(ctx, query, passwordHash, id, token)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return ErrInvalidInvitation
	}
	return nil
}
