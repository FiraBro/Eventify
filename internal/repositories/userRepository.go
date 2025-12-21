package repositories

import (
	"database/sql"
	"log"
	"strings"

	"github.com/FiraBro/local-go/internal/db"
	"github.com/FiraBro/local-go/internal/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository() *UserRepository {
	return &UserRepository{db: db.DB}
}

func (r *UserRepository) CreateUser(user *models.User) error {
	query := `INSERT INTO users (id, username, email, password, role) VALUES (?, ?, ?, ?, ?)`
	_, err := r.db.Exec(query, user.ID, user.Username, user.Email, user.Password, user.Role)
	return err
}

func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	log.Println("Querying for email:", email) // Debug log

	row := r.db.QueryRow("SELECT id, username, email, password, role FROM users WHERE LOWER(email)=?", email)
	var u models.User
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role); err != nil {
		log.Println("GetByEmail scan error:", err)
		return nil, err
	}
	log.Println("Found user:", u)
	return &u, nil
}



func (r *UserRepository) GetByID(id string) (*models.User, error) {
	row := r.db.QueryRow("SELECT id, username, email, password, role FROM users WHERE id=?", id)
	var u models.User
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetAll() ([]models.User, error) {
	rows, err := r.db.Query("SELECT id, username, email, password, role FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *UserRepository) DeleteUser(id string) error {
	_, err := r.db.Exec("DELETE FROM users WHERE id=?", id)
	return err
}
