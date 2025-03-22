package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application/usecase/testutil"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchTasksUseCase(t *testing.T) {
	// Create a mock repository
	mockRepo := testutil.NewMockTaskRepository()
	useCase := NewFetchTasksUseCase(mockRepo)

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
				mockRepo.SetFindByProjectAndSprintFunc(func(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
					assert.Equal(t, "TEST", project)
					assert.Equal(t, "Sprint 1", sprint)
					return testTasks, nil
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
			name:     "repository error",
			project:  "TEST",
			sprint:   "Sprint 1",
			platform: "jira",
			setupMock: func() {
				mockRepo.SetFindByProjectAndSprintFunc(func(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
					return nil, errors.New("repository error")
				})
			},
			wantErr: true,
			errMsg:  "failed to fetch tasks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock before each test
			mockRepo.Reset()

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
