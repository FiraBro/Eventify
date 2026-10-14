package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/FiraBro/local-go/internal/models"
)

type SettingsRepository struct {
	db *sql.DB
}

func NewSettingsRepository(db *sql.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

const settingsColumns = `business_name, contact_email, contact_phone, currency, timezone,
	min_booking_notice_hours, max_booking_advance_days, cancellation_window_hours,
	COALESCE(updated_by::text, ''), updated_at`

// --------------------
// GET SETTINGS (creates the default row on first use)
// --------------------
func (r *SettingsRepository) Get(ctx context.Context) (*models.Settings, error) {
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING`); err != nil {
		return nil, err
	}

	var s models.Settings
	err := r.db.QueryRowContext(ctx, `SELECT `+settingsColumns+` FROM settings WHERE id = 1`).Scan(
		&s.BusinessName, &s.ContactEmail, &s.ContactPhone, &s.Currency, &s.Timezone,
		&s.MinBookingNoticeHours, &s.MaxBookingAdvanceDays, &s.CancellationWindowHours,
		&s.UpdatedBy, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// --------------------
// UPDATE SETTINGS
// --------------------
func (r *SettingsRepository) Update(ctx context.Context, s *models.Settings) error {
	query := `
		UPDATE settings
		SET business_name = $1, contact_email = $2, contact_phone = $3, currency = $4, timezone = $5,
		    min_booking_notice_hours = $6, max_booking_advance_days = $7, cancellation_window_hours = $8,
		    updated_by = NULLIF($9::text, '')::uuid, updated_at = $10
		WHERE id = 1
	`
	_, err := r.db.ExecContext(ctx, query,
		s.BusinessName, s.ContactEmail, s.ContactPhone, s.Currency, s.Timezone,
		s.MinBookingNoticeHours, s.MaxBookingAdvanceDays, s.CancellationWindowHours,
		s.UpdatedBy, time.Now(),
	)
	return err
}