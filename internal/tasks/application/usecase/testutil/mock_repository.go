package testutil

import (
	"context"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// MockTaskRepository is a mock implementation of TaskRepository for testing
type MockTaskRepository struct {
	findByProjectAndSprintFunc func(ctx context.Context, project, sprint string) ([]*domain.Task, error)
}

// NewMockTaskRepository creates a new mock task repository
func NewMockTaskRepository() *MockTaskRepository {
	return &MockTaskRepository{}
}

// Reset resets all mock functions
func (m *MockTaskRepository) Reset() {
	m.findByProjectAndSprintFunc = nil
}

// SetFindByProjectAndSprintFunc sets the mock function for FindByProjectAndSprint
func (m *MockTaskRepository) SetFindByProjectAndSprintFunc(f func(ctx context.Context, project, sprint string) ([]*domain.Task, error)) {
	m.findByProjectAndSprintFunc = f
}

// Save saves a task to the repository
func (m *MockTaskRepository) Save(ctx context.Context, task *domain.Task) error {
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
