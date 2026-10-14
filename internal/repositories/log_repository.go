package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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

// --------------------
// LIST LOGS (filtered, newest first)
// --------------------
func (r *LogRepository) List(ctx context.Context, f *models.LogFilter) ([]models.ActivityLog, int, error) {
	where := []string{"1=1"}
	args := []any{}
	add := func(cond string, v any) {
		args = append(args, v)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}

	if f.UserID != "" {
		add("user_id = $%d", f.UserID)
	}
	if f.Method != "" {
		add("method = $%d", f.Method)
	}
	if f.Status != 0 {
		add("status_code = $%d", f.Status)
	}
	if f.Q != "" {
		add(`(path ILIKE $%[1]d OR route ILIKE $%[1]d)`, "%"+f.Q+"%")
	}
	if f.From != nil {
		add("created_at >= $%d", *f.From)
	}
	if f.To != nil {
		add("created_at < $%d", *f.To)
	}
	clause := strings.Join(where, " AND ")

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM activity_logs WHERE `+clause, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.Limit, f.Offset)
	query := fmt.Sprintf(`
		SELECT id, COALESCE(user_id::text, ''), role, method, path, route, status_code, ip, duration_ms, details, created_at
		FROM activity_logs
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, clause, len(args)-1, len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

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