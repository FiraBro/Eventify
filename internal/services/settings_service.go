package services

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
	_ "time/tzdata" // embed the timezone database so validation works in minimal Docker images

	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/google/uuid"
)

var (
	ErrSettingsForbidden    = errors.New("only admins can change settings")
	ErrSettingsInvalidInput = errors.New("invalid settings data")
)

var (
	currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)
	phonePattern    = regexp.MustCompile(`^[0-9+()\-\s]{0,30}$`)
)

func settingsInvalid(msg string) error {
	return fmt.Errorf("%w: %s", ErrSettingsInvalidInput, msg)
}

type SettingsService struct {
	repo *repositories.SettingsRepository
}

func NewSettingsService(repo *repositories.SettingsRepository) *SettingsService {
	return &SettingsService{repo: repo}
}

func (s *SettingsService) Get(ctx context.Context) (*models.Settings, error) {
	return s.repo.Get(ctx)
}

// Update validates and applies only the provided fields. It returns what really changed
// (old -> new) so the handler can attach it to the activity log.
func (s *SettingsService) Update(ctx context.Context, userID, role string, req *models.UpdateSettingsRequest) (*models.Settings, models.SettingsChanges, error) {
	// 1. Permission + input checks
	if role != roleAdmin {
		return nil, nil, ErrSettingsForbidden
	}
	if req.BusinessName == nil && req.ContactEmail == nil && req.ContactPhone == nil &&
		req.Currency == nil && req.Timezone == nil && req.MinBookingNoticeHours == nil &&
		req.MaxBookingAdvanceDays == nil && req.CancellationWindowHours == nil {
		return nil, nil, settingsInvalid("at least one field is required")
	}

	// 2. Fetch current values from REPO
	current, err := s.repo.Get(ctx)
	if err != nil {
		return nil, nil, err
	}
	updated := *current

	// 3. Merge + validate only the fields that were provided
	if req.BusinessName != nil {
		v := strings.TrimSpace(*req.BusinessName)
		if v == "" || len(v) > 100 {
			return nil, nil, settingsInvalid("business_name must be 1-100 characters")
		}
		updated.BusinessName = v
	}
	if req.ContactEmail != nil {
		v := strings.TrimSpace(*req.ContactEmail)
		if v != "" {
			addr, err := mail.ParseAddress(v)
			if err != nil || addr.Address != v || len(v) > 150 {
				return nil, nil, settingsInvalid("contact_email is not a valid email address")
			}
		}
		updated.ContactEmail = v
	}
	if req.ContactPhone != nil {
		v := strings.TrimSpace(*req.ContactPhone)
		if !phonePattern.MatchString(v) {
			return nil, nil, settingsInvalid("contact_phone may only contain digits, + ( ) - and spaces (max 30)")
		}
		updated.ContactPhone = v
	}
	if req.Currency != nil {
		v := strings.ToUpper(strings.TrimSpace(*req.Currency))
		if !currencyPattern.MatchString(v) {
			return nil, nil, settingsInvalid("currency must be a 3-letter code such as ETB or USD")
		}
		updated.Currency = v
	}
	if req.Timezone != nil {
		v := strings.TrimSpace(*req.Timezone)
		if v == "" || v == "Local" {
			return nil, nil, settingsInvalid("timezone must be an IANA name such as UTC or Africa/Addis_Ababa")
		}
		if _, err := time.LoadLocation(v); err != nil {
			return nil, nil, settingsInvalid("unknown timezone")
		}
		updated.Timezone = v
	}
	if req.MinBookingNoticeHours != nil {
		if *req.MinBookingNoticeHours < 0 || *req.MinBookingNoticeHours > 720 {
			return nil, nil, settingsInvalid("min_booking_notice_hours must be between 0 and 720")
		}
		updated.MinBookingNoticeHours = *req.MinBookingNoticeHours
	}
	if req.MaxBookingAdvanceDays != nil {
		if *req.MaxBookingAdvanceDays < 1 || *req.MaxBookingAdvanceDays > 730 {
			return nil, nil, settingsInvalid("max_booking_advance_days must be between 1 and 730")
		}
		updated.MaxBookingAdvanceDays = *req.MaxBookingAdvanceDays
	}
	if req.CancellationWindowHours != nil {
		if *req.CancellationWindowHours < 0 || *req.CancellationWindowHours > 720 {
			return nil, nil, settingsInvalid("cancellation_window_hours must be between 0 and 720")
		}
		updated.CancellationWindowHours = *req.CancellationWindowHours
	}

	// the minimum notice must fit inside the booking horizon
	if updated.MinBookingNoticeHours >= updated.MaxBookingAdvanceDays*24 {
		return nil, nil, settingsInvalid("min_booking_notice_hours must be shorter than max_booking_advance_days")
	}

	// 4. Work out what really changed
	changes := models.SettingsChanges{}
	record := func(field string, oldVal, newVal any) {
		if oldVal != newVal {
			changes[field] = models.FieldChange{Old: oldVal, New: newVal}
		}
	}
	record("business_name", current.BusinessName, updated.BusinessName)
	record("contact_email", current.ContactEmail, updated.ContactEmail)
	record("contact_phone", current.ContactPhone, updated.ContactPhone)
	record("currency", current.Currency, updated.Currency)
	record("timezone", current.Timezone, updated.Timezone)
	record("min_booking_notice_hours", current.MinBookingNoticeHours, updated.MinBookingNoticeHours)
	record("max_booking_advance_days", current.MaxBookingAdvanceDays, updated.MaxBookingAdvanceDays)
	record("cancellation_window_hours", current.CancellationWindowHours, updated.CancellationWindowHours)

	if len(changes) == 0 {
		return current, changes, nil // nothing to save
	}

	// 5. Save via REPO
	if _, err := uuid.Parse(userID); err != nil {
		userID = ""
	}
	updated.UpdatedBy = userID
	updated.UpdatedAt = time.Now()
	if err := s.repo.Update(ctx, &updated); err != nil {
		return nil, nil, err
	}
	return &updated, changes, nil
}
