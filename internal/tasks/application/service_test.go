package application

import (
	"context"
	"errors"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application/usecase"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application/usecase/testutil"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/stretchr/testify/assert"
)

func TestTasksService_FetchTasks(t *testing.T) {
	remoteRepo := testutil.NewMockTaskRepository()
	localRepo := testutil.NewMockTaskRepository()
	service := NewTasksService(remoteRepo, localRepo, nil, nil, nil)

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

func TestTasksService_ClassifyTasks(t *testing.T) {
	remoteRepo := testutil.NewMockTaskRepository()
	localRepo := testutil.NewMockTaskRepository()
	classifier := testutil.NewMockTaskClassifier()
	userInput := testutil.NewMockUserInput()
	taskFetcher := testutil.NewMockTaskFetcher()
	service := NewTasksService(remoteRepo, localRepo, classifier, userInput, taskFetcher)

	tests := []struct {
		name    string
		input   usecase.ClassifyTasksInput
		setup   func()
		wantErr bool
	}{
		{
			name: "successful classification",
			input: usecase.ClassifyTasksInput{
				Project: "PROJ",
				Sprint:  "Sprint 1",
			},
			setup: func() {
				localRepo.Reset()
				classifier.Reset()
				userInput.Reset()
				taskFetcher.Reset()

				// Setup local repo to return existing tasks
				localRepo.SetFindByProjectAndSprintFunc(func(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
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

				// Setup classifier to return work types
				classifier.SetClassifyTasksFunc(func(tasks []*domain.Task) (map[string]domain.WorkType, error) {
					return map[string]domain.WorkType{
						"PROJ-1": domain.WorkTypeDevelopment,
					}, nil
				})

				// Setup local repo to handle task updates
				localRepo.SetSaveFunc(func(ctx context.Context, task *domain.Task) error {
					assert.Equal(t, "PROJ-1", task.Key)
					assert.Equal(t, domain.WorkTypeDevelopment, task.WorkType)
					return nil
				})
			},
			wantErr: false,
		},
		{
			name: "no tasks found, user chooses to fetch",
			input: usecase.ClassifyTasksInput{
				Project: "PROJ",
				Sprint:  "Sprint 1",
			},
			setup: func() {
				localRepo.Reset()
				classifier.Reset()
				userInput.Reset()
				taskFetcher.Reset()

				// Setup local repo to return no tasks
				localRepo.SetFindByProjectAndSprintFunc(func(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
					return []*domain.Task{}, nil
				})

				// Setup user input to confirm fetching
				userInput.SetConfirmFunc(func(prompt string, args ...interface{}) (bool, error) {
					return true, nil
				})

				// Setup task fetcher to return tasks
				taskFetcher.SetFetchTasksFunc(func(project, sprint string) ([]*domain.Task, error) {
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

				// Setup classifier to return work types
				classifier.SetClassifyTasksFunc(func(tasks []*domain.Task) (map[string]domain.WorkType, error) {
					return map[string]domain.WorkType{
						"PROJ-1": domain.WorkTypeDevelopment,
					}, nil
				})

				// Setup local repo to handle task saves
				localRepo.SetSaveFunc(func(ctx context.Context, task *domain.Task) error {
					assert.Equal(t, "PROJ-1", task.Key)
					return nil
				})
			},
			wantErr: false,
		},
		{
			name: "no tasks found, user chooses not to fetch",
			input: usecase.ClassifyTasksInput{
				Project: "PROJ",
				Sprint:  "Sprint 1",
			},
			setup: func() {
				localRepo.Reset()
				classifier.Reset()
				userInput.Reset()
				taskFetcher.Reset()

				// Setup local repo to return no tasks
				localRepo.SetFindByProjectAndSprintFunc(func(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
					return []*domain.Task{}, nil
				})

				// Setup user input to decline fetching
				userInput.SetConfirmFunc(func(prompt string, args ...interface{}) (bool, error) {
					return false, nil
				})
			},
			wantErr: true,
		},
		{
			name: "classifier error",
			input: usecase.ClassifyTasksInput{
				Project: "PROJ",
				Sprint:  "Sprint 1",
			},
			setup: func() {
				localRepo.Reset()
				classifier.Reset()
				userInput.Reset()
				taskFetcher.Reset()

				// Setup local repo to return existing tasks
				localRepo.SetFindByProjectAndSprintFunc(func(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
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

				// Setup classifier to return error
				classifier.SetClassifyTasksFunc(func(tasks []*domain.Task) (map[string]domain.WorkType, error) {
					return nil, errors.New("classification error")
				})
			},
			wantErr: true,
		},
		{
			name: "local repository error",
			input: usecase.ClassifyTasksInput{
				Project: "PROJ",
				Sprint:  "Sprint 1",
			},
			setup: func() {
				localRepo.Reset()
				classifier.Reset()
				userInput.Reset()
				taskFetcher.Reset()

				// Setup local repo to return error
				localRepo.SetFindByProjectAndSprintFunc(func(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
					return nil, errors.New("repository error")
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := service.ClassifyTasks(context.Background(), tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
