package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/FiraBro/local-go/internal/models"
	"github.com/google/uuid"
)

type ServiceRepository struct {
	db *sql.DB
}

func NewServiceRepository(db *sql.DB) *ServiceRepository {
	return &ServiceRepository{db: db}
}

// --------------------
// CREATE SERVICE
// --------------------
func (r *ServiceRepository) Create(ctx context.Context, service *models.Service) error {
	service.ID = uuid.New().String()
	service.CreatedAt = time.Now()
	service.UpdatedAt = time.Now()

	query := `
		INSERT INTO services (
			id, name, description, category, price, created_at, updated_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`

	_, err := r.db.ExecContext(ctx, query,
		service.ID,
		service.Name,
		service.Description,
		service.Category,
		service.Price,
		service.CreatedAt,
		service.UpdatedAt,
	)

	return err
}

// --------------------
// GET ALL SERVICES
// --------------------
func (r *ServiceRepository) GetAll(ctx context.Context) ([]models.Service, error) {
	query := `
		SELECT id, name, description, category, price, created_at, updated_at
		FROM services
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []models.Service
	for rows.Next() {
		var s models.Service
		if err := rows.Scan(
			&s.ID,
			&s.Name,
			&s.Description,
			&s.Category,
			&s.Price,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		services = append(services, s)
	}

	return services, nil
}

// --------------------
// GET SERVICE BY ID
// --------------------
func (r *ServiceRepository) GetByIDs(ctx context.Context, id string) (*models.Service, error) {
	query := `
		SELECT id, name, description, category, price, created_at, updated_at
		FROM services
		WHERE id = $1
	`

	var s models.Service
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID,
		&s.Name,
		&s.Description,
		&s.Category,
		&s.Price,
		&s.CreatedAt,
		&s.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // not found
		}
		return nil, err
	}

	return &s, nil
}


// --------------------
// UPDATE SERVICE
// --------------------
func (r *ServiceRepository) Update(ctx context.Context, id string, s *models.Service) error {
    query := `
        UPDATE services
        SET name = $1, description = $2, category = $3, price = $4, updated_at = $5
        WHERE id = $6
    `
    _, err := r.db.ExecContext(ctx, query, 
        s.Name, s.Description, s.Category, s.Price, time.Now(), id,
    )
    return err
}

// --------------------
// DELETE SERVICE
// --------------------
func (r *ServiceRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(
		ctx,
		`DELETE FROM services WHERE id = $1`,
		id,
	)
	return err
}

// --------------------
// GET SERVICE CATEGORIES
// --------------------
func (r *ServiceRepository) GetCategories(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT DISTINCT category FROM services ORDER BY category`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}

	return categories, nil
}
