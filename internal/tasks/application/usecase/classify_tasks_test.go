package usecase

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	assetsdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application/usecase/testutil"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// createTestAsset creates a test asset for testing purposes
func createTestAsset(name string) *assetsdomain.Asset {
	asset, _ := assetsdomain.NewAsset(name, "Test description")
	return asset
}

const (
	testProject = "TEST"
	testSprint  = "Sprint 1"
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

// SaveAll defers to Save so existing test expectations set on Save
// (with Times(N), specific task args, or mock.Anything) cover the new
// batch path without needing to be rewritten.
func (m *MockTaskRepository) SaveAll(ctx context.Context, tasks []*domain.Task) error {
	for _, task := range tasks {
		if err := m.Save(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

func (m *MockTaskRepository) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockTaskRepository) DeleteByProjectAndSprint(ctx context.Context, project, sprint string) error {
	args := m.Called(ctx, project, sprint)
	return args.Error(0)
}

func (m *MockTaskRepository) UpdateLabels(ctx context.Context, taskKey string, addLabels, removeLabels []string) error {
	args := m.Called(ctx, taskKey, addLabels, removeLabels)
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

// MockComprehensiveTaskClassifier is a mock implementation of ComprehensiveTaskClassifier
type MockComprehensiveTaskClassifier struct {
	MockTaskClassifier
}

func (m *MockComprehensiveTaskClassifier) ClassifyTasksComprehensive(tasks []*domain.Task) ([]*ports.ComprehensiveClassificationResult, error) {
	args := m.Called(tasks)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ports.ComprehensiveClassificationResult), args.Error(1)
}

// MockUserInput is a mock implementation of UserInput
type MockUserInput struct {
	mock.Mock
}

func (m *MockUserInput) Confirm(format string, args ...interface{}) (bool, error) {
	// Create call arguments slice starting with format string
	callArgs := []interface{}{format}
	// Add variadic args if any
	callArgs = append(callArgs, args...)
	// Call with all arguments
	mockedArgs := m.Called(callArgs...)
	return mockedArgs.Bool(0), mockedArgs.Error(1)
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
		input         domain.ClassifyTasksInput
		existingTasks []*domain.Task
		shouldFetch   bool
		fetchedTasks  []*domain.Task
		workTypes     map[string]domain.WorkType
		expectedError bool
		expectedCalls func(*MockTaskRepository, *MockTaskRepository, *MockTaskClassifier, *MockUserInput)
	}{
		{
			name: "successfully classify existing tasks",
			input: domain.ClassifyTasksInput{
				Project: testProject,
				Sprint:  testSprint,
				DryRun:  false,
				Apply:   true,
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
			expectedCalls: func(localRepo, remoteRepo *MockTaskRepository, classifier *MockTaskClassifier, _ *MockUserInput) {
				localRepo.On("FindByProjectAndSprint", ctx, "TEST", "Sprint 1").Return([]*domain.Task{
					{Key: "TEST-1", Summary: "Task 1"},
					{Key: "TEST-2", Summary: "Task 2"},
				}, nil)
				classifier.On("ClassifyTasks", mock.Anything).Return(map[string]domain.WorkType{
					"TEST-1": domain.WorkTypeDevelopment,
					"TEST-2": domain.WorkTypeMaintenance,
				}, nil)
				localRepo.On("Save", ctx, mock.Anything).Return(nil).Times(2)
				remoteRepo.On("UpdateLabels", ctx, "TEST-1", []string{"cap-development"}, []string(nil)).Return(nil)
				remoteRepo.On("UpdateLabels", ctx, "TEST-2", []string{"cap-maintenance"}, []string(nil)).Return(nil)
			},
		},
		{
			name: "fetch and classify new tasks",
			input: domain.ClassifyTasksInput{
				Project: testProject,
				Sprint:  testSprint,
				DryRun:  false,
				Apply:   true,
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
				localRepo.On("FindByProjectAndSprint", ctx, "TEST", "Sprint 1").Return(nil, nil)
				userInput.On("Confirm", "No tasks found for project %s and sprint %s. Would you like to fetch them?", "TEST", "Sprint 1").Return(true, nil)
				remoteRepo.On("FindByProjectAndSprint", ctx, "TEST", "Sprint 1").Return([]*domain.Task{
					{Key: "TEST-3", Summary: "Task 3"},
					{Key: "TEST-4", Summary: "Task 4"},
				}, nil)
				localRepo.On("Save", ctx, mock.Anything).Return(nil).Times(4)
				classifier.On("ClassifyTasks", mock.Anything).Return(map[string]domain.WorkType{
					"TEST-3": domain.WorkTypeDiscovery,
					"TEST-4": domain.WorkTypeDevelopment,
				}, nil)
				remoteRepo.On("UpdateLabels", ctx, "TEST-3", []string{"cap-discovery"}, []string(nil)).Return(nil)
				remoteRepo.On("UpdateLabels", ctx, "TEST-4", []string{"cap-development"}, []string(nil)).Return(nil)
			},
		},
		{
			name: "dry run classification",
			input: domain.ClassifyTasksInput{
				Project: testProject,
				Sprint:  testSprint,
				DryRun:  true,
				Apply:   false,
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
			expectedCalls: func(localRepo, _ *MockTaskRepository, classifier *MockTaskClassifier, _ *MockUserInput) {
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
			assetService := testutil.NewMockAssetService()
			uc := NewClassifyTasksUseCase(localRepo, remoteRepo, classifier, userInput, assetService, nil)

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
		assetService := testutil.NewMockAssetService()
		uc := NewClassifyTasksUseCase(mockLocalRepo, mockRemoteRepo, mockClassifier, mockUserInput, assetService, nil)

		// Arrange
		project := testProject
		sprint := testSprint
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
		assetService := testutil.NewMockAssetService()
		uc := NewClassifyTasksUseCase(mockLocalRepo, mockRemoteRepo, mockClassifier, mockUserInput, assetService, nil)

		// Arrange
		project := testProject
		sprint := testSprint
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
		assetService := testutil.NewMockAssetService()
		uc := NewClassifyTasksUseCase(mockLocalRepo, mockRemoteRepo, mockClassifier, mockUserInput, assetService, nil)

		// Arrange
		project := testProject
		sprint := testSprint

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
		assetService := testutil.NewMockAssetService()
		uc := NewClassifyTasksUseCase(mockLocalRepo, mockRemoteRepo, mockClassifier, mockUserInput, assetService, nil)

		// Arrange
		project := testProject
		sprint := testSprint

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
		assetService := testutil.NewMockAssetService()
		uc := NewClassifyTasksUseCase(mockLocalRepo, mockRemoteRepo, mockClassifier, mockUserInput, assetService, nil)

		// Arrange
		project := testProject
		sprint := testSprint
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

func TestGetAllTasks(t *testing.T) {
	ctx := context.Background()

	t.Run("should return all tasks from local repository", func(t *testing.T) {
		// Create mocks
		mockLocalRepo := new(MockTaskRepository)
		mockRemoteRepo := new(MockTaskRepository)
		mockClassifier := new(MockTaskClassifier)
		mockUserInput := new(MockUserInput)

		// Create use case
		assetService := testutil.NewMockAssetService()
		uc := NewClassifyTasksUseCase(mockLocalRepo, mockRemoteRepo, mockClassifier, mockUserInput, assetService, nil)

		// Arrange
		expectedTasks := []*domain.Task{
			{Key: "TEST-1", Summary: "Task 1"},
			{Key: "TEST-2", Summary: "Task 2"},
			{Key: "DEV-1", Summary: "Dev Task"},
		}

		mockLocalRepo.On("FindAll", ctx).Return(expectedTasks, nil)

		// Act
		tasks, err := uc.GetAllTasks(ctx)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, expectedTasks, tasks)
		mockLocalRepo.AssertExpectations(t)
	})

	t.Run("should return error when repository fails", func(t *testing.T) {
		// Create mocks
		mockLocalRepo := new(MockTaskRepository)
		mockRemoteRepo := new(MockTaskRepository)
		mockClassifier := new(MockTaskClassifier)
		mockUserInput := new(MockUserInput)

		// Create use case
		assetService := testutil.NewMockAssetService()
		uc := NewClassifyTasksUseCase(mockLocalRepo, mockRemoteRepo, mockClassifier, mockUserInput, assetService, nil)

		// Arrange
		mockLocalRepo.On("FindAll", ctx).Return(nil, fmt.Errorf("repository error"))

		// Act
		tasks, err := uc.GetAllTasks(ctx)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, tasks)
		assert.Contains(t, err.Error(), "repository error")
		mockLocalRepo.AssertExpectations(t)
	})

	t.Run("should return empty slice when no tasks exist", func(t *testing.T) {
		// Create mocks
		mockLocalRepo := new(MockTaskRepository)
		mockRemoteRepo := new(MockTaskRepository)
		mockClassifier := new(MockTaskClassifier)
		mockUserInput := new(MockUserInput)

		// Create use case
		assetService := testutil.NewMockAssetService()
		uc := NewClassifyTasksUseCase(mockLocalRepo, mockRemoteRepo, mockClassifier, mockUserInput, assetService, nil)

		// Arrange
		mockLocalRepo.On("FindAll", ctx).Return([]*domain.Task{}, nil)

		// Act
		tasks, err := uc.GetAllTasks(ctx)

		// Assert
		assert.NoError(t, err)
		assert.Empty(t, tasks)
		mockLocalRepo.AssertExpectations(t)
	})
}

func TestGetLocalRepository(t *testing.T) {
	t.Run("should return the local repository instance", func(t *testing.T) {
		// Create mocks
		mockLocalRepo := new(MockTaskRepository)
		mockRemoteRepo := new(MockTaskRepository)
		mockClassifier := new(MockTaskClassifier)
		mockUserInput := new(MockUserInput)

		// Create use case
		assetService := testutil.NewMockAssetService()
		uc := NewClassifyTasksUseCase(mockLocalRepo, mockRemoteRepo, mockClassifier, mockUserInput, assetService, nil)

		// Act
		repo := uc.GetLocalRepository()

		// Assert
		assert.Equal(t, mockLocalRepo, repo)
		assert.NotNil(t, repo)
	})
}

func TestFormatWorkType(t *testing.T) {
	tests := []struct {
		name     string
		workType domain.WorkType
		expected string
	}{
		{
			name:     "should format discovery work type",
			workType: domain.WorkTypeDiscovery,
			expected: "🔍 DISCOVERY",
		},
		{
			name:     "should format development work type",
			workType: domain.WorkTypeDevelopment,
			expected: "🚀 DEVELOPMENT",
		},
		{
			name:     "should format maintenance work type",
			workType: domain.WorkTypeMaintenance,
			expected: "🔧 MAINTENANCE",
		},
		{
			name:     "should format unknown work type",
			workType: domain.WorkType("unknown"),
			expected: "❓ UNKNOWN",
		},
		{
			name:     "should format empty work type as unknown",
			workType: domain.WorkType(""),
			expected: "❓ UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatWorkType(tt.workType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildLabelChanges(t *testing.T) {
	// Create mocks for the use case (needed to call the method)
	mockLocalRepo := new(MockTaskRepository)
	mockRemoteRepo := new(MockTaskRepository)
	mockClassifier := new(MockTaskClassifier)
	mockUserInput := new(MockUserInput)
	assetService := testutil.NewMockAssetService()
	uc := NewClassifyTasksUseCase(mockLocalRepo, mockRemoteRepo, mockClassifier, mockUserInput, assetService, nil)

	tests := []struct {
		name           string
		existingLabels []string
		workType       domain.WorkType
		assetResult    *ports.AssetClassificationResult
		expectedAdd    []string
		expectedRemove []string
	}{
		{
			name:           "should add new work type label to empty labels",
			existingLabels: []string{},
			workType:       domain.WorkTypeDevelopment,
			assetResult:    nil,
			expectedAdd:    []string{"cap-development"},
			expectedRemove: nil,
		},
		{
			name:           "should replace existing work type label",
			existingLabels: []string{"cap-maintenance", "other-label"},
			workType:       domain.WorkTypeDevelopment,
			assetResult:    nil,
			expectedAdd:    []string{"cap-development"},
			expectedRemove: []string{"cap-maintenance"},
		},
		{
			name:           "should add asset label when provided",
			existingLabels: []string{"existing-label"},
			workType:       domain.WorkTypeDevelopment,
			assetResult: &ports.AssetClassificationResult{
				Asset:      createTestAsset("Test Asset"),
				Confidence: 0.85,
				Reason:     "keyword match",
			},
			expectedAdd:    []string{"cap-development", "cap-asset-test-asset"},
			expectedRemove: nil,
		},
		{
			name:           "should preserve existing asset label when confidence is high and reason indicates preservation",
			existingLabels: []string{"cap-asset-existing", "cap-maintenance", "other-label"},
			workType:       domain.WorkTypeDevelopment,
			assetResult: &ports.AssetClassificationResult{
				Asset:      createTestAsset("Existing Asset"),
				Confidence: 0.95,
				Reason:     "existing asset label preserved",
			},
			expectedAdd:    []string{"cap-development"},
			expectedRemove: []string{"cap-maintenance"},
		},
		{
			name:           "should replace asset label when confidence is low",
			existingLabels: []string{"cap-asset-old", "cap-maintenance", "other-label"},
			workType:       domain.WorkTypeDevelopment,
			assetResult: &ports.AssetClassificationResult{
				Asset:      createTestAsset("New Asset"),
				Confidence: 0.75,
				Reason:     "keyword match",
			},
			expectedAdd:    []string{"cap-development", "cap-asset-new-asset"},
			expectedRemove: []string{"cap-maintenance", "cap-asset-old"},
		},
		{
			name:           "should handle multiple work type labels and replace all",
			existingLabels: []string{"cap-development", "cap-maintenance", "cap-discovery", "other-label"},
			workType:       domain.WorkTypeMaintenance,
			assetResult:    nil,
			expectedAdd:    []string{"cap-maintenance"},
			expectedRemove: []string{"cap-development", "cap-discovery"},
		},
		{
			name:           "should handle multiple asset labels and replace with new one",
			existingLabels: []string{"cap-asset-old1", "cap-asset-old2", "other-label"},
			workType:       domain.WorkTypeDevelopment,
			assetResult: &ports.AssetClassificationResult{
				Asset:      createTestAsset("New Asset"),
				Confidence: 0.90,
				Reason:     "best match found",
			},
			expectedAdd:    []string{"cap-development", "cap-asset-new-asset"},
			expectedRemove: []string{"cap-asset-old1", "cap-asset-old2"},
		},
		{
			name:           "should not touch non-cap labels",
			existingLabels: []string{"bug", "priority-high", "team-backend", "cap-development"},
			workType:       domain.WorkTypeDiscovery,
			assetResult:    nil,
			expectedAdd:    []string{"cap-discovery"},
			expectedRemove: []string{"cap-development"},
		},
		{
			name:           "should handle asset result with nil asset",
			existingLabels: []string{"existing-label"},
			workType:       domain.WorkTypeDevelopment,
			assetResult: &ports.AssetClassificationResult{
				Asset:      nil,
				Confidence: 0.0,
				Reason:     "no match found",
			},
			expectedAdd:    []string{"cap-development"},
			expectedRemove: nil,
		},
		{
			name:           "should handle complex scenario with preservation and replacement",
			existingLabels: []string{"cap-asset-preserved", "cap-maintenance", "bug", "priority-high"},
			workType:       domain.WorkTypeDevelopment,
			assetResult: &ports.AssetClassificationResult{
				Asset:      createTestAsset("Preserved Asset"),
				Confidence: 0.98,
				Reason:     "existing asset label preserved",
			},
			expectedAdd:    []string{"cap-development"},
			expectedRemove: []string{"cap-maintenance"},
		},
		{
			name:           "should not remove the same label being added",
			existingLabels: []string{"cap-development"},
			workType:       domain.WorkTypeDevelopment,
			assetResult:    nil,
			expectedAdd:    []string{"cap-development"},
			expectedRemove: nil,
		},
		{
			name:           "should never touch non-cap labels like workstream",
			existingLabels: []string{"workstream", "cap-maintenance", "team-backend", "cap-asset-old-feature"},
			workType:       domain.WorkTypeDevelopment,
			assetResult: &ports.AssetClassificationResult{
				Asset:      createTestAsset("New Feature"),
				Confidence: 0.85,
				Reason:     "keyword match",
			},
			expectedAdd:    []string{"cap-development", "cap-asset-new-feature"},
			expectedRemove: []string{"cap-maintenance", "cap-asset-old-feature"},
		},
		{
			name:           "should only produce cap-prefixed labels in add and remove lists",
			existingLabels: []string{"bug", "priority-high", "workstream", "sprint-goal", "cap-discovery", "cap-asset-payments"},
			workType:       domain.WorkTypeMaintenance,
			assetResult: &ports.AssetClassificationResult{
				Asset:      createTestAsset("Infrastructure"),
				Confidence: 0.80,
				Reason:     "keyword match",
			},
			expectedAdd:    []string{"cap-maintenance", "cap-asset-infrastructure"},
			expectedRemove: []string{"cap-discovery", "cap-asset-payments"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addLabels, removeLabels := uc.buildLabelChanges(tt.existingLabels, tt.workType, tt.assetResult)
			assert.ElementsMatch(t, tt.expectedAdd, addLabels, "add labels mismatch")
			if tt.expectedRemove == nil {
				assert.Empty(t, removeLabels, "remove labels should be empty")
			} else {
				assert.ElementsMatch(t, tt.expectedRemove, removeLabels, "remove labels mismatch")
			}
		})
	}
}

func TestGetAssetLabel(t *testing.T) {
	// Create mocks for the use case (needed to call the method)
	mockLocalRepo := new(MockTaskRepository)
	mockRemoteRepo := new(MockTaskRepository)
	mockClassifier := new(MockTaskClassifier)
	mockUserInput := new(MockUserInput)
	assetService := testutil.NewMockAssetService()
	uc := NewClassifyTasksUseCase(mockLocalRepo, mockRemoteRepo, mockClassifier, mockUserInput, assetService, nil)

	tests := []struct {
		name      string
		assetName string
		expected  string
	}{
		{
			name:      "should format simple asset name",
			assetName: "TestAsset",
			expected:  "cap-asset-testasset",
		},
		{
			name:      "should format asset name with spaces",
			assetName: "My Test Asset",
			expected:  "cap-asset-my-test-asset",
		},
		{
			name:      "should format asset name with mixed case",
			assetName: "CamelCaseAsset",
			expected:  "cap-asset-camelcaseasset",
		},
		{
			name:      "should format asset name with multiple spaces",
			assetName: "Asset  With   Multiple Spaces",
			expected:  "cap-asset-asset--with---multiple-spaces",
		},
		{
			name:      "should format single word asset",
			assetName: "Asset",
			expected:  "cap-asset-asset",
		},
		{
			name:      "should handle empty asset name",
			assetName: "",
			expected:  "cap-asset-",
		},
		{
			name:      "should format asset name with special characters",
			assetName: "Asset With-Hyphens_And_Underscores",
			expected:  "cap-asset-asset-with-hyphens_and_underscores",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uc.getAssetLabel(tt.assetName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetAssetLabel_WithAssetObject(t *testing.T) {
	mockLocalRepo := new(MockTaskRepository)
	mockRemoteRepo := new(MockTaskRepository)
	mockClassifier := new(MockTaskClassifier)
	mockUserInput := new(MockUserInput)
	assetService := testutil.NewMockAssetService()
	uc := NewClassifyTasksUseCase(mockLocalRepo, mockRemoteRepo, mockClassifier, mockUserInput, assetService, nil)

	t.Run("should use asset ID when it has cap-asset prefix", func(t *testing.T) {
		asset := &assetsdomain.Asset{ID: "cap-asset-mode-comparison-optimization", Name: "Mode Comparison Optimization"}
		result := uc.getAssetLabel(asset)
		assert.Equal(t, "cap-asset-mode-comparison-optimization", result)
	})

	t.Run("should generate label from asset Name when ID has no cap-asset prefix", func(t *testing.T) {
		asset := &assetsdomain.Asset{ID: "some-other-id", Name: "Mode Comparison Optimization"}
		result := uc.getAssetLabel(asset)
		assert.Equal(t, "cap-asset-mode-comparison-optimization", result)
	})

	t.Run("should generate label from asset Name when ID is empty", func(t *testing.T) {
		asset := &assetsdomain.Asset{Name: "Home Page Optimization"}
		result := uc.getAssetLabel(asset)
		assert.Equal(t, "cap-asset-home-page-optimization", result)
	})

	t.Run("should handle asset with special characters in name", func(t *testing.T) {
		asset := &assetsdomain.Asset{Name: "Dynamic Currency Conversion (DCC)"}
		result := uc.getAssetLabel(asset)
		assert.Equal(t, "cap-asset-dynamic-currency-conversion-dcc", result)
	})

	t.Run("should return unknown for nil asset", func(t *testing.T) {
		result := uc.getAssetLabel(nil)
		assert.Equal(t, "cap-asset-unknown", result)
	})
}

func TestFormatAssetLabel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple name", "Home Page", "cap-asset-home-page"},
		{"with ampersand", "Search & Filter", "cap-asset-search-and-filter"},
		{"with parentheses", "Dynamic Currency Conversion (DCC)", "cap-asset-dynamic-currency-conversion-dcc"},
		{"already lowercase", "booking success", "cap-asset-booking-success"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatAssetLabel(tt.input))
		})
	}
}

func TestClassifyTasksComprehensive(t *testing.T) {
	ctx := context.Background()

	t.Run("should use comprehensive classification when available", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		comprehensiveClassifier := new(MockComprehensiveTaskClassifier)
		userInput := new(MockUserInput)

		// Create use case
		assetService := testutil.NewMockAssetService()
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, comprehensiveClassifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  false,
			Apply:   true,
		}

		tasks := []*domain.Task{
			{Key: "TEST-1", Summary: "Task 1"},
			{Key: "TEST-2", Summary: "Task 2"},
		}

		// Create test asset with proper ID
		testAsset, err := assetsdomain.NewAsset("Test Asset", "Test asset description")
		require.NoError(t, err)

		comprehensiveResults := []*ports.ComprehensiveClassificationResult{
			{
				Task:     tasks[0],
				WorkType: domain.WorkTypeDevelopment,
				Asset: &ports.AssetClassificationResult{
					Asset:      testAsset,
					Confidence: 0.95,
					Reason:     "Keyword match in summary",
				},
				WorkTypeReason: "Development task indicators found",
			},
			{
				Task:           tasks[1],
				WorkType:       domain.WorkTypeMaintenance,
				Asset:          nil, // No asset assignment
				WorkTypeReason: "Maintenance task pattern detected",
			},
		}

		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return(tasks, nil)
		comprehensiveClassifier.On("ClassifyTasksComprehensive", tasks).Return(comprehensiveResults, nil)
		localRepo.On("Save", ctx, mock.Anything).Return(nil).Times(2)
		remoteRepo.On("UpdateLabels", ctx, "TEST-1", []string{"cap-development", "cap-asset-test-asset"}, []string(nil)).Return(nil)
		remoteRepo.On("UpdateLabels", ctx, "TEST-2", []string{"cap-maintenance"}, []string(nil)).Return(nil)

		// Act
		err = uc.Execute(ctx, input)

		// Assert
		assert.NoError(t, err)
		localRepo.AssertExpectations(t)
		remoteRepo.AssertExpectations(t)
		comprehensiveClassifier.AssertExpectations(t)
	})

	t.Run("should handle comprehensive classification error", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		comprehensiveClassifier := new(MockComprehensiveTaskClassifier)
		userInput := new(MockUserInput)

		// Create use case
		assetService := testutil.NewMockAssetService()
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, comprehensiveClassifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  false,
			Apply:   false,
		}

		tasks := []*domain.Task{{Key: "TEST-1", Summary: "Test task"}}

		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return(tasks, nil)
		comprehensiveClassifier.On("ClassifyTasksComprehensive", tasks).Return(nil, fmt.Errorf("comprehensive classification error"))

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to classify tasks comprehensively")
		localRepo.AssertExpectations(t)
		comprehensiveClassifier.AssertExpectations(t)
	})

	t.Run("should fall back to simple classification when comprehensive not available", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		simpleClassifier := new(MockTaskClassifier) // Regular classifier, not comprehensive
		userInput := new(MockUserInput)

		// Create use case
		assetService := testutil.NewMockAssetService()
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, simpleClassifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  false,
			Apply:   true,
		}

		tasks := []*domain.Task{{Key: "TEST-1", Summary: "Test task"}}
		workTypes := map[string]domain.WorkType{
			"TEST-1": domain.WorkTypeDevelopment,
		}

		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return(tasks, nil)
		simpleClassifier.On("ClassifyTasks", tasks).Return(workTypes, nil)
		localRepo.On("Save", ctx, mock.Anything).Return(nil)
		remoteRepo.On("UpdateLabels", ctx, "TEST-1", []string{"cap-development"}, []string(nil)).Return(nil)

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.NoError(t, err)
		localRepo.AssertExpectations(t)
		remoteRepo.AssertExpectations(t)
		simpleClassifier.AssertExpectations(t)
	})
}

func TestClassifyTasksEdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("should handle classification with empty tasks", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		classifier := new(MockTaskClassifier)
		userInput := new(MockUserInput)

		// Create use case
		assetService := testutil.NewMockAssetService()
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, classifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  false,
			Apply:   false,
		}

		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return([]*domain.Task{}, nil)
		// Mock the user input for "no tasks found" scenario
		userInput.On("Confirm", "No tasks found for project %s and sprint %s. Would you like to fetch them?", testProject, testSprint).Return(false, nil)

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no tasks available for classification")
		localRepo.AssertExpectations(t)
		userInput.AssertExpectations(t)
	})

	t.Run("should handle classification error", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		classifier := new(MockTaskClassifier)
		userInput := new(MockUserInput)

		// Create use case
		assetService := testutil.NewMockAssetService()
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, classifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  false,
			Apply:   false,
		}

		tasks := []*domain.Task{{Key: "TEST-1", Summary: "Test task"}}
		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return(tasks, nil)
		classifier.On("ClassifyTasks", tasks).Return(nil, fmt.Errorf("classification error"))

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to classify tasks")
		localRepo.AssertExpectations(t)
		classifier.AssertExpectations(t)
	})

	t.Run("should handle user refusing to fetch tasks", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		classifier := new(MockTaskClassifier)
		userInput := new(MockUserInput)

		// Create use case
		assetService := testutil.NewMockAssetService()
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, classifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  false,
			Apply:   false,
		}

		// Mock no existing tasks and user refusing to fetch
		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return([]*domain.Task{}, nil)
		userInput.On("Confirm", "No tasks found for project %s and sprint %s. Would you like to fetch them?", testProject, testSprint).Return(false, nil)

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no tasks available for classification")
		localRepo.AssertExpectations(t)
		userInput.AssertExpectations(t)
	})

	t.Run("should successfully apply classifications when apply is true", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		classifier := new(MockTaskClassifier)
		userInput := new(MockUserInput)

		// Create use case
		assetService := testutil.NewMockAssetService()
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, classifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  false,
			Apply:   true,
		}

		tasks := []*domain.Task{
			{Key: "TEST-1", Summary: "Test task"},
		}
		workTypes := map[string]domain.WorkType{
			"TEST-1": domain.WorkTypeDevelopment,
		}

		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return(tasks, nil)
		classifier.On("ClassifyTasks", tasks).Return(workTypes, nil)
		localRepo.On("Save", ctx, mock.Anything).Return(nil)
		remoteRepo.On("UpdateLabels", ctx, "TEST-1", []string{"cap-development"}, []string(nil)).Return(nil)

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.NoError(t, err)
		localRepo.AssertExpectations(t)
		classifier.AssertExpectations(t)
		remoteRepo.AssertExpectations(t)
	})

	t.Run("should handle user input error during fetch confirmation", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		classifier := new(MockTaskClassifier)
		userInput := new(MockUserInput)

		// Create use case
		assetService := testutil.NewMockAssetService()
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, classifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  false,
			Apply:   false,
		}

		// Mock no existing tasks, which triggers user input for fetching
		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return([]*domain.Task{}, nil)
		userInput.On("Confirm", "No tasks found for project %s and sprint %s. Would you like to fetch them?", testProject, testSprint).Return(false, fmt.Errorf("user input error"))

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get user confirmation")
		localRepo.AssertExpectations(t)
		userInput.AssertExpectations(t)
	})

	t.Run("should handle remote repository update error", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		classifier := new(MockTaskClassifier)
		userInput := new(MockUserInput)

		// Create use case
		assetService := testutil.NewMockAssetService()
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, classifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  false,
			Apply:   true,
		}

		tasks := []*domain.Task{
			{Key: "TEST-1", Summary: "Test task"},
		}
		workTypes := map[string]domain.WorkType{
			"TEST-1": domain.WorkTypeDevelopment,
		}

		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return(tasks, nil)
		classifier.On("ClassifyTasks", tasks).Return(workTypes, nil)
		localRepo.On("Save", ctx, mock.Anything).Return(nil)
		remoteRepo.On("UpdateLabels", ctx, "TEST-1", []string{"cap-development"}, []string(nil)).Return(fmt.Errorf("update error"))

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to apply labels to task TEST-1")
		localRepo.AssertExpectations(t)
		remoteRepo.AssertExpectations(t)
		classifier.AssertExpectations(t)
	})

	t.Run("should handle local repository save error during classification", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		classifier := new(MockTaskClassifier)
		userInput := new(MockUserInput)

		// Create use case
		assetService := testutil.NewMockAssetService()
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, classifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  false,
			Apply:   false,
		}

		tasks := []*domain.Task{
			{Key: "TEST-1", Summary: "Test task"},
		}
		workTypes := map[string]domain.WorkType{
			"TEST-1": domain.WorkTypeDevelopment,
		}

		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return(tasks, nil)
		classifier.On("ClassifyTasks", tasks).Return(workTypes, nil)
		// Save delegation under MockTaskRepository.SaveAll covers this expectation.
		localRepo.On("Save", ctx, mock.Anything).Return(fmt.Errorf("save error"))

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to persist classified tasks")
		localRepo.AssertExpectations(t)
		classifier.AssertExpectations(t)
	})
}

