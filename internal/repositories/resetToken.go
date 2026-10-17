package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/FiraBro/local-go/internal/models"
)

type ResetTokenRepository struct {
	db *sql.DB
}

func NewResetTokenRepository(db *sql.DB) *ResetTokenRepository {
	return &ResetTokenRepository{db: db}
}

// Save stores or replaces a secure hashed password reset token state.
func (r *ResetTokenRepository) Save(rt *models.ResetToken) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		INSERT INTO reset_tokens (email, token_hash, expires_at, is_used)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (email) DO UPDATE
		SET token_hash = EXCLUDED.token_hash, expires_at = EXCLUDED.expires_at, is_used = EXCLUDED.is_used
	`
	_, err := r.db.ExecContext(ctx, query, rt.Email, rt.TokenHash, rt.ExpiresAt, rt.IsUsed)
	return err
}

// GetAndConsumeAtomic atomically fetches and invalidates an active token in a single operation.
// This completely neutralizes concurrent read-then-write race conditions.
func (r *ResetTokenRepository) GetAndConsumeAtomic(email, hash string) (*models.ResetToken, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
		UPDATE reset_tokens
		SET is_used = TRUE
		WHERE email = $1 AND token_hash = $2 AND is_used = FALSE AND expires_at > NOW()
		RETURNING email, token_hash, expires_at, is_used
	`
	row := r.db.QueryRowContext(ctx, query, email, hash)

	var rt models.ResetToken
	err := row.Scan(&rt.Email, &rt.TokenHash, &rt.ExpiresAt, &rt.IsUsed)
	if err != nil {
		return nil, err
	}
	return &rt, nil
}
func (r *ResetTokenRepository) Delete(email string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Fixed: Removed the undefined "query" argument reference here
	_, err := r.db.ExecContext(ctx, `DELETE FROM reset_tokens WHERE email = $1`, email)
	return err
}
