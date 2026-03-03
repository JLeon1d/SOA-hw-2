package promos

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

func (r *repositoryImpl) Create(promo *domain.PromoCode) error {
	query := `
		INSERT INTO promo_codes (id, code, discount_type, discount_value, min_order_amount, 
		                         max_uses, current_uses, valid_from, valid_until, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.Exec(query,
		promo.ID,
		promo.Code,
		promo.DiscountType,
		promo.DiscountValue,
		promo.MinOrderAmount,
		promo.MaxUses,
		promo.CurrentUses,
		promo.ValidFrom,
		promo.ValidUntil,
		promo.Active,
	)
	if err != nil {
		return fmt.Errorf("failed to create promo code: %w", err)
	}
	return nil
}

func (r *repositoryImpl) GetByCode(code string) (*domain.PromoCode, error) {
	var promo domain.PromoCode
	query := `SELECT * FROM promo_codes WHERE code = $1`
	err := r.db.Get(&promo, query, code)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("promo code not found")
		}
		return nil, fmt.Errorf("failed to get promo code: %w", err)
	}
	return &promo, nil
}

func (r *repositoryImpl) GetByID(id uuid.UUID) (*domain.PromoCode, error) {
	var promo domain.PromoCode
	query := `SELECT * FROM promo_codes WHERE id = $1`
	err := r.db.Get(&promo, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("promo code not found")
		}
		return nil, fmt.Errorf("failed to get promo code: %w", err)
	}
	return &promo, nil
}

func (r *repositoryImpl) Update(promo *domain.PromoCode) error {
	query := `
		UPDATE promo_codes 
		SET discount_value = $1, min_order_amount = $2, max_uses = $3, 
		    valid_until = $4, active = $5
		WHERE id = $6
	`
	_, err := r.db.Exec(query,
		promo.DiscountValue,
		promo.MinOrderAmount,
		promo.MaxUses,
		promo.ValidUntil,
		promo.Active,
		promo.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update promo code: %w", err)
	}
	return nil
}

func (r *repositoryImpl) Delete(id uuid.UUID) error {
	query := `DELETE FROM promo_codes WHERE id = $1`
	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete promo code: %w", err)
	}
	return nil
}

func (r *repositoryImpl) IncrementUses(tx *sqlx.Tx, id uuid.UUID) error {
	query := `
		UPDATE promo_codes 
		SET current_uses = current_uses + 1
		WHERE id = $1
	`
	var err error
	if tx != nil {
		_, err = tx.Exec(query, id)
	} else {
		_, err = r.db.Exec(query, id)
	}

	if err != nil {
		return fmt.Errorf("failed to increment promo code uses: %w", err)
	}
	return nil
}

func (r *repositoryImpl) DecrementUses(tx *sqlx.Tx, id uuid.UUID) error {
	query := `
		UPDATE promo_codes 
		SET current_uses = current_uses - 1
		WHERE id = $1 AND current_uses > 0
	`
	var err error
	if tx != nil {
		_, err = tx.Exec(query, id)
	} else {
		_, err = r.db.Exec(query, id)
	}

	if err != nil {
		return fmt.Errorf("failed to decrement promo code uses: %w", err)
	}
	return nil
}