func TestPreviewClassificationsWithAssetSync(t *testing.T) {
	ctx := context.Background()

	t.Run("should trigger asset sync when unassigned tasks found in comprehensive preview", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		comprehensiveClassifier := new(MockComprehensiveTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()

		// Create use case
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, comprehensiveClassifier, userInput, assetService, nil)

		// Arrange - dry run input that triggers preview
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  true,
			Apply:   false,
		}

		tasks := []*domain.Task{
			{Key: "TEST-1", Summary: "Task 1", Type: "Story", Status: "To Do"},
			{Key: "TEST-2", Summary: "Task 2", Type: "Bug", Status: "In Progress"},
		}

		// First call - with unassigned tasks
		firstResults := []*ports.ComprehensiveClassificationResult{
			{
				Task:           tasks[0],
				WorkType:       domain.WorkTypeDevelopment,
				Asset:          nil, // No asset assignment - triggers sync
				WorkTypeReason: "Development task pattern",
			},
			{
				Task:           tasks[1],
				WorkType:       domain.WorkTypeMaintenance,
				Asset:          nil, // No asset assignment - triggers sync
				WorkTypeReason: "Maintenance task pattern",
			},
		}

		// Second call - after sync with asset assignments
		secondResults := []*ports.ComprehensiveClassificationResult{
			{
				Task:     tasks[0],
				WorkType: domain.WorkTypeDevelopment,
				Asset: &ports.AssetClassificationResult{
					Asset:      &assetsdomain.Asset{Name: "Synced Asset 1"},
					Confidence: 0.85,
					Reason:     "Post-sync keyword match",
				},
				WorkTypeReason: "Development task pattern",
			},
			{
				Task:     tasks[1],
				WorkType: domain.WorkTypeMaintenance,
				Asset: &ports.AssetClassificationResult{
					Asset:      &assetsdomain.Asset{Name: "Synced Asset 2"},
					Confidence: 0.90,
					Reason:     "Post-sync description match",
				},
				WorkTypeReason: "Maintenance task pattern",
			},
		}

		syncResult := &assetsdomain.SyncResult{
			SyncedAssets: []*assetsdomain.Asset{
				{Name: "Synced Asset 1"},
				{Name: "Synced Asset 2"},
			},
			NotSyncedAssets: []*assetsdomain.NotSyncedAsset{},
		}

		// Setup expectations
		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return(tasks, nil)
		comprehensiveClassifier.On("ClassifyTasksComprehensive", tasks).Return(firstResults, nil).Once()
		userInput.On("Confirm", "Would you like to sync assets from Confluence to potentially improve classification?").Return(true, nil).Once()
		assetService.SetSyncFromConfluenceFunc(func(spaceKey, label string, debug bool) (*assetsdomain.SyncResult, error) {
			assert.Equal(t, "CAP", spaceKey)
			assert.Equal(t, "cap-asset", label)
			assert.False(t, debug)
			return syncResult, nil
		})
		comprehensiveClassifier.On("ClassifyTasksComprehensive", tasks).Return(secondResults, nil).Once()

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.NoError(t, err)
		localRepo.AssertExpectations(t)
		comprehensiveClassifier.AssertExpectations(t)
		userInput.AssertExpectations(t)
	})

	t.Run("should handle asset sync error gracefully and continue with classification", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		comprehensiveClassifier := new(MockComprehensiveTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()

		// Create use case
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, comprehensiveClassifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  true,
			Apply:   false,
		}

		tasks := []*domain.Task{{Key: "TEST-1", Summary: "Task 1"}}
		results := []*ports.ComprehensiveClassificationResult{
			{
				Task:           tasks[0],
				WorkType:       domain.WorkTypeDevelopment,
				Asset:          nil, // No asset assignment
				WorkTypeReason: "Development pattern",
			},
		}

		// Setup expectations
		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return(tasks, nil)
		comprehensiveClassifier.On("ClassifyTasksComprehensive", tasks).Return(results, nil)
		userInput.On("Confirm", "Would you like to sync assets from Confluence to potentially improve classification?").Return(true, nil)
		assetService.SetSyncFromConfluenceFunc(func(_, _ string, _ bool) (*assetsdomain.SyncResult, error) {
			return nil, fmt.Errorf("sync failed")
		})

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.NoError(t, err) // Should continue despite sync error
		localRepo.AssertExpectations(t)
		comprehensiveClassifier.AssertExpectations(t)
		userInput.AssertExpectations(t)
	})

	t.Run("should skip asset sync when user declines", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		comprehensiveClassifier := new(MockComprehensiveTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()

		// Create use case
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, comprehensiveClassifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  true,
			Apply:   false,
		}

		tasks := []*domain.Task{{Key: "TEST-1", Summary: "Task 1"}}
		results := []*ports.ComprehensiveClassificationResult{
			{
				Task:           tasks[0],
				WorkType:       domain.WorkTypeDevelopment,
				Asset:          nil, // No asset assignment
				WorkTypeReason: "Development pattern",
			},
		}

		// Setup expectations
		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return(tasks, nil)
		comprehensiveClassifier.On("ClassifyTasksComprehensive", tasks).Return(results, nil)
		userInput.On("Confirm", "Would you like to sync assets from Confluence to potentially improve classification?").Return(false, nil)

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.NoError(t, err)
		localRepo.AssertExpectations(t)
		comprehensiveClassifier.AssertExpectations(t)
		userInput.AssertExpectations(t)
	})

	t.Run("should handle user confirmation error during asset sync", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		comprehensiveClassifier := new(MockComprehensiveTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()

		// Create use case
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, comprehensiveClassifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  true,
			Apply:   false,
		}

		tasks := []*domain.Task{{Key: "TEST-1", Summary: "Task 1"}}
		results := []*ports.ComprehensiveClassificationResult{
			{
				Task:           tasks[0],
				WorkType:       domain.WorkTypeDevelopment,
				Asset:          nil,
				WorkTypeReason: "Development pattern",
			},
		}

		// Setup expectations
		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return(tasks, nil)
		comprehensiveClassifier.On("ClassifyTasksComprehensive", tasks).Return(results, nil)
		userInput.On("Confirm", "Would you like to sync assets from Confluence to potentially improve classification?").Return(false, fmt.Errorf("user input error"))

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get user confirmation for asset sync")
		localRepo.AssertExpectations(t)
		comprehensiveClassifier.AssertExpectations(t)
		userInput.AssertExpectations(t)
	})

	t.Run("should not trigger asset sync when all tasks have assets assigned", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		comprehensiveClassifier := new(MockComprehensiveTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()

		// Create use case
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, comprehensiveClassifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  true,
			Apply:   false,
		}

		tasks := []*domain.Task{{Key: "TEST-1", Summary: "Task 1"}}
		results := []*ports.ComprehensiveClassificationResult{
			{
				Task:     tasks[0],
				WorkType: domain.WorkTypeDevelopment,
				Asset: &ports.AssetClassificationResult{
					Asset:      &assetsdomain.Asset{Name: "Existing Asset"},
					Confidence: 0.95,
					Reason:     "keyword match",
				},
				WorkTypeReason: "Development pattern",
			},
		}

		// Setup expectations
		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return(tasks, nil)
		comprehensiveClassifier.On("ClassifyTasksComprehensive", tasks).Return(results, nil)
		// No user input or asset service calls expected

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.NoError(t, err)
		localRepo.AssertExpectations(t)
		comprehensiveClassifier.AssertExpectations(t)
		userInput.AssertNotCalled(t, "Confirm")
	})

	t.Run("should skip asset sync retry when hasTriedSync is true", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		comprehensiveClassifier := new(MockComprehensiveTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()

		// Create use case
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, comprehensiveClassifier, userInput, assetService, nil)

		tasks := []*domain.Task{{Key: "TEST-1", Summary: "Task 1"}}
		results := []*ports.ComprehensiveClassificationResult{
			{
				Task:           tasks[0],
				WorkType:       domain.WorkTypeDevelopment,
				Asset:          nil, // No asset assignment
				WorkTypeReason: "Development pattern",
			},
		}

		comprehensiveClassifier.On("ClassifyTasksComprehensive", tasks).Return(results, nil)

		// Act - call previewClassificationsWithRetry directly with hasTriedSync=true
		err := uc.previewClassificationsWithRetry(tasks, true, false)

		// Assert
		assert.NoError(t, err)
		comprehensiveClassifier.AssertExpectations(t)
		userInput.AssertNotCalled(t, "Confirm") // Should not ask for sync when hasTriedSync=true
	})
}

func TestAdditionalErrorHandlingAndEdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("should handle UpdateWorkType error during classification", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		comprehensiveClassifier := new(MockComprehensiveTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()

		// Create use case
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, comprehensiveClassifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  false,
			Apply:   false,
		}

		// Create a task that will cause UpdateWorkType to fail
		task := &domain.Task{Key: "TEST-1", Summary: "Test task"}

		results := []*ports.ComprehensiveClassificationResult{
			{
				Task:           task,
				WorkType:       "INVALID_WORK_TYPE", // This should cause UpdateWorkType to fail
				Asset:          nil,
				WorkTypeReason: "Test reason",
			},
		}

		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return([]*domain.Task{task}, nil)
		comprehensiveClassifier.On("ClassifyTasksComprehensive", mock.Anything).Return(results, nil)

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update work type for task TEST-1")
		localRepo.AssertExpectations(t)
		comprehensiveClassifier.AssertExpectations(t)
	})

	t.Run("should handle comprehensive classification with simple classifier fallback error", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		simpleClassifier := new(MockTaskClassifier) // Simple classifier, not comprehensive
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()

		// Create use case
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, simpleClassifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  false,
			Apply:   false,
		}

		tasks := []*domain.Task{{Key: "TEST-1", Summary: "Test task"}}

		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return(tasks, nil)
		simpleClassifier.On("ClassifyTasks", tasks).Return(nil, fmt.Errorf("simple classification failed"))

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to classify tasks")
		localRepo.AssertExpectations(t)
		simpleClassifier.AssertExpectations(t)
	})

	t.Run("should handle empty task list from repository", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		classifier := new(MockTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()

		// Create use case
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, classifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  false,
			Apply:   false,
		}

		// Repository returns nil (empty) tasks
		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return([]*domain.Task(nil), nil)
		userInput.On("Confirm", "No tasks found for project %s and sprint %s. Would you like to fetch them?", testProject, testSprint).Return(false, nil)

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no tasks available for classification")
		localRepo.AssertExpectations(t)
		userInput.AssertExpectations(t)
	})

	t.Run("should handle GetTasks with nil return from remote repository", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		classifier := new(MockTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()

		// Create use case
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, classifier, userInput, assetService, nil)

		// Arrange
		project := testProject
		sprint := testSprint

		localRepo.On("FindByProjectAndSprint", ctx, project, sprint).Return([]*domain.Task{}, nil)
		remoteRepo.On("FindByProjectAndSprint", ctx, project, sprint).Return([]*domain.Task(nil), nil)

		// Act
		tasks, err := uc.GetTasks(ctx, project, sprint)

		// Assert
		assert.NoError(t, err)
		assert.Empty(t, tasks)
		localRepo.AssertExpectations(t)
		remoteRepo.AssertExpectations(t)
	})

	t.Run("should handle classification with mixed success and failure", func(t *testing.T) {
		// Under the original semantics this exercised an interleaved Save/Apply
		// loop where Save(TEST-1) + Apply(TEST-1) succeeded before Save(TEST-2)
		// failed. The batch-then-push design persists every task in one
		// SaveAll call BEFORE any remote update, so a save failure now aborts
		// before any UpdateLabels call. The test therefore asserts that:
		//   - SaveAll returns the second task's failure (delegation routes
		//     to Save(tasks[1]) which is wired to fail);
		//   - the error message reports the batch-level persistence failure;
		//   - no UpdateLabels expectation fires because phase 3 never runs.
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		comprehensiveClassifier := new(MockComprehensiveTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()

		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, comprehensiveClassifier, userInput, assetService, nil)

		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  false,
			Apply:   true,
		}

		tasks := []*domain.Task{
			{Key: "TEST-1", Summary: "Task 1"},
			{Key: "TEST-2", Summary: "Task 2"},
		}

		results := []*ports.ComprehensiveClassificationResult{
			{
				Task:           tasks[0],
				WorkType:       domain.WorkTypeDevelopment,
				Asset:          nil,
				WorkTypeReason: "Development pattern",
			},
			{
				Task:           tasks[1],
				WorkType:       domain.WorkTypeMaintenance,
				Asset:          nil,
				WorkTypeReason: "Maintenance pattern",
			},
		}

		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return(tasks, nil)
		comprehensiveClassifier.On("ClassifyTasksComprehensive", tasks).Return(results, nil)
		localRepo.On("Save", ctx, tasks[0]).Return(nil)
		localRepo.On("Save", ctx, tasks[1]).Return(fmt.Errorf("save error"))

		err := uc.Execute(ctx, input)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to persist classified tasks")
		localRepo.AssertExpectations(t)
		remoteRepo.AssertExpectations(t)
		comprehensiveClassifier.AssertExpectations(t)
	})

	t.Run("should handle preview with comprehensive classification error", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		comprehensiveClassifier := new(MockComprehensiveTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()

		// Create use case
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, comprehensiveClassifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  true,
			Apply:   false,
		}

		tasks := []*domain.Task{{Key: "TEST-1", Summary: "Task 1"}}

		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return(tasks, nil)
		comprehensiveClassifier.On("ClassifyTasksComprehensive", tasks).Return(nil, fmt.Errorf("comprehensive preview error"))

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to classify tasks comprehensively")
		localRepo.AssertExpectations(t)
		comprehensiveClassifier.AssertExpectations(t)
	})

	t.Run("should handle GetAllTasks with nil result", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		classifier := new(MockTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()

		// Create use case
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, classifier, userInput, assetService, nil)

		// Arrange
		localRepo.On("FindAll", ctx).Return([]*domain.Task(nil), nil)

		// Act
		tasks, err := uc.GetAllTasks(ctx)

		// Assert
		assert.NoError(t, err)
		assert.Nil(t, tasks)
		localRepo.AssertExpectations(t)
	})

	t.Run("should handle task with existing labels in comprehensive mode", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		comprehensiveClassifier := new(MockComprehensiveTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()

		// Create use case
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, comprehensiveClassifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  false,
			Apply:   true,
		}

		task := &domain.Task{
			Key:     "TEST-1",
			Summary: "Task 1",
			Labels:  []string{"existing-label", "cap-maintenance", "cap-asset-old"},
		}

		results := []*ports.ComprehensiveClassificationResult{
			{
				Task:     task,
				WorkType: domain.WorkTypeDevelopment,
				Asset: &ports.AssetClassificationResult{
					Asset:      createTestAsset("Test Asset"),
					Confidence: 0.90,
					Reason:     "keyword match",
				},
				WorkTypeReason: "Development pattern",
			},
		}

		expectedAddLabels := []string{"cap-development", "cap-asset-test-asset"}
		expectedRemoveLabels := []string{"cap-maintenance", "cap-asset-old"}

		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return([]*domain.Task{task}, nil)
		comprehensiveClassifier.On("ClassifyTasksComprehensive", []*domain.Task{task}).Return(results, nil)
		localRepo.On("Save", ctx, mock.Anything).Return(nil)
		remoteRepo.On("UpdateLabels", ctx, "TEST-1", expectedAddLabels, expectedRemoveLabels).Return(nil)

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.NoError(t, err)
		localRepo.AssertExpectations(t)
		remoteRepo.AssertExpectations(t)
		comprehensiveClassifier.AssertExpectations(t)
	})

	t.Run("should never include non-cap labels in add or remove operations", func(t *testing.T) {
		// This test reproduces the original bug: COP-147 had a "workstream" label
		// that was overwritten when classify --apply used PUT with full label replacement.
		// With add/remove operations, "workstream" must never appear in either list.
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		comprehensiveClassifier := new(MockComprehensiveTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()

		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, comprehensiveClassifier, userInput, assetService, nil)

		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  false,
			Apply:   true,
		}

		task := &domain.Task{
			Key:     "COP-147",
			Summary: "Fix payment processing",
			Labels:  []string{"workstream", "cap-maintenance"},
		}

		results := []*ports.ComprehensiveClassificationResult{
			{
				Task:     task,
				WorkType: domain.WorkTypeDevelopment,
				Asset: &ports.AssetClassificationResult{
					Asset:      createTestAsset("Payment Processing"),
					Confidence: 0.90,
					Reason:     "keyword match",
				},
				WorkTypeReason: "Development pattern",
			},
		}

		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return([]*domain.Task{task}, nil)
		comprehensiveClassifier.On("ClassifyTasksComprehensive", []*domain.Task{task}).Return(results, nil)
		localRepo.On("Save", ctx, mock.Anything).Return(nil)

		// The key assertion: only cap-prefixed labels appear in add/remove.
		// "workstream" must NOT appear in either list.
		remoteRepo.On("UpdateLabels", ctx, "COP-147",
			[]string{"cap-development", "cap-asset-payment-processing"},
			[]string{"cap-maintenance"},
		).Return(nil)

		err := uc.Execute(ctx, input)

		assert.NoError(t, err)
		localRepo.AssertExpectations(t)
		remoteRepo.AssertExpectations(t)
		comprehensiveClassifier.AssertExpectations(t)
	})
}

