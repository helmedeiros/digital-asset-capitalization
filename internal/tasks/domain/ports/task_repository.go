package ports

import (
	"context"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

// TaskRepository defines the interface for task persistence operations
type TaskRepository interface {
	// Save persists a task
	Save(ctx context.Context, task *domain.Task) error

	// SaveAll persists multiple tasks atomically with respect to the
	// repository's internal load/store cycle. For implementations backed
	// by a file (e.g. JSON storage) this is dramatically cheaper than
	// calling Save in a loop, which would re-read and re-write the
	// entire file once per task. Implementations MAY treat SaveAll as
	// equivalent to Save-in-a-loop when no batch path applies.
	SaveAll(ctx context.Context, tasks []*domain.Task) error

	// FindByKey retrieves a task by its key
	FindByKey(ctx context.Context, key string) (*domain.Task, error)

	// FindByProjectAndSprint retrieves tasks for a specific project and sprint
	FindByProjectAndSprint(ctx context.Context, project, sprint string) ([]*domain.Task, error)

	// FindByProject retrieves all tasks for a specific project
	FindByProject(ctx context.Context, project string) ([]*domain.Task, error)

	// FindBySprint retrieves all tasks for a specific sprint
	FindBySprint(ctx context.Context, sprint string) ([]*domain.Task, error)

	// FindByPlatform retrieves all tasks for a specific platform
	FindByPlatform(ctx context.Context, platform string) ([]*domain.Task, error)

	// FindAll retrieves all tasks
	FindAll(ctx context.Context) ([]*domain.Task, error)

	// Delete removes a task
	Delete(ctx context.Context, key string) error

	// DeleteByProjectAndSprint removes all tasks for a specific project and sprint
	DeleteByProjectAndSprint(ctx context.Context, project, sprint string) error

	// UpdateLabels updates the labels of a task in the remote repository
	// using add/remove operations to avoid overwriting non-cap labels
	UpdateLabels(ctx context.Context, taskKey string, addLabels, removeLabels []string) error
}
