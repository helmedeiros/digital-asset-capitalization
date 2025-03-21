package command

import (
	"context"
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

// FetchTasksCommand represents the command to fetch tasks
type FetchTasksCommand struct {
	Project  string
	Sprint   string
	Platform string
}

// FetchTasksHandler handles the fetch tasks command
type FetchTasksHandler struct {
	taskRepository domain.TaskRepository
}

// NewFetchTasksHandler creates a new fetch tasks handler
func NewFetchTasksHandler(repo domain.TaskRepository) *FetchTasksHandler {
	return &FetchTasksHandler{
		taskRepository: repo,
	}
}

// Handle executes the fetch tasks command
func (h *FetchTasksHandler) Handle(ctx context.Context, cmd FetchTasksCommand) error {
	if cmd.Project == "" {
		return fmt.Errorf("project is required")
	}

	if cmd.Platform == "" {
		return fmt.Errorf("platform is required")
	}

	tasks, err := h.taskRepository.FetchTasks(ctx, cmd.Project, cmd.Sprint)
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
		fmt.Printf("- %s: %s (%s)%s\n", task.Key, task.Summary, task.Status, sprintInfo)
	}

	return nil
}
