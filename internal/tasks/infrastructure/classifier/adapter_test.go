package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// MockClassificationChain is a mock implementation of ClassificationChain
type MockClassificationChain struct {
	mock.Mock
}

func (m *MockClassificationChain) ClassifyTask(task *taskdomain.Task) (*ports.ComprehensiveClassificationResult, error) {
	args := m.Called(task)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ports.ComprehensiveClassificationResult), args.Error(1)
}

func (m *MockClassificationChain) ClassifyTasks(tasks []*taskdomain.Task) ([]*ports.ComprehensiveClassificationResult, error) {
	args := m.Called(tasks)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ports.ComprehensiveClassificationResult), args.Error(1)
}

func TestComprehensiveClassifierAdapter_ClassifyTask(t *testing.T) {
	tests := []struct {
		name                string
		task                *taskdomain.Task
		comprehensiveResult *ports.ComprehensiveClassificationResult
		chainError          error
		expectedWorkType    taskdomain.WorkType
		expectError         bool
	}{
		{
			name: "successful classification with asset assignment",
			task: &taskdomain.Task{
				Key:         "TEST-1",
				Summary:     "Fix Payment Gateway bug",
				Description: "Resolve timeout issue",
			},
			comprehensiveResult: &ports.ComprehensiveClassificationResult{
				Task: &taskdomain.Task{Key: "TEST-1"},
				Asset: &ports.AssetClassificationResult{
					Asset: &assetdomain.Asset{
						Name:        "Payment Gateway",
						Description: "Processes payments",
					},
					Confidence: 0.9,
					Reason:     "asset name match in summary",
				},
				WorkType:       taskdomain.WorkTypeMaintenance,
				WorkTypeReason: "bug fix for Payment Gateway",
			},
			chainError:       nil,
			expectedWorkType: taskdomain.WorkTypeMaintenance,
			expectError:      false,
		},
		{
			name: "successful classification without asset assignment",
			task: &taskdomain.Task{
				Key:         "TEST-2",
				Summary:     "Research new features",
				Description: "Investigate new capabilities",
				Labels:      []string{"research", "spike"},
			},
			comprehensiveResult: &ports.ComprehensiveClassificationResult{
				Task: &taskdomain.Task{Key: "TEST-2"},
				Asset: &ports.AssetClassificationResult{
					Asset:      nil,
					Confidence: 0.1,
					Reason:     "no matching asset found",
				},
				WorkType:       taskdomain.WorkTypeDiscovery,
				WorkTypeReason: "spike/research task detected",
			},
			chainError:       nil,
			expectedWorkType: taskdomain.WorkTypeDiscovery,
			expectError:      false,
		},
		{
			name: "chain classification error",
			task: &taskdomain.Task{
				Key:     "TEST-3",
				Summary: "Test task",
			},
			comprehensiveResult: nil,
			chainError:          assert.AnError,
			expectedWorkType:    "",
			expectError:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockChain := new(MockClassificationChain)
			mockChain.On("ClassifyTask", tt.task).Return(tt.comprehensiveResult, tt.chainError)

			// Create adapter
			adapter := NewComprehensiveClassifierAdapter(mockChain)

			// Execute
			workType, err := adapter.ClassifyTask(tt.task)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, taskdomain.WorkType(""), workType)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedWorkType, workType)
			}

			mockChain.AssertExpectations(t)
		})
	}
}

