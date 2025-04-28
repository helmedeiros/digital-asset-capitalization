package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application/usecase"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// TaskService provides all task-related operations
type TaskService struct {
	fetchTasksUseCase    *usecase.FetchTasksUseCase
	classifyTasksUseCase *usecase.ClassifyTasksUseCase
}

// NewTasksService creates a new TasksService
func NewTasksService(remoteRepo, localRepo ports.TaskRepository, classifier ports.TaskClassifier, userInput ports.UserInput) *TaskService {
	return &TaskService{
		fetchTasksUseCase:    usecase.NewFetchTasksUseCase(remoteRepo, localRepo),
		classifyTasksUseCase: usecase.NewClassifyTasksUseCase(localRepo, remoteRepo, classifier, userInput),
	}
}

// FetchTasks fetches tasks from a platform
func (s *TaskService) FetchTasks(ctx context.Context, project, sprint, platform string) error {
	return s.fetchTasksUseCase.Execute(ctx, project, sprint, platform)
}

// ClassifyTasks classifies tasks for a project and sprint
func (s *TaskService) ClassifyTasks(ctx context.Context, input domain.ClassifyTasksInput) error {
	return s.classifyTasksUseCase.Execute(ctx, input)
}

// GetTasks retrieves tasks for a project and sprint
func (s *TaskService) GetTasks(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
	return s.classifyTasksUseCase.GetTasks(ctx, project, sprint)
}

// GetTasksByAsset retrieves tasks associated with a specific asset
func (s *TaskService) GetTasksByAsset(ctx context.Context, assetName string) ([]*domain.Task, error) {
	tasks, err := s.classifyTasksUseCase.GetAllTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	// Handle both asset names and full asset IDs
	assetID := assetName
	if !strings.HasPrefix(assetName, "cap-asset-") {
		// For multi-word asset names, use just the first word
		words := strings.Fields(assetName)
		assetID = "cap-asset-" + strings.ToLower(words[0])
	}

	var assetTasks []*domain.Task
	for _, task := range tasks {
		for _, label := range task.Labels {
			if label == assetID {
				assetTasks = append(assetTasks, task)
				break
			}
		}
	}

	return assetTasks, nil
}

func (s *TaskService) GetLocalRepository() ports.TaskRepository {
	return s.classifyTasksUseCase.GetLocalRepository()
}
