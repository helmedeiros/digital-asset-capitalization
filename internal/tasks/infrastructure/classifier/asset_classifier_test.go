package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

// MockAssetRepository for testing
type MockAssetRepositoryForAssetClassifier struct {
	mock.Mock
}

func (m *MockAssetRepositoryForAssetClassifier) Save(asset *assetdomain.Asset) error {
	args := m.Called(asset)
	return args.Error(0)
}

func (m *MockAssetRepositoryForAssetClassifier) FindAll() ([]*assetdomain.Asset, error) {
	args := m.Called()
	return args.Get(0).([]*assetdomain.Asset), args.Error(1)
}

func (m *MockAssetRepositoryForAssetClassifier) FindByID(id string) (*assetdomain.Asset, error) {
	args := m.Called(id)
	return args.Get(0).(*assetdomain.Asset), args.Error(1)
}

func (m *MockAssetRepositoryForAssetClassifier) FindByName(name string) (*assetdomain.Asset, error) {
	args := m.Called(name)
	return args.Get(0).(*assetdomain.Asset), args.Error(1)
}

func (m *MockAssetRepositoryForAssetClassifier) Delete(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func TestContentBasedAssetClassifier_ClassifyTaskAsset(t *testing.T) {
	tests := []struct {
		name              string
		task              *taskdomain.Task
		assets            []*assetdomain.Asset
		expectedAssetName string
		expectedMinConf   float64
		expectedMaxConf   float64
		expectedReason    string
		expectError       bool
	}{
		{
			name: "exact asset name match in summary",
			task: &taskdomain.Task{
				Key:         "TEST-1",
				Summary:     "Fix Payment Gateway timeout issue",
				Description: "The gateway sometimes fails to process payments",
				Epic:        "Payment Processing",
			},
			assets: []*assetdomain.Asset{
				{
					Name:        "Payment Gateway",
					Description: "Processes online payments",
					Keywords:    []string{"payment", "gateway", "transaction"},
				},
				{
					Name:        "User Management",
					Description: "Manages user accounts",
					Keywords:    []string{"user", "account", "profile"},
				},
			},
			expectedAssetName: "Payment Gateway",
			expectedMinConf:   0.9,
			expectedMaxConf:   1.0,
			expectedReason:    "detected as primary subject based on title emphasis",
		},
		{
			name: "keyword match in description",
			task: &taskdomain.Task{
				Key:         "TEST-2",
				Summary:     "Update transaction processing logic",
				Description: "Need to improve the payment processing efficiency",
				Epic:        "Financial Services",
			},
			assets: []*assetdomain.Asset{
				{
					Name:        "Payment Gateway",
					Description: "Processes online payments",
					Keywords:    []string{"payment", "gateway", "transaction"},
				},
			},
			expectedAssetName: "Payment Gateway",
			expectedMinConf:   0.6,
			expectedMaxConf:   0.9,
			expectedReason:    "keyword match in task title",
		},
		{
			name: "epic name match",
			task: &taskdomain.Task{
				Key:         "TEST-3",
				Summary:     "Implement new feature",
				Description: "Add functionality to the system",
				Epic:        "Payment Gateway Improvements",
			},
			assets: []*assetdomain.Asset{
				{
					Name:        "Payment Gateway",
					Description: "Processes online payments",
					Keywords:    []string{"payment", "gateway", "transaction"},
				},
			},
			expectedAssetName: "Payment Gateway",
			expectedMinConf:   0.7,
			expectedMaxConf:   0.9,
			expectedReason:    "asset name match in epic name",
		},
		{
			name: "label-based asset match",
			task: &taskdomain.Task{
				Key:         "TEST-4",
				Summary:     "Generic task description",
				Description: "Some work needs to be done",
				Epic:        "General Work",
				Labels:      []string{"cap-asset-payment-gateway", "urgent"},
			},
			assets: []*assetdomain.Asset{
				{
					Name:        "Payment Gateway",
					Description: "Processes online payments",
					Keywords:    []string{"payment", "gateway", "transaction"},
				},
			},
			expectedAssetName: "Payment Gateway",
			expectedMinConf:   0.85,
			expectedMaxConf:   1.0,
			expectedReason:    "existing asset label preserved",
		},
		{
			name: "multiple keyword matches - high confidence",
			task: &taskdomain.Task{
				Key:         "TEST-5",
				Summary:     "Fix payment gateway transaction timeout",
				Description: "The payment processing gateway fails on large transactions",
				Epic:        "Payment System",
			},
			assets: []*assetdomain.Asset{
				{
					Name:        "Payment Gateway",
					Description: "Processes online payments",
					Keywords:    []string{"payment", "gateway", "transaction"},
				},
			},
			expectedAssetName: "Payment Gateway",
			expectedMinConf:   0.9,
			expectedMaxConf:   1.0,
			expectedReason:    "detected as primary subject based on title emphasis",
		},
		{
			name: "no asset match",
			task: &taskdomain.Task{
				Key:         "TEST-6",
				Summary:     "Update documentation",
				Description: "Need to update the project documentation",
				Epic:        "Documentation Work",
			},
			assets: []*assetdomain.Asset{
				{
					Name:        "Payment Gateway",
					Description: "Processes online payments",
					Keywords:    []string{"payment", "gateway", "transaction"},
				},
			},
			expectedAssetName: "",
			expectedMinConf:   0.0,
			expectedMaxConf:   0.1,
			expectedReason:    "no matching asset found",
		},
		{
			name: "existing label preserved even when natural classification is strong",
			task: &taskdomain.Task{
				Key:         "TEST-8",
				Summary:     "AB Mode Comparison on SRP - Desktop Web",
				Description: "Implement mode comparison feature on search results page",
				Epic:        "SRP Improvements",
				Labels:      []string{"cap-asset-mode-comparison", "cap-development"},
			},
			assets: []*assetdomain.Asset{
				{
					Name:        "Mode Comparison",
					Description: "Compare different travel modes",
					Keywords:    []string{"mode", "comparison", "srp"},
				},
				{
					Name:        "Search Results Page",
					Description: "Search results page optimization",
					Keywords:    []string{"search", "results", "srp", "filtering"},
				},
			},
			expectedAssetName: "Mode Comparison",
			expectedMinConf:   0.85,
			expectedMaxConf:   0.85,
			expectedReason:    "existing asset label preserved",
		},
		{
			name: "case insensitive matching",
			task: &taskdomain.Task{
				Key:         "TEST-7",
				Summary:     "PAYMENT GATEWAY bug fix",
				Description: "Fix issue with TRANSACTION processing",
				Epic:        "payment system",
			},
			assets: []*assetdomain.Asset{
				{
					Name:        "Payment Gateway",
					Description: "Processes online payments",
					Keywords:    []string{"payment", "gateway", "transaction"},
				},
			},
			expectedAssetName: "Payment Gateway",
			expectedMinConf:   0.9,
			expectedMaxConf:   1.0,
			expectedReason:    "detected as primary subject based on title emphasis", // Asset name + multiple keywords match
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock repository
			mockRepo := new(MockAssetRepositoryForAssetClassifier)
			mockRepo.On("FindAll").Return(tt.assets, nil)

			// Create classifier
			classifier := NewContentBasedAssetClassifier(mockRepo)

			// Execute classification
			result, err := classifier.ClassifyTaskAsset(tt.task)

			// Verify error expectation
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			// Verify successful classification
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.task, result.Task)

			if tt.expectedAssetName != "" {
				assert.NotNil(t, result.Asset)
				assert.Equal(t, tt.expectedAssetName, result.Asset.Name)
				assert.GreaterOrEqual(t, result.Confidence, tt.expectedMinConf)
				assert.LessOrEqual(t, result.Confidence, tt.expectedMaxConf)
			} else {
				assert.Nil(t, result.Asset)
				assert.LessOrEqual(t, result.Confidence, tt.expectedMaxConf)
			}

			assert.Contains(t, result.Reason, tt.expectedReason)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestContentBasedAssetClassifier_ClassifyTasksAssets(t *testing.T) {
	// Setup test data
	assets := []*assetdomain.Asset{
		{
			Name:        "Payment Gateway",
			Description: "Processes online payments",
			Keywords:    []string{"payment", "gateway", "transaction"},
		},
		{
			Name:        "User Management",
			Description: "Manages user accounts",
			Keywords:    []string{"user", "account", "profile"},
		},
	}

	tasks := []*taskdomain.Task{
		{
			Key:         "TEST-1",
			Summary:     "Fix Payment Gateway timeout",
			Description: "Gateway timeout issue",
			Epic:        "Payment Processing",
		},
		{
			Key:         "TEST-2",
			Summary:     "Update user profile page",
			Description: "Improve user account management",
			Epic:        "User Experience",
		},
	}

	// Setup mock repository
	mockRepo := new(MockAssetRepositoryForAssetClassifier)
	mockRepo.On("FindAll").Return(assets, nil)

	// Create classifier
	classifier := NewContentBasedAssetClassifier(mockRepo)

	// Execute batch classification
	results, err := classifier.ClassifyTasksAssets(tasks)

	// Verify results
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	// Verify first task classified to Payment Gateway
	assert.Equal(t, "TEST-1", results[0].Task.Key)
	assert.NotNil(t, results[0].Asset)
	assert.Equal(t, "Payment Gateway", results[0].Asset.Name)
	assert.Greater(t, results[0].Confidence, 0.8)

	// Verify second task classified to User Management
	assert.Equal(t, "TEST-2", results[1].Task.Key)
	assert.NotNil(t, results[1].Asset)
	assert.Equal(t, "User Management", results[1].Asset.Name)
	assert.Greater(t, results[1].Confidence, 0.6)

	mockRepo.AssertExpectations(t)
}

func TestContentBasedAssetClassifier_RepositoryError(t *testing.T) {
	// Setup mock repository with error
	mockRepo := new(MockAssetRepositoryForAssetClassifier)
	mockRepo.On("FindAll").Return([]*assetdomain.Asset(nil), assert.AnError)

	// Create classifier
	classifier := NewContentBasedAssetClassifier(mockRepo)

	// Create test task
	task := &taskdomain.Task{
		Key:     "TEST-1",
		Summary: "Test task",
	}

	// Execute classification
	result, err := classifier.ClassifyTaskAsset(task)

	// Verify error handling
	assert.Error(t, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestContentBasedAssetClassifier_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		task        *taskdomain.Task
		assets      []*assetdomain.Asset
		expectError bool
	}{
		{
			name:        "nil task",
			task:        nil,
			assets:      []*assetdomain.Asset{},
			expectError: true,
		},
		{
			name: "empty asset list",
			task: &taskdomain.Task{
				Key:     "TEST-1",
				Summary: "Test task",
			},
			assets:      []*assetdomain.Asset{},
			expectError: false,
		},
		{
			name: "task with empty fields",
			task: &taskdomain.Task{
				Key:         "TEST-1",
				Summary:     "",
				Description: "",
				Epic:        "",
			},
			assets: []*assetdomain.Asset{
				{
					Name:     "Test Asset",
					Keywords: []string{"test"},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock repository
			mockRepo := new(MockAssetRepositoryForAssetClassifier)
			if tt.task != nil {
				mockRepo.On("FindAll").Return(tt.assets, nil)
			}

			// Create classifier
			classifier := NewContentBasedAssetClassifier(mockRepo)

			// Execute classification
			result, err := classifier.ClassifyTaskAsset(tt.task)

			// Verify error expectation
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			if tt.task != nil {
				mockRepo.AssertExpectations(t)
			}
		})
	}
}

func TestContentBasedAssetClassifier_DetectPrimarySubject(t *testing.T) {
	mockRepo := new(MockAssetRepositoryForAssetClassifier)
	classifier := NewContentBasedAssetClassifier(mockRepo).(*ContentBasedAssetClassifier)

	tests := []struct {
		name              string
		task              *taskdomain.Task
		assets            []*assetdomain.Asset
		expectedAssetName string
	}{
		{
			name: "primary subject detected with multiple mentions",
			task: &taskdomain.Task{
				Summary: "Fix payment gateway issues",
			},
			assets: []*assetdomain.Asset{
				{
					Name:     "Payment Gateway",
					Keywords: []string{"payment", "gateway"},
				},
				{
					Name:     "User Management",
					Keywords: []string{"user"},
				},
			},
			expectedAssetName: "Payment Gateway",
		},
		{
			name: "no primary subject - single mention",
			task: &taskdomain.Task{
				Summary: "Update payment logic",
			},
			assets: []*assetdomain.Asset{
				{
					Name:     "Payment Gateway",
					Keywords: []string{"payment", "gateway"},
				},
			},
			expectedAssetName: "",
		},
		{
			name: "primary subject with short keywords filtered out",
			task: &taskdomain.Task{
				Summary: "Fix API auth issue",
			},
			assets: []*assetdomain.Asset{
				{
					Name:     "API Gateway",
					Keywords: []string{"api", "auth", "gateway"},
				},
			},
			expectedAssetName: "",
		},
		{
			name: "no primary subject with empty task",
			task: &taskdomain.Task{
				Summary: "",
			},
			assets: []*assetdomain.Asset{
				{
					Name:     "Payment Gateway",
					Keywords: []string{"payment", "gateway"},
				},
			},
			expectedAssetName: "",
		},
		{
			name: "title match with asset name",
			task: &taskdomain.Task{
				Summary: "eSIM eSIM enhancement",
			},
			assets: []*assetdomain.Asset{
				{
					Name:     "eSIM",
					Keywords: []string{"esim"},
				},
			},
			expectedAssetName: "eSIM",
		},
		{
			name: "multiple assets compete",
			task: &taskdomain.Task{
				Summary: "payment gateway payment",
			},
			assets: []*assetdomain.Asset{
				{
					Name:     "Payment Gateway",
					Keywords: []string{"payment", "gateway"},
				},
				{
					Name:     "Payment System",
					Keywords: []string{"payment", "system"},
				},
			},
			expectedAssetName: "Payment Gateway",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.detectPrimarySubject(tt.task, tt.assets)

			if tt.expectedAssetName == "" {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedAssetName, result.Name)
			}
		})
	}
}

func TestContentBasedAssetClassifier_ShouldPrioritizeSecondaryAsset(t *testing.T) {
	mockRepo := new(MockAssetRepositoryForAssetClassifier)
	classifier := NewContentBasedAssetClassifier(mockRepo).(*ContentBasedAssetClassifier)

	tests := []struct {
		name        string
		taskSummary string
		assetName   string
		expected    bool
	}{
		{
			name:        "X-based Y experiment pattern - should prioritize",
			taskSummary: "dynamic markup-based rounding experiment",
			assetName:   "Dynamic Rounding",
			expected:    true,
		},
		{
			name:        "X using Y pattern - should prioritize",
			taskSummary: "implement pricing using service fee",
			assetName:   "Service Fee",
			expected:    true,
		},
		{
			name:        "rounding experiment pattern - should prioritize",
			taskSummary: "rounding experiment for fare calculation",
			assetName:   "Dynamic Rounding",
			expected:    true,
		},
		{
			name:        "X test pattern - should prioritize",
			taskSummary: "service test for new market",
			assetName:   "Payment Service",
			expected:    true,
		},
		{
			name:        "special case rounding experiment - should prioritize",
			taskSummary: "configure rounding experiment parameters",
			assetName:   "Rounding Rules",
			expected:    true,
		},
		{
			name:        "no special pattern - should not prioritize",
			taskSummary: "update fare calculation logic",
			assetName:   "Service Fee",
			expected:    false,
		},
		{
			name:        "experiment without using/based - should not prioritize",
			taskSummary: "new experiment for pricing",
			assetName:   "Service Fee",
			expected:    false,
		},
		{
			name:        "using without matching asset - should not prioritize",
			taskSummary: "implement pricing using markup algorithm",
			assetName:   "Service Fee",
			expected:    false,
		},
		{
			name:        "short asset words filtered - should not prioritize",
			taskSummary: "test api for new endpoint",
			assetName:   "API",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.shouldPrioritizeSecondaryAsset(tt.taskSummary, tt.assetName)
			assert.Equal(t, tt.expected, result, "Expected shouldPrioritizeSecondaryAsset('%s', '%s') to be %t",
				tt.taskSummary, tt.assetName, tt.expected)
		})
	}
}

func TestContentBasedAssetClassifier_TitleWeighting(t *testing.T) {
	tests := []struct {
		name              string
		task              *taskdomain.Task
		assets            []*assetdomain.Asset
		expectedAssetName string
		minConfidence     float64
	}{
		{
			name: "title match overrides description noise",
			task: &taskdomain.Task{
				Key:         "TEST-1",
				Summary:     "Show eSim on booking funnel",
				Description: "Update price breakdown to show eSIM as free item. The price breakdown service fee calculation needs adjustment.",
			},
			assets: []*assetdomain.Asset{
				{
					Name:     "eSIM",
					Keywords: []string{"esim"},
				},
				{
					Name:     "Service Fee",
					Keywords: []string{"service", "fee", "price", "breakdown"},
				},
			},
			expectedAssetName: "eSIM",
			minConfidence:     0.9,
		},
		{
			name: "title keyword match with high priority",
			task: &taskdomain.Task{
				Key:         "TEST-2",
				Summary:     "Fix transaction processing",
				Description: "The payment gateway sometimes fails",
			},
			assets: []*assetdomain.Asset{
				{
					Name:     "Payment Gateway",
					Keywords: []string{"payment", "gateway", "transaction"},
				},
			},
			expectedAssetName: "Payment Gateway",
			minConfidence:     0.6,
		},
		{
			name: "multiple title keywords boost confidence",
			task: &taskdomain.Task{
				Key:         "TEST-3",
				Summary:     "Payment gateway transaction timeout",
				Description: "Issue with processing",
			},
			assets: []*assetdomain.Asset{
				{
					Name:     "Payment Gateway",
					Keywords: []string{"payment", "gateway", "transaction"},
				},
			},
			expectedAssetName: "Payment Gateway",
			minConfidence:     0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAssetRepositoryForAssetClassifier)
			mockRepo.On("FindAll").Return(tt.assets, nil)

			classifier := NewContentBasedAssetClassifier(mockRepo)
			result, err := classifier.ClassifyTaskAsset(tt.task)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.NotNil(t, result.Asset)
			assert.Equal(t, tt.expectedAssetName, result.Asset.Name)
			assert.GreaterOrEqual(t, result.Confidence, tt.minConfidence)
			mockRepo.AssertExpectations(t)
		})
	}
}
