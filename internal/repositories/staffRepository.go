package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/google/uuid"
)

type StaffRepository struct {
    db *sql.DB
}

func NewStaffRepository(db *sql.DB) *StaffRepository {
    return &StaffRepository{db: db}
}

// ------------------- STAFF CRUD -------------------

func (r *StaffRepository) GetAll(ctx context.Context) ([]models.Staff, error) {
    query := `SELECT id, name, email, phone, role FROM staff ORDER BY name ASC`
    rows, err := r.db.QueryContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var staffList []models.Staff
    for rows.Next() {
        var s models.Staff
        if err := rows.Scan(&s.ID, &s.Name, &s.Email, &s.Phone, &s.Role); err != nil {
            return nil, err
        }
        staffList = append(staffList, s)
    }
    return staffList, nil
}

func (r *StaffRepository) Create(ctx context.Context, staff *models.Staff) error {
    if staff.ID == "" {
        staff.ID = uuid.New().String()
    }

    query := `
        INSERT INTO staff (id, name, email, phone, role)
        VALUES ($1, $2, $3, $4, $5)
    `
    _, err := r.db.ExecContext(ctx, query, staff.ID, staff.Name, staff.Email, staff.Phone, staff.Role)
    return err
}

func (r *StaffRepository) GetByID(ctx context.Context, id string) (*models.Staff, error) {
    query := `SELECT id, name, email, phone, role FROM staff WHERE id = $1`
    var s models.Staff
    err := r.db.QueryRowContext(ctx, query, id).Scan(&s.ID, &s.Name, &s.Email, &s.Phone, &s.Role)
    
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, nil
        }
        return nil, err
    }
    return &s, nil
}

func (r *StaffRepository) Update(ctx context.Context, id string, staff *models.Staff) error {
    query := `UPDATE staff SET name = $1, email = $2, phone = $3, role = $4 WHERE id = $5`
    _, err := r.db.ExecContext(ctx, query, staff.Name, staff.Email, staff.Phone, staff.Role, id)
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
    defer rows.Close()

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

// AssignServices uses a Sync pattern: Delete existing associations and re-insert
func (r *StaffRepository) AssignServices(ctx context.Context, staffID string, serviceIDs []string) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

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

func (r *StaffRepository) GetSchedule(ctx context.Context, staffID string) ([]map[string]string, error) {
    query := `SELECT day_of_week, start_time, end_time FROM staff_schedule WHERE staff_id = $1 ORDER BY day_of_week`
    rows, err := r.db.QueryContext(ctx, query, staffID)
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

func (r *StaffRepository) SetSchedule(ctx context.Context, staffID string, entries []map[string]string) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

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

// ------------------- HOLIDAYS -------------------

func (r *StaffRepository) AddHoliday(ctx context.Context, staffID, date, reason string) error {
    query := `INSERT INTO staff_holidays (id, staff_id, date, reason) VALUES ($1, $2, $3, $4)`
    _, err := r.db.ExecContext(ctx, query, uuid.New().String(), staffID, date, reason)
    return err
}