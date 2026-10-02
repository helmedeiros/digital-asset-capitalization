package application

import (
	"context"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application/usecase"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// TaskService defines the interface for task management operations
type TaskService interface {
	// FetchTasks fetches tasks from a platform
	FetchTasks(ctx context.Context, project, sprint, platform string) error

	// FetchTaskByKey fetches a single task by its key from a platform
	FetchTaskByKey(ctx context.Context, key, platform string) error

	// ClassifyTasks classifies tasks for a project and sprint
	ClassifyTasks(ctx context.Context, input domain.ClassifyTasksInput) error

	// GetTasks retrieves tasks for a project and sprint
	GetTasks(ctx context.Context, project, sprint string) ([]*domain.Task, error)

	// GetTasksByAsset retrieves tasks associated with a specific asset
	GetTasksByAsset(ctx context.Context, assetName string) ([]*domain.Task, error)

	// GetTaskByKey retrieves a single task by its key
	GetTaskByKey(ctx context.Context, key string) (*domain.Task, error)

	// GetLocalRepository returns the local task repository
	GetLocalRepository() ports.TaskRepository

	// SetCapitalizationLabelProvider configures the per-project real
	// accounting label lookup (e.g. an IAS38-style CapEx/OpEx label) used
	// when classifying with --apply. Optional: when never called, no
	// project has a configured label and only the internal
	// cap-development/cap-maintenance/cap-discovery label is written.
	SetCapitalizationLabelProvider(provider usecase.CapitalizationLabelProvider)
}
