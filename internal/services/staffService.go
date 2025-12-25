package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
)

type StaffService struct {
	staffRepo   *repositories.StaffRepository
	serviceRepo *repositories.ServiceRepository
}

func NewStaffService(
	staffRepo *repositories.StaffRepository,
	serviceRepo *repositories.ServiceRepository,
) *StaffService {
	return &StaffService{
		staffRepo:   staffRepo,
		serviceRepo: serviceRepo,
	}
}

// ---------- STAFF CRUD ----------

func (s *StaffService) Create(ctx context.Context, staff *models.Staff) error {
	if staff.Name == "" || staff.Email == "" {
		return errors.New("staff name and email are required")
	}
	return s.staffRepo.Create(ctx, staff)
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
	return s.staffRepo.Delete(ctx, id)
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
	return s.staffRepo.AssignServices(ctx, staffID, serviceIDs)
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

return s.staffRepo.SetSchedule(ctx, id, entries)

}

func (s *StaffService) AddHoliday(ctx context.Context, id, date, reason string) error {
	return s.staffRepo.AddHoliday(ctx, id, date, reason)
}

// ---------- AVAILABILITY SERVICE ----------

type AvailabilityService struct {
	repo *repositories.StaffRepository
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