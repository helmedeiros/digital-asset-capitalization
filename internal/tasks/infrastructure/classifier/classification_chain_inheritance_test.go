package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

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

func TestSubtaskInheritance_needsInheritance(t *testing.T) {
	chain := &ComprehensiveClassificationChainWithInheritance{}

	tests := []struct {
		name        string
		assetResult *ports.AssetClassificationResult
		workType    taskdomain.WorkType
		expected    bool
	}{
		{
			name:        "no asset found - needs inheritance",
			assetResult: nil,
			workType:    taskdomain.WorkTypeDevelopment,
			expected:    true,
		},
		{
			name: "low confidence asset - needs inheritance",
			assetResult: &ports.AssetClassificationResult{
				Asset:      &assetdomain.Asset{Name: "Test"},
				Confidence: 0.3,
			},
			workType: taskdomain.WorkTypeDevelopment,
			expected: true,
		},
		{
			name: "high confidence asset - no inheritance needed",
			assetResult: &ports.AssetClassificationResult{
				Asset:      &assetdomain.Asset{Name: "Test"},
				Confidence: 0.8,
			},
			workType: taskdomain.WorkTypeMaintenance,
			expected: false,
		},
		{
			name:        "no asset but specific work type - needs inheritance",
			assetResult: nil,
			workType:    taskdomain.WorkTypeDiscovery,
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := chain.needsInheritance(tt.assetResult, tt.workType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
