package operations

import (
	"fmt"
	"time"

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

func (r *repositoryImpl) Create(tx *sqlx.Tx, operation *domain.UserOperation) error {
	query := `
		INSERT INTO user_operations (id, user_id, operation_type, created_at)
		VALUES ($1, $2, $3, $4)
	`

	var err error
	if tx != nil {
		_, err = tx.Exec(query,
			operation.ID,
			operation.UserID,
			operation.OperationType,
			operation.CreatedAt,
		)
	} else {
		_, err = r.db.Exec(query,
			operation.ID,
			operation.UserID,
			operation.OperationType,
			operation.CreatedAt,
		)
	}

	if err != nil {
		return fmt.Errorf("failed to create user operation: %w", err)
	}

	return nil
}

func (r *repositoryImpl) GetLastOperation(userID uuid.UUID, operationType domain.OperationType) (*domain.UserOperation, error) {
	var operation domain.UserOperation

	query := `
		SELECT * FROM user_operations 
		WHERE user_id = $1 AND operation_type = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	err := r.db.Get(&operation, query, userID, operationType)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get last operation: %w", err)
	}

	return &operation, nil
}

func (r *repositoryImpl) CheckRateLimit(userID uuid.UUID, operationType domain.OperationType, windowMinutes int) (bool, time.Duration, error) {
	lastOp, err := r.GetLastOperation(userID, operationType)
	if err != nil {
		return false, 0, err
	}

	if lastOp == nil {
		return true, 0, nil
	}

	elapsed := time.Since(lastOp.CreatedAt)
	window := time.Duration(windowMinutes) * time.Minute

	if elapsed < window {
		remaining := window - elapsed
		return false, remaining, nil
	}

	return true, 0, nil
}
