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

func (r *RefreshTokenRepository) Get(token string) (*models.RefreshToken, error) {
	row := r.db.QueryRow(`
		SELECT token, user_id, expires_at, is_used, parent_tag
		FROM refresh_tokens
		WHERE token = $1
	`, token)

	var rt models.RefreshToken
	err := row.Scan(&rt.Token, &rt.UserID, &rt.ExpiresAt, &rt.IsUsed, &rt.ParentTag)
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

func (r *RefreshTokenRepository) Save(rt *models.RefreshToken) error {
	_, err := r.db.Exec(`
		INSERT INTO refresh_tokens (token, user_id, expires_at, is_used, parent_tag)
		VALUES ($1, $2, $3, $4, $5)
	`, rt.Token, rt.UserID, rt.ExpiresAt, rt.IsUsed, rt.ParentTag)
	return err
}

func (r *RefreshTokenRepository) Update(rt *models.RefreshToken) error {
	_, err := r.db.Exec(`
		UPDATE refresh_tokens
		SET is_used = $1, parent_tag = $2
		WHERE token = $3
	`, rt.IsUsed, rt.ParentTag, rt.Token)
	return err
}

func (r *RefreshTokenRepository) Delete(token string) error {
	_, err := r.db.Exec(`DELETE FROM refresh_tokens WHERE token = $1`, token)
	return err
}

func (r *RefreshTokenRepository) DeleteAllForUser(userID string) error {
	_, err := r.db.Exec(`DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	return err
}
