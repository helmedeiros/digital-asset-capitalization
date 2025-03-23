package ports

import (
	"context"
	"fmt"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRepository implements the Repository interface for testing
type mockRepository struct {
	tasks map[string]*domain.Task
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		tasks: make(map[string]*domain.Task),
	}
}

func (m *mockRepository) Save(ctx context.Context, task *domain.Task) error {
	m.tasks[task.Key] = task
	return nil
}

func (m *mockRepository) FindByKey(ctx context.Context, key string) (*domain.Task, error) {
	if task, exists := m.tasks[key]; exists {
		return task, nil
	}
	return nil, nil
}

func (m *mockRepository) FindByProjectAndSprint(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
	var tasks []*domain.Task
	for _, task := range m.tasks {
		if task.Project == project && task.Sprint == sprint {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func (m *mockRepository) FindByProject(ctx context.Context, project string) ([]*domain.Task, error) {
	var tasks []*domain.Task
	for _, task := range m.tasks {
		if task.Project == project {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func (m *mockRepository) FindBySprint(ctx context.Context, sprint string) ([]*domain.Task, error) {
	var tasks []*domain.Task
	for _, task := range m.tasks {
		if task.Sprint == sprint {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func (m *mockRepository) FindByPlatform(ctx context.Context, platform string) ([]*domain.Task, error) {
	var tasks []*domain.Task
	for _, task := range m.tasks {
		if task.Platform == platform {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func (m *mockRepository) FindAll(ctx context.Context) ([]*domain.Task, error) {
	tasks := make([]*domain.Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (m *mockRepository) Delete(ctx context.Context, key string) error {
	delete(m.tasks, key)
	return nil
}

func (m *mockRepository) DeleteByProjectAndSprint(ctx context.Context, project, sprint string) error {
	for key, task := range m.tasks {
		if task.Project == project && task.Sprint == sprint {
			delete(m.tasks, key)
		}
	}
	return nil
}

// UpdateLabels updates the labels of a task in the remote repository
func (m *mockRepository) UpdateLabels(ctx context.Context, taskKey string, labels []string) error {
	if taskKey == "" {
		return fmt.Errorf("task key cannot be empty")
	}

	task, exists := m.tasks[taskKey]
	if !exists {
		return fmt.Errorf("task %s not found", taskKey)
	}

	task.Labels = labels
	return nil
}

// Ensure mockRepository implements Repository
var _ TaskRepository = (*mockRepository)(nil)

func TestRepositoryOperations(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()

	// Create test tasks
	task1, err := domain.NewTask("TASK-1", "Test Task 1", "PROJECT-1", "SPRINT-1", "JIRA")
	require.NoError(t, err)
	require.NotNil(t, task1)

	task2, err := domain.NewTask("TASK-2", "Test Task 2", "PROJECT-1", "SPRINT-1", "JIRA")
	require.NoError(t, err)
	require.NotNil(t, task2)

	// Test Save
	err = repo.Save(ctx, task1)
	require.NoError(t, err)

	err = repo.Save(ctx, task2)
	require.NoError(t, err)

	// Test FindByKey
	found, err := repo.FindByKey(ctx, "TASK-1")
	require.NoError(t, err)
	assert.Equal(t, task1, found)

	// Test FindByProjectAndSprint
	tasks, err := repo.FindByProjectAndSprint(ctx, "PROJECT-1", "SPRINT-1")
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
	assert.Contains(t, tasks, task1)
	assert.Contains(t, tasks, task2)

	// Test FindByProject
	tasks, err = repo.FindByProject(ctx, "PROJECT-1")
	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	// Test FindBySprint
	tasks, err = repo.FindBySprint(ctx, "SPRINT-1")
	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	// Test FindByPlatform
	tasks, err = repo.FindByPlatform(ctx, "JIRA")
	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	// Test FindAll
	tasks, err = repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, tasks, 2)

	// Test Delete
	err = repo.Delete(ctx, "TASK-1")
	require.NoError(t, err)

	tasks, err = repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)

	// Test DeleteByProjectAndSprint
	err = repo.DeleteByProjectAndSprint(ctx, "PROJECT-1", "SPRINT-1")
	require.NoError(t, err)

	tasks, err = repo.FindAll(ctx)
	require.NoError(t, err)
	assert.Empty(t, tasks)

	// Test UpdateLabels
	// First save a task to update
	task3, err := domain.NewTask("TASK-3", "Test Task 3", "PROJECT-2", "SPRINT-2", "JIRA")
	require.NoError(t, err)
	require.NotNil(t, task3)

	err = repo.Save(ctx, task3)
	require.NoError(t, err)

	// Test updating labels
	newLabels := []string{"label1", "label2"}
	err = repo.UpdateLabels(ctx, "TASK-3", newLabels)
	require.NoError(t, err)

	// Verify the labels were updated
	updatedTask, err := repo.FindByKey(ctx, "TASK-3")
	require.NoError(t, err)
	assert.Equal(t, newLabels, updatedTask.Labels)

	// Test updating labels for non-existent task
	err = repo.UpdateLabels(ctx, "NON-EXISTENT", newLabels)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task NON-EXISTENT not found")

	// Test updating labels with empty task key
	err = repo.UpdateLabels(ctx, "", newLabels)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task key cannot be empty")
}
