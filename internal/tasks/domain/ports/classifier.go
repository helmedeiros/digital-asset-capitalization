package ports

import "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"

// TaskClassifier defines the interface for classifying tasks by work type
type TaskClassifier interface {
	// ClassifyTask determines the work type of a task based on its content
	ClassifyTask(task *domain.Task) (domain.WorkType, error)

	// ClassifyTasks determines the work type for multiple tasks
	ClassifyTasks(tasks []*domain.Task) (map[string]domain.WorkType, error)
}
