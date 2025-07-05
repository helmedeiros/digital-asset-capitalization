package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application/usecase"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application/usecase/testutil"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

func TestTasksService_FetchTasks(t *testing.T) {
	remoteRepo := testutil.NewMockTaskRepository()
	localRepo := testutil.NewMockTaskRepository()
	assetService := testutil.NewMockAssetService()
	service := NewTasksService(remoteRepo, localRepo, nil, nil, assetService)

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
	assetService := testutil.NewMockAssetService()
	service := NewTasksService(remoteRepo, localRepo, classifier, userInput, assetService)

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
	assetService := testutil.NewMockAssetService()
	service := NewTasksService(jiraRepo, localRepo, classifier, userInput, assetService)

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

func TestTaskService_GetTasks(t *testing.T) {
	// Create test tasks
	expectedTasks := []*domain.Task{
		{
			Key:     "TEST-1",
			Summary: "Test Task 1",
			Project: "TEST",
			Sprint:  "Sprint 1",
		},
		{
			Key:     "TEST-2",
			Summary: "Test Task 2",
			Project: "TEST",
			Sprint:  "Sprint 1",
		},
	}

	t.Run("should return tasks for project and sprint", func(t *testing.T) {
		// Set up mock dependencies for this test
		jiraRepo := testutil.NewMockTaskRepository()
		localRepo := testutil.NewMockTaskRepository()
		// Create a mock use case
		mockUseCase := &MockClassifyTasksUseCase{}
		mockUseCase.On("GetTasks", mock.Anything, "TEST", "Sprint 1").Return(expectedTasks, nil)

		// Create testable service with mock use case
		fetchUseCase := usecase.NewFetchTasksUseCase(jiraRepo, localRepo)
		testableService := TestableTaskService(fetchUseCase, mockUseCase)

		// Test
		tasks, err := testableService.GetTasks(context.Background(), "TEST", "Sprint 1")

		// Verify
		assert.NoError(t, err, "Should not return error")
		assert.Equal(t, expectedTasks, tasks, "Should return expected tasks")
		mockUseCase.AssertExpectations(t)
	})

	t.Run("should return error when use case fails", func(t *testing.T) {
		// Set up mock dependencies for this test
		jiraRepo := testutil.NewMockTaskRepository()
		localRepo := testutil.NewMockTaskRepository()

		// Create a mock use case that returns error
		mockUseCase := &MockClassifyTasksUseCase{}
		mockUseCase.On("GetTasks", mock.Anything, "TEST", "Sprint 1").Return(nil, fmt.Errorf("use case error"))

		// Create testable service with mock use case
		fetchUseCase := usecase.NewFetchTasksUseCase(jiraRepo, localRepo)
		testableService := TestableTaskService(fetchUseCase, mockUseCase)

		// Test
		tasks, err := testableService.GetTasks(context.Background(), "TEST", "Sprint 1")

		// Verify
		assert.Error(t, err, "Should return error")
		assert.Nil(t, tasks, "Should return nil tasks")
		assert.Contains(t, err.Error(), "use case error", "Error should contain use case error")
		mockUseCase.AssertExpectations(t)
	})

	t.Run("should handle empty results", func(t *testing.T) {
		// Set up mock dependencies for this test
		jiraRepo := testutil.NewMockTaskRepository()
		localRepo := testutil.NewMockTaskRepository()

		// Create a mock use case that returns empty slice
		mockUseCase := &MockClassifyTasksUseCase{}
		mockUseCase.On("GetTasks", mock.Anything, "TEST", "Sprint 1").Return([]*domain.Task{}, nil)

		// Create testable service with mock use case
		fetchUseCase := usecase.NewFetchTasksUseCase(jiraRepo, localRepo)
		testableService := TestableTaskService(fetchUseCase, mockUseCase)

		// Test
		tasks, err := testableService.GetTasks(context.Background(), "TEST", "Sprint 1")

		// Verify
		assert.NoError(t, err, "Should not return error")
		assert.Empty(t, tasks, "Should return empty slice")
		mockUseCase.AssertExpectations(t)
	})
}

func TestTaskService_GetLocalRepository(t *testing.T) {
	t.Run("should return the local repository", func(t *testing.T) {
		// Set up mock dependencies
		jiraRepo := testutil.NewMockTaskRepository()
		localRepo := testutil.NewMockTaskRepository()
		classifier := testutil.NewMockTaskClassifier()
		userInput := testutil.NewMockUserInput()
		assetService := testutil.NewMockAssetService()

		// Create service
		service := NewTasksService(jiraRepo, localRepo, classifier, userInput, assetService)

		// Test
		result := service.GetLocalRepository()

		// Verify that we get a repository (it will be wrapped in the use case)
		assert.NotNil(t, result, "Should return a repository")

		// Since the repository is wrapped in the use case, we can't directly compare
		// But we can verify it's not nil which means the method works
	})

	t.Run("should return consistent repository instance", func(t *testing.T) {
		// Set up mock dependencies
		jiraRepo := testutil.NewMockTaskRepository()
		localRepo := testutil.NewMockTaskRepository()
		classifier := testutil.NewMockTaskClassifier()
		userInput := testutil.NewMockUserInput()
		assetService := testutil.NewMockAssetService()

		// Create service
		service := NewTasksService(jiraRepo, localRepo, classifier, userInput, assetService)

		// Test multiple calls
		result1 := service.GetLocalRepository()
		result2 := service.GetLocalRepository()

		// Verify both calls return the same instance
		assert.NotNil(t, result1, "First call should return repository")
		assert.NotNil(t, result2, "Second call should return repository")
		assert.Equal(t, result1, result2, "Should return same repository instance")
	})
}

// ClassifyTasksUseCaseInterface defines the interface for the classify tasks use case
type ClassifyTasksUseCaseInterface interface {
	Execute(ctx context.Context, input domain.ClassifyTasksInput) error
	GetTasks(ctx context.Context, project, sprint string) ([]*domain.Task, error)
	GetAllTasks(ctx context.Context) ([]*domain.Task, error)
	GetLocalRepository() ports.TaskRepository
}

// Mock for ClassifyTasksUseCase
type MockClassifyTasksUseCase struct {
	mock.Mock
}

func (m *MockClassifyTasksUseCase) Execute(ctx context.Context, input domain.ClassifyTasksInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockClassifyTasksUseCase) GetTasks(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
	args := m.Called(ctx, project, sprint)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Task), args.Error(1)
}

