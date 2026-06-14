package classifier

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// MockAssetClassifierForInheritance for testing
type MockAssetClassifierForInheritance struct {
	mock.Mock
}

func (m *MockAssetClassifierForInheritance) ClassifyTaskAsset(task *taskdomain.Task) (*ports.AssetClassificationResult, error) {
	args := m.Called(task)
	return args.Get(0).(*ports.AssetClassificationResult), args.Error(1)
}

func (m *MockAssetClassifierForInheritance) ClassifyTasksAssets(tasks []*taskdomain.Task) ([]*ports.AssetClassificationResult, error) {
	args := m.Called(tasks)
	return args.Get(0).([]*ports.AssetClassificationResult), args.Error(1)
}

// MockWorkTypeClassifierForInheritance for testing
type MockWorkTypeClassifierForInheritance struct {
	mock.Mock
}

func (m *MockWorkTypeClassifierForInheritance) ClassifyTask(task *taskdomain.Task) (taskdomain.WorkType, error) {
	args := m.Called(task)
	return args.Get(0).(taskdomain.WorkType), args.Error(1)
}

func (m *MockWorkTypeClassifierForInheritance) ClassifyTasks(tasks []*taskdomain.Task) (map[string]taskdomain.WorkType, error) {
	args := m.Called(tasks)
	return args.Get(0).(map[string]taskdomain.WorkType), args.Error(1)
}

