package ports

import "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"

// TaskClassifier defines the interface for classifying tasks by work type
type TaskClassifier interface {
	// ClassifyTask determines the work type of a task based on its content
	ClassifyTask(task *domain.Task) (domain.WorkType, error)

	// ClassifyTasks determines the work type for multiple tasks
	ClassifyTasks(tasks []*domain.Task) (map[string]domain.WorkType, error)
}

// ComprehensiveTaskClassifier extends TaskClassifier to support comprehensive classification results
// This allows access to both work type and asset assignment information
type ComprehensiveTaskClassifier interface {
	TaskClassifier

	// ClassifyTasksComprehensive returns detailed classification results including asset assignments
	ClassifyTasksComprehensive(tasks []*domain.Task) ([]*ComprehensiveClassificationResult, error)
}
