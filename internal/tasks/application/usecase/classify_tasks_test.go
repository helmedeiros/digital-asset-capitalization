package usecase

import (
	"context"
	"fmt"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTaskRepository is a mock implementation of TaskRepository
type MockTaskRepository struct {
	mock.Mock
}

func (m *MockTaskRepository) FindAll(ctx context.Context) ([]*domain.Task, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Task), args.Error(1)
}

func (m *MockTaskRepository) FindByKey(ctx context.Context, key string) (*domain.Task, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Task), args.Error(1)
}

func (m *MockTaskRepository) FindByPlatform(ctx context.Context, platform string) ([]*domain.Task, error) {
	args := m.Called(ctx, platform)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Task), args.Error(1)
}

func (m *MockTaskRepository) FindByProject(ctx context.Context, project string) ([]*domain.Task, error) {
	args := m.Called(ctx, project)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Task), args.Error(1)
}

func (m *MockTaskRepository) FindByProjectAndSprint(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
	args := m.Called(ctx, project, sprint)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Task), args.Error(1)
}

func (m *MockTaskRepository) FindBySprint(ctx context.Context, sprint string) ([]*domain.Task, error) {
	args := m.Called(ctx, sprint)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Task), args.Error(1)
}

func (m *MockTaskRepository) Save(ctx context.Context, task *domain.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockTaskRepository) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockTaskRepository) DeleteByProjectAndSprint(ctx context.Context, project, sprint string) error {
	args := m.Called(ctx, project, sprint)
	return args.Error(0)
}

func (m *MockTaskRepository) UpdateLabels(ctx context.Context, taskKey string, labels []string) error {
	args := m.Called(ctx, taskKey, labels)
	return args.Error(0)
}

// MockTaskClassifier is a mock implementation of TaskClassifier
type MockTaskClassifier struct {
	mock.Mock
}

func (m *MockTaskClassifier) ClassifyTask(task *domain.Task) (domain.WorkType, error) {
	args := m.Called(task)
	return args.Get(0).(domain.WorkType), args.Error(1)
}

func (m *MockTaskClassifier) ClassifyTasks(tasks []*domain.Task) (map[string]domain.WorkType, error) {
	args := m.Called(tasks)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]domain.WorkType), args.Error(1)
}

// MockUserInput is a mock implementation of UserInput
type MockUserInput struct {
	mock.Mock
}

func (m *MockUserInput) Confirm(format string, args ...interface{}) (bool, error) {
	callArgs := m.Called(format, args)
	return callArgs.Bool(0), callArgs.Error(1)
}

// MockTaskFetcher is a mock implementation of TaskFetcher
type MockTaskFetcher struct {
	mock.Mock
}

func (m *MockTaskFetcher) FetchTasks(project, sprint string) ([]*domain.Task, error) {
	args := m.Called(project, sprint)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Task), args.Error(1)
}

