package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/FiraBro/local-go/internal/models"
)

// ReportRepository only reads. All ranges are [start, end) in UTC.
type ReportRepository struct {
	db *sql.DB
}

func NewReportRepository(db *sql.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

// --------------------
// BOOKINGS: count per status
// --------------------
func (r *ReportRepository) BookingsByStatus(ctx context.Context, start, end time.Time) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT status, COUNT(*)
		FROM bookings
		WHERE scheduled_at >= $1 AND scheduled_at < $2
		GROUP BY status
	`, start, end)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	counts := map[string]int{}
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return nil, err
		}
		counts[status] = n
	}
	return counts, rows.Err()
}

// --------------------
// BOOKINGS: count per day
// --------------------
func (r *ReportRepository) BookingsDaily(ctx context.Context, start, end time.Time) ([]models.DailyCount, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT to_char(scheduled_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS day, COUNT(*)
		FROM bookings
		WHERE scheduled_at >= $1 AND scheduled_at < $2
		GROUP BY day
		ORDER BY day
	`, start, end)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	daily := []models.DailyCount{}
	for rows.Next() {
		var d models.DailyCount
		if err := rows.Scan(&d.Date, &d.Count); err != nil {
			return nil, err
		}
		daily = append(daily, d)
	}
	return daily, rows.Err()
}

// --------------------
// REVENUE: totals (gross by paid_at, refunds by refunded_at)
// --------------------
func (r *ReportRepository) RevenueTotals(ctx context.Context, start, end time.Time) (*models.RevenueTotals, error) {
	var t models.RevenueTotals
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE paid_at >= $1 AND paid_at < $2), 0),
			COUNT(*)             FILTER (WHERE paid_at >= $1 AND paid_at < $2),
			COALESCE(SUM(amount) FILTER (WHERE status = 'refunded' AND refunded_at >= $1 AND refunded_at < $2), 0),
			COUNT(*)             FILTER (WHERE status = 'refunded' AND refunded_at >= $1 AND refunded_at < $2)
		FROM payments
	`, start, end).Scan(&t.Gross, &t.Payments, &t.Refunded, &t.Refunds)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// --------------------
// REVENUE: time series. unit MUST be validated by the caller (day | week | month).
// --------------------
func (r *ReportRepository) RevenueSeries(ctx context.Context, start, end time.Time, unit string) ([]models.RevenuePoint, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT to_char(period, 'YYYY-MM-DD'), COALESCE(SUM(gross), 0), COALESCE(SUM(refunded), 0)
		FROM (
			SELECT date_trunc($3::text, paid_at AT TIME ZONE 'UTC') AS period,
			       amount AS gross, 0::numeric AS refunded
			FROM payments
			WHERE paid_at >= $1 AND paid_at < $2
			UNION ALL
			SELECT date_trunc($3::text, refunded_at AT TIME ZONE 'UTC'),
			       0::numeric, amount
			FROM payments
			WHERE status = 'refunded' AND refunded_at >= $1 AND refunded_at < $2
		) t
		GROUP BY period
		ORDER BY period
	`, start, end, unit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	series := []models.RevenuePoint{}
	for rows.Next() {
		var p models.RevenuePoint
		if err := rows.Scan(&p.Period, &p.Gross, &p.Refunded); err != nil {
			return nil, err
		}
		series = append(series, p)
	}
	return series, rows.Err()
}

