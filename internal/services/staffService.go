package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/FiraBro/local-go/internal/cacheaside"
	"github.com/FiraBro/local-go/internal/config"
	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/FiraBro/local-go/internal/utils"
	"golang.org/x/sync/singleflight"
)

type StaffInvitationMailer interface {
	SendStaffInvitation(to, staffName, businessName, acceptanceURL string, expiryHours int) error
}

var (
	ErrStaffAlreadyExists      = errors.New("staff email already exists")
	ErrInvitationEmailDelivery = errors.New("staff invitation email delivery failed")
)

type StaffService struct {
	staffRepo   *repositories.StaffRepository
	serviceRepo *repositories.ServiceRepository
	cache       cacheaside.Store
	mailer      StaffInvitationMailer
}

func NewStaffService(
	staffRepo *repositories.StaffRepository,
	serviceRepo *repositories.ServiceRepository,
	mailers ...StaffInvitationMailer,
) *StaffService {
	mailer := StaffInvitationMailer(utils.SMTPMailer{})
	if len(mailers) > 0 && mailers[0] != nil {
		mailer = mailers[0]
	}
	return &StaffService{
		staffRepo:   staffRepo,
		serviceRepo: serviceRepo,
		mailer:      mailer,
	}
}

// SetReadCache enables best-effort invalidation for availability-affecting writes.
func (s *StaffService) SetReadCache(store cacheaside.Store) { s.cache = store }

// ---------- STAFF CRUD ----------

func (s *StaffService) Create(ctx context.Context, staff *models.Staff) error {
	staff.Name = strings.TrimSpace(staff.Name)
	staff.Email = strings.ToLower(strings.TrimSpace(staff.Email))
	if staff.Name == "" || staff.Email == "" {
		return errors.New("staff name and email are required")
	}
	if config.AppConfig == nil {
		return errors.New("application configuration is unavailable")
	}
	rawToken, tokenHash, err := GenerateStaffInvitationToken()
	if err != nil {
		return fmt.Errorf("generate staff invitation: %w", err)
	}
	expiresAt := time.Now().Add(config.AppConfig.StaffInvitationExpiry)
	acceptanceURL, err := invitationLink(config.AppConfig.StaffInvitationURL, rawToken)
	if err != nil {
		return err
	}

	err = mutateAndInvalidate(
		ctx,
		s.cache,
		[]string{availabilityCacheNamespace},
		func(ctx context.Context) error {
			return s.staffRepo.CreateInvitedStaff(ctx, staff, tokenHash, expiresAt)
		},
	)
	if errors.Is(err, repositories.ErrStaffEmailAlreadyExists) {
		return ErrStaffAlreadyExists
	}
	if err != nil {
		return err
	}

	expiryHours := int(config.AppConfig.StaffInvitationExpiry / time.Hour)
	if err := s.mailer.SendStaffInvitation(
		staff.Email,
		staff.Name,
		config.AppConfig.BusinessName,
		acceptanceURL,
		expiryHours,
	); err != nil {
		slog.ErrorContext(ctx, "staff invitation email delivery failed", "staff_id", staff.ID)
		return ErrInvitationEmailDelivery
	}
	return nil
}

func invitationLink(baseURL, token string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return "", errors.New("staff invitation URL must be an absolute HTTP or HTTPS URL")
	}
	if parsed.Scheme != "https" && (parsed.Scheme != "http" || config.AppConfig == nil || config.AppConfig.AppEnv == "production") {
		return "", errors.New("staff invitation URL must be an absolute HTTP or HTTPS URL")
	}
	query := parsed.Query()
	query.Set("token", token)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

// ResendInvitation rotates the token only for pending staff. Ineligible and
// unknown addresses are intentionally indistinguishable to the HTTP caller.
func (s *StaffService) ResendInvitation(ctx context.Context, email string) error {
	if config.AppConfig == nil {
		return errors.New("application configuration is unavailable")
	}
	rawToken, tokenHash, err := GenerateStaffInvitationToken()
	if err != nil {
		return fmt.Errorf("generate staff invitation: %w", err)
	}
	acceptanceURL, err := invitationLink(config.AppConfig.StaffInvitationURL, rawToken)
	if err != nil {
		return err
	}
	expiresAt := time.Now().Add(config.AppConfig.StaffInvitationExpiry)
	invitation, eligible, err := s.staffRepo.ReplacePendingInvitationByEmail(ctx, email, tokenHash, expiresAt)
	if err != nil || !eligible {
		return err
	}

	expiryHours := int(config.AppConfig.StaffInvitationExpiry / time.Hour)
	if err := s.mailer.SendStaffInvitation(
		invitation.Email,
		invitation.Name,
		config.AppConfig.BusinessName,
		acceptanceURL,
		expiryHours,
	); err != nil {
		// Do not log the email address, token, link, or adapter error text.
		slog.ErrorContext(ctx, "staff invitation email delivery failed", "staff_id", invitation.StaffID)
		return ErrInvitationEmailDelivery
	}
	return nil
}

