package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// Result represents the result of a migration operation
type Result struct {
	TasksProcessed int                    `json:"tasks_processed"`
	TasksMigrated  int                    `json:"tasks_migrated"`
	TasksSkipped   int                    `json:"tasks_skipped"`
	BackupCreated  bool                   `json:"backup_created"`
	BackupPath     string                 `json:"backup_path"`
	MigratedTasks  []string               `json:"migrated_tasks"`
	SkippedTasks   []string               `json:"skipped_tasks"`
	Errors         []string               `json:"errors"`
	Duration       time.Duration          `json:"duration"`
	Statistics     map[string]interface{} `json:"statistics"`
}

// SprintMigration handles the migration of sprint data from string to array format
type SprintMigration struct {
	repository ports.TaskRepository
	backupDir  string
}

// NewSprintMigration creates a new SprintMigration instance
func NewSprintMigration(repository ports.TaskRepository, backupDir string) *SprintMigration {
	return &SprintMigration{
		repository: repository,
		backupDir:  backupDir,
	}
}

// MigrateToArrayFormat migrates tasks from comma-separated sprint strings to array format
func (m *SprintMigration) MigrateToArrayFormat(_ context.Context, dryRun bool) (*Result, error) {
	start := time.Now()

	result := &Result{
		Statistics: make(map[string]interface{}),
	}

	// Get all tasks
	tasks, err := m.repository.FindAll(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks: %w", err)
	}

	result.TasksProcessed = len(tasks)

	// Create backup if not dry run
	if !dryRun {
		backupPath, err := m.createBackup(tasks)
		if err != nil {
			return nil, fmt.Errorf("failed to create backup: %w", err)
		}
		result.BackupCreated = true
		result.BackupPath = backupPath
	}

	// Process each task
	for _, task := range tasks {
		migrated, err := m.migrateTask(task)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Task %s: %s", task.Key, err.Error()))
			continue
		}

		if migrated {
			result.TasksMigrated++
			result.MigratedTasks = append(result.MigratedTasks, task.Key)

			// Save the migrated task if not dry run
			if !dryRun {
				if err := m.repository.Save(context.Background(), task); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Failed to save task %s: %s", task.Key, err.Error()))
					continue
				}
			}
		} else {
			result.TasksSkipped++
			result.SkippedTasks = append(result.SkippedTasks, task.Key)
		}
	}

	result.Duration = time.Since(start)

	// Add statistics
	result.Statistics["success_rate"] = float64(result.TasksMigrated) / float64(result.TasksProcessed) * 100
	result.Statistics["migration_time"] = result.Duration.String()

	return result, nil
}

// migrateTask migrates a single task from string to array format
func (m *SprintMigration) migrateTask(task *domain.Task) (bool, error) {
	// Check if task has no sprint data at all
	if task.Sprint == "" {
		return false, nil // No sprint data to migrate
	}

	// Check if task has been explicitly migrated by checking if the Sprint field
	// exactly matches the joined Sprints array (indicating explicit migration)
	if len(task.Sprints) > 0 {
		expectedSprintString := strings.Join(task.Sprints, ", ")
		if task.Sprint == expectedSprintString {
			// This indicates the task was explicitly migrated (Sprint field was set by SetSprints)
			return false, nil // No migration needed
		}
	}

	// At this point, we have a task with Sprint data that needs migration
	// Parse sprint string and populate sprints array
	sprints := task.GetSprints() // This will parse the Sprint string
	if len(sprints) == 0 {
		return false, fmt.Errorf("failed to parse sprint string: %s", task.Sprint)
	}

	// Set the sprints array (this will also update the Sprint field for consistency)
	task.SetSprints(sprints)

	return true, nil
}

