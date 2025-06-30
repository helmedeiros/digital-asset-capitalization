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
			expectedReason:    "asset name match in task summary",
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
			expectedReason:    "keyword match in task content",
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
				Labels:      []string{"cap-asset-payment", "urgent"},
			},
			assets: []*assetdomain.Asset{
				{
					Name:        "Payment Gateway",
					Description: "Processes online payments",
					Keywords:    []string{"payment", "gateway", "transaction"},
				},
			},
			expectedAssetName: "Payment Gateway",
			expectedMinConf:   0.8,
			expectedMaxConf:   1.0,
			expectedReason:    "explicit asset label match",
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
			expectedReason:    "multiple strong matches",
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
			expectedReason:    "multiple strong matches", // Asset name + multiple keywords match
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
