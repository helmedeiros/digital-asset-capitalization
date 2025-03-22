package application

import (
	"context"
	"errors"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application/usecase/testutil"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/stretchr/testify/assert"
)

func TestTasksService_FetchTasks(t *testing.T) {
	mockRepo := testutil.NewMockTaskRepository()
	service := NewTasksService(mockRepo)

	tests := []struct {
		name     string
		project  string
		sprint   string
		platform string
		setup    func()
		wantErr  bool
	}{
		{
			name:     "successful fetch",
			project:  "PROJ",
			sprint:   "Sprint 1",
			platform: "JIRA",
			setup: func() {
				mockRepo.Reset()
				mockRepo.SetFindByProjectAndSprintFunc(func(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
					return []*domain.Task{
						{
							Key:     "PROJ-1",
							Type:    "Story",
							Summary: "Test Task",
							Status:  "In Progress",
							Sprint:  "Sprint 1",
						},
					}, nil
				})
			},
			wantErr: false,
		},
		{
			name:     "empty project",
			project:  "",
			sprint:   "Sprint 1",
			platform: "JIRA",
			setup:    func() {},
			wantErr:  true,
		},
		{
			name:     "empty platform",
			project:  "PROJ",
			sprint:   "Sprint 1",
			platform: "",
			setup:    func() {},
			wantErr:  true,
		},
		{
			name:     "repository error",
			project:  "PROJ",
			sprint:   "Sprint 1",
			platform: "JIRA",
			setup: func() {
				mockRepo.Reset()
				mockRepo.SetFindByProjectAndSprintFunc(func(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
					return nil, errors.New("repository error")
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := service.FetchTasks(context.Background(), tt.project, tt.sprint, tt.platform)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
