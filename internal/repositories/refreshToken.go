package repositories

import (
	"database/sql"

	"github.com/FiraBro/local-go/internal/models"
)

type RefreshTokenRepository struct {
	db *sql.DB
}

func NewRefreshTokenRepository(db *sql.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Save(token *models.RefreshToken) error {
	_, err := r.db.Exec(`INSERT INTO refresh_tokens (token, user_id, expires_at) VALUES (?, ?, ?)`,
		token.Token, token.UserID, token.ExpiresAt)
	return err
}

func (r *RefreshTokenRepository) Get(token string) (*models.RefreshToken, error) {
	row := r.db.QueryRow(`SELECT token, user_id, expires_at FROM refresh_tokens WHERE token=?`, token)
	var t models.RefreshToken
	if err := row.Scan(&t.Token, &t.UserID, &t.ExpiresAt); err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *RefreshTokenRepository) Delete(token string) error {
	_, err := r.db.Exec(`DELETE FROM refresh_tokens WHERE token=?`, token)
	return err
}

func (r *RefreshTokenRepository) DeleteByUser(userID string) error {
	_, err := r.db.Exec(`DELETE FROM refresh_tokens WHERE user_id=?`, userID)
	return err
}
