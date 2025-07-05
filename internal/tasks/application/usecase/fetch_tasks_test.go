package usecase

import (
	"context"
	"errors"
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
	useCase := NewFetchTasksUseCase(remoteRepo, localRepo)

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
	useCase := NewFetchTasksUseCase(remoteRepo, localRepo)

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
