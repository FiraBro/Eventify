package services

import (
	"context"
	"errors"
	"fmt"

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
    // 1. Check if staff exists
    existing, err := s.staffRepo.GetByID(ctx, id)
    if err != nil {
        return err
    }
    if existing == nil {
        return errors.New("staff member not found")
    }

    // 2. Update logic (this is simplified; in real code, you'd have an Update method in the repo)
    existing.Name = staff.Name
    existing.Email = staff.Email
    existing.Phone = staff.Phone
    existing.Role = staff.Role

    // Here you would call an Update method on the repository
    // For brevity, let's assume it's done directly
    return s.staffRepo.Create(ctx, existing) // Replace with actual update logic

}

func (s *StaffService) Delete(ctx context.Context, id string) error {
    return s.staffRepo.Delete(ctx, id)
}

// ADD THESE TO StaffService in internal/services/staff_service.go

func (s *StaffService) GetServices(ctx context.Context, staffID string) ([]models.Service, error) {
    // 1. Get the IDs from the junction table
    serviceIDs, err := s.staffRepo.GetServiceIDs(ctx, staffID)
    if err != nil {
        return nil, err
    }

    if len(serviceIDs) == 0 {
        return []models.Service{}, nil
    }

    // 2. Fetch the actual service details for those IDs
    // Note: You might need to add a "GetByMultipleIDs" method to your serviceRepo
    // or loop through GetByIDs for now.
    var services []models.Service
    for _, id := range serviceIDs {
        svc, err := s.serviceRepo.GetByIDs(ctx, id)
        if err == nil && svc != nil {
            services = append(services, *svc)
        }
    }

    return services, nil
}

func (s *StaffService) GetSchedule(ctx context.Context, staffID string) ([]map[string]string, error) {
    return s.staffRepo.GetSchedule(ctx, staffID)
}

// ---------- RELATIONSHIP LOGIC (The "Orchestrator" part) ----------

func (s *StaffService) AssignServices(ctx context.Context, staffID string, serviceIDs []string) error {
    // 1. Verify Staff exists
    staff, err := s.staffRepo.GetByID(ctx, staffID)
    if err != nil || staff == nil {
        return errors.New("cannot assign services: staff member not found")
    }

    // 2. (Optional but recommended) Verify all serviceIDs actually exist in the DB
    // This prevents foreign key violations or "ghost" assignments
    for _, sID := range serviceIDs {
        found, err := s.serviceRepo.GetByIDs(ctx, sID) // Assuming GetByIDs returns a single service or error
        if err != nil || found == nil {
            return fmt.Errorf("service ID %s does not exist", sID)
        }
    }

    return s.staffRepo.AssignServices(ctx, staffID, serviceIDs)
}

// ---------- SCHEDULE & HOLIDAYS ----------

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
    // Check if the date is in the past
    // time.Parse(...) etc.
    return s.staffRepo.AddHoliday(ctx, id, date, reason)
}