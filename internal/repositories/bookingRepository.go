package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type BookingRepository struct {
	db *sql.DB
}

func NewBookingRepository(db *sql.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

// ErrSlotConflict means the database rejected a booking because it overlaps another ACTIVE booking
// of the same staff member (exclusion constraint bookings_no_staff_overlap).
var ErrSlotConflict = errors.New("booking slot conflict")

const (
	exclusionViolation       pq.ErrorCode = "23P01"
	bookingOverlapConstraint              = "bookings_no_staff_overlap"
)

// mapBookingWriteError turns the overlap violation into ErrSlotConflict; other errors pass through.
func mapBookingWriteError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == exclusionViolation && pqErr.Constraint == bookingOverlapConstraint {
		return ErrSlotConflict
	}
	return err
}

const bookingColumns = `id, user_id, service_id, COALESCE(staff_id::text, ''), scheduled_at, end_at, status, notes, created_at, updated_at`

func scanBooking(scan func(dest ...any) error) (models.Booking, error) {
	var b models.Booking
	err := scan(&b.ID, &b.UserID, &b.ServiceID, &b.StaffID, &b.ScheduledAt, &b.EndAt, &b.Status, &b.Notes, &b.CreatedAt, &b.UpdatedAt)
	return b, err
}

func (r *BookingRepository) queryList(ctx context.Context, query string, args ...any) ([]models.Booking, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bookings := []models.Booking{}
	for rows.Next() {
		b, err := scanBooking(rows.Scan)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, b)
	}
	return bookings, rows.Err()
}

// --------------------
// CREATE BOOKING
// --------------------
func (r *BookingRepository) Create(ctx context.Context, b *models.Booking) error {
	b.ID = uuid.New().String()
	b.CreatedAt = time.Now()
	b.UpdatedAt = time.Now()

	query := `
		INSERT INTO bookings (id, user_id, service_id, staff_id, scheduled_at, end_at, status, notes, created_at, updated_at)
		VALUES ($1,$2,$3,NULLIF($4::text,'')::uuid,$5,$6,$7,$8,$9,$10)
	`
	_, err := r.db.ExecContext(ctx, query,
		b.ID, b.UserID, b.ServiceID, b.StaffID, b.ScheduledAt, b.EndAt, b.Status, b.Notes, b.CreatedAt, b.UpdatedAt,
	)
	return mapBookingWriteError(err)
}

// --------------------
// GET ALL BOOKINGS (optional status filter)
// --------------------
func (r *BookingRepository) GetAll(ctx context.Context, status string) ([]models.Booking, error) {
	if status != "" {
		return r.queryList(ctx,
			`SELECT `+bookingColumns+` FROM bookings WHERE status = $1 ORDER BY created_at DESC`, status)
	}
	return r.queryList(ctx,
		`SELECT `+bookingColumns+` FROM bookings ORDER BY created_at DESC`)
}

// --------------------
// GET BOOKINGS BY USER
// --------------------
func (r *BookingRepository) GetByUserID(ctx context.Context, userID string) ([]models.Booking, error) {
	return r.queryList(ctx,
		`SELECT `+bookingColumns+` FROM bookings WHERE user_id = $1 ORDER BY created_at DESC`, userID)
}

// --------------------
// GET BOOKING BY ID
// --------------------
func (r *BookingRepository) GetByID(ctx context.Context, id string) (*models.Booking, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+bookingColumns+` FROM bookings WHERE id = $1`, id)

	b, err := scanBooking(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // not found
		}
		return nil, err
	}
	return &b, nil
}

// --------------------
// UPDATE BOOKING
// --------------------
func (r *BookingRepository) Update(ctx context.Context, id string, b *models.Booking) error {
	query := `
		UPDATE bookings
		SET scheduled_at = $1, end_at = $2, notes = $3, staff_id = NULLIF($4::text,'')::uuid, updated_at = $5
		WHERE id = $6
	`
	_, err := r.db.ExecContext(ctx, query, b.ScheduledAt, b.EndAt, b.Notes, b.StaffID, time.Now(), id)
	return mapBookingWriteError(err)
}

// --------------------
// CHECK STAFF EXISTS
// --------------------
func (r *BookingRepository) StaffExists(ctx context.Context, staffID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM staff WHERE id = $1)`, staffID).Scan(&exists)
	return exists, err
}

// --------------------
// CHECK STAFF OFFERS SERVICE (staff_services link table)
// --------------------
func (r *BookingRepository) StaffOffersService(ctx context.Context, staffID, serviceID string) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM staff_services WHERE staff_id = $1 AND service_id = $2)`,
		staffID, serviceID).Scan(&ok)
	return ok, err
}

// --------------------
// UPDATE BOOKING STATUS
// --------------------
func (r *BookingRepository) UpdateStatus(ctx context.Context, id, status string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE bookings SET status = $1, updated_at = $2 WHERE id = $3`,
		status, time.Now(), id,
	)
	return err
}

// --------------------
// DELETE BOOKING
// --------------------
func (r *BookingRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM bookings WHERE id = $1`, id)
	return err
}
