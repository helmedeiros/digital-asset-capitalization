package ports

import (
	"context"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

// TaskService defines the interface for task management operations
type TaskService interface {
	// FetchTasks fetches tasks from a platform
	FetchTasks(ctx context.Context, project, sprint, platform string) error

	// ClassifyTasks classifies tasks for a project and sprint
	ClassifyTasks(ctx context.Context, input domain.ClassifyTasksInput) error

	// GetTasks retrieves tasks for a project and sprint
	GetTasks(ctx context.Context, project, sprint string) ([]*domain.Task, error)

	// GetTasksByAsset retrieves tasks associated with a specific asset
	GetTasksByAsset(ctx context.Context, assetName string) ([]*domain.Task, error)

	// GetLocalRepository returns the local task repository
	GetLocalRepository() TaskRepository
}
