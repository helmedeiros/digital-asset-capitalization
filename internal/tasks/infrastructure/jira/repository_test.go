package jira

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockClient struct {
	fetchTasksFunc func(ctx context.Context, project, sprint string) ([]*domain.Task, error)
}

func (m *mockClient) FetchTasks(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
	return m.fetchTasksFunc(ctx, project, sprint)
}

func TestNewRepository(t *testing.T) {
	// Save the original functions and restore them after the test
	originalNewClient := NewClient
	originalNewConfig := NewConfig
	defer func() {
		NewClient = originalNewClient
		NewConfig = originalNewConfig
	}()

	// Create a mock client
	mockClient := &mockClient{}

	// Set up the mock functions
	NewConfig = func() (*Config, error) {
		return &Config{
			baseURL: "https://test.atlassian.net",
			email:   "test@example.com",
			token:   "test-token",
		}, nil
	}
	NewClient = func(config *Config) (Client, error) {
		return mockClient, nil
	}

	repo, err := NewRepository()
	require.NoError(t, err, "Should not return error")
	assert.NotNil(t, repo, "Repository should not be nil")
}

func TestRepository_FetchTasks(t *testing.T) {
	ctx := context.Background()

	// Save the original functions and restore them after the test
	originalNewClient := NewClient
	originalNewConfig := NewConfig
	defer func() {
		NewClient = originalNewClient
		NewConfig = originalNewConfig
	}()

	t.Run("client error", func(t *testing.T) {
		// Set up the mock client
		mockClient := &mockClient{
			fetchTasksFunc: func(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
				return nil, errors.New("client error")
			},
		}

		// Set up the mock functions
		NewConfig = func() (*Config, error) {
			return &Config{
				baseURL: "https://test.atlassian.net",
				email:   "test@example.com",
				token:   "test-token",
			}, nil
		}
		NewClient = func(config *Config) (Client, error) {
			return mockClient, nil
		}

		repo, err := NewRepository()
		require.NoError(t, err, "Should not return error")

		tasks, err := repo.FetchTasks(ctx, "TEST", "Sprint 1")
		require.Error(t, err, "Should return error")
		assert.Nil(t, tasks, "Tasks should be nil")
		assert.Contains(t, err.Error(), "client error", "Error message should be propagated")
	})

	t.Run("successful fetch", func(t *testing.T) {
		now := time.Now()
		expectedTasks := []*domain.Task{
			{
				Key:       "TEST-1",
				Summary:   "Test Task",
				Status:    domain.TaskStatusInProgress,
				Project:   "TEST",
				Sprint:    "Sprint 1",
				Platform:  "JIRA",
				CreatedAt: now,
				UpdatedAt: now,
				Version:   1,
			},
		}

		// Set up the mock client
		mockClient := &mockClient{
			fetchTasksFunc: func(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
				assert.Equal(t, "TEST", project, "Project should match")
				assert.Equal(t, "Sprint 1", sprint, "Sprint should match")
				return expectedTasks, nil
			},
		}

		// Set up the mock functions
		NewConfig = func() (*Config, error) {
			return &Config{
				baseURL: "https://test.atlassian.net",
				email:   "test@example.com",
				token:   "test-token",
			}, nil
		}
		NewClient = func(config *Config) (Client, error) {
			return mockClient, nil
		}

		repo, err := NewRepository()
		require.NoError(t, err, "Should not return error")

		tasks, err := repo.FetchTasks(ctx, "TEST", "Sprint 1")
		require.NoError(t, err, "Should not return error")
		assert.Equal(t, expectedTasks, tasks, "Tasks should match")
	})
}

func TestRepository_NotImplementedMethods(t *testing.T) {
	// Save the original functions and restore them after the test
	originalNewClient := NewClient
	originalNewConfig := NewConfig
	defer func() {
		NewClient = originalNewClient
		NewConfig = originalNewConfig
	}()

	// Set up the mock client
	mockClient := &mockClient{}

	// Set up the mock functions
	NewConfig = func() (*Config, error) {
		return &Config{
			baseURL: "https://test.atlassian.net",
			email:   "test@example.com",
			token:   "test-token",
		}, nil
	}
	NewClient = func(config *Config) (Client, error) {
		return mockClient, nil
	}

	repo, err := NewRepository()
	require.NoError(t, err, "Should not return error")

	t.Run("Save", func(t *testing.T) {
		err := repo.Save(&domain.Task{})
		require.Error(t, err, "Should return error")
		assert.Equal(t, "not implemented", err.Error(), "Error message should match")
	})

	t.Run("FindByKey", func(t *testing.T) {
		task, err := repo.FindByKey("TEST-1")
		require.Error(t, err, "Should return error")
		assert.Nil(t, task, "Task should be nil")
		assert.Equal(t, "not implemented", err.Error(), "Error message should match")
	})

	t.Run("FindByProjectAndSprint", func(t *testing.T) {
		tasks, err := repo.FindByProjectAndSprint("TEST", "Sprint 1")
		require.Error(t, err, "Should return error")
		assert.Nil(t, tasks, "Tasks should be nil")
		assert.Equal(t, "not implemented", err.Error(), "Error message should match")
	})

	t.Run("FindByProject", func(t *testing.T) {
		tasks, err := repo.FindByProject("TEST")
		require.Error(t, err, "Should return error")
		assert.Nil(t, tasks, "Tasks should be nil")
		assert.Equal(t, "not implemented", err.Error(), "Error message should match")
	})

	t.Run("FindBySprint", func(t *testing.T) {
		tasks, err := repo.FindBySprint("Sprint 1")
		require.Error(t, err, "Should return error")
		assert.Nil(t, tasks, "Tasks should be nil")
		assert.Equal(t, "not implemented", err.Error(), "Error message should match")
	})

	t.Run("FindByPlatform", func(t *testing.T) {
		tasks, err := repo.FindByPlatform("JIRA")
		require.Error(t, err, "Should return error")
		assert.Nil(t, tasks, "Tasks should be nil")
		assert.Equal(t, "not implemented", err.Error(), "Error message should match")
	})

	t.Run("FindAll", func(t *testing.T) {
		tasks, err := repo.FindAll()
		require.Error(t, err, "Should return error")
		assert.Nil(t, tasks, "Tasks should be nil")
		assert.Equal(t, "not implemented", err.Error(), "Error message should match")
	})

	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete("TEST-1")
		require.Error(t, err, "Should return error")
		assert.Equal(t, "not implemented", err.Error(), "Error message should match")
	})

	t.Run("DeleteByProjectAndSprint", func(t *testing.T) {
		err := repo.DeleteByProjectAndSprint("TEST", "Sprint 1")
		require.Error(t, err, "Should return error")
		assert.Equal(t, "not implemented", err.Error(), "Error message should match")
	})
}
