package repositories

import (
	"database/sql"
	"strings"
	"time"

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
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	query := `
		INSERT INTO users (id, username, email, password, role, deleted_at, delete_deadline)
		VALUES ($1, $2, $3, $4, $5, NULL, NULL)
	`

	_, err := r.db.Exec(
		query,
		user.ID,
		user.Username,
		strings.ToLower(user.Email),
		user.Password,
		user.Role,
	)
	return err
}

// ----------------------------
// GET USER BY EMAIL
// ----------------------------
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	row := r.db.QueryRow(`
		SELECT id, username, email, password, role
		FROM users
		WHERE LOWER(email) = $1 AND deleted_at IS NULL
	`, email)

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
	row := r.db.QueryRow(`
		SELECT id, username, email, password, role
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, id)

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
	_, err := r.db.Exec(`
		UPDATE users
		SET username = $1, email = $2
		WHERE id = $3 AND deleted_at IS NULL
	`, user.Username, strings.ToLower(user.Email), id)

	return err
}

// ----------------------------
// UPDATE PASSWORD
// ----------------------------
func (r *UserRepository) UpdatePassword(id string, hashedPassword string) error {
	_, err := r.db.Exec(`
		UPDATE users
		SET password = $1
		WHERE id = $2 AND deleted_at IS NULL
	`, hashedPassword, id)

	return err
}

// ----------------------------
// HARD DELETE
// ----------------------------
func (r *UserRepository) DeleteUser(id string) error {
	_, err := r.db.Exec(`DELETE FROM users WHERE id = $1`, id)
	return err
}

// ----------------------------
// CHECK IF EMAIL EXISTS
// ----------------------------
func (r *UserRepository) ExistsByEmail(email string) (bool, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	row := r.db.QueryRow(`
		SELECT COUNT(*)
		FROM users
		WHERE LOWER(email) = $1
	`, email)

	var count int
	if err := row.Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}

// ----------------------------
// FETCH ALL USERS
// ----------------------------
func (r *UserRepository) FetchAllUsers() ([]models.User, error) {
	rows, err := r.db.Query(`
		SELECT id, username, email, role
		FROM users
		WHERE deleted_at IS NULL
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// ----------------------------
// SOFT DELETE USER
// ----------------------------
func (r *UserRepository) SoftDeleteUser(id string) error {
	now := time.Now()
	deadline := now.Add(14 * 24 * time.Hour)

	_, err := r.db.Exec(`
		UPDATE users
		SET deleted_at = $1, delete_deadline = $2
		WHERE id = $3 AND deleted_at IS NULL
	`, now, deadline, id)

	return err
}

// ----------------------------
// RESTORE USER
// ----------------------------
func (r *UserRepository) RestoreUser(id string) error {
	_, err := r.db.Exec(`
		UPDATE users
		SET deleted_at = NULL, delete_deadline = NULL
		WHERE id = $1
	`, id)

	return err
}

// ----------------------------
// CHECK IF USER IS DELETED
// ----------------------------
func (r *UserRepository) IsUserDeleted(id string) (bool, error) {
	row := r.db.QueryRow(`
		SELECT deleted_at
		FROM users
		WHERE id = $1
	`, id)

	var deletedAt *time.Time
	err := row.Scan(&deletedAt)
	if err != nil {
		return false, err
	}

	return deletedAt != nil, nil
}

// ----------------------------
// GET ACTIVE USER BY ID
// ----------------------------
func (r *UserRepository) GetActiveByID(id string) (*models.User, error) {
	row := r.db.QueryRow(`
		SELECT id, username, email, role, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, id)

	var u models.User
	var deletedAt sql.NullTime
	if err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Role, &deletedAt); err != nil {
		return nil, err
	}

	if deletedAt.Valid {
		u.DeletedAt = &deletedAt.Time
	}
	return &u, nil
}

// ----------------------------
// PERMANENT DELETE EXPIRED
// ----------------------------
func (r *UserRepository) PermanentlyDeleteExpired() error {
	_, err := r.db.Exec(`
		DELETE FROM users
		WHERE deleted_at IS NOT NULL
		AND delete_deadline <= NOW()
	`)
	return err
}