// --------------------
// SERVICES: bookings + net revenue per service
// (a booking has at most one live payment, so the LEFT JOIN never double counts)
// --------------------
func (r *ReportRepository) ServicePerformance(ctx context.Context, start, end time.Time, limit int) ([]models.ServiceReportRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT s.id::text, s.name, s.category, s.price,
		       COALESCE(b.total, 0), COALESCE(b.completed, 0), COALESCE(b.cancelled, 0), COALESCE(b.revenue, 0)
		FROM services s
		LEFT JOIN (
			SELECT bk.service_id,
			       COUNT(*) AS total,
			       COUNT(*) FILTER (WHERE bk.status = 'completed') AS completed,
			       COUNT(*) FILTER (WHERE bk.status = 'cancelled') AS cancelled,
			       COALESCE(SUM(p.amount), 0) AS revenue
			FROM bookings bk
			LEFT JOIN payments p ON p.booking_id = bk.id AND p.status IN ('paid', 'refund_pending')
			WHERE bk.scheduled_at >= $1 AND bk.scheduled_at < $2
			GROUP BY bk.service_id
		) b ON b.service_id = s.id
		ORDER BY COALESCE(b.total, 0) DESC, s.name
		LIMIT $3
	`, start, end, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := []models.ServiceReportRow{}
	for rows.Next() {
		var s models.ServiceReportRow
		if err := rows.Scan(&s.ServiceID, &s.Name, &s.Category, &s.Price,
			&s.Bookings, &s.Completed, &s.Cancelled, &s.Revenue); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// --------------------
// STAFF: needs bookings.staff_id (see migrations/add_booking_staff_id.sql)
// --------------------
func (r *ReportRepository) StaffPerformance(ctx context.Context, start, end time.Time, limit int) ([]models.StaffReportRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT st.id::text, st.name, st.email,
		       COUNT(*),
		       COUNT(*) FILTER (WHERE bk.status = 'completed'),
		       COUNT(*) FILTER (WHERE bk.status = 'cancelled'),
		       COALESCE(SUM(p.amount), 0)
		FROM bookings bk
		JOIN staff st ON st.id = bk.staff_id
		LEFT JOIN payments p ON p.booking_id = bk.id AND p.status IN ('paid', 'refund_pending')
		WHERE bk.scheduled_at >= $1 AND bk.scheduled_at < $2
		GROUP BY st.id, st.name, st.email
		ORDER BY COUNT(*) FILTER (WHERE bk.status = 'completed') DESC, COUNT(*) DESC
		LIMIT $3
	`, start, end, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := []models.StaffReportRow{}
	for rows.Next() {
		var s models.StaffReportRow
		if err := rows.Scan(&s.StaffID, &s.Name, &s.Email, &s.Bookings, &s.Completed, &s.Cancelled, &s.Revenue); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// --------------------
// CUSTOMERS: top customers by net spend
// --------------------
func (r *ReportRepository) CustomerPerformance(ctx context.Context, start, end time.Time, limit int) ([]models.CustomerReportRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id::text, u.username, u.email,
		       COUNT(*),
		       COUNT(*) FILTER (WHERE bk.status = 'completed'),
		       COUNT(*) FILTER (WHERE bk.status = 'cancelled'),
		       COALESCE(SUM(p.amount), 0),
		       MAX(bk.scheduled_at)
		FROM bookings bk
		JOIN users u ON u.id = bk.user_id
		LEFT JOIN payments p ON p.booking_id = bk.id AND p.status IN ('paid', 'refund_pending')
		WHERE bk.scheduled_at >= $1 AND bk.scheduled_at < $2
		GROUP BY u.id, u.username, u.email
		ORDER BY COALESCE(SUM(p.amount), 0) DESC, COUNT(*) DESC
		LIMIT $3
	`, start, end, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := []models.CustomerReportRow{}
	for rows.Next() {
		var c models.CustomerReportRow
		if err := rows.Scan(&c.UserID, &c.Username, &c.Email, &c.Bookings, &c.Completed, &c.Cancelled, &c.TotalSpent, &c.LastBookingAt); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

// CUSTOMERS: how many distinct customers booked, and how many booked 2+ times
func (r *ReportRepository) CustomerSummary(ctx context.Context, start, end time.Time) (total, repeat int, err error) {
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE c >= 2)
		FROM (
			SELECT user_id, COUNT(*) AS c
			FROM bookings
			WHERE scheduled_at >= $1 AND scheduled_at < $2
			GROUP BY user_id
		) t
	`, start, end).Scan(&total, &repeat)
	return total, repeat, err
}
