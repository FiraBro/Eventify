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

const bookingOverlapConstraint = "bookings_no_staff_overlap"

// mapBookingWriteError turns the overlap violation and deadlocks into ErrSlotConflict; other errors pass through.
func mapBookingWriteError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		// Catch standard exclusion overlap OR concurrency deadlocks caused by parallel inserts
		if pqErr.Code == "40P01" || (pqErr.Code == "23P01" && pqErr.Constraint == bookingOverlapConstraint) {
			return ErrSlotConflict
		}
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
	defer func() {
		_ = rows.Close()
	}()

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

// ShiftRow is one working window of a staff member (table staff_schedule), exactly as stored.
type ShiftRow struct {
	StaffID string
	Day     string // weekday name, e.g. "Monday"
	Start   string // clock time, e.g. "09:00"
	End     string
}

// StaffCandidate is a staff member who offers a service, with that day's number of active bookings.
type StaffCandidate struct {
	ID   string
	Load int
}

// --------------------
// DOES ANY STAFF MEMBER OFFER THIS SERVICE
// --------------------
func (r *BookingRepository) ServiceHasStaff(ctx context.Context, serviceID string) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM staff_services WHERE service_id = $1)`, serviceID).Scan(&ok)
	return ok, err
}

const staffCandidatesSQL = `
	SELECT st.id::text, COALESCE(day_load.n, 0)
	FROM staff_services ss
	JOIN staff st ON st.id = ss.staff_id
	LEFT JOIN (
		SELECT staff_id, COUNT(*) AS n
		FROM bookings
		WHERE staff_id IS NOT NULL
		  AND status IN ('pending', 'confirmed')
		  AND scheduled_at >= $2 AND scheduled_at < $3
		GROUP BY staff_id
	) day_load ON day_load.staff_id = st.id
	WHERE ss.service_id = $1
	ORDER BY COALESCE(day_load.n, 0) ASC, st.id ASC`

// --------------------
// STAFF WHO OFFER A SERVICE, LEAST BUSY FIRST (between dayStart and dayEnd)
// --------------------
func (r *BookingRepository) StaffCandidates(ctx context.Context, serviceID string, dayStart, dayEnd time.Time) ([]StaffCandidate, error) {
	rows, err := r.db.QueryContext(ctx, staffCandidatesSQL, serviceID, dayStart, dayEnd)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	var out []StaffCandidate
	for rows.Next() {
		var c StaffCandidate
		if err := rows.Scan(&c.ID, &c.Load); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// --------------------
// WORKING WINDOWS OF THESE STAFF MEMBERS
// --------------------
func (r *BookingRepository) StaffShifts(ctx context.Context, staffIDs []string) (map[string][]ShiftRow, error) {
	shifts := map[string][]ShiftRow{}
	if len(staffIDs) == 0 {
		return shifts, nil
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT staff_id::text, day_of_week, start_time, end_time FROM staff_schedule WHERE staff_id = ANY($1::uuid[])`,
		pq.Array(staffIDs))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var s ShiftRow
		if err := rows.Scan(&s.StaffID, &s.Day, &s.Start, &s.End); err != nil {
			return nil, err
		}
		shifts[s.StaffID] = append(shifts[s.StaffID], s)
	}
	return shifts, rows.Err()
}

// --------------------
// WHICH OF THESE STAFF MEMBERS ARE ON HOLIDAY ON THAT DATE (YYYY-MM-DD)
// --------------------
func (r *BookingRepository) StaffOnHoliday(ctx context.Context, staffIDs []string, date string) (map[string]bool, error) {
	onHoliday := map[string]bool{}
	if len(staffIDs) == 0 {
		return onHoliday, nil
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT staff_id::text FROM staff_holidays WHERE staff_id = ANY($1::uuid[]) AND date = $2::date`,
		pq.Array(staffIDs), date)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		onHoliday[id] = true
	}
	return onHoliday, rows.Err()
}

// --------------------
// BUSINESS TIMEZONE (settings), "UTC" when settings were never saved
// --------------------
func (r *BookingRepository) BusinessTimezone(ctx context.Context) (string, error) {
	var tz string
	err := r.db.QueryRowContext(ctx, `SELECT timezone FROM settings WHERE id = 1`).Scan(&tz)
	if errors.Is(err, sql.ErrNoRows) {
		return "UTC", nil
	}
	return tz, err
}
