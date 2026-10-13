package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
	"github.com/google/uuid"
)

var (
	ErrNotificationNotFound     = errors.New("notification not found")
	ErrNotificationForbidden    = errors.New("only admins can send notifications")
	ErrNotificationInvalidInput = errors.New("invalid notification data")
	ErrNotificationUserNotFound = errors.New("recipient user not found")
)

const (
	maxNotificationTitle   = 150
	maxNotificationMessage = 1000
)

var notificationTypes = map[string]bool{
	"info": true, "booking": true, "payment": true, "system": true,
}

type NotificationService struct {
	repo *repositories.NotificationRepository
}

func NewNotificationService(repo *repositories.NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

// --------------------
// LIST MY NOTIFICATIONS
// --------------------
func (s *NotificationService) GetMine(ctx context.Context, userID string, unreadOnly bool, page, limit int) ([]models.Notification, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	items, err := s.repo.GetByUserID(ctx, userID, unreadOnly, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}
	unread, err := s.repo.CountUnread(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	return items, unread, nil
}

// --------------------
// SEND (admin only; route middleware guards it, service re-checks)
// --------------------
func (s *NotificationService) Send(ctx context.Context, role string, req *models.SendNotificationRequest) (*models.Notification, error) {
	if role != roleAdmin {
		return nil, ErrNotificationForbidden
	}
	return s.Notify(ctx, req.UserID, req.Type, req.Title, req.Message, req.ReferenceID)
}

// Notify creates a notification. Other services (bookings, payments) can call this
// directly to notify a user about an event.
func (s *NotificationService) Notify(ctx context.Context, userID, nType, title, message, referenceID string) (*models.Notification, error) {
	// 1. Business Validation
	title = strings.TrimSpace(title)
	message = strings.TrimSpace(message)
	nType = strings.ToLower(strings.TrimSpace(nType))
	if nType == "" {
		nType = "info"
	}

	if _, err := uuid.Parse(userID); err != nil {
		return nil, fmt.Errorf("%w: user_id must be a valid id", ErrNotificationInvalidInput)
	}
	if title == "" || message == "" {
		return nil, fmt.Errorf("%w: title and message are required", ErrNotificationInvalidInput)
	}
	if len(title) > maxNotificationTitle || len(message) > maxNotificationMessage {
		return nil, fmt.Errorf("%w: title max %d and message max %d characters",
			ErrNotificationInvalidInput, maxNotificationTitle, maxNotificationMessage)
	}
	if !notificationTypes[nType] {
		return nil, fmt.Errorf("%w: type must be info, booking, payment or system", ErrNotificationInvalidInput)
	}

	// 2. Recipient must exist
	exists, err := s.repo.UserExists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotificationUserNotFound
	}

	// 3. Call Repo
	n := &models.Notification{
		UserID:      userID,
		Title:       title,
		Message:     message,
		Type:        nType,
		ReferenceID: strings.TrimSpace(referenceID),
	}
	if err := s.repo.Create(ctx, n); err != nil {
		return nil, err
	}
	return n, nil
}

// --------------------
// MARK AS READ (owner only)
// --------------------
func (s *NotificationService) MarkRead(ctx context.Context, id, userID string) (*models.Notification, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrNotificationNotFound
	}

	// the repo only matches rows owned by userID, so other people's ids look like "not found"
	n, err := s.repo.MarkRead(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if n == nil {
		return nil, ErrNotificationNotFound
	}
	return n, nil
}