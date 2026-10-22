package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/FiraBro/local-go/internal/cacheaside"
	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
	"golang.org/x/sync/singleflight"
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

// ============================================================================
// SERVICE SERVICE
// ============================================================================

type ServiceService struct {
	repo       *repositories.ServiceRepository
	cache      cacheaside.Store
	cacheGroup singleflight.Group
}

// SetReadCache enables best-effort caching for the services-list read path.
func (s *ServiceService) SetReadCache(store cacheaside.Store) { s.cache = store }

func servicesListCacheKey() string {
	sum := sha256.Sum256([]byte("all-services-v1"))
	return servicesListCacheNamespace + ":" + hex.EncodeToString(sum[:])
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

	// Invalidate only after the repository confirms the write succeeded.
	return mutateAndInvalidate(ctx, s.cache, []string{servicesListCacheNamespace, availabilityCacheNamespace}, func(ctx context.Context) error {
		return s.repo.Create(ctx, service)
	})
}

func (s *ServiceService) GetAll(ctx context.Context) ([]models.Service, error) {
	return cacheaside.LoadVersioned(ctx, s.cache, &s.cacheGroup, servicesListCacheNamespace, strings.TrimPrefix(servicesListCacheKey(), servicesListCacheNamespace+":"), cacheaside.ServicesTTL, s.repo.GetAll)
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

	// 3. Save, then invalidate the shared list and availability namespaces.
	return cacheaside.Mutate(ctx, s.cache, []string{servicesListCacheNamespace, availabilityCacheNamespace}, func(ctx context.Context) (*models.Service, error) {
		if err := s.repo.Update(ctx, id, existing); err != nil {
			return nil, err
		}
		return existing, nil
	})
}

func (s *ServiceService) Delete(ctx context.Context, id string) error {
	return mutateAndInvalidate(ctx, s.cache, []string{servicesListCacheNamespace, availabilityCacheNamespace}, func(ctx context.Context) error {
		return s.repo.Delete(ctx, id)
	})
}

func (s *ServiceService) GetCategories(ctx context.Context) ([]string, error) {
	return s.repo.GetCategories(ctx)
}

// SetReadCache enables short-lived caching for service availability only.
func (s *AvailabilityService) SetReadCache(store cacheaside.Store) {
	s.cache = store
}

// availabilityCacheKey creates a cryptographically secure hash representing specific service lookup contexts.
func availabilityCacheKey(serviceID, date string) string {
	sum := sha256.Sum256([]byte("service=" + serviceID + "\x00date=" + date))
	return availabilityCacheNamespace + ":" + hex.EncodeToString(sum[:])
}

// GetServiceSlots checks cache layers before executing the heavy query resolver function.
func (s *AvailabilityService) GetServiceSlots(ctx context.Context, serviceID, date string) ([]string, error) {
	return cacheaside.LoadVersioned(ctx, s.cache, &s.cacheGroup, availabilityCacheNamespace, strings.TrimPrefix(availabilityCacheKey(serviceID, date), availabilityCacheNamespace+":"), cacheaside.AvailabilityTTL, func(loadCtx context.Context) ([]string, error) {
		staffList, err := s.repo.GetStaffByService(loadCtx, serviceID)
		if err != nil {
			return nil, err
		}
		unique := make(map[string]struct{})
		for _, staff := range staffList {
			slots, slotErr := s.GetStaffSlots(loadCtx, staff.ID, date)
			if slotErr != nil {
				continue
			} // preserve existing partial-result semantics
			for _, slot := range slots {
				unique[slot] = struct{}{}
			}
		}
		result := make([]string, 0, len(unique))
		for slot := range unique {
			result = append(result, slot)
		}
		sort.Strings(result)
		return result, nil
	})
}
