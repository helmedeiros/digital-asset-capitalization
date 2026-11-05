package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// FetchTasksUseCase represents the use case for fetching tasks
type FetchTasksUseCase struct {
	remoteRepo     ports.TaskRepository
	localRepo      ports.TaskRepository
	sprintResolver *SprintResolver
	boardSource    ports.BoardIssueSource
	excludedLabels ports.ExcludedLabelProvider
}

// SetBoardIssueSource adds the issues of the project's configured boards
// (e.g. a kanban board) to what a sprint fetch returns. Optional.
func (u *FetchTasksUseCase) SetBoardIssueSource(source ports.BoardIssueSource) {
	u.boardSource = source
}

// SetExcludedLabelProvider makes fetches skip the cards carrying one of the
// project's excluded labels. Optional.
func (u *FetchTasksUseCase) SetExcludedLabelProvider(provider ports.ExcludedLabelProvider) {
	u.excludedLabels = provider
}

// addBoardTasks fetches issues that sit on the project's configured boards
// for the sprint but were not returned by the sprint query, and tags them
// with the sprint so classification treats them like any other sprint task.
func (u *FetchTasksUseCase) addBoardTasks(ctx context.Context, project, sprint string, tasks []*domain.Task) ([]*domain.Task, error) {
	if u.boardSource == nil {
		return tasks, nil
	}
	keys, err := u.boardSource.FindIssueKeysForSprint(ctx, project, sprint)
	if err != nil {
		return nil, fmt.Errorf("failed to list board issues: %w", err)
	}
	seen := make(map[string]bool, len(tasks))
	for _, task := range tasks {
		seen[task.Key] = true
	}
	for _, key := range keys {
		if seen[key] {
			continue
		}
		seen[key] = true
		task, err := u.remoteRepo.FindByKey(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch board task %s: %w", key, err)
		}
		if task == nil {
			continue
		}
		if !task.HasSprint(sprint) {
			task.AddSprint(sprint)
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// NewFetchTasksUseCase creates a new fetch tasks use case
func NewFetchTasksUseCase(remoteRepo, localRepo ports.TaskRepository, sprintResolver *SprintResolver) *FetchTasksUseCase {
	return &FetchTasksUseCase{
		remoteRepo:     remoteRepo,
		localRepo:      localRepo,
		sprintResolver: sprintResolver,
	}
}

// Execute fetches tasks for a given project and sprint
func (u *FetchTasksUseCase) Execute(ctx context.Context, project, sprint, platform string) error {
	if project == "" {
		return fmt.Errorf("project is required")
	}

	if sprint == "" {
		return fmt.Errorf("sprint is required")
	}

	if platform == "" {
		return fmt.Errorf("platform is required")
	}

	// Resolve sprint name to sprint ID using interactive resolution if needed
	resolvedSprintID := sprint
	if u.sprintResolver != nil {
		var err error
		resolvedSprintID, err = u.sprintResolver.ResolveSprint(ctx, project, sprint)
		if err != nil {
			return fmt.Errorf("failed to resolve sprint '%s': %w", sprint, err)
		}
	}

	// Fetch tasks from remote repository using resolved sprint ID
	tasks, err := u.remoteRepo.FindByProjectAndSprint(ctx, project, resolvedSprintID)
	if err != nil {
		return fmt.Errorf("failed to fetch tasks: %w", err)
	}

	tasks, err = u.addBoardTasks(ctx, project, resolvedSprintID, tasks)
	if err != nil {
		return err
	}

	tasks, excluded := dropExcludedTasks(u.excludedLabels, project, tasks)
	if len(excluded) > 0 {
		fmt.Printf("Skipped %d excluded task(s): %s\n", len(excluded), strings.Join(excluded, ", "))
	}

	// Save tasks to local storage in one batch -- per-task Save would
	// re-read and re-write the entire JSON file once per task.
	if err := u.localRepo.SaveAll(ctx, tasks); err != nil {
		return fmt.Errorf("failed to save tasks: %w", err)
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

// ExecuteByKey fetches a single task by its key
func (u *FetchTasksUseCase) ExecuteByKey(ctx context.Context, key, platform string) error {
	if key == "" {
		return fmt.Errorf("task key is required")
	}

	if platform == "" {
		return fmt.Errorf("platform is required")
	}

	// Fetch task from remote repository (e.g., Jira)
	task, err := u.remoteRepo.FindByKey(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to fetch task %s: %w", key, err)
	}

	if task == nil {
		return fmt.Errorf("failed to fetch task %s: task not found", key)
	}

	// Save task to local storage
	if err := u.localRepo.Save(ctx, task); err != nil {
		return fmt.Errorf("failed to save task %s: %w", task.Key, err)
	}

	// Display task
	fmt.Printf("Found and saved 1 task\n")
	sprintInfo := ""
	if task.Sprint != "" {
		sprintInfo = fmt.Sprintf(" [Sprints: %s]", task.Sprint)
	}
	fmt.Printf("- %s: [%s] %s (%s)%s\n", task.Key, task.Type, task.Summary, task.Status, sprintInfo)

	return nil
}

// GetRemoteRepository returns the remote repository instance
func (u *FetchTasksUseCase) GetRemoteRepository() ports.TaskRepository {
	return u.remoteRepo
}
