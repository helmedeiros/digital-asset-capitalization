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
	remoteRepo := testutil.NewMockTaskRepository()
	localRepo := testutil.NewMockTaskRepository()
	service := NewTasksService(remoteRepo, localRepo)

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
				remoteRepo.Reset()
				localRepo.Reset()
				remoteRepo.SetFindByProjectAndSprintFunc(func(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
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
				localRepo.SetSaveFunc(func(ctx context.Context, task *domain.Task) error {
					assert.Equal(t, "PROJ-1", task.Key)
					return nil
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
			name:     "remote repository error",
			project:  "PROJ",
			sprint:   "Sprint 1",
			platform: "JIRA",
			setup: func() {
				remoteRepo.Reset()
				remoteRepo.SetFindByProjectAndSprintFunc(func(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
					return nil, errors.New("repository error")
				})
			},
			wantErr: true,
		},
		{
			name:     "local repository error",
			project:  "PROJ",
			sprint:   "Sprint 1",
			platform: "JIRA",
			setup: func() {
				remoteRepo.Reset()
				localRepo.Reset()
				remoteRepo.SetFindByProjectAndSprintFunc(func(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
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
				localRepo.SetSaveFunc(func(ctx context.Context, task *domain.Task) error {
					return errors.New("save error")
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
