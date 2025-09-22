package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application/usecase/testutil"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

func TestFetchTasksUseCase(t *testing.T) {
	// Create mock repositories
	remoteRepo := testutil.NewMockTaskRepository()
	localRepo := testutil.NewMockTaskRepository()
	useCase := NewFetchTasksUseCase(remoteRepo, localRepo, nil)

	// Create test tasks
	now := time.Now()
	testTasks := []*domain.Task{
		{
			Key:       "TEST-1",
			Summary:   "Test Task",
			Status:    domain.TaskStatusInProgress,
			Project:   "TEST",
			Sprint:    "Sprint 1",
			Platform:  "jira",
			CreatedAt: now,
			UpdatedAt: now,
			Version:   1,
		},
	}

	tests := []struct {
		name      string
		project   string
		sprint    string
		platform  string
		setupMock func()
		wantErr   bool
		errMsg    string
	}{
		{
			name:     "successful fetch",
			project:  "TEST",
			sprint:   "Sprint 1",
			platform: "jira",
			setupMock: func() {
				remoteRepo.SetFindByProjectAndSprintFunc(func(_ context.Context, project, sprint string) ([]*domain.Task, error) {
					assert.Equal(t, "TEST", project)
					assert.Equal(t, "Sprint 1", sprint)
					return testTasks, nil
				})
				localRepo.SetSaveFunc(func(_ context.Context, task *domain.Task) error {
					assert.Equal(t, testTasks[0].Key, task.Key)
					return nil
				})
			},
			wantErr: false,
		},
		{
			name:     "empty project",
			project:  "",
			sprint:   "Sprint 1",
			platform: "jira",
			wantErr:  true,
			errMsg:   "project is required",
		},
		{
			name:     "empty platform",
			project:  "TEST",
			sprint:   "Sprint 1",
			platform: "",
			wantErr:  true,
			errMsg:   "platform is required",
		},
		{
			name:     "remote repository error",
			project:  "TEST",
			sprint:   "Sprint 1",
			platform: "jira",
			setupMock: func() {
				remoteRepo.SetFindByProjectAndSprintFunc(func(_ context.Context, _, _ string) ([]*domain.Task, error) {
					return nil, errors.New("repository error")
				})
			},
			wantErr: true,
			errMsg:  "failed to fetch tasks",
		},
		{
			name:     "local repository error",
			project:  "TEST",
			sprint:   "Sprint 1",
			platform: "jira",
			setupMock: func() {
				remoteRepo.SetFindByProjectAndSprintFunc(func(_ context.Context, _, _ string) ([]*domain.Task, error) {
					return testTasks, nil
				})
				localRepo.SetSaveFunc(func(_ context.Context, _ *domain.Task) error {
					return errors.New("repository error")
				})
			},
			wantErr: true,
			errMsg:  "failed to save task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks before each test
			remoteRepo.Reset()
			localRepo.Reset()

			// Setup mock if needed
			if tt.setupMock != nil {
				tt.setupMock()
			}

			// Execute use case
			err := useCase.Execute(context.Background(), tt.project, tt.sprint, tt.platform)

			// Verify results
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestFetchTasksUseCase_ExecuteByKey(t *testing.T) {
	// Create mock repositories
	remoteRepo := testutil.NewMockTaskRepository()
	localRepo := testutil.NewMockTaskRepository()
	useCase := NewFetchTasksUseCase(remoteRepo, localRepo, nil)

	// Create test task
	now := time.Now()
	testTask := &domain.Task{
		Key:       "FN-1015",
		Summary:   "Enable rounding with new Journey Details Service",
		Status:    domain.TaskStatusDone,
		Project:   "FN",
		Sprint:    "The Hulk",
		Platform:  "jira",
		CreatedAt: now,
		UpdatedAt: now,
		Version:   1,
	}

	tests := []struct {
		name      string
		key       string
		platform  string
		setupMock func()
		wantErr   bool
		errMsg    string
	}{
		{
			name:     "successful fetch by key",
			key:      "FN-1015",
			platform: "jira",
			setupMock: func() {
				remoteRepo.SetFindByKeyFunc(func(_ context.Context, key string) (*domain.Task, error) {
					assert.Equal(t, "FN-1015", key)
					return testTask, nil
				})
				localRepo.SetSaveFunc(func(_ context.Context, task *domain.Task) error {
					assert.Equal(t, testTask.Key, task.Key)
					return nil
				})
			},
			wantErr: false,
		},
		{
			name:     "empty key",
			key:      "",
			platform: "jira",
			wantErr:  true,
			errMsg:   "task key is required",
		},
		{
			name:     "empty platform",
			key:      "FN-1015",
			platform: "",
			wantErr:  true,
			errMsg:   "platform is required",
		},
		{
			name:     "task not found in remote",
			key:      "FN-1015",
			platform: "jira",
			setupMock: func() {
				remoteRepo.SetFindByKeyFunc(func(_ context.Context, _ string) (*domain.Task, error) {
					return nil, errors.New("task not found")
				})
			},
			wantErr: true,
			errMsg:  "failed to fetch task",
		},
		{
			name:     "local repository save error",
			key:      "FN-1015",
			platform: "jira",
			setupMock: func() {
				remoteRepo.SetFindByKeyFunc(func(_ context.Context, _ string) (*domain.Task, error) {
					return testTask, nil
				})
				localRepo.SetSaveFunc(func(_ context.Context, _ *domain.Task) error {
					return errors.New("save error")
				})
			},
			wantErr: true,
			errMsg:  "failed to save task",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks before each test
			remoteRepo.Reset()
			localRepo.Reset()

			// Setup mock if needed
			if tt.setupMock != nil {
				tt.setupMock()
			}

			// Execute use case
			err := useCase.ExecuteByKey(context.Background(), tt.key, tt.platform)

			// Verify results
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestFetchTasksUseCase_GetRemoteRepository(t *testing.T) {
	// Create mock repositories
	remoteRepo := testutil.NewMockTaskRepository()
	localRepo := testutil.NewMockTaskRepository()
	useCase := NewFetchTasksUseCase(remoteRepo, localRepo, nil)

	// Test GetRemoteRepository method
	result := useCase.GetRemoteRepository()

	// Verify it returns the remote repository
	assert.Equal(t, remoteRepo, result)
	assert.NotNil(t, result)
}

func TestFetchTasksUseCase_EdgeCases(t *testing.T) {
	t.Run("should handle nil task from remote repository", func(t *testing.T) {
		// Create mock repositories
		remoteRepo := testutil.NewMockTaskRepository()
		localRepo := testutil.NewMockTaskRepository()
		useCase := NewFetchTasksUseCase(remoteRepo, localRepo, nil)

		remoteRepo.SetFindByKeyFunc(func(_ context.Context, _ string) (*domain.Task, error) {
			return nil, nil // Return nil task without error
		})

		// Execute use case
		err := useCase.ExecuteByKey(context.Background(), "TEST-1", "jira")

		// Should handle nil task gracefully
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch task")
	})

	t.Run("should handle empty sprint name", func(t *testing.T) {
		// Create mock repositories
		remoteRepo := testutil.NewMockTaskRepository()
		localRepo := testutil.NewMockTaskRepository()
		useCase := NewFetchTasksUseCase(remoteRepo, localRepo, nil)

		// Execute use case with empty sprint
		err := useCase.Execute(context.Background(), "TEST", "", "jira")

		// Should validate sprint name
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sprint is required")
	})

	t.Run("should handle no tasks found scenario", func(t *testing.T) {
		// Create mock repositories
		remoteRepo := testutil.NewMockTaskRepository()
		localRepo := testutil.NewMockTaskRepository()
		useCase := NewFetchTasksUseCase(remoteRepo, localRepo, nil)

		// Setup mock to return empty tasks
		remoteRepo.SetFindByProjectAndSprintFunc(func(_ context.Context, _, _ string) ([]*domain.Task, error) {
			return []*domain.Task{}, nil // Empty slice
		})

		// Execute use case
		err := useCase.Execute(context.Background(), "TEST", "Sprint1", "jira")

		// Should succeed even with no tasks
		assert.NoError(t, err)
	})

	t.Run("should handle multiple tasks successfully", func(t *testing.T) {
		// Create mock repositories
		remoteRepo := testutil.NewMockTaskRepository()
		localRepo := testutil.NewMockTaskRepository()
		useCase := NewFetchTasksUseCase(remoteRepo, localRepo, nil)

		// Create multiple test tasks
		now := time.Now()
		testTasks := []*domain.Task{
			{Key: "TEST-1", Summary: "Task 1", Platform: "jira", CreatedAt: now, UpdatedAt: now},
			{Key: "TEST-2", Summary: "Task 2", Platform: "jira", CreatedAt: now, UpdatedAt: now},
			{Key: "TEST-3", Summary: "Task 3", Platform: "jira", CreatedAt: now, UpdatedAt: now},
		}

		remoteRepo.SetFindByProjectAndSprintFunc(func(_ context.Context, _, _ string) ([]*domain.Task, error) {
			return testTasks, nil
		})

		// Track how many times save is called
		saveCount := 0
		localRepo.SetSaveFunc(func(_ context.Context, _ *domain.Task) error {
			saveCount++
			return nil
		})

		// Execute use case
		err := useCase.Execute(context.Background(), "TEST", "Sprint1", "jira")

		// Should save all tasks
		assert.NoError(t, err)
		assert.Equal(t, 3, saveCount)
	})

	t.Run("should handle partial save failures", func(t *testing.T) {
		// Create mock repositories
		remoteRepo := testutil.NewMockTaskRepository()
		localRepo := testutil.NewMockTaskRepository()
		useCase := NewFetchTasksUseCase(remoteRepo, localRepo, nil)

		// Create multiple test tasks
		now := time.Now()
		testTasks := []*domain.Task{
			{Key: "TEST-1", Summary: "Task 1", Platform: "jira", CreatedAt: now, UpdatedAt: now},
			{Key: "TEST-2", Summary: "Task 2", Platform: "jira", CreatedAt: now, UpdatedAt: now},
		}

		remoteRepo.SetFindByProjectAndSprintFunc(func(_ context.Context, _, _ string) ([]*domain.Task, error) {
			return testTasks, nil
		})

		// Fail on second save
		saveCount := 0
		localRepo.SetSaveFunc(func(_ context.Context, task *domain.Task) error {
			saveCount++
			if saveCount == 2 {
				return fmt.Errorf("save error for %s", task.Key)
			}
			return nil
		})

		// Execute use case
		err := useCase.Execute(context.Background(), "TEST", "Sprint1", "jira")

		// Should fail on second task save
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to save task")
		assert.Contains(t, err.Error(), "TEST-2")
	})

	t.Run("should validate all required parameters", func(t *testing.T) {
		// Create mock repositories
		remoteRepo := testutil.NewMockTaskRepository()
		localRepo := testutil.NewMockTaskRepository()
		useCase := NewFetchTasksUseCase(remoteRepo, localRepo, nil)

		// Test all parameter validations
		testCases := []struct {
			name     string
			project  string
			sprint   string
			platform string
			key      string
			method   string
			errMsg   string
		}{
			{"empty project in Execute", "", "Sprint1", "jira", "", "Execute", "project is required"},
			{"empty sprint in Execute", "TEST", "", "jira", "", "Execute", "sprint is required"},
			{"empty platform in Execute", "TEST", "Sprint1", "", "", "Execute", "platform is required"},
			{"empty key in ExecuteByKey", "", "", "jira", "", "ExecuteByKey", "task key is required"},
			{"empty platform in ExecuteByKey", "", "", "", "TEST-1", "ExecuteByKey", "platform is required"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var err error
				if tc.method == "Execute" {
					err = useCase.Execute(context.Background(), tc.project, tc.sprint, tc.platform)
				} else {
					err = useCase.ExecuteByKey(context.Background(), tc.key, tc.platform)
				}

				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMsg)
			})
		}
	})
}
