package testutil

import (
	"context"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// MockTaskRepository is a mock implementation of TaskRepository for testing
type MockTaskRepository struct {
	findByProjectAndSprintFunc func(ctx context.Context, project, sprint string) ([]*domain.Task, error)
	saveFunc                   func(ctx context.Context, task *domain.Task) error
}

// NewMockTaskRepository creates a new mock task repository
func NewMockTaskRepository() *MockTaskRepository {
	return &MockTaskRepository{}
}

// Reset resets all mock functions
func (m *MockTaskRepository) Reset() {
	m.findByProjectAndSprintFunc = nil
	m.saveFunc = nil
}

// SetFindByProjectAndSprintFunc sets the mock function for FindByProjectAndSprint
func (m *MockTaskRepository) SetFindByProjectAndSprintFunc(f func(ctx context.Context, project, sprint string) ([]*domain.Task, error)) {
	m.findByProjectAndSprintFunc = f
}

// SetSaveFunc sets the mock function for Save
func (m *MockTaskRepository) SetSaveFunc(f func(ctx context.Context, task *domain.Task) error) {
	m.saveFunc = f
}

// Save saves a task to the repository
func (m *MockTaskRepository) Save(ctx context.Context, task *domain.Task) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, task)
	}
	return nil
}

// FindByKey finds a task by its key
func (m *MockTaskRepository) FindByKey(ctx context.Context, key string) (*domain.Task, error) {
	return nil, nil
}

// FindByProjectAndSprint finds tasks by project and sprint
func (m *MockTaskRepository) FindByProjectAndSprint(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
	if m.findByProjectAndSprintFunc != nil {
		return m.findByProjectAndSprintFunc(ctx, project, sprint)
	}
	return nil, nil
}

// FindByProject finds tasks by project
func (m *MockTaskRepository) FindByProject(ctx context.Context, project string) ([]*domain.Task, error) {
	return nil, nil
}

// FindBySprint finds tasks by sprint
func (m *MockTaskRepository) FindBySprint(ctx context.Context, sprint string) ([]*domain.Task, error) {
	return nil, nil
}

// FindByPlatform finds tasks by platform
func (m *MockTaskRepository) FindByPlatform(ctx context.Context, platform string) ([]*domain.Task, error) {
	return nil, nil
}

// FindAll returns all tasks
func (m *MockTaskRepository) FindAll(ctx context.Context) ([]*domain.Task, error) {
	return nil, nil
}

// Delete deletes a task by key
func (m *MockTaskRepository) Delete(ctx context.Context, key string) error {
	return nil
}

// DeleteByProjectAndSprint deletes tasks by project and sprint
func (m *MockTaskRepository) DeleteByProjectAndSprint(ctx context.Context, project, sprint string) error {
	return nil
}

// Ensure MockTaskRepository implements TaskRepository
var _ ports.TaskRepository = (*MockTaskRepository)(nil)

// MockTaskClassifier is a mock implementation of TaskClassifier
type MockTaskClassifier struct {
	classifyTasksFunc func(tasks []*domain.Task) (map[string]domain.WorkType, error)
	classifyTaskFunc  func(task *domain.Task) (domain.WorkType, error)
}

// NewMockTaskClassifier creates a new mock task classifier
func NewMockTaskClassifier() *MockTaskClassifier {
	return &MockTaskClassifier{}
}

// Reset resets the mock classifier
func (m *MockTaskClassifier) Reset() {
	m.classifyTasksFunc = nil
	m.classifyTaskFunc = nil
}

// SetClassifyTasksFunc sets the function to be called when ClassifyTasks is called
func (m *MockTaskClassifier) SetClassifyTasksFunc(f func(tasks []*domain.Task) (map[string]domain.WorkType, error)) {
	m.classifyTasksFunc = f
}

// SetClassifyTaskFunc sets the function to be called when ClassifyTask is called
func (m *MockTaskClassifier) SetClassifyTaskFunc(f func(task *domain.Task) (domain.WorkType, error)) {
	m.classifyTaskFunc = f
}

// ClassifyTasks implements TaskClassifier.ClassifyTasks
func (m *MockTaskClassifier) ClassifyTasks(tasks []*domain.Task) (map[string]domain.WorkType, error) {
	if m.classifyTasksFunc != nil {
		return m.classifyTasksFunc(tasks)
	}
	return nil, nil
}

// ClassifyTask implements TaskClassifier.ClassifyTask
func (m *MockTaskClassifier) ClassifyTask(task *domain.Task) (domain.WorkType, error) {
	if m.classifyTaskFunc != nil {
		return m.classifyTaskFunc(task)
	}
	return "", nil
}

// MockUserInput is a mock implementation of UserInput
type MockUserInput struct {
	confirmFunc func(prompt string, args ...interface{}) (bool, error)
}

// NewMockUserInput creates a new mock user input
func NewMockUserInput() *MockUserInput {
	return &MockUserInput{}
}

// Reset resets the mock user input
func (m *MockUserInput) Reset() {
	m.confirmFunc = nil
}

// SetConfirmFunc sets the function to be called when Confirm is called
func (m *MockUserInput) SetConfirmFunc(f func(prompt string, args ...interface{}) (bool, error)) {
	m.confirmFunc = f
}

// Confirm implements UserInput.Confirm
func (m *MockUserInput) Confirm(prompt string, args ...interface{}) (bool, error) {
	if m.confirmFunc != nil {
		return m.confirmFunc(prompt, args...)
	}
	return false, nil
}
