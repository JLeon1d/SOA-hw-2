package users

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"marketplace-backend/internal/domain"
)

type repositoryImpl struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &repositoryImpl{db: db}
}

func (r *repositoryImpl) Create(user *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Role,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *repositoryImpl) GetByID(id uuid.UUID) (*domain.User, error) {
	var user domain.User
	query := `SELECT * FROM users WHERE id = $1`
	err := r.db.Get(&user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}

func (r *repositoryImpl) GetByEmail(email string) (*domain.User, error) {
	var user domain.User
	query := `SELECT * FROM users WHERE email = $1`
	err := r.db.Get(&user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}
