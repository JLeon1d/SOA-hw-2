package operations

import (
	"time"

	"marketplace-backend/internal/domain"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository interface {
	Create(tx *sqlx.Tx, operation *domain.UserOperation) error
	GetLastOperation(userID uuid.UUID, operationType domain.OperationType) (*domain.UserOperation, error)
	CheckRateLimit(userID uuid.UUID, operationType domain.OperationType, windowMinutes int) (bool, time.Duration, error)
}
