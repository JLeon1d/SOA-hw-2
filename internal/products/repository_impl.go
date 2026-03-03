package products

import (
	"database/sql"
	"fmt"
	"strings"

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

func (r *repositoryImpl) Create(product *domain.Product) error {
	query := `
		INSERT INTO products (id, name, description, price, stock, category, status, seller_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := r.db.Exec(query,
		product.ID,
		product.Name,
		product.Description,
		product.Price,
		product.Stock,
		product.Category,
		product.Status,
		product.SellerID,
		product.CreatedAt,
		product.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	return nil
}

func (r *repositoryImpl) GetByID(id uuid.UUID) (*domain.Product, error) {
	var product domain.Product

	query := `SELECT * FROM products WHERE id = $1`

	err := r.db.Get(&product, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	return &product, nil
}

func (r *repositoryImpl) Update(product *domain.Product) error {
	query := `
		UPDATE products 
		SET name = $2, description = $3, price = $4, stock = $5, 
		    category = $6, status = $7, updated_at = $8
		WHERE id = $1
	`

	result, err := r.db.Exec(query,
		product.ID,
		product.Name,
		product.Description,
		product.Price,
		product.Stock,
		product.Category,
		product.Status,
		product.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update product: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("product not found")
	}

	return nil
}

func (r *repositoryImpl) Delete(id uuid.UUID) error {
	query := `UPDATE products SET status = $1, updated_at = NOW() WHERE id = $2`

	result, err := r.db.Exec(query, domain.ProductArchived, id)
	if err != nil {
		return fmt.Errorf("failed to archive product: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("product not found")
	}

	return nil
}

func (r *repositoryImpl) List(filter Filter) (*ListResult, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.Category != nil {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, *filter.Category)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	var totalElements int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM products %s", whereClause)
	err := r.db.Get(&totalElements, countQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to count products: %w", err)
	}

	offset := filter.Page * filter.Size
	args = append(args, filter.Size, offset)

	var products []domain.Product

	query := fmt.Sprintf(`
		SELECT * FROM products %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	err = r.db.Select(&products, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}

	return &ListResult{
		Products:      products,
		TotalElements: totalElements,
	}, nil
}

func (r *repositoryImpl) UpdateStock(tx *sqlx.Tx, productID uuid.UUID, delta int) error {
	query := `
		UPDATE products 
		SET stock = stock + $1, updated_at = NOW()
		WHERE id = $2
		RETURNING stock
	`

	var newStock int
	var err error
	if tx != nil {
		err = tx.QueryRow(query, delta, productID).Scan(&newStock)
	} else {
		err = r.db.QueryRow(query, delta, productID).Scan(&newStock)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("product not found")
		}
		return fmt.Errorf("failed to update stock: %w", err)
	}

	if newStock < 0 {
		return fmt.Errorf("insufficient stock")
	}

	return nil
}