func TestSubtaskInheritance_ClassifyTasks(t *testing.T) {
	tests := []struct {
		name            string
		tasks           []*taskdomain.Task
		expectedResults map[string]expectedClassification
		setupMocks      func(*MockAssetClassifierForInheritance, *MockWorkTypeClassifierForInheritance)
	}{
		{
			name: "subtask inherits asset from parent with cap-asset label",
			tasks: []*taskdomain.Task{
				{
					Key:     "FN-910",
					Summary: "Configurable Cabins Markup for Ferry Ticket Purchases",
					Type:    taskdomain.TaskTypeStory,
					Epic:    "FN-873",
					Labels:  []string{"cap-asset-cabin-markup", "cap-development"},
				},
				{
					Key:     "FN-954",
					Summary: "Bugbash and go-live",
					Type:    taskdomain.TaskTypeSubtask,
					Epic:    "FN-910",
					Labels:  []string{"cap-development"},
				},
			},
			expectedResults: map[string]expectedClassification{
				"FN-910": {
					hasAsset:      true,
					workType:      taskdomain.WorkTypeDevelopment,
					inheritedFrom: "",
					assetName:     "Cabins Markup",
				},
				"FN-954": {
					hasAsset:      true,
					workType:      taskdomain.WorkTypeDevelopment,
					inheritedFrom: "FN-910",
					assetName:     "cabin-markup", // inherited from parent label
				},
			},
			setupMocks: func(assetClassifier *MockAssetClassifierForInheritance, workTypeClassifier *MockWorkTypeClassifierForInheritance) {
				// Parent task classification
				assetClassifier.On("ClassifyTaskAsset", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "FN-910"
				})).Return(&ports.AssetClassificationResult{
					Asset: &assetdomain.Asset{
						Name:     "Cabins Markup",
						Keywords: []string{"cabin", "markup", "ferry"},
					},
					Confidence: 0.9,
					Reason:     "asset name match in task summary",
				}, nil)

				workTypeClassifier.On("ClassifyTask", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "FN-910"
				})).Return(taskdomain.WorkTypeDevelopment, nil)

				// Subtask classification (should be weak and trigger inheritance)
				assetClassifier.On("ClassifyTaskAsset", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "FN-954"
				})).Return(&ports.AssetClassificationResult{
					Asset:      nil,
					Confidence: 0.1,
					Reason:     "no matching asset found",
				}, nil)

				workTypeClassifier.On("ClassifyTask", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "FN-954"
				})).Return(taskdomain.WorkTypeDevelopment, nil)
			},
		},
		{
			name: "subtask inherits work type from parent",
			tasks: []*taskdomain.Task{
				{
					Key:     "BUG-100",
					Summary: "Fix payment gateway timeout",
					Type:    taskdomain.TaskTypeStory,
					Epic:    "",
					Labels:  []string{"cap-asset-payment-gateway", "cap-maintenance"},
				},
				{
					Key:     "BUG-101",
					Summary: "Update error messages",
					Type:    taskdomain.TaskTypeSubtask,
					Epic:    "BUG-100",
					Labels:  []string{"cap-development"}, // Wrong work type, should inherit
				},
			},
			expectedResults: map[string]expectedClassification{
				"BUG-100": {
					hasAsset:      true,
					workType:      taskdomain.WorkTypeMaintenance,
					inheritedFrom: "",
				},
				"BUG-101": {
					hasAsset:      true,
					workType:      taskdomain.WorkTypeMaintenance, // Should inherit from parent
					inheritedFrom: "BUG-100",
				},
			},
			setupMocks: func(assetClassifier *MockAssetClassifierForInheritance, workTypeClassifier *MockWorkTypeClassifierForInheritance) {
				// Parent task classification
				assetClassifier.On("ClassifyTaskAsset", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "BUG-100"
				})).Return(&ports.AssetClassificationResult{
					Asset: &assetdomain.Asset{
						Name:     "Payment Gateway",
						Keywords: []string{"payment", "gateway"},
					},
					Confidence: 0.9,
					Reason:     "explicit asset label match",
				}, nil)

				workTypeClassifier.On("ClassifyTask", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "BUG-100"
				})).Return(taskdomain.WorkTypeMaintenance, nil)

				// Subtask classification
				assetClassifier.On("ClassifyTaskAsset", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "BUG-101"
				})).Return(&ports.AssetClassificationResult{
					Asset:      nil,
					Confidence: 0.2,
					Reason:     "no clear asset match",
				}, nil)

				workTypeClassifier.On("ClassifyTask", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "BUG-101"
				})).Return(taskdomain.WorkTypeDevelopment, nil) // Wrong classification
			},
		},
		{
			name: "subtask with good classification doesn't inherit",
			tasks: []*taskdomain.Task{
				{
					Key:     "FEAT-200",
					Summary: "New user dashboard",
					Type:    taskdomain.TaskTypeStory,
					Epic:    "",
					Labels:  []string{"cap-asset-user-dashboard", "cap-development"},
				},
				{
					Key:     "FEAT-201",
					Summary: "Create API endpoint for user stats",
					Type:    taskdomain.TaskTypeSubtask,
					Epic:    "FEAT-200",
					Labels:  []string{"cap-asset-analytics-api", "cap-development"}, // Already well classified
				},
			},
			expectedResults: map[string]expectedClassification{
				"FEAT-200": {
					hasAsset:      true,
					workType:      taskdomain.WorkTypeDevelopment,
					inheritedFrom: "",
				},
				"FEAT-201": {
					hasAsset:      true,
					workType:      taskdomain.WorkTypeDevelopment,
					inheritedFrom: "", // Should NOT inherit - already well classified
					assetName:     "Analytics API",
				},
			},
			setupMocks: func(assetClassifier *MockAssetClassifierForInheritance, workTypeClassifier *MockWorkTypeClassifierForInheritance) {
				// Parent task classification
				assetClassifier.On("ClassifyTaskAsset", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "FEAT-200"
				})).Return(&ports.AssetClassificationResult{
					Asset: &assetdomain.Asset{
						Name:     "User Dashboard",
						Keywords: []string{"user", "dashboard"},
					},
					Confidence: 0.9,
					Reason:     "explicit asset label match",
				}, nil)

				workTypeClassifier.On("ClassifyTask", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "FEAT-200"
				})).Return(taskdomain.WorkTypeDevelopment, nil)

				// Subtask classification (good classification, shouldn't inherit)
				assetClassifier.On("ClassifyTaskAsset", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "FEAT-201"
				})).Return(&ports.AssetClassificationResult{
					Asset: &assetdomain.Asset{
						Name:     "Analytics API",
						Keywords: []string{"analytics", "api"},
					},
					Confidence: 0.85, // High confidence, shouldn't inherit
					Reason:     "explicit asset label match",
				}, nil)

				workTypeClassifier.On("ClassifyTask", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "FEAT-201"
				})).Return(taskdomain.WorkTypeDevelopment, nil)
			},
		},
		{
			name: "subtask without parent falls back to normal classification",
			tasks: []*taskdomain.Task{
				{
					Key:     "ORPHAN-1",
					Summary: "Standalone subtask",
					Type:    taskdomain.TaskTypeSubtask,
					Epic:    "NONEXISTENT-PARENT", // Parent doesn't exist
					Labels:  []string{"cap-development"},
				},
			},
			expectedResults: map[string]expectedClassification{
				"ORPHAN-1": {
					hasAsset:      false, // No parent to inherit from
					workType:      taskdomain.WorkTypeDevelopment,
					inheritedFrom: "", // No inheritance
				},
			},
			setupMocks: func(assetClassifier *MockAssetClassifierForInheritance, workTypeClassifier *MockWorkTypeClassifierForInheritance) {
				// Subtask classification (no parent available)
				assetClassifier.On("ClassifyTaskAsset", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "ORPHAN-1"
				})).Return(&ports.AssetClassificationResult{
					Asset:      nil,
					Confidence: 0.1,
					Reason:     "no matching asset found",
				}, nil)

				workTypeClassifier.On("ClassifyTask", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "ORPHAN-1"
				})).Return(taskdomain.WorkTypeDevelopment, nil)
			},
		},
		{
			name: "discovery task inherits from epic context",
			tasks: []*taskdomain.Task{
				{
					Key:     "FN-100",
					Summary: "Service Fee Configuration",
					Type:    taskdomain.TaskTypeStory,
					Epic:    "",
					Labels:  []string{"cap-asset-service-fee", "cap-development"},
				},
				{
					Key:     "FN-101",
					Summary: "Spike: Research pricing options",
					Type:    taskdomain.TaskTypeTask,
					Epic:    "FN-100",
					Labels:  []string{"cap-discovery"},
				},
			},
			expectedResults: map[string]expectedClassification{
				"FN-100": {
					hasAsset:      true,
					workType:      taskdomain.WorkTypeDevelopment,
					inheritedFrom: "",
					assetName:     "Service Fee",
				},
				"FN-101": {
					hasAsset:      true,
					workType:      taskdomain.WorkTypeDiscovery,
					inheritedFrom: "FN-100", // Should inherit from epic
					assetName:     "Service Fee",
				},
			},
			setupMocks: func(assetClassifier *MockAssetClassifierForInheritance, workTypeClassifier *MockWorkTypeClassifierForInheritance) {
				// Epic task classification
				assetClassifier.On("ClassifyTaskAsset", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "FN-100"
				})).Return(&ports.AssetClassificationResult{
					Asset: &assetdomain.Asset{
						Name:     "Service Fee",
						Keywords: []string{"service", "fee", "pricing"},
					},
					Confidence: 0.9,
					Reason:     "explicit asset label match",
				}, nil)

				workTypeClassifier.On("ClassifyTask", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "FN-100"
				})).Return(taskdomain.WorkTypeDevelopment, nil)

				// Discovery task classification (weak - should trigger inheritance)
				assetClassifier.On("ClassifyTaskAsset", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "FN-101"
				})).Return(&ports.AssetClassificationResult{
					Asset:      nil,
					Confidence: 0.2,
					Reason:     "no clear asset match in spike task",
				}, nil)

				workTypeClassifier.On("ClassifyTask", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "FN-101"
				})).Return(taskdomain.WorkTypeDiscovery, nil)
			},
		},
		{
			name: "research task with weak classification inherits from epic",
			tasks: []*taskdomain.Task{
				{
					Key:     "ESIM-500",
					Summary: "eSIM Integration for Travelers",
					Type:    taskdomain.TaskTypeStory,
					Epic:    "",
					Labels:  []string{"cap-asset-esim", "cap-development"},
				},
				{
					Key:     "ESIM-501",
					Summary: "Investigation: eSIM provider APIs",
					Type:    taskdomain.TaskTypeTask,
					Epic:    "ESIM-500",
					Labels:  []string{"cap-discovery"},
				},
			},
			expectedResults: map[string]expectedClassification{
				"ESIM-500": {
					hasAsset:      true,
					workType:      taskdomain.WorkTypeDevelopment,
					inheritedFrom: "",
					assetName:     "eSIM",
				},
				"ESIM-501": {
					hasAsset:      true,
					workType:      taskdomain.WorkTypeDiscovery,
					inheritedFrom: "ESIM-500", // Should inherit from epic
					assetName:     "eSIM",
				},
			},
			setupMocks: func(assetClassifier *MockAssetClassifierForInheritance, workTypeClassifier *MockWorkTypeClassifierForInheritance) {
				// Epic task classification
				assetClassifier.On("ClassifyTaskAsset", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "ESIM-500"
				})).Return(&ports.AssetClassificationResult{
					Asset: &assetdomain.Asset{
						Name:     "eSIM",
						Keywords: []string{"esim", "sim", "traveler"},
					},
					Confidence: 0.9,
					Reason:     "explicit asset label match",
				}, nil)

				workTypeClassifier.On("ClassifyTask", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "ESIM-500"
				})).Return(taskdomain.WorkTypeDevelopment, nil)

				// Research task classification (weak - should trigger inheritance)
				assetClassifier.On("ClassifyTaskAsset", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "ESIM-501"
				})).Return(&ports.AssetClassificationResult{
					Asset:      nil,
					Confidence: 0.3,
					Reason:     "unclear context in research task",
				}, nil)

				workTypeClassifier.On("ClassifyTask", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "ESIM-501"
				})).Return(taskdomain.WorkTypeDiscovery, nil)
			},
		},
		{
			name: "discovery task with strong classification doesn't inherit",
			tasks: []*taskdomain.Task{
				{
					Key:     "API-200",
					Summary: "Payment API Refactoring",
					Type:    taskdomain.TaskTypeStory,
					Epic:    "",
					Labels:  []string{"cap-asset-payment-api", "cap-development"},
				},
				{
					Key:     "API-201",
					Summary: "Spike: Analyze user authentication patterns",
					Type:    taskdomain.TaskTypeTask,
					Epic:    "API-200",
					Labels:  []string{"cap-asset-authentication", "cap-discovery"},
				},
			},
			expectedResults: map[string]expectedClassification{
				"API-200": {
					hasAsset:      true,
					workType:      taskdomain.WorkTypeDevelopment,
					inheritedFrom: "",
					assetName:     "Payment API",
				},
				"API-201": {
					hasAsset:      true,
					workType:      taskdomain.WorkTypeDiscovery,
					inheritedFrom: "", // Should NOT inherit - has strong classification
					assetName:     "Authentication",
				},
			},
			setupMocks: func(assetClassifier *MockAssetClassifierForInheritance, workTypeClassifier *MockWorkTypeClassifierForInheritance) {
				// Epic task classification
				assetClassifier.On("ClassifyTaskAsset", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "API-200"
				})).Return(&ports.AssetClassificationResult{
					Asset: &assetdomain.Asset{
						Name:     "Payment API",
						Keywords: []string{"payment", "api"},
					},
					Confidence: 0.9,
					Reason:     "explicit asset label match",
				}, nil)

				workTypeClassifier.On("ClassifyTask", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "API-200"
				})).Return(taskdomain.WorkTypeDevelopment, nil)

				// Discovery task classification (strong - should NOT inherit)
				assetClassifier.On("ClassifyTaskAsset", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "API-201"
				})).Return(&ports.AssetClassificationResult{
					Asset: &assetdomain.Asset{
						Name:     "Authentication",
						Keywords: []string{"authentication", "auth", "user"},
					},
					Confidence: 0.85,
					Reason:     "explicit asset label match",
				}, nil)

				workTypeClassifier.On("ClassifyTask", mock.MatchedBy(func(task *taskdomain.Task) bool {
					return task.Key == "API-201"
				})).Return(taskdomain.WorkTypeDiscovery, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assetClassifier := new(MockAssetClassifierForInheritance)
			workTypeClassifier := new(MockWorkTypeClassifierForInheritance)

			tt.setupMocks(assetClassifier, workTypeClassifier)

			chain := NewComprehensiveClassificationChainWithInheritance(assetClassifier, workTypeClassifier)

			results, err := chain.ClassifyTasks(tt.tasks)
			assert.NoError(t, err)
			assert.Len(t, results, len(tt.tasks))

			// Verify results match expectations
			for _, result := range results {
				expected, exists := tt.expectedResults[result.Task.Key]
				assert.True(t, exists, "Unexpected task result: %s", result.Task.Key)

				// Check asset assignment
				if expected.hasAsset {
					assert.NotNil(t, result.Asset, "Task %s should have asset assigned", result.Task.Key)
					assert.NotNil(t, result.Asset.Asset, "Task %s should have asset assigned", result.Task.Key)
					if expected.assetName != "" {
						// For inherited assets, check the reason contains parent info
						if expected.inheritedFrom != "" {
							assert.Contains(t, result.Asset.Reason, expected.inheritedFrom,
								"Task %s should show inheritance from %s", result.Task.Key, expected.inheritedFrom)
						}
					}
				} else {
					// Either no asset or very low confidence
					hasNoAsset := result.Asset == nil || result.Asset.Asset == nil || result.Asset.Confidence < 0.3
					assert.True(t, hasNoAsset, "Task %s should not have asset assigned", result.Task.Key)
				}

				// Check work type
				assert.Equal(t, expected.workType, result.WorkType,
					"Task %s should have work type %s", result.Task.Key, expected.workType)

				// Check inheritance
				if expected.inheritedFrom != "" {
					assert.Contains(t, result.WorkTypeReason, expected.inheritedFrom,
						"Task %s should show inheritance from %s in reason", result.Task.Key, expected.inheritedFrom)
				}
			}

			assetClassifier.AssertExpectations(t)
			workTypeClassifier.AssertExpectations(t)
		})
	}
}