func TestSortingAndDisplayLogic(t *testing.T) {
	ctx := context.Background()

	t.Run("should sort tasks alphabetically during classification", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		comprehensiveClassifier := new(MockComprehensiveTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()

		// Create use case
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, comprehensiveClassifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  false,
			Apply:   true,
		}

		// Tasks in non-alphabetical order
		tasks := []*domain.Task{
			{Key: "TEST-3", Summary: "Task 3"},
			{Key: "TEST-1", Summary: "Task 1"},
			{Key: "TEST-2", Summary: "Task 2"},
		}

		results := []*ports.ComprehensiveClassificationResult{
			{
				Task:           tasks[0],
				WorkType:       domain.WorkTypeDevelopment,
				WorkTypeReason: "Development pattern",
			},
			{
				Task:           tasks[1],
				WorkType:       domain.WorkTypeMaintenance,
				WorkTypeReason: "Maintenance pattern",
			},
			{
				Task:           tasks[2],
				WorkType:       domain.WorkTypeDiscovery,
				WorkTypeReason: "Discovery pattern",
			},
		}

		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return(tasks, nil)
		comprehensiveClassifier.On("ClassifyTasksComprehensive", tasks).Return(results, nil)

		// Expect saves and updates in alphabetical order (TEST-1, TEST-2, TEST-3)
		localRepo.On("Save", ctx, mock.MatchedBy(func(task *domain.Task) bool {
			return task.Key == "TEST-1"
		})).Return(nil).Once()
		remoteRepo.On("UpdateLabels", ctx, "TEST-1", []string{"cap-maintenance"}, []string(nil)).Return(nil).Once()

		localRepo.On("Save", ctx, mock.MatchedBy(func(task *domain.Task) bool {
			return task.Key == "TEST-2"
		})).Return(nil).Once()
		remoteRepo.On("UpdateLabels", ctx, "TEST-2", []string{"cap-discovery"}, []string(nil)).Return(nil).Once()

		localRepo.On("Save", ctx, mock.MatchedBy(func(task *domain.Task) bool {
			return task.Key == "TEST-3"
		})).Return(nil).Once()
		remoteRepo.On("UpdateLabels", ctx, "TEST-3", []string{"cap-development"}, []string(nil)).Return(nil).Once()

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.NoError(t, err)
		localRepo.AssertExpectations(t)
		remoteRepo.AssertExpectations(t)
		comprehensiveClassifier.AssertExpectations(t)
	})

	t.Run("should sort unassigned tasks alphabetically in preview", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		comprehensiveClassifier := new(MockComprehensiveTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()

		// Create use case
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, comprehensiveClassifier, userInput, assetService, nil)

		// Tasks with no asset assignments in non-alphabetical order
		tasks := []*domain.Task{
			{Key: "TEST-C", Summary: "Task C"},
			{Key: "TEST-A", Summary: "Task A"},
			{Key: "TEST-B", Summary: "Task B"},
		}

		results := []*ports.ComprehensiveClassificationResult{
			{
				Task:           tasks[0],
				WorkType:       domain.WorkTypeDevelopment,
				Asset:          nil, // No asset - will be in unassigned list
				WorkTypeReason: "Development pattern",
			},
			{
				Task:           tasks[1],
				WorkType:       domain.WorkTypeMaintenance,
				Asset:          nil, // No asset - will be in unassigned list
				WorkTypeReason: "Maintenance pattern",
			},
			{
				Task:           tasks[2],
				WorkType:       domain.WorkTypeDiscovery,
				Asset:          nil, // No asset - will be in unassigned list
				WorkTypeReason: "Discovery pattern",
			},
		}

		comprehensiveClassifier.On("ClassifyTasksComprehensive", tasks).Return(results, nil)
		userInput.On("Confirm", "Would you like to sync assets from Confluence to potentially improve classification?").Return(false, nil)

		// Act - call previewClassificationsWithRetry directly to test sorting logic
		err := uc.previewClassificationsWithRetry(tasks, false, false)

		// Assert
		assert.NoError(t, err)
		comprehensiveClassifier.AssertExpectations(t)
		userInput.AssertExpectations(t)
	})

	t.Run("should group tasks by work type in preview display", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		comprehensiveClassifier := new(MockComprehensiveTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()

		// Create use case
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, comprehensiveClassifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  true,
			Apply:   false,
		}

		tasks := []*domain.Task{
			{Key: "DEV-2", Summary: "Dev Task 2"},
			{Key: "MAINT-1", Summary: "Maintenance Task"},
			{Key: "DEV-1", Summary: "Dev Task 1"},
			{Key: "DISC-1", Summary: "Discovery Task"},
		}

		results := []*ports.ComprehensiveClassificationResult{
			{
				Task:     tasks[0],
				WorkType: domain.WorkTypeDevelopment,
				Asset: &ports.AssetClassificationResult{
					Asset:      &assetsdomain.Asset{Name: "Dev Asset"},
					Confidence: 0.90,
					Reason:     "keyword match",
				},
				WorkTypeReason: "Development pattern",
			},
			{
				Task:           tasks[1],
				WorkType:       domain.WorkTypeMaintenance,
				Asset:          nil,
				WorkTypeReason: "Maintenance pattern",
			},
			{
				Task:     tasks[2],
				WorkType: domain.WorkTypeDevelopment,
				Asset: &ports.AssetClassificationResult{
					Asset:      &assetsdomain.Asset{Name: "Dev Asset 2"},
					Confidence: 0.85,
					Reason:     "description match",
				},
				WorkTypeReason: "Development pattern",
			},
			{
				Task:           tasks[3],
				WorkType:       domain.WorkTypeDiscovery,
				Asset:          nil,
				WorkTypeReason: "Discovery pattern",
			},
		}

		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return(tasks, nil)
		comprehensiveClassifier.On("ClassifyTasksComprehensive", tasks).Return(results, nil)
		userInput.On("Confirm", "Would you like to sync assets from Confluence to potentially improve classification?").Return(false, nil)

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.NoError(t, err)
		localRepo.AssertExpectations(t)
		comprehensiveClassifier.AssertExpectations(t)
		userInput.AssertExpectations(t)
	})

	t.Run("should fall back to simple display when comprehensive classifier not available", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		simpleClassifier := new(MockTaskClassifier) // Simple classifier, not comprehensive
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()

		// Create use case
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, simpleClassifier, userInput, assetService, nil)

		// Arrange
		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  true,
			Apply:   false,
		}

		// Tasks in non-alphabetical order to test sorting
		tasks := []*domain.Task{
			{Key: "TEST-3", Summary: "Task 3"},
			{Key: "TEST-1", Summary: "Task 1"},
			{Key: "TEST-2", Summary: "Task 2"},
		}

		workTypes := map[string]domain.WorkType{
			"TEST-1": domain.WorkTypeDevelopment,
			"TEST-2": domain.WorkTypeMaintenance,
			"TEST-3": domain.WorkTypeDiscovery,
		}

		localRepo.On("FindByProjectAndSprint", ctx, testProject, testSprint).Return(tasks, nil)
		simpleClassifier.On("ClassifyTasks", tasks).Return(workTypes, nil)

		// Act
		err := uc.Execute(ctx, input)

		// Assert
		assert.NoError(t, err)
		localRepo.AssertExpectations(t)
		simpleClassifier.AssertExpectations(t)
	})

	t.Run("should handle empty work type groups in comprehensive preview", func(t *testing.T) {
		// Create mocks
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		comprehensiveClassifier := new(MockComprehensiveTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()

		// Create use case
		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, comprehensiveClassifier, userInput, assetService, nil)

		// Empty tasks list
		tasks := []*domain.Task{}
		results := []*ports.ComprehensiveClassificationResult{}

		comprehensiveClassifier.On("ClassifyTasksComprehensive", tasks).Return(results, nil)

		// Act
		err := uc.previewClassificationsWithRetry(tasks, false, false)

		// Assert
		assert.NoError(t, err)
		comprehensiveClassifier.AssertExpectations(t)
	})
}

