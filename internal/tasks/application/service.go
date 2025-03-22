package application

import (
	"context"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application/usecase"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// TaskService provides all task-related operations
type TaskService struct {
	fetchTasksUseCase *usecase.FetchTasksUseCase
}

// NewTasksService creates a new TasksService
func NewTasksService(taskRepo ports.TaskRepository) *TaskService {
	return &TaskService{
		fetchTasksUseCase: usecase.NewFetchTasksUseCase(taskRepo),
	}
}

// FetchTasks fetches tasks from a platform
func (s *TaskService) FetchTasks(ctx context.Context, project, sprint, platform string) error {
	return s.fetchTasksUseCase.Execute(ctx, project, sprint, platform)
}
