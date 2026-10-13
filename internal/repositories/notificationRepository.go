package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/google/uuid"
)

type NotificationRepository struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

const notificationColumns = `id, user_id, title, message, type, reference_id, is_read, read_at, created_at`

func scanNotification(scan func(dest ...any) error) (*models.Notification, error) {
	var n models.Notification
	err := scan(&n.ID, &n.UserID, &n.Title, &n.Message, &n.Type, &n.ReferenceID, &n.IsRead, &n.ReadAt, &n.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // not found
		}
		return nil, err
	}
	return &n, nil
}

// --------------------
// CREATE NOTIFICATION
// --------------------
func (r *NotificationRepository) Create(ctx context.Context, n *models.Notification) error {
	n.ID = uuid.New().String()
	n.CreatedAt = time.Now()
	n.IsRead = false

	query := `
		INSERT INTO notifications (id, user_id, title, message, type, reference_id, is_read, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`
	_, err := r.db.ExecContext(ctx, query,
		n.ID, n.UserID, n.Title, n.Message, n.Type, n.ReferenceID, n.IsRead, n.CreatedAt,
	)
	return err
}

// --------------------
// GET NOTIFICATIONS OF A USER
// --------------------
func (r *NotificationRepository) GetByUserID(ctx context.Context, userID string, unreadOnly bool, limit, offset int) ([]models.Notification, error) {
	query := `SELECT ` + notificationColumns + ` FROM notifications WHERE user_id = $1`
	if unreadOnly {
		query += ` AND is_read = false`
	}
	query += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []models.Notification{}
	for rows.Next() {
		n, err := scanNotification(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, *n)
	}
	return items, rows.Err()
}

// --------------------
// COUNT UNREAD
// --------------------
func (r *NotificationRepository) CountUnread(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`, userID).Scan(&count)
	return count, err
}

// --------------------
// MARK AS READ
// --------------------
// Only matches a row owned by userID. Safe to call twice: the first read_at is kept.
// Returns nil, nil when the notification does not exist or belongs to someone else.
func (r *NotificationRepository) MarkRead(ctx context.Context, id, userID string) (*models.Notification, error) {
	row := r.db.QueryRowContext(ctx,
		`UPDATE notifications
		 SET is_read = true, read_at = COALESCE(read_at, $1)
		 WHERE id = $2 AND user_id = $3
		 RETURNING `+notificationColumns,
		time.Now(), id, userID)
	return scanNotification(row.Scan)
}

// --------------------
// CHECK RECIPIENT EXISTS
// --------------------
func (r *NotificationRepository) UserExists(ctx context.Context, userID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&exists)
	return exists, err
}