func TestClassifyTasksUseCase_Execute(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		input         ClassifyTasksInput
		existingTasks []*domain.Task
		shouldFetch   bool
		fetchedTasks  []*domain.Task
		workTypes     map[string]domain.WorkType
		expectedError bool
		expectedCalls func(*MockTaskRepository, *MockTaskRepository, *MockTaskClassifier, *MockUserInput)
	}{
		{
			name: "successfully classify existing tasks",
			input: ClassifyTasksInput{
				Project: "TEST",
				Sprint:  "Sprint 1",
			},
			existingTasks: []*domain.Task{
				{Key: "TEST-1", Summary: "Task 1"},
				{Key: "TEST-2", Summary: "Task 2"},
			},
			shouldFetch: false,
			workTypes: map[string]domain.WorkType{
				"TEST-1": domain.WorkTypeDevelopment,
				"TEST-2": domain.WorkTypeMaintenance,
			},
			expectedError: false,
			expectedCalls: func(localRepo, remoteRepo *MockTaskRepository, classifier *MockTaskClassifier, userInput *MockUserInput) {
				localRepo.On("FindByProjectAndSprint", ctx, "TEST", "Sprint 1").Return([]*domain.Task{
					{Key: "TEST-1", Summary: "Task 1"},
					{Key: "TEST-2", Summary: "Task 2"},
				}, nil)
				classifier.On("ClassifyTasks", mock.Anything).Return(map[string]domain.WorkType{
					"TEST-1": domain.WorkTypeDevelopment,
					"TEST-2": domain.WorkTypeMaintenance,
				}, nil)
				localRepo.On("Save", ctx, mock.Anything).Return(nil).Times(2)
			},
		},
		{
			name: "fetch and classify new tasks",
			input: ClassifyTasksInput{
				Project: "TEST",
				Sprint:  "Sprint 2",
			},
			existingTasks: nil,
			shouldFetch:   true,
			fetchedTasks: []*domain.Task{
				{Key: "TEST-3", Summary: "Task 3"},
				{Key: "TEST-4", Summary: "Task 4"},
			},
			workTypes: map[string]domain.WorkType{
				"TEST-3": domain.WorkTypeDiscovery,
				"TEST-4": domain.WorkTypeDevelopment,
			},
			expectedError: false,
			expectedCalls: func(localRepo, remoteRepo *MockTaskRepository, classifier *MockTaskClassifier, userInput *MockUserInput) {
				localRepo.On("FindByProjectAndSprint", ctx, "TEST", "Sprint 2").Return(nil, nil)
				userInput.On("Confirm", mock.Anything, mock.Anything).Return(true, nil)
				remoteRepo.On("FindByProjectAndSprint", ctx, "TEST", "Sprint 2").Return([]*domain.Task{
					{Key: "TEST-3", Summary: "Task 3"},
					{Key: "TEST-4", Summary: "Task 4"},
				}, nil)
				localRepo.On("Save", ctx, mock.Anything).Return(nil).Times(4)
				classifier.On("ClassifyTasks", mock.Anything).Return(map[string]domain.WorkType{
					"TEST-3": domain.WorkTypeDiscovery,
					"TEST-4": domain.WorkTypeDevelopment,
				}, nil)
			},
		},
		{
			name: "dry run classification",
			input: ClassifyTasksInput{
				Project: "TEST",
				Sprint:  "Sprint 1",
				DryRun:  true,
			},
			existingTasks: []*domain.Task{
				{Key: "TEST-1", Summary: "Task 1"},
				{Key: "TEST-2", Summary: "Task 2"},
			},
			shouldFetch: false,
			workTypes: map[string]domain.WorkType{
				"TEST-1": domain.WorkTypeDevelopment,
				"TEST-2": domain.WorkTypeMaintenance,
			},
			expectedError: false,
			expectedCalls: func(localRepo, remoteRepo *MockTaskRepository, classifier *MockTaskClassifier, userInput *MockUserInput) {
				localRepo.On("FindByProjectAndSprint", ctx, "TEST", "Sprint 1").Return([]*domain.Task{
					{Key: "TEST-1", Summary: "Task 1"},
					{Key: "TEST-2", Summary: "Task 2"},
				}, nil)
				classifier.On("ClassifyTasks", mock.Anything).Return(map[string]domain.WorkType{
					"TEST-1": domain.WorkTypeDevelopment,
					"TEST-2": domain.WorkTypeMaintenance,
				}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			localRepo := new(MockTaskRepository)
			remoteRepo := new(MockTaskRepository)
			classifier := new(MockTaskClassifier)
			userInput := new(MockUserInput)

			// Set up expected calls
			tt.expectedCalls(localRepo, remoteRepo, classifier, userInput)

			// Create use case
			uc := NewClassifyTasksUseCase(localRepo, remoteRepo, classifier, userInput)

			// Execute use case
			err := uc.Execute(ctx, tt.input)

			// Verify results
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				localRepo.AssertExpectations(t)
				remoteRepo.AssertExpectations(t)
				classifier.AssertExpectations(t)
				userInput.AssertExpectations(t)
			}
		})
	}
}

