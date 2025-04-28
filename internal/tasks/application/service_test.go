package application

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application/usecase/testutil"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

func TestTasksService_FetchTasks(t *testing.T) {
	remoteRepo := testutil.NewMockTaskRepository()
	localRepo := testutil.NewMockTaskRepository()
	service := NewTasksService(remoteRepo, localRepo, nil, nil)

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
				remoteRepo.SetFindByProjectAndSprintFunc(func(_ context.Context, _, _ string) ([]*domain.Task, error) {
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
				localRepo.SetSaveFunc(func(_ context.Context, task *domain.Task) error {
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
				remoteRepo.SetFindByProjectAndSprintFunc(func(_ context.Context, _, _ string) ([]*domain.Task, error) {
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
				remoteRepo.SetFindByProjectAndSprintFunc(func(_ context.Context, _, _ string) ([]*domain.Task, error) {
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
				localRepo.SetSaveFunc(func(_ context.Context, _ *domain.Task) error {
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
	service := NewTasksService(remoteRepo, localRepo, classifier, userInput)

	tests := []struct {
		name    string
		input   domain.ClassifyTasksInput
		setup   func()
		wantErr bool
	}{
		{
			name: "successful classification",
			input: domain.ClassifyTasksInput{
				Project: "TEST",
				Sprint:  "Sprint 1",
				DryRun:  false,
				Apply:   true,
			},
			setup: func() {
				localRepo.Reset()
				classifier.Reset()
				userInput.Reset()

				// Setup local repo to return existing tasks
				localRepo.SetFindByProjectAndSprintFunc(func(_ context.Context, _, _ string) ([]*domain.Task, error) {
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
				classifier.SetClassifyTasksFunc(func(_ []*domain.Task) (map[string]domain.WorkType, error) {
					return map[string]domain.WorkType{
						"PROJ-1": domain.WorkTypeDevelopment,
					}, nil
				})

				// Setup local repo to handle task updates
				localRepo.SetSaveFunc(func(_ context.Context, task *domain.Task) error {
					assert.Equal(t, "PROJ-1", task.Key)
					assert.Equal(t, domain.WorkTypeDevelopment, task.WorkType)
					return nil
				})
			},
			wantErr: false,
		},
		{
			name: "no tasks found, user chooses to fetch",
			input: domain.ClassifyTasksInput{
				Project: "PROJ",
				Sprint:  "Sprint 1",
				DryRun:  false,
				Apply:   true,
			},
			setup: func() {
				localRepo.Reset()
				classifier.Reset()
				userInput.Reset()

				// Setup local repo to return no tasks
				localRepo.SetFindByProjectAndSprintFunc(func(_ context.Context, _, _ string) ([]*domain.Task, error) {
					return []*domain.Task{}, nil
				})

				// Setup user input to confirm fetching
				userInput.SetConfirmFunc(func(_ string, _ ...interface{}) (bool, error) {
					return true, nil
				})

				// Setup remote repo to return tasks
				remoteRepo.SetFindByProjectAndSprintFunc(func(_ context.Context, _, _ string) ([]*domain.Task, error) {
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
				classifier.SetClassifyTasksFunc(func(_ []*domain.Task) (map[string]domain.WorkType, error) {
					return map[string]domain.WorkType{
						"PROJ-1": domain.WorkTypeDevelopment,
					}, nil
				})

				// Setup local repo to handle task saves
				localRepo.SetSaveFunc(func(_ context.Context, task *domain.Task) error {
					assert.Equal(t, "PROJ-1", task.Key)
					return nil
				})
			},
			wantErr: false,
		},
		{
			name: "no tasks found, user chooses not to fetch",
			input: domain.ClassifyTasksInput{
				Project: "PROJ",
				Sprint:  "Sprint 1",
				DryRun:  false,
				Apply:   false,
			},
			setup: func() {
				localRepo.Reset()
				classifier.Reset()
				userInput.Reset()

				// Setup local repo to return no tasks
				localRepo.SetFindByProjectAndSprintFunc(func(_ context.Context, _, _ string) ([]*domain.Task, error) {
					return []*domain.Task{}, nil
				})

				// Setup user input to decline fetching
				userInput.SetConfirmFunc(func(_ string, _ ...interface{}) (bool, error) {
					return false, nil
				})
			},
			wantErr: true,
		},
		{
			name: "classifier error",
			input: domain.ClassifyTasksInput{
				Project: "PROJ",
				Sprint:  "Sprint 1",
				DryRun:  false,
				Apply:   true,
			},
			setup: func() {
				localRepo.Reset()
				classifier.Reset()
				userInput.Reset()

				// Setup local repo to return existing tasks
				localRepo.SetFindByProjectAndSprintFunc(func(_ context.Context, _, _ string) ([]*domain.Task, error) {
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
				classifier.SetClassifyTasksFunc(func(_ []*domain.Task) (map[string]domain.WorkType, error) {
					return nil, errors.New("classification error")
				})
			},
			wantErr: true,
		},
		{
			name: "local repository error",
			input: domain.ClassifyTasksInput{
				Project: "PROJ",
				Sprint:  "Sprint 1",
				DryRun:  false,
				Apply:   true,
			},
			setup: func() {
				localRepo.Reset()
				classifier.Reset()
				userInput.Reset()

				// Setup local repo to return error
				localRepo.SetFindByProjectAndSprintFunc(func(_ context.Context, _, _ string) ([]*domain.Task, error) {
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

func TestTaskService_GetTasksByAsset(t *testing.T) {
	// Set up mock dependencies
	jiraRepo := testutil.NewMockTaskRepository()
	localRepo := testutil.NewMockTaskRepository()
	classifier := testutil.NewMockTaskClassifier()
	userInput := testutil.NewMockUserInput()

	// Create test tasks
	tasks := []*domain.Task{
		{
			Key:     "TEST-1",
			Type:    "Story",
			Summary: "Test Task 1",
			Status:  "In Progress",
			Labels:  []string{"cap-asset-insurance", "cap-development"},
		},
		{
			Key:     "TEST-2",
			Type:    "Story",
			Summary: "Test Task 2",
			Status:  "In Progress",
			Labels:  []string{"cap-asset-insurance", "cap-development"},
		},
		{
			Key:     "TEST-3",
			Type:    "Story",
			Summary: "Test Task 3",
			Status:  "In Progress",
			Labels:  []string{"cap-asset-other", "cap-development"},
		},
	}

	// Set up mock behavior for GetAllTasks
	localRepo.SetFindAllFunc(func(_ context.Context) ([]*domain.Task, error) {
		return tasks, nil
	})

	// Create service
	service := NewTasksService(jiraRepo, localRepo, classifier, userInput)

	tests := []struct {
		name      string
		assetName string
		wantCount int
		wantErr   bool
	}{
		{
			name:      "find tasks by asset name",
			assetName: "Insurance",
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "find tasks by asset ID",
			assetName: "cap-asset-insurance",
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "find tasks by full asset name",
			assetName: "Insurance Platform",
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "no tasks found for asset",
			assetName: "NonExistentAsset",
			wantCount: 0,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := service.GetTasksByAsset(context.Background(), tt.assetName)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, got, tt.wantCount, "Expected %d tasks but got %d", tt.wantCount, len(got))

			// Verify task contents if we expect tasks
			if tt.wantCount > 0 {
				for _, task := range got {
					assert.Contains(t, task.Labels, "cap-asset-insurance", "Task should have insurance label")
				}
			}
		})
	}
}
