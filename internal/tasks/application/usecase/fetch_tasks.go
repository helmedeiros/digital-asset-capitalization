package usecase

import (
	"context"
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// FetchTasksUseCase represents the use case for fetching tasks
type FetchTasksUseCase struct {
	remoteRepo ports.TaskRepository
	localRepo  ports.TaskRepository
}

// NewFetchTasksUseCase creates a new fetch tasks use case
func NewFetchTasksUseCase(remoteRepo, localRepo ports.TaskRepository) *FetchTasksUseCase {
	return &FetchTasksUseCase{
		remoteRepo: remoteRepo,
		localRepo:  localRepo,
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

	// Fetch tasks from remote repository (e.g., Jira)
	tasks, err := u.remoteRepo.FindByProjectAndSprint(ctx, project, sprint)
	if err != nil {
		return fmt.Errorf("failed to fetch tasks: %w", err)
	}

	// Save tasks to local storage
	for _, task := range tasks {
		if err := u.localRepo.Save(ctx, task); err != nil {
			return fmt.Errorf("failed to save task %s: %w", task.Key, err)
		}
	}

	// Display tasks
	fmt.Printf("Found and saved %d tasks\n", len(tasks))
	for _, task := range tasks {
		sprintInfo := ""
		if task.Sprint != "" {
			sprintInfo = fmt.Sprintf(" [Sprints: %s]", task.Sprint)
		}
		fmt.Printf("- %s: [%s] %s (%s)%s\n", task.Key, task.Type, task.Summary, task.Status, sprintInfo)
	}

	return nil
}
