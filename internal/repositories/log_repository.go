package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/google/uuid"
)

type LogRepository struct {
	db *sql.DB
}

func NewLogRepository(db *sql.DB) *LogRepository {
	return &LogRepository{db: db}
}

// --------------------
// CREATE LOG ENTRY
// --------------------
func (r *LogRepository) Create(ctx context.Context, l *models.ActivityLog) error {
	l.ID = uuid.New().String()
	l.CreatedAt = time.Now()

	query := `
		INSERT INTO activity_logs (id, user_id, role, method, path, route, status_code, ip, duration_ms, details, created_at)
		VALUES ($1, NULLIF($2::text,'')::uuid, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.ExecContext(ctx, query,
		l.ID, l.UserID, l.Role, l.Method, l.Path, l.Route, l.StatusCode, l.IP, l.DurationMs,
		string(l.Details), l.CreatedAt,
	)
	return err
}

// Every filter is optional: a NULL parameter switches its condition off. The SQL below is a
// constant, so no value is ever formatted into it; all user input travels as bound parameters.
const logWhere = `
	WHERE ($1::uuid IS NULL OR user_id = $1)
	  AND ($2::text IS NULL OR method = $2)
	  AND ($3::int IS NULL OR status_code = $3)
	  AND ($4::text IS NULL OR path ILIKE $4 OR route ILIKE $4)
	  AND ($5::timestamptz IS NULL OR created_at >= $5)
	  AND ($6::timestamptz IS NULL OR created_at < $6)`

const logCountSQL = `SELECT COUNT(*) FROM activity_logs` + logWhere

const logListSQL = `
	SELECT id, COALESCE(user_id::text, ''), role, method, path, route, status_code, ip, duration_ms, details, created_at
	FROM activity_logs` + logWhere + `
	ORDER BY created_at DESC
	LIMIT $7 OFFSET $8`

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullIfZero(n int) any {
	if n == 0 {
		return nil
	}
	return n
}

func nullTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

// --------------------
// LIST LOGS (filtered, newest first)
// --------------------
func (r *LogRepository) List(ctx context.Context, f *models.LogFilter) ([]models.ActivityLog, int, error) {
	filters := []any{
		nullIfEmpty(f.UserID),
		nullIfEmpty(f.Method),
		nullIfZero(f.Status),
		nullIfEmpty(likePattern(f.Q)),
		nullTime(f.From),
		nullTime(f.To),
	}

	var total int
	if err := r.db.QueryRowContext(ctx, logCountSQL, filters...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listArgs := append(append([]any{}, filters...), f.Limit, f.Offset)
	rows, err := r.db.QueryContext(ctx, logListSQL, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		_ = rows.Close()
	}()

	items := []models.ActivityLog{}
	for rows.Next() {
		var l models.ActivityLog
		var details string
		if err := rows.Scan(&l.ID, &l.UserID, &l.Role, &l.Method, &l.Path, &l.Route,
			&l.StatusCode, &l.IP, &l.DurationMs, &details, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		if details != "" {
			l.Details = json.RawMessage(details)
		}
		items = append(items, l)
	}
	return items, total, rows.Err()
}

// likePattern wraps an already-escaped search term for ILIKE; an empty term means "no filter".
func likePattern(q string) string {
	if q == "" {
		return ""
	}
	return "%" + q + "%"
}