func (s *StaffService) GetAll(ctx context.Context) ([]models.Staff, error) {
	return s.staffRepo.GetAll(ctx)
}

func (s *StaffService) GetByID(ctx context.Context, id string) (*models.Staff, error) {
	return s.staffRepo.GetByID(ctx, id)
}

func (s *StaffService) Update(ctx context.Context, id string, staff *models.Staff) error {
	existing, err := s.staffRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("staff member not found")
	}

	// Use the actual Update method in repo, not Create
	return s.staffRepo.Update(ctx, id, staff)
}

func (s *StaffService) Delete(ctx context.Context, id string) error {
	return mutateAndInvalidate(ctx, s.cache, []string{availabilityCacheNamespace}, func(ctx context.Context) error {
		return s.staffRepo.Delete(ctx, id)
	})
}

// ---------- SERVICES RELATIONSHIP ----------

func (s *StaffService) GetServices(ctx context.Context, staffID string) ([]models.Service, error) {
	serviceIDs, err := s.staffRepo.GetServiceIDs(ctx, staffID)
	if err != nil {
		return nil, err
	}

	if len(serviceIDs) == 0 {
		return []models.Service{}, nil
	}

	var services []models.Service
	for _, id := range serviceIDs {
		// Use your serviceRepo to get full details
		svc, err := s.serviceRepo.GetByIDs(ctx, id)
		if err == nil && svc != nil {
			services = append(services, *svc)
		}
	}
	return services, nil
}

func (s *StaffService) GetStaffByService(ctx context.Context, serviceID string) ([]models.Staff, error) {
	return s.staffRepo.GetStaffByService(ctx, serviceID)
}

func (s *StaffService) AssignServices(ctx context.Context, staffID string, serviceIDs []string) error {
	staff, err := s.staffRepo.GetByID(ctx, staffID)
	if err != nil || staff == nil {
		return errors.New("cannot assign services: staff member not found")
	}

	for _, sID := range serviceIDs {
		found, err := s.serviceRepo.GetByIDs(ctx, sID)
		if err != nil || found == nil {
			return fmt.Errorf("service ID %s does not exist", sID)
		}
	}
	return mutateAndInvalidate(ctx, s.cache, []string{availabilityCacheNamespace}, func(ctx context.Context) error {
		return s.staffRepo.AssignServices(ctx, staffID, serviceIDs)
	})
}

// ---------- SCHEDULE & HOLIDAYS ----------

func (s *StaffService) GetSchedule(ctx context.Context, staffID string) ([]map[string]string, error) {
	// FIXED: Your repo's GetAvailabilityData returns ([]map, []string, error)
	// We only want the first return value here.
	schedule, _, err := s.staffRepo.GetAvailabilityData(ctx, staffID)
	return schedule, err
}

func (s *StaffService) SetSchedule(ctx context.Context, id string, entries []map[string]string) error {

	// Business logic: Ensure hours are valid (e.g., 09:00 - 17:00)

	for _, entry := range entries {

		if entry["day"] == "" || entry["start"] == "" || entry["end"] == "" {

			return errors.New("invalid schedule format: day, start, and end are required")

		}

	}

	return mutateAndInvalidate(ctx, s.cache, []string{availabilityCacheNamespace}, func(ctx context.Context) error {
		return s.staffRepo.SetSchedule(ctx, id, entries)
	})

}

func (s *StaffService) AddHoliday(ctx context.Context, id, date, reason string) error {
	return mutateAndInvalidate(ctx, s.cache, []string{availabilityCacheNamespace}, func(ctx context.Context) error {
		return s.staffRepo.AddHoliday(ctx, id, date, reason)
	})
}

// ---------- AVAILABILITY SERVICE ----------

type AvailabilityService struct {
	repo       *repositories.StaffRepository
	cache      cacheaside.Store
	cacheGroup singleflight.Group
}

func NewAvailabilityService(repo *repositories.StaffRepository) *AvailabilityService {
	return &AvailabilityService{repo: repo}
}

func (s *AvailabilityService) GetStaffSlots(ctx context.Context, staffID, date string) ([]string, error) {
	schedule, holidays, err := s.repo.GetAvailabilityData(ctx, staffID)
	if err != nil {
		return nil, err
	}

	for _, h := range holidays {
		if h == date {
			return []string{}, nil
		}
	}

	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, errors.New("invalid date format, use YYYY-MM-DD")
	}
	dayName := t.Weekday().String()

	var slots []string
	for _, sch := range schedule {
		if sch["day"] == dayName {
			slots = s.generateTimeSlots(sch["start"], sch["end"], 30)
		}
	}
	return slots, nil
}

func (s *AvailabilityService) generateTimeSlots(start, end string, interval int) []string {
	slots := []string{}
	curr, errStart := time.Parse("15:04", start)
	finish, errEnd := time.Parse("15:04", end)

	if errStart != nil || errEnd != nil {
		return slots
	}

	for curr.Before(finish) {
		slots = append(slots, curr.Format("15:04"))
		curr = curr.Add(time.Minute * time.Duration(interval))
	}
	return slots
}
