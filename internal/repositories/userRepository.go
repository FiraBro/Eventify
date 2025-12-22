package repositories

import (
	"database/sql"
	"strings"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/google/uuid"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// ----------------------------
// CREATE USER
// ----------------------------
func (r *UserRepository) CreateUser(user *models.User) error {
	query := `INSERT INTO users (id, username, email, password, role) VALUES (?, ?, ?, ?, ?)`
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	_, err := r.db.Exec(query, user.ID, user.Username, strings.ToLower(user.Email), user.Password, user.Role)
	return err
}

// ----------------------------
// GET USER BY EMAIL
// ----------------------------
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	row := r.db.QueryRow(`SELECT id, username, email, password, role FROM users WHERE LOWER(email)=?`, email)
	var u models.User
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role); err != nil {
		return nil, err
	}
	return &u, nil
}

// ----------------------------
// GET USER BY ID
// ----------------------------
func (r *UserRepository) GetByID(id string) (*models.User, error) {
	row := r.db.QueryRow(`SELECT id, username, email, password, role FROM users WHERE id=?`, id)
	var u models.User
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &u.Role); err != nil {
		return nil, err
	}
	return &u, nil
}

// ----------------------------
// UPDATE USER PROFILE
// ----------------------------
func (r *UserRepository) UpdateUser(id string, user *models.User) error {
	_, err := r.db.Exec(`UPDATE users SET username=?, email=? WHERE id=?`, user.Username, user.Email, id)
	return err
}

// ----------------------------
// UPDATE PASSWORD
// ----------------------------
func (r *UserRepository) UpdatePassword(id string, hashedPassword string) error {
	_, err := r.db.Exec(`UPDATE users SET password=? WHERE id=?`, hashedPassword, id)
	return err
}

// ----------------------------
// DELETE USER
// ----------------------------
func (r *UserRepository) DeleteUser(id string) error {
	_, err := r.db.Exec(`DELETE FROM users WHERE id=?`, id)
	return err
}

// ----------------------------
// OPTIONAL: Check if email exists (for forgot password validation)
// ----------------------------
func (r *UserRepository) ExistsByEmail(email string) (bool, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	row := r.db.QueryRow(`SELECT COUNT(*) FROM users WHERE LOWER(email)=?`, email)
	var count int
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}
