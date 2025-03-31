package usecase

import (
	"context"
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// ClassifyTasksInput represents the input parameters for classifying tasks
type ClassifyTasksInput struct {
	Project string
	Sprint  string
	DryRun  bool
	Apply   bool
}

// ClassifyTasksUseCase handles the classification of tasks for a project/sprint
type ClassifyTasksUseCase struct {
	localRepo  ports.TaskRepository
	remoteRepo ports.TaskRepository
	classifier ports.TaskClassifier
	userInput  ports.UserInput
}

// NewClassifyTasksUseCase creates a new instance of ClassifyTasksUseCase
func NewClassifyTasksUseCase(
	localRepo ports.TaskRepository,
	remoteRepo ports.TaskRepository,
	classifier ports.TaskClassifier,
	userInput ports.UserInput,
) *ClassifyTasksUseCase {
	return &ClassifyTasksUseCase{
		localRepo:  localRepo,
		remoteRepo: remoteRepo,
		classifier: classifier,
		userInput:  userInput,
	}
}

// Execute runs the task classification process
func (uc *ClassifyTasksUseCase) Execute(ctx context.Context, input ClassifyTasksInput) error {
	// First, try to find existing tasks for the project/sprint
	tasks, err := uc.localRepo.FindByProjectAndSprint(ctx, input.Project, input.Sprint)
	if err != nil {
		return fmt.Errorf("failed to find existing tasks: %w", err)
	}

	// If no tasks found, ask user if they want to fetch them
	if len(tasks) == 0 {
		shouldFetch, err := uc.userInput.Confirm("No tasks found for project %s and sprint %s. Would you like to fetch them?", input.Project, input.Sprint)
		if err != nil {
			return fmt.Errorf("failed to get user confirmation: %w", err)
		}

		if shouldFetch {
			// Fetch tasks from the platform
			fetchedTasks, err := uc.remoteRepo.FindByProjectAndSprint(ctx, input.Project, input.Sprint)
			if err != nil {
				return fmt.Errorf("failed to fetch tasks: %w", err)
			}

			// Save fetched tasks to repository
			for _, task := range fetchedTasks {
				if err := uc.localRepo.Save(ctx, task); err != nil {
					return fmt.Errorf("failed to save fetched task %s: %w", task.Key, err)
				}
			}
			tasks = fetchedTasks
		} else {
			return fmt.Errorf("no tasks available for classification")
		}
	}

	// Classify all tasks
	workTypes, err := uc.classifier.ClassifyTasks(tasks)
	if err != nil {
		return fmt.Errorf("failed to classify tasks: %w", err)
	}

	// Preview classifications if in dry run mode
	if input.DryRun {
		fmt.Println("\nPreview of task classifications:")
		for _, task := range tasks {
			workType := workTypes[task.Key]
			fmt.Printf("- %s: %s (%s)\n", task.Key, workType, task.Summary)
		}
		return nil
	}

	// Update tasks with their classifications
	for _, task := range tasks {
		workType := workTypes[task.Key]
		if err := task.UpdateWorkType(workType); err != nil {
			return fmt.Errorf("failed to update work type for task %s: %w", task.Key, err)
		}

		// Save updated task locally
		if err := uc.localRepo.Save(ctx, task); err != nil {
			return fmt.Errorf("failed to save classified task %s: %w", task.Key, err)
		}

		// Apply labels to Jira if requested
		if input.Apply {
			if err := uc.remoteRepo.UpdateLabels(ctx, task.Key, []string{string(workType)}); err != nil {
				return fmt.Errorf("failed to apply labels to task %s: %w", task.Key, err)
			}
		}
	}

	return nil
}

// GetTasks retrieves tasks for a project and sprint
func (uc *ClassifyTasksUseCase) GetTasks(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
	// Try to get tasks from local repository first
	tasks, err := uc.localRepo.FindByProjectAndSprint(ctx, project, sprint)
	if err != nil {
		return nil, fmt.Errorf("failed to find existing tasks: %w", err)
	}

	// If no tasks found locally, try to fetch from remote
	if len(tasks) == 0 {
		remoteTasks, err := uc.remoteRepo.FindByProjectAndSprint(ctx, project, sprint)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch tasks from remote: %w", err)
		}

		// Save remote tasks to local repository
		for _, task := range remoteTasks {
			if err := uc.localRepo.Save(ctx, task); err != nil {
				return nil, fmt.Errorf("failed to save fetched task: %w", err)
			}
		}

		return remoteTasks, nil
	}

	return tasks, nil
}

// GetAllTasks retrieves all tasks from the local repository
func (uc *ClassifyTasksUseCase) GetAllTasks(ctx context.Context) ([]*domain.Task, error) {
	return uc.localRepo.FindAll(ctx)
}

func (u *ClassifyTasksUseCase) GetLocalRepository() ports.TaskRepository {
	return u.localRepo
}
