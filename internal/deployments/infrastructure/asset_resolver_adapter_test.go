package infrastructure

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	assetsDomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/deployments/domain/ports"
	tasksDomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	tasksPorts "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// Mock implementations
type MockTaskRepository struct {
	mock.Mock
}

func (m *MockTaskRepository) FindByKey(ctx context.Context, key string) (*tasksDomain.Task, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*tasksDomain.Task), args.Error(1)
}

func (m *MockTaskRepository) FindByProjectAndSprint(ctx context.Context, project, sprint string) ([]*tasksDomain.Task, error) {
	args := m.Called(ctx, project, sprint)
	return args.Get(0).([]*tasksDomain.Task), args.Error(1)
}

func (m *MockTaskRepository) FindByProject(ctx context.Context, projectKey string) ([]*tasksDomain.Task, error) {
	args := m.Called(ctx, projectKey)
	return args.Get(0).([]*tasksDomain.Task), args.Error(1)
}

func (m *MockTaskRepository) FindBySprint(ctx context.Context, sprintName string) ([]*tasksDomain.Task, error) {
	args := m.Called(ctx, sprintName)
	return args.Get(0).([]*tasksDomain.Task), args.Error(1)
}

func (m *MockTaskRepository) FindByPlatform(ctx context.Context, platform string) ([]*tasksDomain.Task, error) {
	args := m.Called(ctx, platform)
	return args.Get(0).([]*tasksDomain.Task), args.Error(1)
}

func (m *MockTaskRepository) FindAll(ctx context.Context) ([]*tasksDomain.Task, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*tasksDomain.Task), args.Error(1)
}

func (m *MockTaskRepository) Save(ctx context.Context, task *tasksDomain.Task) error {
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

func (m *MockTaskRepository) UpdateLabels(ctx context.Context, taskKey string, addLabels, removeLabels []string) error {
	args := m.Called(ctx, taskKey, addLabels, removeLabels)
	return args.Error(0)
}

type MockTaskClassifier struct {
	mock.Mock
}

func (m *MockTaskClassifier) ClassifyTask(task *tasksDomain.Task) (tasksDomain.WorkType, error) {
	args := m.Called(task)
	return args.Get(0).(tasksDomain.WorkType), args.Error(1)
}

func (m *MockTaskClassifier) ClassifyTasks(tasks []*tasksDomain.Task) (map[string]tasksDomain.WorkType, error) {
	args := m.Called(tasks)
	return args.Get(0).(map[string]tasksDomain.WorkType), args.Error(1)
}

func (m *MockTaskClassifier) ClassifyTasksComprehensive(tasks []*tasksDomain.Task) ([]*tasksPorts.ComprehensiveClassificationResult, error) {
	args := m.Called(tasks)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*tasksPorts.ComprehensiveClassificationResult), args.Error(1)
}

func TestNewAssetResolverAdapter(t *testing.T) {
	mockRepo := &MockTaskRepository{}
	mockClassifier := &MockTaskClassifier{}

	adapter := NewAssetResolverAdapter(mockRepo, mockClassifier)
	assert.NotNil(t, adapter)

	// Check that it implements the interface
	var _ ports.AssetResolver = adapter
}

