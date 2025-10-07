package ports

import (
	"context"

	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain"
)

// DeploymentRepository defines the interface for deployment persistence
type DeploymentRepository interface {
	// Save persists a deployment
	Save(ctx context.Context, deployment *domain.Deployment) error

	// FindByID retrieves a deployment by its ID
	FindByID(ctx context.Context, id string) (*domain.Deployment, error)

	// FindByTaskKey retrieves all deployments containing the specified task key
	FindByTaskKey(ctx context.Context, taskKey string) ([]*domain.Deployment, error)

	// FindByTimeRange retrieves all deployments within the specified time range
	FindByTimeRange(ctx context.Context, timeRange domain.TimeRange) ([]*domain.Deployment, error)

	// FindByEnvironmentAndTimeRange retrieves deployments for a specific environment within a time range
	FindByEnvironmentAndTimeRange(ctx context.Context, environment domain.Environment, timeRange domain.TimeRange) ([]*domain.Deployment, error)

	// ListAll retrieves all deployments
	ListAll(ctx context.Context) ([]*domain.Deployment, error)

	// Update updates an existing deployment
	Update(ctx context.Context, deployment *domain.Deployment) error

	// Delete removes a deployment by ID
	Delete(ctx context.Context, id string) error

	// Count returns the total number of deployments
	Count(ctx context.Context) (int, error)
}
