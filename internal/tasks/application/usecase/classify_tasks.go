package usecase

import (
	"context"
	"fmt"

	assetsapp "github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// ClassifyTasksUseCase handles the classification of tasks for a project/sprint
type ClassifyTasksUseCase struct {
	localRepo    ports.TaskRepository
	remoteRepo   ports.TaskRepository
	classifier   ports.TaskClassifier
	userInput    ports.UserInput
	assetService assetsapp.AssetService
}

// NewClassifyTasksUseCase creates a new instance of ClassifyTasksUseCase
func NewClassifyTasksUseCase(
	localRepo ports.TaskRepository,
	remoteRepo ports.TaskRepository,
	classifier ports.TaskClassifier,
	userInput ports.UserInput,
	assetService assetsapp.AssetService,
) *ClassifyTasksUseCase {
	return &ClassifyTasksUseCase{
		localRepo:    localRepo,
		remoteRepo:   remoteRepo,
		classifier:   classifier,
		userInput:    userInput,
		assetService: assetService,
	}
}

// Execute runs the task classification process
func (uc *ClassifyTasksUseCase) Execute(ctx context.Context, input domain.ClassifyTasksInput) error {
	// First, try to find existing tasks for the project/sprint
	tasks, err := uc.localRepo.FindByProjectAndSprint(ctx, input.Project, input.Sprint)
	if err != nil {
		return fmt.Errorf("failed to find existing tasks: %w", err)
	}

	// If no tasks found, ask user if they want to fetch them
	if len(tasks) == 0 {
		shouldFetch, confirmErr := uc.userInput.Confirm("No tasks found for project %s and sprint %s. Would you like to fetch them?", input.Project, input.Sprint)
		if confirmErr != nil {
			return fmt.Errorf("failed to get user confirmation: %w", confirmErr)
		}

		if shouldFetch {
			// Fetch tasks from the platform
			var fetchedTasks []*domain.Task
			var fetchErr error
			fetchedTasks, fetchErr = uc.remoteRepo.FindByProjectAndSprint(ctx, input.Project, input.Sprint)
			if fetchErr != nil {
				return fmt.Errorf("failed to fetch tasks: %w", fetchErr)
			}

			// Save fetched tasks to repository
			for _, task := range fetchedTasks {
				if saveErr := uc.localRepo.Save(ctx, task); saveErr != nil {
					return fmt.Errorf("failed to save fetched task %s: %w", task.Key, saveErr)
				}
			}
			tasks = fetchedTasks
		} else {
			return fmt.Errorf("no tasks available for classification")
		}
	}

	// Preview classifications if in dry run mode
	if input.DryRun {
		return uc.previewClassifications(tasks)
	}

	// Classify all tasks for actual execution
	workTypes, err := uc.classifier.ClassifyTasks(tasks)
	if err != nil {
		return fmt.Errorf("failed to classify tasks: %w", err)
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

// previewClassifications shows classification preview with enhanced output when comprehensive results are available
// Includes intelligent asset syncing when unassigned tasks are detected
func (uc *ClassifyTasksUseCase) previewClassifications(tasks []*domain.Task) error {
	return uc.previewClassificationsWithRetry(tasks, false)
}

// previewClassificationsWithRetry handles the classification preview with optional asset sync retry
func (uc *ClassifyTasksUseCase) previewClassificationsWithRetry(tasks []*domain.Task, hasTriedSync bool) error {
	fmt.Println("\nPreview of task classifications:")

	// Check if classifier supports comprehensive results
	if comprehensiveClassifier, ok := uc.classifier.(ports.ComprehensiveTaskClassifier); ok {
		// Use comprehensive classification for detailed preview
		results, err := comprehensiveClassifier.ClassifyTasksComprehensive(tasks)
		if err != nil {
			return fmt.Errorf("failed to classify tasks comprehensively: %w", err)
		}

		// Count unassigned tasks
		var unassignedTasks []string
		for _, result := range results {
			assetInfo := "No asset assigned"
			if result.Asset != nil && result.Asset.Asset != nil {
				assetInfo = fmt.Sprintf("Asset: %s (%.0f%% confidence)", result.Asset.Asset.Name, result.Asset.Confidence*100)
			} else {
				unassignedTasks = append(unassignedTasks, result.Task.Key)
			}

			fmt.Printf("- %s: %s | %s (%s)\n",
				result.Task.Key,
				result.WorkType,
				assetInfo,
				result.Task.Summary)
		}

		// If there are unassigned tasks and we haven't tried syncing yet, offer to sync assets
		if len(unassignedTasks) > 0 && !hasTriedSync {
			fmt.Printf("\nFound %d task(s) without asset assignments: %v\n", len(unassignedTasks), unassignedTasks)

			shouldSync, confirmErr := uc.userInput.Confirm("Would you like to sync assets from Confluence to potentially improve classification?")
			if confirmErr != nil {
				return fmt.Errorf("failed to get user confirmation for asset sync: %w", confirmErr)
			}

			if shouldSync {
				fmt.Println("Syncing assets from Confluence...")

				// Sync assets with default parameters (CAP space, cap-asset label)
				syncResult, syncErr := uc.assetService.SyncFromConfluence("CAP", "cap-asset", false)
				if syncErr != nil {
					fmt.Printf("Warning: Asset sync failed: %v\n", syncErr)
					fmt.Println("Continuing with current classification results...")
				} else {
					fmt.Printf("Asset sync completed: %d assets synced, %d not synced\n",
						len(syncResult.SyncedAssets), len(syncResult.NotSyncedAssets))

					// Re-run classification with updated assets (only once to avoid loops)
					fmt.Println("\nRe-running classification with updated assets...")
					return uc.previewClassificationsWithRetry(tasks, true)
				}
			}
		}
	} else {
		// Fallback to simple classification for backward compatibility
		workTypes, err := uc.classifier.ClassifyTasks(tasks)
		if err != nil {
			return fmt.Errorf("failed to classify tasks: %w", err)
		}

		for _, task := range tasks {
			workType := workTypes[task.Key]
			fmt.Printf("- %s: %s (%s)\n", task.Key, workType, task.Summary)
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
		remoteTasks, fetchErr := uc.remoteRepo.FindByProjectAndSprint(ctx, project, sprint)
		if fetchErr != nil {
			return nil, fmt.Errorf("failed to fetch tasks from remote: %w", fetchErr)
		}

		// Save remote tasks to local repository
		for _, task := range remoteTasks {
			if saveErr := uc.localRepo.Save(ctx, task); saveErr != nil {
				return nil, fmt.Errorf("failed to save fetched task: %w", saveErr)
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

func (uc *ClassifyTasksUseCase) GetLocalRepository() ports.TaskRepository {
	return uc.localRepo
}