type expectedClassification struct {
	hasAsset      bool
	workType      taskdomain.WorkType
	inheritedFrom string // If not empty, should inherit from this task
	assetName     string // Expected asset name (if relevant)
}

func TestSubtaskInheritance_inheritFromParent_NoEpic(t *testing.T) {
	chain := &ComprehensiveClassificationChainWithInheritance{}

	task := &taskdomain.Task{
		Key:     "TEST-1",
		Type:    taskdomain.TaskTypeSubtask,
		Summary: "Test subtask",
		Epic:    "", // No epic
	}

	inheritedAsset, inheritedWorkType, inheritedReason := chain.inheritFromParent(task)

	assert.Nil(t, inheritedAsset)
	assert.Equal(t, taskdomain.WorkType(""), inheritedWorkType)
	assert.Equal(t, "", inheritedReason)
}

func TestComprehensiveClassificationChainWithInheritance_ErrorHandling(t *testing.T) {
	tests := []struct {
		name               string
		task               *taskdomain.Task
		assetClassifyError error
		workTypeError      error
		expectError        bool
		errorContains      string
	}{
		{
			name:          "nil task",
			task:          nil,
			expectError:   true,
			errorContains: "task cannot be nil",
		},
		{
			name: "asset classification error",
			task: &taskdomain.Task{
				Key:     "TEST-1",
				Summary: "Test task",
			},
			assetClassifyError: assert.AnError,
			expectError:        true,
			errorContains:      "asset classification failed",
		},
		{
			name: "work type classification error",
			task: &taskdomain.Task{
				Key:     "TEST-2",
				Summary: "Test task",
			},
			workTypeError: assert.AnError,
			expectError:   true,
			errorContains: "work type classification failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assetClassifier := new(MockAssetClassifierForInheritance)
			workTypeClassifier := new(MockWorkTypeClassifierForInheritance)

			if tt.task != nil && tt.assetClassifyError == nil {
				assetClassifier.On("ClassifyTaskAsset", tt.task).Return(&ports.AssetClassificationResult{
					Asset:      nil,
					Confidence: 0.1,
				}, tt.assetClassifyError)
			} else if tt.task != nil {
				assetClassifier.On("ClassifyTaskAsset", tt.task).Return((*ports.AssetClassificationResult)(nil), tt.assetClassifyError)
			}

			if tt.task != nil && tt.workTypeError == nil && tt.assetClassifyError == nil {
				workTypeClassifier.On("ClassifyTask", tt.task).Return(taskdomain.WorkTypeDevelopment, tt.workTypeError)
			} else if tt.task != nil && tt.assetClassifyError == nil {
				workTypeClassifier.On("ClassifyTask", tt.task).Return(taskdomain.WorkType(""), tt.workTypeError)
			}

			chain := NewComprehensiveClassificationChainWithInheritance(assetClassifier, workTypeClassifier)

			result, err := chain.ClassifyTask(tt.task)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			if tt.task != nil {
				assetClassifier.AssertExpectations(t)
				if tt.assetClassifyError == nil {
					workTypeClassifier.AssertExpectations(t)
				}
			}
		})
	}
}

