package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
)

var (
	ErrBookingNotFound     = errors.New("booking not found")
	ErrBookingForbidden    = errors.New("you do not have access to this booking")
	ErrBookingNotEditable  = errors.New("booking can no longer be modified")
	ErrInvalidTransition   = errors.New("invalid status transition")
	ErrBookingInvalidInput = errors.New("invalid booking data")
)

const roleAdmin = "admin"

const (
	bookingPending   = "pending"
	bookingConfirmed = "confirmed"
	bookingCancelled = "cancelled"
	bookingCompleted = "completed"
)

func validStatus(st string) bool {
	switch st {
	case bookingPending, bookingConfirmed, bookingCancelled, bookingCompleted:
		return true
	}
	return false
}

var statusTransitions = map[string][]string{
	bookingPending:   {bookingConfirmed, bookingCancelled},
	bookingConfirmed: {bookingCompleted, bookingCancelled},
}

func canTransition(from, to string) bool {
	for _, s := range statusTransitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

type BookingService struct {
	repo        *repositories.BookingRepository
	serviceRepo *repositories.ServiceRepository
}

func NewBookingService(repo *repositories.BookingRepository, serviceRepo *repositories.ServiceRepository) *BookingService {
	return &BookingService{repo: repo, serviceRepo: serviceRepo}
}

func (s *BookingService) Create(ctx context.Context, userID string, booking *models.Booking) error {
	// 1. Business Validation
	if booking.ServiceID == "" {
		return fmt.Errorf("%w: service_id is required", ErrBookingInvalidInput)
	}
	if !booking.ScheduledAt.After(time.Now()) {
		return fmt.Errorf("%w: scheduled_at must be in the future", ErrBookingInvalidInput)
	}

	// 2. The service being booked must exist
	svc, err := s.serviceRepo.GetByIDs(ctx, booking.ServiceID)
	if err != nil {
		return err
	}
	if svc == nil {
		return ErrServiceNotFound
	}

	// 3. Never trust client-provided ownership/status
	booking.UserID = userID
	booking.Status = bookingPending

	// 4. Call Repo
	return s.repo.Create(ctx, booking)
}

// Admin only (route middleware guards it, service re-checks)
func (s *BookingService) GetAll(ctx context.Context, role, status string) ([]models.Booking, error) {
	if role != roleAdmin {
		return nil, ErrBookingForbidden
	}
	return s.repo.GetAll(ctx, status)
}

func (s *BookingService) GetMine(ctx context.Context, userID string) ([]models.Booking, error) {
	return s.repo.GetByUserID(ctx, userID)
}

func (s *BookingService) GetByID(ctx context.Context, id, userID, role string) (*models.Booking, error) {
	booking, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if booking == nil {
		return nil, ErrBookingNotFound
	}
	if role != roleAdmin && booking.UserID != userID {
		return nil, ErrBookingForbidden
	}
	return booking, nil
}

func (s *BookingService) Update(ctx context.Context, id, userID, role string, req *models.UpdateBookingRequest) (*models.Booking, error) {
	// 1. Fetch + ownership check
	existing, err := s.GetByID(ctx, id, userID, role)
	if err != nil {
		return nil, err
	}

	// 2. Editing rules
	if role != roleAdmin && existing.Status != bookingPending {
		return nil, ErrBookingNotEditable
	}
	if existing.Status == bookingCancelled || existing.Status == bookingCompleted {
		return nil, ErrBookingNotEditable
	}

	// 3. Merge only provided fields
	if req.ScheduledAt != nil {
		if !req.ScheduledAt.After(time.Now()) {
			return nil, fmt.Errorf("%w: scheduled_at must be in the future", ErrBookingInvalidInput)
		}
		existing.ScheduledAt = *req.ScheduledAt
	}
	if req.Notes != nil {
		existing.Notes = *req.Notes
	}

	// 4. Save
	if err := s.repo.Update(ctx, id, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *BookingService) Delete(ctx context.Context, id, userID, role string) error {
	existing, err := s.GetByID(ctx, id, userID, role)
	if err != nil {
		return err
	}
	if role != roleAdmin && existing.Status != bookingPending {
		return ErrBookingNotEditable
	}
	return s.repo.Delete(ctx, id)
}

func (s *BookingService) UpdateStatus(ctx context.Context, id, userID, role, newStatus string) (*models.Booking, error) {
	existing, err := s.GetByID(ctx, id, userID, role)
	if err != nil {
		return nil, err
	}

	if !validStatus(newStatus) {
		return nil, fmt.Errorf("%w: status must be pending, confirmed, cancelled or completed", ErrBookingInvalidInput)
	}

	// customers can only cancel their own booking
	if role != roleAdmin && newStatus != bookingCancelled {
		return nil, ErrBookingForbidden
	}
	if !canTransition(existing.Status, newStatus) {
		return nil, ErrInvalidTransition
	}

	if err := s.repo.UpdateStatus(ctx, id, newStatus); err != nil {
		return nil, err
	}
	existing.Status = newStatus
	existing.UpdatedAt = time.Now()
	return existing, nil
}