package ports

import "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"

// TaskFetcher defines the interface for fetching tasks from external platforms
type TaskFetcher interface {
	// FetchTasks retrieves tasks for a given project and sprint
	FetchTasks(project, sprint string) ([]*domain.Task, error)
}
