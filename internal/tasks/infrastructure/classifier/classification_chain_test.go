package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// Mock classifiers for testing
type MockAssetClassifier struct {
	mock.Mock
}

func (m *MockAssetClassifier) ClassifyTaskAsset(task *taskdomain.Task) (*ports.AssetClassificationResult, error) {
	args := m.Called(task)
	return args.Get(0).(*ports.AssetClassificationResult), args.Error(1)
}

func (m *MockAssetClassifier) ClassifyTasksAssets(tasks []*taskdomain.Task) ([]*ports.AssetClassificationResult, error) {
	args := m.Called(tasks)
	return args.Get(0).([]*ports.AssetClassificationResult), args.Error(1)
}

type MockWorkTypeClassifier struct {
	mock.Mock
}

func (m *MockWorkTypeClassifier) ClassifyTask(task *taskdomain.Task) (taskdomain.WorkType, error) {
	args := m.Called(task)
	return args.Get(0).(taskdomain.WorkType), args.Error(1)
}

func (m *MockWorkTypeClassifier) ClassifyTasks(tasks []*taskdomain.Task) (map[string]taskdomain.WorkType, error) {
	args := m.Called(tasks)
	return args.Get(0).(map[string]taskdomain.WorkType), args.Error(1)
}

func TestComprehensiveClassificationChain_ClassifyTask(t *testing.T) {
	tests := []struct {
		name             string
		task             *taskdomain.Task
		assetResult      *ports.AssetClassificationResult
		assetError       error
		workType         taskdomain.WorkType
		workTypeError    error
		expectedAsset    *assetdomain.Asset
		expectedWorkType taskdomain.WorkType
		expectError      bool
	}{
		{
			name: "successful comprehensive classification",
			task: &taskdomain.Task{
				Key:         "TEST-1",
				Summary:     "Fix Payment Gateway timeout",
				Description: "Resolve gateway timeout issues",
				Epic:        "Payment Processing",
			},
			assetResult: &ports.AssetClassificationResult{
				Task: &taskdomain.Task{Key: "TEST-1"},
				Asset: &assetdomain.Asset{
					Name:        "Payment Gateway",
					Description: "Processes payments",
					Keywords:    []string{"payment", "gateway"},
				},
				Confidence: 0.9,
				Reason:     "asset name match in task summary",
			},
			assetError:       nil,
			workType:         taskdomain.WorkTypeMaintenance,
			workTypeError:    nil,
			expectedWorkType: taskdomain.WorkTypeMaintenance,
			expectError:      false,
		},
		{
			name: "asset classification without matching asset",
			task: &taskdomain.Task{
				Key:         "TEST-2",
				Summary:     "Update documentation",
				Description: "Update project documentation",
				Epic:        "Documentation",
			},
			assetResult: &ports.AssetClassificationResult{
				Task:       &taskdomain.Task{Key: "TEST-2"},
				Asset:      nil,
				Confidence: 0.1,
				Reason:     "no matching asset found",
			},
			assetError:       nil,
			workType:         taskdomain.WorkTypeDevelopment,
			workTypeError:    nil,
			expectedWorkType: taskdomain.WorkTypeDevelopment,
			expectError:      false,
		},
		{
			name: "discovery work type classification",
			task: &taskdomain.Task{
				Key:         "TEST-3",
				Summary:     "Research new payment methods",
				Description: "Investigate blockchain payment solutions",
				Epic:        "Payment Innovation",
				Labels:      []string{"research", "spike"},
			},
			assetResult: &ports.AssetClassificationResult{
				Task: &taskdomain.Task{Key: "TEST-3"},
				Asset: &assetdomain.Asset{
					Name:        "Payment Gateway",
					Description: "Processes payments",
					Keywords:    []string{"payment", "gateway"},
				},
				Confidence: 0.7,
				Reason:     "keyword match in task content",
			},
			assetError:       nil,
			workType:         taskdomain.WorkTypeDiscovery,
			workTypeError:    nil,
			expectedWorkType: taskdomain.WorkTypeDiscovery,
			expectError:      false,
		},
		{
			name: "asset classifier error",
			task: &taskdomain.Task{
				Key:         "TEST-4",
				Summary:     "Test task",
				Description: "Test description",
			},
			assetResult:      nil,
			assetError:       assert.AnError,
			workType:         taskdomain.WorkTypeDevelopment,
			workTypeError:    nil,
			expectedWorkType: "",
			expectError:      true,
		},
		{
			name: "work type classifier error",
			task: &taskdomain.Task{
				Key:         "TEST-5",
				Summary:     "Test task",
				Description: "Test description",
			},
			assetResult: &ports.AssetClassificationResult{
				Task:       &taskdomain.Task{Key: "TEST-5"},
				Asset:      nil,
				Confidence: 0.1,
				Reason:     "no matching asset found",
			},
			assetError:       nil,
			workType:         "",
			workTypeError:    assert.AnError,
			expectedWorkType: "",
			expectError:      true,
		},
		{
			name: "both classifiers error",
			task: &taskdomain.Task{
				Key:         "TEST-6",
				Summary:     "Test task",
				Description: "Test description",
			},
			assetResult:      nil,
			assetError:       assert.AnError,
			workType:         "",
			workTypeError:    assert.AnError,
			expectedWorkType: "",
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockAssetClassifier := new(MockAssetClassifier)
			mockWorkTypeClassifier := new(MockWorkTypeClassifier)

			mockAssetClassifier.On("ClassifyTaskAsset", tt.task).Return(tt.assetResult, tt.assetError)
			if tt.assetError == nil {
				mockWorkTypeClassifier.On("ClassifyTask", tt.task).Return(tt.workType, tt.workTypeError)
			}

			// Create chain
			chain := NewComprehensiveClassificationChain(mockAssetClassifier, mockWorkTypeClassifier)

			// Execute classification
			result, err := chain.ClassifyTask(tt.task)

			// Verify error expectation
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.task, result.Task)
				assert.Equal(t, tt.expectedWorkType, result.WorkType)
				assert.NotNil(t, result.Asset)

				if tt.assetResult.Asset != nil {
					assert.Equal(t, tt.assetResult.Asset.Name, result.Asset.Asset.Name)
				} else {
					assert.Nil(t, result.Asset.Asset)
				}
			}

			mockAssetClassifier.AssertExpectations(t)
			mockWorkTypeClassifier.AssertExpectations(t)
		})
	}
}

