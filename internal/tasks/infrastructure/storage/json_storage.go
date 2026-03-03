package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// JSONStorage implements TaskRepository using JSON files
type JSONStorage struct {
	dir  string
	file string
}

// NewJSONStorage creates a new JSON storage instance
func NewJSONStorage(dir, file string) *JSONStorage {
	return &JSONStorage{
		dir:  dir,
		file: file,
	}
}

// Save persists a task
func (s *JSONStorage) Save(_ context.Context, task *domain.Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	// Load existing tasks
	tasks, err := s.loadTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// Update or add the task
	tasks[task.Key] = task

	// Save back to file
	return s.saveTasks(tasks)
}

// FindByKey retrieves a task by its key
func (s *JSONStorage) FindByKey(_ context.Context, key string) (*domain.Task, error) {
	if key == "" {
		return nil, fmt.Errorf("task key cannot be empty")
	}

	tasks, err := s.loadTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	task, exists := tasks[key]
	if !exists {
		return nil, fmt.Errorf("task %s not found", key)
	}

	return task, nil
}

// FindByProjectAndSprint retrieves tasks for a specific project and sprint
func (s *JSONStorage) FindByProjectAndSprint(_ context.Context, project, sprint string) ([]*domain.Task, error) {
	tasks, err := s.loadTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	var result []*domain.Task
	for _, task := range tasks {
		if task.Project == project && task.HasSprint(sprint) {
			result = append(result, task)
		}
	}

	return result, nil
}

// FindByProject retrieves all tasks for a specific project
func (s *JSONStorage) FindByProject(_ context.Context, project string) ([]*domain.Task, error) {
	tasks, err := s.loadTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	var result []*domain.Task
	for _, task := range tasks {
		if task.Project == project {
			result = append(result, task)
		}
	}

	return result, nil
}

// FindBySprint retrieves all tasks for a specific sprint
func (s *JSONStorage) FindBySprint(_ context.Context, sprint string) ([]*domain.Task, error) {
	tasks, err := s.loadTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	var result []*domain.Task
	for _, task := range tasks {
		if task.HasSprint(sprint) {
			result = append(result, task)
		}
	}

	return result, nil
}

// FindByPlatform retrieves all tasks for a specific platform
func (s *JSONStorage) FindByPlatform(_ context.Context, platform string) ([]*domain.Task, error) {
	tasks, err := s.loadTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	var result []*domain.Task
	for _, task := range tasks {
		if task.Platform == platform {
			result = append(result, task)
		}
	}

	return result, nil
}

// FindAll retrieves all tasks
func (s *JSONStorage) FindAll(_ context.Context) ([]*domain.Task, error) {
	tasks, err := s.loadTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to load tasks: %w", err)
	}

	result := make([]*domain.Task, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, task)
	}

	return result, nil
}

// Delete removes a task
func (s *JSONStorage) Delete(_ context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("task key cannot be empty")
	}

	tasks, err := s.loadTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	if _, exists := tasks[key]; !exists {
		return fmt.Errorf("task %s not found", key)
	}

	delete(tasks, key)
	return s.saveTasks(tasks)
}

// DeleteByProjectAndSprint removes all tasks for a specific project and sprint
func (s *JSONStorage) DeleteByProjectAndSprint(_ context.Context, project, sprint string) error {
	tasks, err := s.loadTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// Create a new map with tasks to keep
	newTasks := make(map[string]*domain.Task)
	for key, task := range tasks {
		if task.Project != project || !task.HasSprint(sprint) {
			newTasks[key] = task
		}
	}

	return s.saveTasks(newTasks)
}

// UpdateLabels updates the labels of a task in the local repository
// using add/remove semantics to stay consistent with the port interface
func (s *JSONStorage) UpdateLabels(_ context.Context, taskKey string, addLabels, removeLabels []string) error {
	if taskKey == "" {
		return fmt.Errorf("task key cannot be empty")
	}

	tasks, err := s.loadTasks()
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	task, exists := tasks[taskKey]
	if !exists {
		return fmt.Errorf("task %s not found", taskKey)
	}

	// Build a set of labels to remove for fast lookup
	removeSet := make(map[string]bool, len(removeLabels))
	for _, label := range removeLabels {
		removeSet[label] = true
	}

	// Filter out removed labels
	filtered := make([]string, 0, len(task.Labels))
	for _, label := range task.Labels {
		if !removeSet[label] {
			filtered = append(filtered, label)
		}
	}

	// Add new labels (avoid duplicates)
	existing := make(map[string]bool, len(filtered))
	for _, label := range filtered {
		existing[label] = true
	}
	for _, label := range addLabels {
		if !existing[label] {
			filtered = append(filtered, label)
		}
	}

	task.Labels = filtered
	return s.saveTasks(tasks)
}

// loadTasks loads all tasks from the JSON file
func (s *JSONStorage) loadTasks() (map[string]*domain.Task, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	filePath := filepath.Join(s.dir, s.file)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*domain.Task), nil
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var tasks map[string]*domain.Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tasks: %w", err)
	}

	return tasks, nil
}

// saveTasks saves all tasks to the JSON file
func (s *JSONStorage) saveTasks(tasks map[string]*domain.Task) error {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks: %w", err)
	}

	filePath := filepath.Join(s.dir, s.file)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Ensure JSONStorage implements TaskRepository
var _ ports.TaskRepository = (*JSONStorage)(nil)