func TestSubtaskInheritance_hasExplicitWorkTypeLabel(t *testing.T) {
	chain := &ComprehensiveClassificationChainWithInheritance{}

	tests := []struct {
		name     string
		task     *taskdomain.Task
		expected bool
	}{
		{
			name: "task with cap-development label",
			task: &taskdomain.Task{
				Key:    "TEST-1",
				Labels: []string{"cap-development", "other-label"},
			},
			expected: true,
		},
		{
			name: "task with cap-discovery label",
			task: &taskdomain.Task{
				Key:    "TEST-2",
				Labels: []string{"cap-discovery"},
			},
			expected: true,
		},
		{
			name: "task with cap-maintenance label",
			task: &taskdomain.Task{
				Key:    "TEST-3",
				Labels: []string{"some-label", "cap-maintenance"},
			},
			expected: true,
		},
		{
			name: "task without work type label",
			task: &taskdomain.Task{
				Key:    "TEST-4",
				Labels: []string{"cap-asset-test", "other-label"},
			},
			expected: false,
		},
		{
			name: "task with no labels",
			task: &taskdomain.Task{
				Key:    "TEST-5",
				Labels: []string{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chain.hasExplicitWorkTypeLabel(tt.task)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSubtaskInheritance_needsInheritance(t *testing.T) {
	chain := &ComprehensiveClassificationChainWithInheritance{}

	tests := []struct {
		name        string
		task        *taskdomain.Task
		assetResult *ports.AssetClassificationResult
		workType    taskdomain.WorkType
		expected    bool
	}{
		{
			name: "no asset found - needs inheritance",
			task: &taskdomain.Task{
				Key:     "TEST-1",
				Type:    taskdomain.TaskTypeSubtask,
				Summary: "Test subtask",
			},
			assetResult: nil,
			workType:    taskdomain.WorkTypeDevelopment,
			expected:    true,
		},
		{
			name: "low confidence asset - needs inheritance",
			task: &taskdomain.Task{
				Key:     "TEST-2",
				Type:    taskdomain.TaskTypeSubtask,
				Summary: "Test subtask",
			},
			assetResult: &ports.AssetClassificationResult{
				Asset:      &assetdomain.Asset{Name: "Test"},
				Confidence: 0.3,
			},
			workType: taskdomain.WorkTypeDevelopment,
			expected: true,
		},
		{
			name: "high confidence asset - no inheritance needed",
			task: &taskdomain.Task{
				Key:     "TEST-3",
				Type:    taskdomain.TaskTypeSubtask,
				Summary: "Test subtask",
			},
			assetResult: &ports.AssetClassificationResult{
				Asset:      &assetdomain.Asset{Name: "Test"},
				Confidence: 0.8,
			},
			workType: taskdomain.WorkTypeMaintenance,
			expected: false,
		},
		{
			name: "no asset but specific work type - needs inheritance",
			task: &taskdomain.Task{
				Key:     "TEST-4",
				Type:    taskdomain.TaskTypeSubtask,
				Summary: "Test subtask",
			},
			assetResult: nil,
			workType:    taskdomain.WorkTypeDiscovery,
			expected:    true,
		},
		{
			name: "discovery task with low confidence - needs inheritance",
			task: &taskdomain.Task{
				Key:     "TEST-5",
				Type:    taskdomain.TaskTypeTask,
				Summary: "Spike: investigate something",
			},
			assetResult: &ports.AssetClassificationResult{
				Asset:      nil,
				Confidence: 0.2,
			},
			workType: taskdomain.WorkTypeDiscovery,
			expected: true,
		},
		{
			name: "primary subject detected - no inheritance",
			task: &taskdomain.Task{
				Key:     "TEST-6",
				Type:    taskdomain.TaskTypeSubtask,
				Summary: "Test subtask",
			},
			assetResult: &ports.AssetClassificationResult{
				Asset:      &assetdomain.Asset{Name: "Test"},
				Confidence: 0.95,
				Reason:     "detected as primary subject based on title emphasis",
			},
			workType: taskdomain.WorkTypeDevelopment,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chain.needsInheritance(tt.task, tt.assetResult, tt.workType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// concurrentAssetClassifier is a hand-rolled fake (no testify Mock,
// which is not safe for concurrent use) that returns a fresh result
// per call so we can drive the parallel classification path under
// -race without setup ceremony.
type concurrentAssetClassifier struct {
	calls int64
}

func (c *concurrentAssetClassifier) ClassifyTaskAsset(task *taskdomain.Task) (*ports.AssetClassificationResult, error) {
	atomic.AddInt64(&c.calls, 1)
	return &ports.AssetClassificationResult{
		Task:       task,
		Asset:      &assetdomain.Asset{Name: "Auto-assigned"},
		Confidence: 0.95,
		Reason:     "stub asset",
	}, nil
}

func (c *concurrentAssetClassifier) ClassifyTasksAssets(tasks []*taskdomain.Task) ([]*ports.AssetClassificationResult, error) {
	out := make([]*ports.AssetClassificationResult, 0, len(tasks))
	for _, task := range tasks {
		r, _ := c.ClassifyTaskAsset(task)
		out = append(out, r)
	}
	return out, nil
}

type concurrentWorkTypeClassifier struct {
	calls int64
}

func (c *concurrentWorkTypeClassifier) ClassifyTask(_ *taskdomain.Task) (taskdomain.WorkType, error) {
	atomic.AddInt64(&c.calls, 1)
	return taskdomain.WorkTypeDevelopment, nil
}

func (c *concurrentWorkTypeClassifier) ClassifyTasks(tasks []*taskdomain.Task) (map[string]taskdomain.WorkType, error) {
	out := make(map[string]taskdomain.WorkType, len(tasks))
	for _, t := range tasks {
		out[t.Key] = taskdomain.WorkTypeDevelopment
	}
	return out, nil
}

// TestComprehensiveClassificationChainWithInheritance_ClassifyTasks_ParallelOrder
// asserts that the parallel pass preserves input order in its output
// slice (non-subtasks first in their input order, then subtasks in theirs)
// and that every task is classified exactly once. Designed to run under
// -race; the high task count makes any per-iteration races likely to
// surface and an out-of-order result obvious.
func TestComprehensiveClassificationChainWithInheritance_ClassifyTasks_ParallelOrder(t *testing.T) {
	const total = 200
	tasks := make([]*taskdomain.Task, 0, total)
	expectedOrder := make([]string, 0, total)

	// Half non-subtasks (no Epic), half subtasks (Epic pointing at a non-subtask).
	for i := 0; i < total/2; i++ {
		key := fmt.Sprintf("STORY-%03d", i)
		tasks = append(tasks, &taskdomain.Task{Key: key, Type: taskdomain.TaskTypeStory})
		expectedOrder = append(expectedOrder, key)
	}
	for i := 0; i < total/2; i++ {
		key := fmt.Sprintf("SUB-%03d", i)
		tasks = append(tasks, &taskdomain.Task{
			Key:  key,
			Type: taskdomain.TaskTypeSubtask,
			Epic: fmt.Sprintf("STORY-%03d", i),
		})
	}
	// After the two-phase split, non-subtasks come first (in their input order)
	// then subtasks (in their input order).
	for i := 0; i < total/2; i++ {
		expectedOrder = append(expectedOrder, fmt.Sprintf("SUB-%03d", i))
	}

	chain := NewComprehensiveClassificationChainWithInheritance(
		&concurrentAssetClassifier{},
		&concurrentWorkTypeClassifier{},
	).(*ComprehensiveClassificationChainWithInheritance)

	results, err := chain.ClassifyTasks(tasks)
	require.NoError(t, err)
	require.Len(t, results, total)

	seen := make(map[string]bool, total)
	for i, r := range results {
		require.NotNil(t, r, "result %d should be non-nil", i)
		require.NotNil(t, r.Task, "result %d task should be non-nil", i)
		assert.Equal(t, expectedOrder[i], r.Task.Key, "order preserved within each phase")
		assert.False(t, seen[r.Task.Key], "task %s classified more than once", r.Task.Key)
		seen[r.Task.Key] = true
	}
	assert.Len(t, seen, total, "every input task should appear exactly once in results")
}
