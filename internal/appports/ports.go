package appports

import (
	"context"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application/usecase"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// TaskService defines the interface for task management operations
type TaskService interface {
	FetchTasks(ctx context.Context, project, sprint, platform string) error
	ClassifyTasks(ctx context.Context, input usecase.ClassifyTasksInput) error
	GetTasks(ctx context.Context, project, sprint string) ([]*domain.Task, error)
	GetTasksByAsset(ctx context.Context, assetName string) ([]*domain.Task, error)
	GetLocalRepository() ports.TaskRepository
}

// SprintService defines the interface for sprint management operations
type SprintService interface {
	ProcessJiraIssues(project, sprint, override string) (string, error)
}

// UserInput defines the interface for user interaction
type UserInput interface {
	Confirm(format string, args ...interface{}) (bool, error)
}

// TaskClassifier defines the interface for task classification
type TaskClassifier interface {
	ClassifyTask(task *domain.Task) (domain.WorkType, error)
	ClassifyTasks(tasks []*domain.Task) (map[string]domain.WorkType, error)
}

// TaskRepository defines the interface for task storage operations
type TaskRepository interface {
	Save(ctx context.Context, task *domain.Task) error
	FindByKey(ctx context.Context, key string) (*domain.Task, error)
	FindByProjectAndSprint(ctx context.Context, project, sprint string) ([]*domain.Task, error)
	FindByProject(ctx context.Context, project string) ([]*domain.Task, error)
	FindBySprint(ctx context.Context, sprint string) ([]*domain.Task, error)
	FindByPlatform(ctx context.Context, platform string) ([]*domain.Task, error)
	FindAll(ctx context.Context) ([]*domain.Task, error)
	Delete(ctx context.Context, key string) error
	DeleteByProjectAndSprint(ctx context.Context, project, sprint string) error
	UpdateLabels(ctx context.Context, taskKey string, labels []string) error
}

// JiraRepository defines the interface for Jira operations
type JiraRepository interface {
	TaskRepository
}
