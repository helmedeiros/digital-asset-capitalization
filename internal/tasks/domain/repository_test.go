package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockTaskRepository is a mock implementation of TaskRepository for testing
type MockTaskRepository struct {
	tasks map[string]*Task
}

func NewMockTaskRepository() *MockTaskRepository {
	return &MockTaskRepository{
		tasks: make(map[string]*Task),
	}
}

func (m *MockTaskRepository) Save(task *Task) error {
	m.tasks[task.Key] = task
	return nil
}

func (m *MockTaskRepository) FindByKey(key string) (*Task, error) {
	if task, exists := m.tasks[key]; exists {
		return task, nil
	}
	return nil, errors.New("task not found")
}

func (m *MockTaskRepository) FindByProjectAndSprint(project, sprint string) ([]*Task, error) {
	var result []*Task
	for _, task := range m.tasks {
		if task.Project == project && task.Sprint == sprint {
			result = append(result, task)
		}
	}
	return result, nil
}

func (m *MockTaskRepository) FindByProject(project string) ([]*Task, error) {
	var result []*Task
	for _, task := range m.tasks {
		if task.Project == project {
			result = append(result, task)
		}
	}
	return result, nil
}

func (m *MockTaskRepository) FindBySprint(sprint string) ([]*Task, error) {
	var result []*Task
	for _, task := range m.tasks {
		if task.Sprint == sprint {
			result = append(result, task)
		}
	}
	return result, nil
}

func (m *MockTaskRepository) FindByPlatform(platform string) ([]*Task, error) {
	var result []*Task
	for _, task := range m.tasks {
		if task.Platform == platform {
			result = append(result, task)
		}
	}
	return result, nil
}

func (m *MockTaskRepository) FindAll() ([]*Task, error) {
	result := make([]*Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		result = append(result, task)
	}
	return result, nil
}

func (m *MockTaskRepository) Delete(key string) error {
	if _, exists := m.tasks[key]; exists {
		delete(m.tasks, key)
		return nil
	}
	return errors.New("task not found")
}

func (m *MockTaskRepository) DeleteByProjectAndSprint(project, sprint string) error {
	for key, task := range m.tasks {
		if task.Project == project && task.Sprint == sprint {
			delete(m.tasks, key)
		}
	}
	return nil
}

func TestTaskRepository(t *testing.T) {
	repo := NewMockTaskRepository()

	// Create test tasks
	task1, err := NewTask("TEST-1", "Task 1", "TEST", "Sprint 1", "jira")
	require.NoError(t, err)

	task2, err := NewTask("TEST-2", "Task 2", "TEST", "Sprint 1", "jira")
	require.NoError(t, err)

	task3, err := NewTask("TEST-3", "Task 3", "TEST", "Sprint 2", "jira")
	require.NoError(t, err)

	t.Run("save and find by key", func(t *testing.T) {
		err := repo.Save(task1)
		require.NoError(t, err)

		found, err := repo.FindByKey("TEST-1")
		require.NoError(t, err)
		assert.Equal(t, task1.Key, found.Key)
		assert.Equal(t, task1.Summary, found.Summary)
	})

	t.Run("find by project and sprint", func(t *testing.T) {
		err := repo.Save(task2)
		require.NoError(t, err)

		tasks, err := repo.FindByProjectAndSprint("TEST", "Sprint 1")
		require.NoError(t, err)
		assert.Len(t, tasks, 2)
	})

	t.Run("find by project", func(t *testing.T) {
		err := repo.Save(task3)
		require.NoError(t, err)

		tasks, err := repo.FindByProject("TEST")
		require.NoError(t, err)
		assert.Len(t, tasks, 3)
	})

	t.Run("find by sprint", func(t *testing.T) {
		tasks, err := repo.FindBySprint("Sprint 1")
		require.NoError(t, err)
		assert.Len(t, tasks, 2)
	})

	t.Run("find by platform", func(t *testing.T) {
		tasks, err := repo.FindByPlatform("jira")
		require.NoError(t, err)
		assert.Len(t, tasks, 3)
	})

	t.Run("find all", func(t *testing.T) {
		tasks, err := repo.FindAll()
		require.NoError(t, err)
		assert.Len(t, tasks, 3)
	})

	t.Run("delete by key", func(t *testing.T) {
		err := repo.Delete("TEST-1")
		require.NoError(t, err)

		_, err = repo.FindByKey("TEST-1")
		assert.Error(t, err)
	})

	t.Run("delete by project and sprint", func(t *testing.T) {
		err := repo.DeleteByProjectAndSprint("TEST", "Sprint 1")
		require.NoError(t, err)

		tasks, err := repo.FindByProjectAndSprint("TEST", "Sprint 1")
		require.NoError(t, err)
		assert.Empty(t, tasks)
	})
}