func TestGetTasks(t *testing.T) {
	ctx := context.Background()

	t.Run("should return tasks from local repository when available", func(t *testing.T) {
		// Create mocks
		mockLocalRepo := new(MockTaskRepository)
		mockRemoteRepo := new(MockTaskRepository)
		mockClassifier := new(MockTaskClassifier)
		mockUserInput := new(MockUserInput)

		// Create use case
		uc := NewClassifyTasksUseCase(mockLocalRepo, mockRemoteRepo, mockClassifier, mockUserInput)

		// Arrange
		project := "TEST"
		sprint := "Sprint 1"
		expectedTasks := []*domain.Task{
			{
				Key:     "TEST-1",
				Type:    "Task",
				Summary: "Test Task 1",
				Status:  "In Progress",
			},
		}

		mockLocalRepo.On("FindByProjectAndSprint", ctx, project, sprint).
			Return(expectedTasks, nil)

		// Act
		tasks, err := uc.GetTasks(ctx, project, sprint)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedTasks, tasks)
		mockLocalRepo.AssertExpectations(t)
		mockRemoteRepo.AssertNotCalled(t, "FindByProjectAndSprint")
	})

	t.Run("should fetch and save tasks from remote when local is empty", func(t *testing.T) {
		// Create mocks
		mockLocalRepo := new(MockTaskRepository)
		mockRemoteRepo := new(MockTaskRepository)
		mockClassifier := new(MockTaskClassifier)
		mockUserInput := new(MockUserInput)

		// Create use case
		uc := NewClassifyTasksUseCase(mockLocalRepo, mockRemoteRepo, mockClassifier, mockUserInput)

		// Arrange
		project := "TEST"
		sprint := "Sprint 1"
		remoteTasks := []*domain.Task{
			{
				Key:     "TEST-1",
				Type:    "Task",
				Summary: "Test Task 1",
				Status:  "In Progress",
			},
		}

		mockLocalRepo.On("FindByProjectAndSprint", ctx, project, sprint).
			Return([]*domain.Task{}, nil).Once()
		mockRemoteRepo.On("FindByProjectAndSprint", ctx, project, sprint).
			Return(remoteTasks, nil).Once()
		mockLocalRepo.On("Save", ctx, remoteTasks[0]).
			Return(nil).Once()

		// Act
		tasks, err := uc.GetTasks(ctx, project, sprint)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, remoteTasks, tasks)
		mockLocalRepo.AssertExpectations(t)
		mockRemoteRepo.AssertExpectations(t)
	})

	t.Run("should return error when local repository fails", func(t *testing.T) {
		// Create mocks
		mockLocalRepo := new(MockTaskRepository)
		mockRemoteRepo := new(MockTaskRepository)
		mockClassifier := new(MockTaskClassifier)
		mockUserInput := new(MockUserInput)

		// Create use case
		uc := NewClassifyTasksUseCase(mockLocalRepo, mockRemoteRepo, mockClassifier, mockUserInput)

		// Arrange
		project := "TEST"
		sprint := "Sprint 1"

		mockLocalRepo.On("FindByProjectAndSprint", ctx, project, sprint).
			Return(nil, fmt.Errorf("repository error")).Once()

		// Act
		tasks, err := uc.GetTasks(ctx, project, sprint)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, tasks)
		assert.Contains(t, err.Error(), "failed to find existing tasks")
		mockLocalRepo.AssertExpectations(t)
		mockRemoteRepo.AssertNotCalled(t, "FindByProjectAndSprint")
	})

	t.Run("should return error when remote repository fails", func(t *testing.T) {
		// Create mocks
		mockLocalRepo := new(MockTaskRepository)
		mockRemoteRepo := new(MockTaskRepository)
		mockClassifier := new(MockTaskClassifier)
		mockUserInput := new(MockUserInput)

		// Create use case
		uc := NewClassifyTasksUseCase(mockLocalRepo, mockRemoteRepo, mockClassifier, mockUserInput)

		// Arrange
		project := "TEST"
		sprint := "Sprint 1"

		mockLocalRepo.On("FindByProjectAndSprint", ctx, project, sprint).
			Return([]*domain.Task{}, nil).Once()
		mockRemoteRepo.On("FindByProjectAndSprint", ctx, project, sprint).
			Return(nil, fmt.Errorf("remote error")).Once()

		// Act
		tasks, err := uc.GetTasks(ctx, project, sprint)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, tasks)
		assert.Contains(t, err.Error(), "failed to fetch tasks")
		mockLocalRepo.AssertExpectations(t)
		mockRemoteRepo.AssertExpectations(t)
	})

	t.Run("should return error when saving fetched task fails", func(t *testing.T) {
		// Create mocks
		mockLocalRepo := new(MockTaskRepository)
		mockRemoteRepo := new(MockTaskRepository)
		mockClassifier := new(MockTaskClassifier)
		mockUserInput := new(MockUserInput)

		// Create use case
		uc := NewClassifyTasksUseCase(mockLocalRepo, mockRemoteRepo, mockClassifier, mockUserInput)

		// Arrange
		project := "TEST"
		sprint := "Sprint 1"
		remoteTasks := []*domain.Task{
			{
				Key:     "TEST-1",
				Type:    "Task",
				Summary: "Test Task 1",
				Status:  "In Progress",
			},
		}

		mockLocalRepo.On("FindByProjectAndSprint", ctx, project, sprint).
			Return([]*domain.Task{}, nil).Once()
		mockRemoteRepo.On("FindByProjectAndSprint", ctx, project, sprint).
			Return(remoteTasks, nil).Once()
		mockLocalRepo.On("Save", ctx, remoteTasks[0]).
			Return(fmt.Errorf("save error")).Once()

		// Act
		tasks, err := uc.GetTasks(ctx, project, sprint)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, tasks)
		assert.Contains(t, err.Error(), "failed to save fetched task")
		mockLocalRepo.AssertExpectations(t)
		mockRemoteRepo.AssertExpectations(t)
	})
}