func TestComprehensiveClassifierAdapter_ClassifyTasks(t *testing.T) {
	tasks := []*taskdomain.Task{
		{
			Key:         "TEST-1",
			Summary:     "Fix Payment Gateway bug",
			Description: "Resolve timeout issue",
		},
		{
			Key:         "TEST-2",
			Summary:     "Research new features",
			Description: "Investigate capabilities",
			Labels:      []string{"research"},
		},
	}

	comprehensiveResults := []*ports.ComprehensiveClassificationResult{
		{
			Task: tasks[0],
			Asset: &ports.AssetClassificationResult{
				Asset: &assetdomain.Asset{
					Name:        "Payment Gateway",
					Description: "Processes payments",
				},
				Confidence: 0.9,
				Reason:     "asset name match in summary",
			},
			WorkType:       taskdomain.WorkTypeMaintenance,
			WorkTypeReason: "bug fix for Payment Gateway",
		},
		{
			Task: tasks[1],
			Asset: &ports.AssetClassificationResult{
				Asset:      nil,
				Confidence: 0.1,
				Reason:     "no matching asset found",
			},
			WorkType:       taskdomain.WorkTypeDiscovery,
			WorkTypeReason: "spike/research task detected",
		},
	}

	tests := []struct {
		name                 string
		tasks                []*taskdomain.Task
		comprehensiveResults []*ports.ComprehensiveClassificationResult
		chainError           error
		expectedWorkTypes    map[string]taskdomain.WorkType
		expectError          bool
	}{
		{
			name:                 "successful batch classification",
			tasks:                tasks,
			comprehensiveResults: comprehensiveResults,
			chainError:           nil,
			expectedWorkTypes: map[string]taskdomain.WorkType{
				"TEST-1": taskdomain.WorkTypeMaintenance,
				"TEST-2": taskdomain.WorkTypeDiscovery,
			},
			expectError: false,
		},
		{
			name:                 "chain classification error",
			tasks:                tasks,
			comprehensiveResults: nil,
			chainError:           assert.AnError,
			expectedWorkTypes:    nil,
			expectError:          true,
		},
		{
			name:                 "empty task list",
			tasks:                []*taskdomain.Task{},
			comprehensiveResults: []*ports.ComprehensiveClassificationResult{},
			chainError:           nil,
			expectedWorkTypes:    map[string]taskdomain.WorkType{},
			expectError:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockChain := new(MockClassificationChain)
			mockChain.On("ClassifyTasks", tt.tasks).Return(tt.comprehensiveResults, tt.chainError)

			// Create adapter
			adapter := NewComprehensiveClassifierAdapter(mockChain)

			// Execute
			workTypes, err := adapter.ClassifyTasks(tt.tasks)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, workTypes)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedWorkTypes, workTypes)
			}

			mockChain.AssertExpectations(t)
		})
	}
}

func TestComprehensiveClassifierAdapter_ClassifyTasksComprehensive(t *testing.T) {
	tasks := []*taskdomain.Task{
		{
			Key:         "TEST-1",
			Summary:     "Fix Payment Gateway bug",
			Description: "Resolve timeout issue",
		},
		{
			Key:         "TEST-2",
			Summary:     "Research new features",
			Description: "Investigate capabilities",
			Labels:      []string{"research"},
		},
	}

	comprehensiveResults := []*ports.ComprehensiveClassificationResult{
		{
			Task: tasks[0],
			Asset: &ports.AssetClassificationResult{
				Asset: &assetdomain.Asset{
					Name:        "Payment Gateway",
					Description: "Processes payments",
				},
				Confidence: 0.9,
				Reason:     "asset name match in summary",
			},
			WorkType:       taskdomain.WorkTypeMaintenance,
			WorkTypeReason: "bug fix for Payment Gateway",
		},
		{
			Task: tasks[1],
			Asset: &ports.AssetClassificationResult{
				Asset:      nil,
				Confidence: 0.1,
				Reason:     "no matching asset found",
			},
			WorkType:       taskdomain.WorkTypeDiscovery,
			WorkTypeReason: "spike/research task detected",
		},
	}

	tests := []struct {
		name                 string
		tasks                []*taskdomain.Task
		comprehensiveResults []*ports.ComprehensiveClassificationResult
		chainError           error
		expectError          bool
	}{
		{
			name:                 "successful comprehensive classification",
			tasks:                tasks,
			comprehensiveResults: comprehensiveResults,
			chainError:           nil,
			expectError:          false,
		},
		{
			name:                 "chain classification error",
			tasks:                tasks,
			comprehensiveResults: nil,
			chainError:           assert.AnError,
			expectError:          true,
		},
		{
			name:                 "empty task list",
			tasks:                []*taskdomain.Task{},
			comprehensiveResults: []*ports.ComprehensiveClassificationResult{},
			chainError:           nil,
			expectError:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock
			mockChain := new(MockClassificationChain)
			mockChain.On("ClassifyTasks", tt.tasks).Return(tt.comprehensiveResults, tt.chainError)

			// Create adapter
			adapter := NewComprehensiveClassifierAdapter(mockChain)

			// Execute
			results, err := adapter.ClassifyTasksComprehensive(tt.tasks)

			// Verify
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, results)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.comprehensiveResults, results)
			}

			mockChain.AssertExpectations(t)
		})
	}
}
