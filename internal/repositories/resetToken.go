package repositories

import (
	"database/sql"

	"github.com/FiraBro/local-go/internal/models"
)

type ResetTokenRepository struct {
	db *sql.DB
}

func NewResetTokenRepository(db *sql.DB) *ResetTokenRepository {
	return &ResetTokenRepository{db: db}
}

func (r *ResetTokenRepository) Save(token *models.ResetToken) error {
	_, err := r.db.Exec(`INSERT INTO reset_tokens (email, otp, expires_at) VALUES (?, ?, ?)`, token.Email, token.OTP, token.ExpiresAt)
	return err
}

func (r *ResetTokenRepository) Get(email, otp string) (*models.ResetToken, error) {
	row := r.db.QueryRow(`SELECT email, otp, expires_at FROM reset_tokens WHERE email=? AND otp=?`, email, otp)
	var t models.ResetToken
	if err := row.Scan(&t.Email, &t.OTP, &t.ExpiresAt); err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *ResetTokenRepository) Delete(email string) error {
	_, err := r.db.Exec(`DELETE FROM reset_tokens WHERE email=?`, email)
	return err
}