// MockSprintLockRepository is a mock implementation of ports.SprintLockRepository
type MockSprintLockRepository struct {
	mock.Mock
}

func (m *MockSprintLockRepository) FindLock(ctx context.Context, project, sprint string) (*domain.SprintLock, error) {
	args := m.Called(ctx, project, sprint)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.SprintLock), args.Error(1)
}

func (m *MockSprintLockRepository) SaveLock(ctx context.Context, lock *domain.SprintLock) error {
	args := m.Called(ctx, lock)
	return args.Error(0)
}

func TestSprintLockBehavior(t *testing.T) {
	tasks := []*domain.Task{
		{Key: "TEST-1", Summary: "Task 1", Labels: []string{"cap-asset-test"}},
	}

	t.Run("apply blocked when sprint is locked", func(t *testing.T) {
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		classifier := new(MockTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()
		lockRepo := new(MockSprintLockRepository)

		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, classifier, userInput, assetService, lockRepo)

		localRepo.On("FindByProjectAndSprint", mock.Anything, testProject, testSprint).Return(tasks, nil)
		existingLock := domain.NewSprintLock(testProject, testSprint, 10)
		lockRepo.On("FindLock", mock.Anything, testProject, testSprint).Return(existingLock, nil)

		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			Apply:   true,
			Force:   false,
		}

		err := uc.Execute(context.Background(), input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already classified")
		assert.Contains(t, err.Error(), "--force")
		lockRepo.AssertExpectations(t)
	})

	t.Run("apply allowed when sprint is not locked", func(t *testing.T) {
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		classifier := new(MockTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()
		lockRepo := new(MockSprintLockRepository)

		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, classifier, userInput, assetService, lockRepo)

		localRepo.On("FindByProjectAndSprint", mock.Anything, testProject, testSprint).Return([]*domain.Task{}, nil)
		userInput.On("Confirm", mock.Anything, mock.Anything, mock.Anything).Return(false, nil)

		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			Apply:   true,
			Force:   false,
		}

		err := uc.Execute(context.Background(), input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no tasks available")
	})

	t.Run("force apply prompts user when locked and proceeds on confirm", func(t *testing.T) {
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		classifier := new(MockTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()
		lockRepo := new(MockSprintLockRepository)

		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, classifier, userInput, assetService, lockRepo)

		localRepo.On("FindByProjectAndSprint", mock.Anything, testProject, testSprint).Return(tasks, nil)
		existingLock := domain.NewSprintLock(testProject, testSprint, 10)
		lockRepo.On("FindLock", mock.Anything, testProject, testSprint).Return(existingLock, nil)
		userInput.On("Confirm", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
		classifier.On("ClassifyTasks", tasks).Return(map[string]domain.WorkType{"TEST-1": "cap-development"}, nil)
		localRepo.On("Save", mock.Anything, mock.Anything).Return(nil)
		remoteRepo.On("UpdateLabels", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		lockRepo.On("SaveLock", mock.Anything, mock.Anything).Return(nil)

		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			Apply:   true,
			Force:   true,
		}

		err := uc.Execute(context.Background(), input)
		assert.NoError(t, err)
		userInput.AssertCalled(t, "Confirm", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		lockRepo.AssertCalled(t, "SaveLock", mock.Anything, mock.Anything)
	})

	t.Run("force apply aborted when user declines", func(t *testing.T) {
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		classifier := new(MockTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()
		lockRepo := new(MockSprintLockRepository)

		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, classifier, userInput, assetService, lockRepo)

		localRepo.On("FindByProjectAndSprint", mock.Anything, testProject, testSprint).Return(tasks, nil)
		existingLock := domain.NewSprintLock(testProject, testSprint, 10)
		lockRepo.On("FindLock", mock.Anything, testProject, testSprint).Return(existingLock, nil)
		userInput.On("Confirm", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)

		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			Apply:   true,
			Force:   true,
		}

		err := uc.Execute(context.Background(), input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "aborted by user")
		lockRepo.AssertExpectations(t)
	})

	t.Run("dry-run always allowed even when locked", func(t *testing.T) {
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		classifier := new(MockTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()
		lockRepo := new(MockSprintLockRepository)

		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, classifier, userInput, assetService, lockRepo)

		localRepo.On("FindByProjectAndSprint", mock.Anything, testProject, testSprint).Return([]*domain.Task{}, nil)
		userInput.On("Confirm", mock.Anything, mock.Anything, mock.Anything).Return(false, nil)

		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			DryRun:  true,
			Apply:   false,
		}

		err := uc.Execute(context.Background(), input)
		assert.Error(t, err)
		lockRepo.AssertNotCalled(t, "FindLock")
	})

	t.Run("lock check error propagated", func(t *testing.T) {
		localRepo := new(MockTaskRepository)
		remoteRepo := new(MockTaskRepository)
		classifier := new(MockTaskClassifier)
		userInput := new(MockUserInput)
		assetService := testutil.NewMockAssetService()
		lockRepo := new(MockSprintLockRepository)

		uc := NewClassifyTasksUseCase(localRepo, remoteRepo, classifier, userInput, assetService, lockRepo)

		localRepo.On("FindByProjectAndSprint", mock.Anything, testProject, testSprint).Return(tasks, nil)
		lockRepo.On("FindLock", mock.Anything, testProject, testSprint).Return(nil, fmt.Errorf("storage error"))

		input := domain.ClassifyTasksInput{
			Project: testProject,
			Sprint:  testSprint,
			Apply:   true,
		}

		err := uc.Execute(context.Background(), input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check sprint lock")
		lockRepo.AssertExpectations(t)
	})
}
