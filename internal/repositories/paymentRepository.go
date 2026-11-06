package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/google/uuid"
)

type PaymentRepository struct {
	db *sql.DB
}

func NewPaymentRepository(db *sql.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

const paymentColumns = `id, booking_id, user_id, tx_ref, chapa_reference, amount, currency, status,
	checkout_url, refund_reference, refund_reason, paid_at, refunded_at, created_at, updated_at`

func scanPayment(scan func(dest ...any) error) (*models.Payment, error) {
	var p models.Payment
	err := scan(&p.ID, &p.BookingID, &p.UserID, &p.TxRef, &p.ChapaReference, &p.Amount, &p.Currency, &p.Status,
		&p.CheckoutURL, &p.RefundReference, &p.RefundReason, &p.PaidAt, &p.RefundedAt, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // not found
		}
		return nil, err
	}
	return &p, nil
}

func rowsChanged(res sql.Result, err error) (bool, error) {
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// --------------------
// CREATE PAYMENT
// --------------------
func (r *PaymentRepository) Create(ctx context.Context, p *models.Payment) error {
	p.ID = uuid.New().String()
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()

	query := `
		INSERT INTO payments (id, booking_id, user_id, tx_ref, amount, currency, status, checkout_url, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`
	_, err := r.db.ExecContext(ctx, query,
		p.ID, p.BookingID, p.UserID, p.TxRef, p.Amount, p.Currency, p.Status, p.CheckoutURL, p.CreatedAt, p.UpdatedAt,
	)
	return err
}

// --------------------
// GET CUSTOMER CONTACT (from the users table)
// --------------------
func (r *PaymentRepository) GetUserContact(ctx context.Context, userID string) (email, username string, err error) {
	err = r.db.QueryRowContext(ctx,
		`SELECT email, username FROM users WHERE id = $1`, userID).Scan(&email, &username)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil
	}
	return email, username, err
}

// --------------------
// GET LATEST PAYMENT OF A BOOKING
// --------------------
func (r *PaymentRepository) GetLatestByBookingID(ctx context.Context, bookingID string) (*models.Payment, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+paymentColumns+` FROM payments WHERE booking_id = $1 ORDER BY created_at DESC LIMIT 1`, bookingID)
	return scanPayment(row.Scan)
}

// --------------------
// GET PAYMENT BY TX_REF
// --------------------
func (r *PaymentRepository) GetByTxRef(ctx context.Context, txRef string) (*models.Payment, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+paymentColumns+` FROM payments WHERE tx_ref = $1`, txRef)
	return scanPayment(row.Scan)
}

// GetAll returns payments with their customer and the assigned or service-qualified staff member.
func (r *PaymentRepository) GetAll(ctx context.Context) ([]models.Payment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.id, p.booking_id, p.user_id, p.tx_ref, p.chapa_reference, p.amount, p.currency, p.status,
		       p.checkout_url, p.refund_reference, p.refund_reason, p.paid_at, p.refunded_at, p.created_at, p.updated_at,
		       u.id::text, u.username, u.email, st.id::text, st.name, st.email
		FROM payments p
		LEFT JOIN users u ON u.id = p.user_id
		LEFT JOIN bookings b ON b.id = p.booking_id
		LEFT JOIN LATERAL (
			SELECT s.id, s.name, s.email
			FROM staff s
			WHERE s.id = b.staff_id OR (NOT EXISTS (SELECT 1 FROM staff assigned WHERE assigned.id = b.staff_id) AND EXISTS (
				SELECT 1 FROM staff_services ss WHERE ss.staff_id = s.id AND ss.service_id = b.service_id
			))
			ORDER BY CASE WHEN s.id = b.staff_id THEN 0 ELSE 1 END, s.id
			LIMIT 1
		) st ON TRUE
		ORDER BY p.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	payments := []models.Payment{}
	for rows.Next() {
		var p models.Payment
		var customerID, customerName, customerEmail, staffID, staffName, staffEmail sql.NullString
		if err := rows.Scan(&p.ID, &p.BookingID, &p.UserID, &p.TxRef, &p.ChapaReference, &p.Amount, &p.Currency, &p.Status,
			&p.CheckoutURL, &p.RefundReference, &p.RefundReason, &p.PaidAt, &p.RefundedAt, &p.CreatedAt, &p.UpdatedAt,
			&customerID, &customerName, &customerEmail, &staffID, &staffName, &staffEmail); err != nil {
			return nil, err
		}
		if customerID.Valid {
			p.Customer = &models.CustomerProfile{ID: customerID.String, Name: customerName.String, Email: customerEmail.String}
		}
		if staffID.Valid {
			p.Staff = &models.BookingStaff{ID: staffID.String, Name: staffName.String, Email: staffEmail.String}
		}
		payments = append(payments, p)
	}
	return payments, rows.Err()
}

// The methods below are compare-and-set transitions: they return false when the row
// was not in the expected state, which makes webhooks and refunds safe to repeat.

// pending -> paid
func (r *PaymentRepository) MarkPaid(ctx context.Context, id, providerRef string) (bool, error) {
	return rowsChanged(r.db.ExecContext(ctx,
		`UPDATE payments SET status = 'paid', chapa_reference = $1, paid_at = $2, updated_at = $2
		 WHERE id = $3 AND status = 'pending'`, providerRef, time.Now(), id))
}

// pending -> failed
func (r *PaymentRepository) MarkFailed(ctx context.Context, id string) (bool, error) {
	return rowsChanged(r.db.ExecContext(ctx,
		`UPDATE payments SET status = 'failed', updated_at = $1 WHERE id = $2 AND status = 'pending'`,
		time.Now(), id))
}

// paid -> refund_pending
func (r *PaymentRepository) BeginRefund(ctx context.Context, id string) (bool, error) {
	return rowsChanged(r.db.ExecContext(ctx,
		`UPDATE payments SET status = 'refund_pending', updated_at = $1 WHERE id = $2 AND status = 'paid'`,
		time.Now(), id))
}

// refund_pending -> paid (provider rejected the refund)
func (r *PaymentRepository) AbortRefund(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE payments SET status = 'paid', updated_at = $1 WHERE id = $2 AND status = 'refund_pending'`,
		time.Now(), id)
	return err
}

// refund_pending -> refunded
func (r *PaymentRepository) CompleteRefund(ctx context.Context, id, refundRef, reason string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE payments
		 SET status = 'refunded', refund_reference = $1, refund_reason = $2, refunded_at = $3, updated_at = $3
		 WHERE id = $4 AND status = 'refund_pending'`,
		refundRef, reason, time.Now(), id)
	return err
}
