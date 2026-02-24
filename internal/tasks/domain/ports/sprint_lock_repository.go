package ports

import (
	"context"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

// SprintLockRepository defines the interface for persisting sprint classification locks
type SprintLockRepository interface {
	// FindLock returns the lock for a project+sprint pair, or nil if not locked
	FindLock(ctx context.Context, project, sprint string) (*domain.SprintLock, error)
	// SaveLock persists a sprint lock
	SaveLock(ctx context.Context, lock *domain.SprintLock) error
}
