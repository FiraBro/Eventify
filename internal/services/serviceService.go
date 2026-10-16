package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
)

var (
	ErrServiceNotFound = errors.New("service not found")
	ErrInvalidInput    = errors.New("invalid input data")
)

const (
	defaultServiceDurationMinutes = 60
	minServiceDurationMinutes     = 5
	maxServiceDurationMinutes     = 24 * 60
)

// checkDuration validates the length of the slot a service occupies.
func checkDuration(minutes int) error {
	if minutes < minServiceDurationMinutes || minutes > maxServiceDurationMinutes {
		return fmt.Errorf("%w: duration_minutes must be between %d and %d",
			ErrInvalidInput, minServiceDurationMinutes, maxServiceDurationMinutes)
	}
	return nil
}

type ServiceService struct {
	repo *repositories.ServiceRepository
}

func NewServiceService(repo *repositories.ServiceRepository) *ServiceService {
	return &ServiceService{repo: repo}
}

func (s *ServiceService) Create(ctx context.Context, service *models.Service) error {
	// 1. Business Validation
	if service.Name == "" || service.Price < 0 {
		return fmt.Errorf("%w: name is required and price must be positive", ErrInvalidInput)
	}
	if service.DurationMinutes == 0 {
		service.DurationMinutes = defaultServiceDurationMinutes // field omitted by the client
	}
	if err := checkDuration(service.DurationMinutes); err != nil {
		return err
	}

	// 2. Call Repo
	return s.repo.Create(ctx, service)
}

func (s *ServiceService) GetAll(ctx context.Context) ([]models.Service, error) {
	return s.repo.GetAll(ctx)
}

func (s *ServiceService) GetByID(ctx context.Context, id string) (*models.Service, error) {
	service, err := s.repo.GetByIDs(ctx, id)
	if err != nil {
		return nil, err
	}
	if service == nil {
		return nil, ErrServiceNotFound
	}
	return service, nil
}

func (s *ServiceService) Update(ctx context.Context, id string, req *models.UpdateServiceRequest) (*models.Service, error) {
	// 1. Fetch current data from REPO
	existing, err := s.repo.GetByIDs(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrServiceNotFound
	}

	// 2. Merge only the fields that were provided
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Category != nil {
		existing.Category = *req.Category
	}
	if req.Price != nil {
		existing.Price = *req.Price
	}
	if req.DurationMinutes != nil {
		if err := checkDuration(*req.DurationMinutes); err != nil {
			return nil, err
		}
		existing.DurationMinutes = *req.DurationMinutes
	}

	// 3. Save to database via REPO
	if err := s.repo.Update(ctx, id, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

func (s *ServiceService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *ServiceService) GetCategories(ctx context.Context) ([]string, error) {
	return s.repo.GetCategories(ctx)
}
