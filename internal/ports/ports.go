package ports

import (
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

// TaskService defines the interface for task management operations
type TaskService interface {
	FetchTasks() error
	IncrementTaskCount(assetName string) error
	DecrementTaskCount(assetName string) error
}

// SprintService defines the interface for sprint management operations
type SprintService interface {
	ProcessJiraIssues(project, sprint, override string) (string, error)
}

// UserInput defines the interface for user interaction
type UserInput interface {
	GetUserInput(prompt string) (string, error)
	GetUserConfirmation(prompt string) (bool, error)
}

// TaskClassifier defines the interface for task classification
type TaskClassifier interface {
	ClassifyTask(task domain.Task) (string, error)
}

// TaskRepository defines the interface for task storage operations
type TaskRepository interface {
	SaveTasks(tasks []domain.Task) error
	LoadTasks() ([]domain.Task, error)
}

// JiraRepository defines the interface for Jira operations
type JiraRepository interface {
	FetchTasks() ([]domain.Task, error)
}