// createBackup creates a backup of all tasks before migration
func (m *SprintMigration) createBackup(tasks []*domain.Task) (string, error) {
	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(m.backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create backup file with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s/tasks_backup_%s.json", m.backupDir, timestamp)

	// Convert tasks to map for JSON serialization
	taskMap := make(map[string]*domain.Task)
	for _, task := range tasks {
		taskMap[task.Key] = task
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(taskMap, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal tasks: %w", err)
	}

	// Write backup file
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	return backupPath, nil
}

// ValidateCompatibility checks if the current tasks are compatible with the new format
func (m *SprintMigration) ValidateCompatibility(_ context.Context) error {
	tasks, err := m.repository.FindAll(context.Background())
	if err != nil {
		return fmt.Errorf("failed to fetch tasks: %w", err)
	}

	var errors []string

	for _, task := range tasks {
		// Check if task has valid sprint data
		if task.Sprint == "" && len(task.Sprints) == 0 {
			errors = append(errors, fmt.Sprintf("Task %s has no sprint data", task.Key))
			continue
		}

		// Check if sprint string can be parsed
		if task.Sprint != "" {
			sprints := task.GetSprints()
			if len(sprints) == 0 {
				errors = append(errors, fmt.Sprintf("Task %s has unparseable sprint string: %s", task.Key, task.Sprint))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	return nil
}

// GetMigrationStats returns statistics about the current migration state
func (m *SprintMigration) GetMigrationStats(_ context.Context) (*Result, error) {
	tasks, err := m.repository.FindAll(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tasks: %w", err)
	}

	result := &Result{
		TasksProcessed: len(tasks),
		Statistics:     make(map[string]interface{}),
	}

	var alreadyMigrated int
	var needsMigration int

	for _, task := range tasks {
		// Check if task has no sprint data at all
		if task.Sprint == "" {
			continue // Skip tasks with no sprint data
		}

		// Check if task has been explicitly migrated by checking if the Sprint field
		// exactly matches the joined Sprints array (indicating explicit migration)
		if len(task.Sprints) > 0 {
			expectedSprintString := strings.Join(task.Sprints, ", ")
			if task.Sprint == expectedSprintString {
				// This indicates the task was explicitly migrated
				alreadyMigrated++
			} else {
				// Task has sprints but they don't match - needs migration
				needsMigration++
			}
		} else {
			// Task has Sprint but no Sprints array - needs migration
			needsMigration++
		}
	}

	result.Statistics["already_migrated"] = alreadyMigrated
	result.Statistics["needs_migration"] = needsMigration
	result.Statistics["migration_percentage"] = float64(alreadyMigrated) / float64(len(tasks)) * 100

	return result, nil
}

// RollbackMigration rolls back the migration using the backup file
func (m *SprintMigration) RollbackMigration(_ context.Context) error {
	// Find the most recent backup file
	files, err := os.ReadDir(m.backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	var latestBackup string
	var latestTime time.Time

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Parse timestamp from filename
		name := file.Name()
		if len(name) < 20 || !strings.HasPrefix(name, "tasks_backup_") {
			continue
		}

		timestampStr := name[13:28] // Extract timestamp part
		timestamp, err := time.Parse("20060102_150405", timestampStr)
		if err != nil {
			continue
		}

		if timestamp.After(latestTime) {
			latestTime = timestamp
			latestBackup = fmt.Sprintf("%s/%s", m.backupDir, name)
		}
	}

	if latestBackup == "" {
		return fmt.Errorf("no backup file found")
	}

	// Read backup file
	data, err := os.ReadFile(latestBackup)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Unmarshal tasks
	var taskMap map[string]*domain.Task
	if err := json.Unmarshal(data, &taskMap); err != nil {
		return fmt.Errorf("failed to unmarshal backup: %w", err)
	}

	// Restore tasks
	for _, task := range taskMap {
		if err := m.repository.Save(context.Background(), task); err != nil {
			return fmt.Errorf("failed to restore task %s: %w", task.Key, err)
		}
	}

	return nil
}

// DefaultTasksFilePath returns the default path for tasks.json
func DefaultTasksFilePath() string {
	return filepath.Join(".assetcap", "tasks.json")
}