func (m *MockClassifyTasksUseCase) GetAllTasks(ctx context.Context) ([]*domain.Task, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Task), args.Error(1)
}

func (m *MockClassifyTasksUseCase) GetLocalRepository() ports.TaskRepository {
	args := m.Called()
	return args.Get(0).(ports.TaskRepository)
}

// Ensure MockClassifyTasksUseCase implements the interface
var _ ClassifyTasksUseCaseInterface = (*MockClassifyTasksUseCase)(nil)

// TestableTaskService creates a TaskService for testing with injectable use case
func TestableTaskService(fetchUseCase *usecase.FetchTasksUseCase, classifyUseCase ClassifyTasksUseCaseInterface) TaskService {
	return &TestableTaskServiceImpl{
		fetchTasksUseCase:    fetchUseCase,
		classifyTasksUseCase: classifyUseCase,
	}
}

// TestableTaskServiceImpl is a testable version of TaskServiceImpl
type TestableTaskServiceImpl struct {
	fetchTasksUseCase    *usecase.FetchTasksUseCase
	classifyTasksUseCase ClassifyTasksUseCaseInterface
}

func (s *TestableTaskServiceImpl) FetchTasks(ctx context.Context, project, sprint, platform string) error {
	return s.fetchTasksUseCase.Execute(ctx, project, sprint, platform)
}

func (s *TestableTaskServiceImpl) FetchTaskByKey(ctx context.Context, key, platform string) error {
	return s.fetchTasksUseCase.ExecuteByKey(ctx, key, platform)
}

func (s *TestableTaskServiceImpl) ClassifyTasks(ctx context.Context, input domain.ClassifyTasksInput) error {
	return s.classifyTasksUseCase.Execute(ctx, input)
}

func (s *TestableTaskServiceImpl) GetTasks(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
	return s.classifyTasksUseCase.GetTasks(ctx, project, sprint)
}