func TestAssetResolverAdapter_ResolveAssetsForTasks(t *testing.T) {
	ctx := context.Background()

	t.Run("resolve with empty task keys", func(t *testing.T) {
		mockRepo := &MockTaskRepository{}
		mockClassifier := &MockTaskClassifier{}
		adapter := NewAssetResolverAdapter(mockRepo, mockClassifier)

		result, err := adapter.ResolveAssetsForTasks(ctx, []string{})
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("resolve with comprehensive classifier", func(t *testing.T) {
		mockRepo := &MockTaskRepository{}
		mockClassifier := &MockTaskClassifier{}
		adapter := NewAssetResolverAdapter(mockRepo, mockClassifier)

		// Setup tasks
		task1 := &tasksDomain.Task{
			Key:         "TASK-1",
			Summary:     "Test task 1",
			Description: "Description for task 1",
			Labels:      []string{"old-label"},
		}

		task2 := &tasksDomain.Task{
			Key:         "TASK-2",
			Summary:     "Test task 2",
			Description: "Description for task 2",
			Labels:      []string{"old-label"},
		}

		// Mock task repository calls
		mockRepo.On("FindByKey", ctx, "TASK-1").Return(task1, nil)
		mockRepo.On("FindByKey", ctx, "TASK-2").Return(task2, nil)

		// Setup classification results
		classificationResults := []*tasksPorts.ComprehensiveClassificationResult{
			{
				Task: task1,
				Asset: &tasksPorts.AssetClassificationResult{
					Asset: &assetsDomain.Asset{
						Name:        "Payment Processing",
						Description: "Handles payment processing",
					},
					Confidence: 0.9,
				},
			},
			{
				Task: task2,
				Asset: &tasksPorts.AssetClassificationResult{
					Asset: &assetsDomain.Asset{
						Name:        "User Management",
						Description: "Manages user accounts",
					},
					Confidence: 0.8,
				},
			},
		}

		mockClassifier.On("ClassifyTasksComprehensive", []*tasksDomain.Task{task1, task2}).Return(classificationResults, nil)

		result, err := adapter.ResolveAssetsForTasks(ctx, []string{"TASK-1", "TASK-2"})
		require.NoError(t, err)
		assert.Len(t, result, 2)

		// Check asset info
		assetNames := make([]string, len(result))
		for i, asset := range result {
			assetNames[i] = asset.Name
			assert.Equal(t, 1, asset.TaskCount)
			assert.Len(t, asset.TaskKeys, 1)
		}

		assert.Contains(t, assetNames, "Payment Processing")
		assert.Contains(t, assetNames, "User Management")

		mockRepo.AssertExpectations(t)
		mockClassifier.AssertExpectations(t)
	})

	t.Run("resolve with same asset for multiple tasks", func(t *testing.T) {
		mockRepo := &MockTaskRepository{}
		mockClassifier := &MockTaskClassifier{}
		adapter := NewAssetResolverAdapter(mockRepo, mockClassifier)

		task1 := &tasksDomain.Task{
			Key:     "TASK-1",
			Summary: "Test task 1",
			Labels:  []string{},
		}

		task2 := &tasksDomain.Task{
			Key:     "TASK-2",
			Summary: "Test task 2",
			Labels:  []string{},
		}

		mockRepo.On("FindByKey", ctx, "TASK-1").Return(task1, nil)
		mockRepo.On("FindByKey", ctx, "TASK-2").Return(task2, nil)

		// Both tasks classified to same asset
		classificationResults := []*tasksPorts.ComprehensiveClassificationResult{
			{
				Task: task1,
				Asset: &tasksPorts.AssetClassificationResult{
					Asset: &assetsDomain.Asset{
						Name: "Payment Processing",
					},
					Confidence: 0.9,
				},
			},
			{
				Task: task2,
				Asset: &tasksPorts.AssetClassificationResult{
					Asset: &assetsDomain.Asset{
						Name: "Payment Processing",
					},
					Confidence: 0.8,
				},
			},
		}

		mockClassifier.On("ClassifyTasksComprehensive", []*tasksDomain.Task{task1, task2}).Return(classificationResults, nil)

		result, err := adapter.ResolveAssetsForTasks(ctx, []string{"TASK-1", "TASK-2"})
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Payment Processing", result[0].Name)
		assert.Equal(t, 2, result[0].TaskCount)
		assert.Len(t, result[0].TaskKeys, 2)
		assert.Contains(t, result[0].TaskKeys, "TASK-1")
		assert.Contains(t, result[0].TaskKeys, "TASK-2")

		mockRepo.AssertExpectations(t)
		mockClassifier.AssertExpectations(t)
	})

	t.Run("resolve with fallback to labels", func(t *testing.T) {
		mockRepo := &MockTaskRepository{}
		// Use basic classifier that doesn't implement ComprehensiveTaskClassifier
		basicClassifier := &struct {
			tasksPorts.TaskClassifier
		}{}
		adapter := NewAssetResolverAdapter(mockRepo, basicClassifier)

		task1 := &tasksDomain.Task{
			Key:     "TASK-1",
			Summary: "Test task 1",
			Labels:  []string{"asset-1", "asset-2"},
		}

		task2 := &tasksDomain.Task{
			Key:     "TASK-2",
			Summary: "Test task 2",
			Labels:  []string{"asset-1", "asset-3"},
		}

		mockRepo.On("FindByKey", ctx, "TASK-1").Return(task1, nil)
		mockRepo.On("FindByKey", ctx, "TASK-2").Return(task2, nil)

		result, err := adapter.ResolveAssetsForTasks(ctx, []string{"TASK-1", "TASK-2"})
		require.NoError(t, err)
		assert.Len(t, result, 3)

		// Check assets are aggregated correctly
		assetMap := make(map[string]ports.AssetInfo)
		for _, asset := range result {
			assetMap[asset.Name] = asset
		}

		assert.Contains(t, assetMap, "asset-1")
		assert.Contains(t, assetMap, "asset-2")
		assert.Contains(t, assetMap, "asset-3")

		// asset-1 should have 2 tasks
		assert.Equal(t, 2, assetMap["asset-1"].TaskCount)
		assert.Len(t, assetMap["asset-1"].TaskKeys, 2)

		// asset-2 and asset-3 should have 1 task each
		assert.Equal(t, 1, assetMap["asset-2"].TaskCount)
		assert.Equal(t, 1, assetMap["asset-3"].TaskCount)

		mockRepo.AssertExpectations(t)
	})

	t.Run("resolve with task not found", func(t *testing.T) {
		mockRepo := &MockTaskRepository{}
		mockClassifier := &MockTaskClassifier{}
		adapter := NewAssetResolverAdapter(mockRepo, mockClassifier)

		task1 := &tasksDomain.Task{
			Key:     "TASK-1",
			Summary: "Test task 1",
			Labels:  []string{"asset-1"},
		}

		// First task found, second not found
		mockRepo.On("FindByKey", ctx, "TASK-1").Return(task1, nil)
		mockRepo.On("FindByKey", ctx, "TASK-2").Return(nil, errors.New("task not found"))

		classificationResults := []*tasksPorts.ComprehensiveClassificationResult{
			{
				Task: task1,
				Asset: &tasksPorts.AssetClassificationResult{
					Asset: &assetsDomain.Asset{
						Name: "Payment Processing",
					},
					Confidence: 0.9,
				},
			},
		}

		mockClassifier.On("ClassifyTasksComprehensive", []*tasksDomain.Task{task1}).Return(classificationResults, nil)

		result, err := adapter.ResolveAssetsForTasks(ctx, []string{"TASK-1", "TASK-2"})
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Payment Processing", result[0].Name)

		mockRepo.AssertExpectations(t)
		mockClassifier.AssertExpectations(t)
	})

	t.Run("resolve with classification error", func(t *testing.T) {
		mockRepo := &MockTaskRepository{}
		mockClassifier := &MockTaskClassifier{}
		adapter := NewAssetResolverAdapter(mockRepo, mockClassifier)

		task1 := &tasksDomain.Task{
			Key:     "TASK-1",
			Summary: "Test task 1",
			Labels:  []string{},
		}

		mockRepo.On("FindByKey", ctx, "TASK-1").Return(task1, nil)
		mockClassifier.On("ClassifyTasksComprehensive", []*tasksDomain.Task{task1}).Return(nil, errors.New("classification error"))

		_, err := adapter.ResolveAssetsForTasks(ctx, []string{"TASK-1"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to classify tasks")

		mockRepo.AssertExpectations(t)
		mockClassifier.AssertExpectations(t)
	})

	t.Run("resolve with no classification results", func(t *testing.T) {
		mockRepo := &MockTaskRepository{}
		mockClassifier := &MockTaskClassifier{}
		adapter := NewAssetResolverAdapter(mockRepo, mockClassifier)

		task1 := &tasksDomain.Task{
			Key:     "TASK-1",
			Summary: "Test task 1",
			Labels:  []string{},
		}

		mockRepo.On("FindByKey", ctx, "TASK-1").Return(task1, nil)

		// Classification results with no asset
		classificationResults := []*tasksPorts.ComprehensiveClassificationResult{
			{
				Task:  task1,
				Asset: nil,
			},
		}

		mockClassifier.On("ClassifyTasksComprehensive", []*tasksDomain.Task{task1}).Return(classificationResults, nil)

		result, err := adapter.ResolveAssetsForTasks(ctx, []string{"TASK-1"})
		require.NoError(t, err)
		assert.Empty(t, result)

		mockRepo.AssertExpectations(t)
		mockClassifier.AssertExpectations(t)
	})
}

func TestAssetResolverAdapter_ResolveAssetsForTask(t *testing.T) {
	ctx := context.Background()

	t.Run("resolve with empty task key", func(t *testing.T) {
		mockRepo := &MockTaskRepository{}
		mockClassifier := &MockTaskClassifier{}
		adapter := NewAssetResolverAdapter(mockRepo, mockClassifier)

		result, err := adapter.ResolveAssetsForTask(ctx, "")
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("resolve with comprehensive classifier", func(t *testing.T) {
		mockRepo := &MockTaskRepository{}
		mockClassifier := &MockTaskClassifier{}
		adapter := NewAssetResolverAdapter(mockRepo, mockClassifier)

		task := &tasksDomain.Task{
			Key:     "TASK-1",
			Summary: "Test task",
			Labels:  []string{"old-label"},
		}

		mockRepo.On("FindByKey", ctx, "TASK-1").Return(task, nil)

		classificationResults := []*tasksPorts.ComprehensiveClassificationResult{
			{
				Task: task,
				Asset: &tasksPorts.AssetClassificationResult{
					Asset: &assetsDomain.Asset{
						Name: "Payment Processing",
					},
					Confidence: 0.9,
				},
			},
		}

		mockClassifier.On("ClassifyTasksComprehensive", []*tasksDomain.Task{task}).Return(classificationResults, nil)

		result, err := adapter.ResolveAssetsForTask(ctx, "TASK-1")
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "Payment Processing", result[0])

		mockRepo.AssertExpectations(t)
		mockClassifier.AssertExpectations(t)
	})

	t.Run("resolve with fallback to labels", func(t *testing.T) {
		mockRepo := &MockTaskRepository{}
		// Use basic classifier
		basicClassifier := &struct {
			tasksPorts.TaskClassifier
		}{}
		adapter := NewAssetResolverAdapter(mockRepo, basicClassifier)

		task := &tasksDomain.Task{
			Key:     "TASK-1",
			Summary: "Test task",
			Labels:  []string{"asset-1", "asset-2"},
		}

		mockRepo.On("FindByKey", ctx, "TASK-1").Return(task, nil)

		result, err := adapter.ResolveAssetsForTask(ctx, "TASK-1")
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Contains(t, result, "asset-1")
		assert.Contains(t, result, "asset-2")

		mockRepo.AssertExpectations(t)
	})

	t.Run("resolve with task not found", func(t *testing.T) {
		mockRepo := &MockTaskRepository{}
		mockClassifier := &MockTaskClassifier{}
		adapter := NewAssetResolverAdapter(mockRepo, mockClassifier)

		mockRepo.On("FindByKey", ctx, "TASK-1").Return(nil, errors.New("task not found"))

		_, err := adapter.ResolveAssetsForTask(ctx, "TASK-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find task")

		mockRepo.AssertExpectations(t)
	})

	t.Run("resolve with classification error", func(t *testing.T) {
		mockRepo := &MockTaskRepository{}
		mockClassifier := &MockTaskClassifier{}
		adapter := NewAssetResolverAdapter(mockRepo, mockClassifier)

		task := &tasksDomain.Task{
			Key:     "TASK-1",
			Summary: "Test task",
			Labels:  []string{},
		}

		mockRepo.On("FindByKey", ctx, "TASK-1").Return(task, nil)
		mockClassifier.On("ClassifyTasksComprehensive", []*tasksDomain.Task{task}).Return(nil, errors.New("classification error"))

		_, err := adapter.ResolveAssetsForTask(ctx, "TASK-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to classify task")

		mockRepo.AssertExpectations(t)
		mockClassifier.AssertExpectations(t)
	})

	t.Run("resolve with no classification results", func(t *testing.T) {
		mockRepo := &MockTaskRepository{}
		mockClassifier := &MockTaskClassifier{}
		adapter := NewAssetResolverAdapter(mockRepo, mockClassifier)

		task := &tasksDomain.Task{
			Key:     "TASK-1",
			Summary: "Test task",
			Labels:  []string{"fallback-asset"},
		}

		mockRepo.On("FindByKey", ctx, "TASK-1").Return(task, nil)

		// No classification results
		classificationResults := []*tasksPorts.ComprehensiveClassificationResult{}
		mockClassifier.On("ClassifyTasksComprehensive", []*tasksDomain.Task{task}).Return(classificationResults, nil)

		result, err := adapter.ResolveAssetsForTask(ctx, "TASK-1")
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, "fallback-asset", result[0])

		mockRepo.AssertExpectations(t)
		mockClassifier.AssertExpectations(t)
	})
}
