package jira

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

// MockHTTPClient is a mock implementation of HTTPClient
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return nil, nil
}

// MockClient is a mock implementation of Client
type MockClient struct {
	FetchTasksFunc   func(ctx context.Context, project, sprint string) ([]*domain.Task, error)
	UpdateLabelsFunc func(ctx context.Context, issueKey string, labels []string) error
}

func (m *MockClient) FetchTasks(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
	if m.FetchTasksFunc != nil {
		return m.FetchTasksFunc(ctx, project, sprint)
	}
	return nil, nil
}

func (m *MockClient) UpdateLabels(ctx context.Context, issueKey string, labels []string) error {
	if m.UpdateLabelsFunc != nil {
		return m.UpdateLabelsFunc(ctx, issueKey, labels)
	}
	return nil
}

type mockClient struct {
	fetchTasksFunc   func(ctx context.Context, project, sprint string) ([]*domain.Task, error)
	updateLabelsFunc func(ctx context.Context, issueKey string, labels []string) error
}

func (m *mockClient) FetchTasks(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
	if m.fetchTasksFunc != nil {
		return m.fetchTasksFunc(ctx, project, sprint)
	}
	return nil, nil
}

func (m *mockClient) UpdateLabels(ctx context.Context, issueKey string, labels []string) error {
	if m.updateLabelsFunc != nil {
		return m.updateLabelsFunc(ctx, issueKey, labels)
	}
	return nil
}

func TestNewRepository(t *testing.T) {
	// Save the original NewClient function and restore it after the test
	originalNewClient := NewClient
	defer func() { NewClient = originalNewClient }()

	tests := []struct {
		name         string
		mockSetup    func()
		wantErr      bool
		errorMessage string
		wantInstance *TaskRepository
	}{
		{
			name:         "successful setup",
			mockSetup:    func() {},
			wantErr:      false,
			errorMessage: "",
			wantInstance: nil,
		},
		{
			name:         "client error",
			mockSetup:    func() { NewClient = func(_ *Config) (Client, error) { return nil, errors.New("client error") } },
			wantErr:      true,
			errorMessage: "failed to create Jira client: client error",
			wantInstance: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()
			repo, err := NewRepository()
			if tt.wantErr {
				assert.Error(t, err, "Should return error")
				assert.Equal(t, tt.errorMessage, err.Error(), "Error message should match")
				assert.Nil(t, repo, "Repository should be nil")
			} else {
				assert.NoError(t, err, "Should not return error")
				assert.NotNil(t, repo, "Repository should not be nil")
			}
		})
	}
}

func TestRepository_FindByProjectAndSprint(t *testing.T) {
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
				BaseURL: "https://test.atlassian.net",
				Email:   "test@example.com",
				Token:   "test-token",
			}, nil
		}
		NewClient = func(config *Config) (Client, error) {
			return mockClient, nil
		}

		repo, err := NewRepository()
		require.NoError(t, err, "Should not return error")

		tasks, err := repo.FindByProjectAndSprint(ctx, "TEST", "Sprint 1")
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
				BaseURL: "https://test.atlassian.net",
				Email:   "test@example.com",
				Token:   "test-token",
			}, nil
		}
		NewClient = func(config *Config) (Client, error) {
			return mockClient, nil
		}

		repo, err := NewRepository()
		require.NoError(t, err, "Should not return error")

		tasks, err := repo.FindByProjectAndSprint(ctx, "TEST", "Sprint 1")
		require.NoError(t, err, "Should not return error")
		assert.Equal(t, expectedTasks, tasks, "Tasks should match")
	})
}

func TestRepository_NotImplementedMethods(t *testing.T) {
	ctx := context.Background()

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
			BaseURL: "https://test.atlassian.net",
			Email:   "test@example.com",
			Token:   "test-token",
		}, nil
	}
	NewClient = func(config *Config) (Client, error) {
		return mockClient, nil
	}

	repo, err := NewRepository()
	require.NoError(t, err, "Should not return error")

	t.Run("Save", func(t *testing.T) {
		err := repo.Save(ctx, &domain.Task{})
		require.Error(t, err, "Should return error")
		assert.Equal(t, "not implemented", err.Error(), "Error message should match")
	})

	t.Run("FindByKey", func(t *testing.T) {
		task, err := repo.FindByKey(ctx, "TEST-1")
		require.Error(t, err, "Should return error")
		assert.Nil(t, task, "Task should be nil")
		assert.Equal(t, "not implemented", err.Error(), "Error message should match")
	})

	t.Run("FindByProject", func(t *testing.T) {
		tasks, err := repo.FindByProject(ctx, "TEST")
		require.Error(t, err, "Should return error")
		assert.Nil(t, tasks, "Tasks should be nil")
		assert.Equal(t, "not implemented", err.Error(), "Error message should match")
	})

	t.Run("FindBySprint", func(t *testing.T) {
		tasks, err := repo.FindBySprint(ctx, "Sprint 1")
		require.Error(t, err, "Should return error")
		assert.Nil(t, tasks, "Tasks should be nil")
		assert.Equal(t, "not implemented", err.Error(), "Error message should match")
	})

	t.Run("FindByPlatform", func(t *testing.T) {
		tasks, err := repo.FindByPlatform(ctx, "JIRA")
		require.Error(t, err, "Should return error")
		assert.Nil(t, tasks, "Tasks should be nil")
		assert.Equal(t, "not implemented", err.Error(), "Error message should match")
	})

	t.Run("FindAll", func(t *testing.T) {
		tasks, err := repo.FindAll(ctx)
		require.Error(t, err, "Should return error")
		assert.Nil(t, tasks, "Tasks should be nil")
		assert.Equal(t, "not implemented", err.Error(), "Error message should match")
	})

	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(ctx, "TEST-1")
		require.Error(t, err, "Should return error")
		assert.Equal(t, "not implemented", err.Error(), "Error message should match")
	})

	t.Run("DeleteByProjectAndSprint", func(t *testing.T) {
		err := repo.DeleteByProjectAndSprint(ctx, "TEST", "Sprint 1")
		require.Error(t, err, "Should return error")
		assert.Equal(t, "not implemented", err.Error(), "Error message should match")
	})
}

func TestJiraTaskRepository_UpdateLabels(t *testing.T) {
	tests := []struct {
		name          string
		taskKey       string
		labels        []string
		mockError     error
		expectedError bool
	}{
		{
			name:          "successful label update",
			taskKey:       "TEST-1",
			labels:        []string{"development"},
			mockError:     nil,
			expectedError: false,
		},
		{
			name:          "failed label update",
			taskKey:       "TEST-1",
			labels:        []string{"development"},
			mockError:     fmt.Errorf("failed to update labels"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			mockClient := &mockClient{
				updateLabelsFunc: func(ctx context.Context, issueKey string, labels []string) error {
					return tt.mockError
				},
			}

			// Create repository with mock client
			repo := &TaskRepository{
				client: mockClient,
			}

			// Test UpdateLabels
			err := repo.UpdateLabels(context.Background(), tt.taskKey, tt.labels)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