func (s *TestableTaskServiceImpl) GetTasksByAsset(ctx context.Context, assetName string) ([]*domain.Task, error) {
	tasks, err := s.classifyTasksUseCase.GetAllTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	// Handle both asset names and full asset IDs
	assetID := assetName
	if !strings.HasPrefix(assetName, "cap-asset-") {
		// For multi-word asset names, use just the first word
		words := strings.Fields(assetName)
		assetID = "cap-asset-" + strings.ToLower(words[0])
	}

	var assetTasks []*domain.Task
	for _, task := range tasks {
		for _, label := range task.Labels {
			if label == assetID {
				assetTasks = append(assetTasks, task)
				break
			}
		}
	}

	return assetTasks, nil
}

func (s *TestableTaskServiceImpl) GetTaskByKey(ctx context.Context, key string) (*domain.Task, error) {
	if key == "" {
		return nil, fmt.Errorf("task key cannot be empty")
	}

	// Try to find the task in the local repository
	localRepo := s.classifyTasksUseCase.GetLocalRepository()
	task, err := localRepo.FindByKey(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to find task with key %s: %w", key, err)
	}

	return task, nil
}

func (s *TestableTaskServiceImpl) GetLocalRepository() ports.TaskRepository {
	return s.classifyTasksUseCase.GetLocalRepository()
}

func TestTaskServiceImpl_GetTasks(t *testing.T) {
	t.Run("should delegate to classify tasks use case", func(t *testing.T) {
		// Set up mock dependencies
		jiraRepo := testutil.NewMockTaskRepository()
		localRepo := testutil.NewMockTaskRepository()
		classifier := testutil.NewMockTaskClassifier()
		userInput := testutil.NewMockUserInput()
		assetService := testutil.NewMockAssetService()

		// Create service using the concrete constructor
		service := NewTasksService(jiraRepo, localRepo, classifier, userInput, assetService)

		// Set up test data
		expectedTasks := []*domain.Task{
			{
				Key:     "TEST-1",
				Summary: "Test Task 1",
				Project: "TEST",
				Sprint:  "Sprint 1",
			},
		}

		// Mock the local repository to return tasks
		localRepo.SetFindByProjectAndSprintFunc(func(_ context.Context, _, _ string) ([]*domain.Task, error) {
			return expectedTasks, nil
		})

		// Test
		tasks, err := service.GetTasks(context.Background(), "TEST", "Sprint 1")

		// Verify
		assert.NoError(t, err, "Should not return error")
		assert.Equal(t, expectedTasks, tasks, "Should return expected tasks")
	})

	t.Run("should return error when use case fails", func(t *testing.T) {
		// Set up mock dependencies
		jiraRepo := testutil.NewMockTaskRepository()
		localRepo := testutil.NewMockTaskRepository()
		classifier := testutil.NewMockTaskClassifier()
		userInput := testutil.NewMockUserInput()
		assetService := testutil.NewMockAssetService()

		// Create service using the concrete constructor
		service := NewTasksService(jiraRepo, localRepo, classifier, userInput, assetService)

		// Mock the local repository to return error
		localRepo.SetFindByProjectAndSprintFunc(func(_ context.Context, _, _ string) ([]*domain.Task, error) {
			return nil, fmt.Errorf("repository error")
		})

		// Test
		tasks, err := service.GetTasks(context.Background(), "TEST", "Sprint 1")

		// Verify
		assert.Error(t, err, "Should return error")
		assert.Nil(t, tasks, "Should return nil tasks")
		assert.Contains(t, err.Error(), "failed to find existing tasks", "Error should contain repository error")
	})
}

