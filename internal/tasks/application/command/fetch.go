package command

import (
	"context"
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// FetchTasksCommand represents the command to fetch tasks
type FetchTasksCommand struct {
	Project  string
	Sprint   string
	Platform string
}

// FetchTasksHandler handles the fetch tasks command
type FetchTasksHandler struct {
	taskRepository ports.Repository
}

// NewFetchTasksHandler creates a new fetch tasks handler
func NewFetchTasksHandler(repo ports.Repository) *FetchTasksHandler {
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

	tasks, err := h.taskRepository.FindByProjectAndSprint(ctx, cmd.Project, cmd.Sprint)
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