func TestComprehensiveClassificationChain_ClassifyTasks(t *testing.T) {
	// Setup test data
	tasks := []*taskdomain.Task{
		{
			Key:         "TEST-1",
			Summary:     "Fix Payment Gateway bug",
			Description: "Fix timeout issue",
			Epic:        "Payment Processing",
		},
		{
			Key:         "TEST-2",
			Summary:     "Research new features",
			Description: "Investigate new payment methods",
			Epic:        "Innovation",
		},
	}

	assetResults := []*ports.AssetClassificationResult{
		{
			Task: tasks[0],
			Asset: &assetdomain.Asset{
				Name:        "Payment Gateway",
				Description: "Processes payments",
				Keywords:    []string{"payment", "gateway"},
			},
			Confidence: 0.9,
			Reason:     "asset name match in task summary",
		},
		{
			Task:       tasks[1],
			Asset:      nil,
			Confidence: 0.1,
			Reason:     "no matching asset found",
		},
	}

	workTypeResults := map[string]taskdomain.WorkType{
		"TEST-1": taskdomain.WorkTypeMaintenance,
		"TEST-2": taskdomain.WorkTypeDiscovery,
	}

	// Setup mocks
	mockAssetClassifier := new(MockAssetClassifier)
	mockWorkTypeClassifier := new(MockWorkTypeClassifier)

	mockAssetClassifier.On("ClassifyTasksAssets", tasks).Return(assetResults, nil)
	mockWorkTypeClassifier.On("ClassifyTasks", tasks).Return(workTypeResults, nil)

	// Create chain
	chain := NewComprehensiveClassificationChain(mockAssetClassifier, mockWorkTypeClassifier)

	// Execute batch classification
	results, err := chain.ClassifyTasks(tasks)

	// Verify results
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	// Verify first task
	assert.Equal(t, "TEST-1", results[0].Task.Key)
	assert.NotNil(t, results[0].Asset.Asset)
	assert.Equal(t, "Payment Gateway", results[0].Asset.Asset.Name)
	assert.Equal(t, taskdomain.WorkTypeMaintenance, results[0].WorkType)

	// Verify second task
	assert.Equal(t, "TEST-2", results[1].Task.Key)
	assert.Nil(t, results[1].Asset.Asset)
	assert.Equal(t, taskdomain.WorkTypeDiscovery, results[1].WorkType)

	mockAssetClassifier.AssertExpectations(t)
	mockWorkTypeClassifier.AssertExpectations(t)
}

func TestComprehensiveClassificationChain_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		task        *taskdomain.Task
		tasks       []*taskdomain.Task
		expectError bool
	}{
		{
			name:        "nil task",
			task:        nil,
			expectError: true,
		},
		{
			name:        "empty task list",
			tasks:       []*taskdomain.Task{},
			expectError: false,
		},
		{
			name: "task with minimal fields",
			task: &taskdomain.Task{
				Key:     "TEST-1",
				Summary: "Minimal task",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAssetClassifier := new(MockAssetClassifier)
			mockWorkTypeClassifier := new(MockWorkTypeClassifier)

			chain := NewComprehensiveClassificationChain(mockAssetClassifier, mockWorkTypeClassifier)

			if tt.task != nil {
				// Setup mocks for single task
				assetResult := &ports.AssetClassificationResult{
					Task:       tt.task,
					Asset:      nil,
					Confidence: 0.1,
					Reason:     "no matching asset found",
				}
				mockAssetClassifier.On("ClassifyTaskAsset", tt.task).Return(assetResult, nil)
				mockWorkTypeClassifier.On("ClassifyTask", tt.task).Return(taskdomain.WorkTypeDevelopment, nil)

				result, err := chain.ClassifyTask(tt.task)
				if tt.expectError {
					assert.Error(t, err)
					assert.Nil(t, result)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, result)
				}
			} else if tt.tasks != nil {
				// Setup mocks for batch processing only if there are tasks to process
				if len(tt.tasks) > 0 {
					mockAssetClassifier.On("ClassifyTasksAssets", tt.tasks).Return([]*ports.AssetClassificationResult{}, nil)
					mockWorkTypeClassifier.On("ClassifyTasks", tt.tasks).Return(map[string]taskdomain.WorkType{}, nil)
				}

				results, err := chain.ClassifyTasks(tt.tasks)
				if tt.expectError {
					assert.Error(t, err)
					assert.Nil(t, results)
				} else {
					assert.NoError(t, err)
					assert.NotNil(t, results)
					assert.Len(t, results, len(tt.tasks))
				}
			} else {
				// Test nil task
				result, err := chain.ClassifyTask(nil)
				assert.Error(t, err)
				assert.Nil(t, result)
			}

			mockAssetClassifier.AssertExpectations(t)
			mockWorkTypeClassifier.AssertExpectations(t)
		})
	}
}