func TestTaskService_GetTaskByKey(t *testing.T) {
	t.Run("should return task when found", func(t *testing.T) {
		// Set up mock dependencies
		jiraRepo := testutil.NewMockTaskRepository()
		localRepo := testutil.NewMockTaskRepository()
		classifier := testutil.NewMockTaskClassifier()
		userInput := testutil.NewMockUserInput()
		assetService := testutil.NewMockAssetService()

		// Create service
		service := NewTasksService(jiraRepo, localRepo, classifier, userInput, assetService)

		// Set up test data
		expectedTask := &domain.Task{
			Key:     "FN-1015",
			Summary: "Enable rounding with new Journey Details Service",
			Project: "FN",
			Sprint:  "The Hulk",
			Status:  "Done",
			Type:    "Task",
		}

		// Mock the local repository to return task
		localRepo.SetFindByKeyFunc(func(_ context.Context, key string) (*domain.Task, error) {
			if key == "FN-1015" {
				return expectedTask, nil
			}
			return nil, fmt.Errorf("task not found")
		})

		// Test
		task, err := service.GetTaskByKey(context.Background(), "FN-1015")

		// Verify
		assert.NoError(t, err, "Should not return error")
		assert.Equal(t, expectedTask, task, "Should return expected task")
	})

	t.Run("should return error when task not found", func(t *testing.T) {
		// Set up mock dependencies
		jiraRepo := testutil.NewMockTaskRepository()
		localRepo := testutil.NewMockTaskRepository()
		classifier := testutil.NewMockTaskClassifier()
		userInput := testutil.NewMockUserInput()
		assetService := testutil.NewMockAssetService()

		// Create service
		service := NewTasksService(jiraRepo, localRepo, classifier, userInput, assetService)

		// Mock the local repository to return error
		localRepo.SetFindByKeyFunc(func(_ context.Context, _ string) (*domain.Task, error) {
			return nil, fmt.Errorf("task not found")
		})

		// Test
		task, err := service.GetTaskByKey(context.Background(), "NON-EXISTENT")

		// Verify
		assert.Error(t, err, "Should return error")
		assert.Nil(t, task, "Should return nil task")
		assert.Contains(t, err.Error(), "task not found", "Error should contain task not found")
	})

	t.Run("should return error when key is empty", func(t *testing.T) {
		// Set up mock dependencies
		jiraRepo := testutil.NewMockTaskRepository()
		localRepo := testutil.NewMockTaskRepository()
		classifier := testutil.NewMockTaskClassifier()
		userInput := testutil.NewMockUserInput()
		assetService := testutil.NewMockAssetService()

		// Create service
		service := NewTasksService(jiraRepo, localRepo, classifier, userInput, assetService)

		// Test
		task, err := service.GetTaskByKey(context.Background(), "")

		// Verify
		assert.Error(t, err, "Should return error")
		assert.Nil(t, task, "Should return nil task")
		assert.Contains(t, err.Error(), "task key cannot be empty", "Error should contain empty key error")
	})
}

func TestTaskService_FetchTaskByKey(t *testing.T) {
	t.Run("should successfully fetch task by key", func(t *testing.T) {
		// Set up mock dependencies
		jiraRepo := testutil.NewMockTaskRepository()
		localRepo := testutil.NewMockTaskRepository()
		classifier := testutil.NewMockTaskClassifier()
		userInput := testutil.NewMockUserInput()
		assetService := testutil.NewMockAssetService()

		// Create service
		service := NewTasksService(jiraRepo, localRepo, classifier, userInput, assetService)

		// Mock successful execution
		jiraRepo.SetFindByKeyFunc(func(_ context.Context, key string) (*domain.Task, error) {
			return &domain.Task{Key: key}, nil
		})
		localRepo.SetSaveFunc(func(_ context.Context, _ *domain.Task) error {
			return nil
		})

		// Execute
		err := service.FetchTaskByKey(context.Background(), "TEST-123", "jira")

		// Verify
		assert.NoError(t, err)
	})

	t.Run("should return error when fetch fails", func(t *testing.T) {
		// Set up mock dependencies
		jiraRepo := testutil.NewMockTaskRepository()
		localRepo := testutil.NewMockTaskRepository()
		classifier := testutil.NewMockTaskClassifier()
		userInput := testutil.NewMockUserInput()
		assetService := testutil.NewMockAssetService()

		// Create service
		service := NewTasksService(jiraRepo, localRepo, classifier, userInput, assetService)

		// Mock failure
		jiraRepo.SetFindByKeyFunc(func(_ context.Context, _ string) (*domain.Task, error) {
			return nil, fmt.Errorf("task not found")
		})

		// Execute
		err := service.FetchTaskByKey(context.Background(), "TEST-123", "jira")

		// Verify
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task not found")
	})
}
