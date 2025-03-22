package usecase

import (
	"context"
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// FetchTasksUseCase represents the use case for fetching tasks
type FetchTasksUseCase struct {
	repo ports.TaskRepository
}

// NewFetchTasksUseCase creates a new fetch tasks use case
func NewFetchTasksUseCase(repo ports.TaskRepository) *FetchTasksUseCase {
	return &FetchTasksUseCase{
		repo: repo,
	}
}

// Execute fetches tasks for a given project and sprint
func (u *FetchTasksUseCase) Execute(ctx context.Context, project, sprint, platform string) error {
	if project == "" {
		return fmt.Errorf("project is required")
	}

	if platform == "" {
		return fmt.Errorf("platform is required")
	}

	tasks, err := u.repo.FindByProjectAndSprint(ctx, project, sprint)
	if err != nil {
		return fmt.Errorf("failed to fetch tasks: %w", err)
	}

	// TODO: Implement task display logic
	fmt.Printf("Found %d tasks\n", len(tasks))
	for _, task := range tasks {
		sprintInfo := ""
		if task.Sprint != "" {
			sprintInfo = fmt.Sprintf(" [Sprints: %s]", task.Sprint)
		}
		fmt.Printf("- %s: [%s] %s (%s)%s\n", task.Key, task.Type, task.Summary, task.Status, sprintInfo)
	}

	return nil
}
