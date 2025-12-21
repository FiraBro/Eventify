package repositories

import (
	"database/sql"
	"github.com/FiraBro/local-go/internal/db"
	"github.com/FiraBro/local-go/internal/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository() *UserRepository {
	return &UserRepository{db: db.DB}
}

// CreateUser saves a new user
func (r *UserRepository) CreateUser(user *models.User) error {
	query := `INSERT INTO users (id, username, password, role) VALUES (?, ?, ?, ?)`
	_, err := r.db.Exec(query, user.ID, user.Username, user.Password, user.Role)
	return err
}

// GetByUsername fetch a user by username
func (r *UserRepository) GetByUsername(username string) (*models.User, error) {
	row := r.db.QueryRow("SELECT id, username, password, role FROM users WHERE username=?", username)
	var user models.User
	if err := row.Scan(&user.ID, &user.Username, &user.Password, &user.Role); err != nil {
		return nil, err
	}
	return &user, nil
}
