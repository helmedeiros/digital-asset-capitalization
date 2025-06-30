package classifier

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

// MockAssetRepository is a mock implementation of AssetRepository
type MockAssetRepository struct {
	mock.Mock
}

func (m *MockAssetRepository) Save(asset *assetdomain.Asset) error {
	args := m.Called(asset)
	return args.Error(0)
}

func (m *MockAssetRepository) FindByName(name string) (*assetdomain.Asset, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*assetdomain.Asset), args.Error(1)
}

func (m *MockAssetRepository) FindByID(id string) (*assetdomain.Asset, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*assetdomain.Asset), args.Error(1)
}

func (m *MockAssetRepository) FindAll() ([]*assetdomain.Asset, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*assetdomain.Asset), args.Error(1)
}

func (m *MockAssetRepository) Delete(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func TestBusinessRulesClassifier_ClassifyTask_SpikeOrResearch(t *testing.T) {
	assetRepo := new(MockAssetRepository)
	classifier := NewBusinessRulesClassifier(assetRepo)

	tests := []struct {
		name       string
		task       *taskdomain.Task
		expectedWt taskdomain.WorkType
	}{
		{
			name: "spike in labels",
			task: &taskdomain.Task{
				Key:     "TEST-1",
				Summary: "Regular task",
				Labels:  []string{"spike", "frontend"},
			},
			expectedWt: taskdomain.WorkTypeDiscovery,
		},
		{
			name: "research in labels",
			task: &taskdomain.Task{
				Key:     "TEST-2",
				Summary: "Regular task",
				Labels:  []string{"research", "backend"},
			},
			expectedWt: taskdomain.WorkTypeDiscovery,
		},
		{
			name: "poc in labels",
			task: &taskdomain.Task{
				Key:     "TEST-3",
				Summary: "Regular task",
				Labels:  []string{"poc", "investigation"},
			},
			expectedWt: taskdomain.WorkTypeDiscovery,
		},
		{
			name: "spike in summary",
			task: &taskdomain.Task{
				Key:     "TEST-4",
				Summary: "Spike: Investigate new technology",
				Labels:  []string{},
			},
			expectedWt: taskdomain.WorkTypeDiscovery,
		},
		{
			name: "research in description",
			task: &taskdomain.Task{
				Key:         "TEST-5",
				Summary:     "Task to do research",
				Description: "Need to research the best approach for this feature",
				Labels:      []string{},
			},
			expectedWt: taskdomain.WorkTypeDiscovery,
		},
		{
			name: "proof of concept in content",
			task: &taskdomain.Task{
				Key:         "TEST-6",
				Summary:     "Create proof of concept",
				Description: "Build a prototype to validate the approach",
				Labels:      []string{},
			},
			expectedWt: taskdomain.WorkTypeDiscovery,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For spike/research tasks, no asset repository calls should be made
			workType, err := classifier.ClassifyTask(tt.task)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedWt, workType)

			// No mock expectations to assert since no calls should be made
		})
	}
}

func TestBusinessRulesClassifier_ClassifyTask_WithinDevelopmentPeriod(t *testing.T) {
	assetRepo := new(MockAssetRepository)
	classifier := NewBusinessRulesClassifier(assetRepo)

	// Create test assets
	now := time.Now()
	recentAsset := &assetdomain.Asset{
		Name:           "recent-asset",
		Keywords:       []string{"payment"},
		LaunchDate:     now.AddDate(0, -3, 0), // 3 months ago
		IsRolledOut100: true,
	}

	notRolledOutAsset := &assetdomain.Asset{
		Name:           "development-asset",
		Keywords:       []string{"booking"},
		LaunchDate:     now.AddDate(0, -12, 0), // 12 months ago
		IsRolledOut100: false,                  // Still in development
	}

	tests := []struct {
		name       string
		task       *taskdomain.Task
		assets     []*assetdomain.Asset
		expectedWt taskdomain.WorkType
	}{
		{
			name: "task for asset within 6 months",
			task: &taskdomain.Task{
				Key:         "TEST-1",
				Summary:     "Payment feature improvement",
				Description: "Improve payment processing",
			},
			assets:     []*assetdomain.Asset{recentAsset},
			expectedWt: taskdomain.WorkTypeDevelopment,
		},
		{
			name: "task for asset not 100% rolled out",
			task: &taskdomain.Task{
				Key:         "TEST-2",
				Summary:     "Booking system update",
				Description: "Update booking logic",
			},
			assets:     []*assetdomain.Asset{notRolledOutAsset},
			expectedWt: taskdomain.WorkTypeDevelopment,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assetRepo.On("FindAll").Return(tt.assets, nil).Once()

			workType, err := classifier.ClassifyTask(tt.task)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedWt, workType)

			assetRepo.AssertExpectations(t)
		})
	}
}

func TestBusinessRulesClassifier_ClassifyTask_NewAPIOrInventory(t *testing.T) {
	assetRepo := new(MockAssetRepository)
	classifier := NewBusinessRulesClassifier(assetRepo)

	tests := []struct {
		name       string
		task       *taskdomain.Task
		expectedWt taskdomain.WorkType
	}{
		{
			name: "add new API endpoint",
			task: &taskdomain.Task{
				Key:         "TEST-1",
				Summary:     "Add new API endpoint",
				Description: "Create REST endpoint for user management",
			},
			expectedWt: taskdomain.WorkTypeDevelopment,
		},
		{
			name: "implement new service",
			task: &taskdomain.Task{
				Key:         "TEST-2",
				Summary:     "Implement new microservice",
				Description: "Build service for inventory management",
			},
			expectedWt: taskdomain.WorkTypeDevelopment,
		},
		{
			name: "create new product catalog",
			task: &taskdomain.Task{
				Key:         "TEST-3",
				Summary:     "Create product catalog",
				Description: "Add new inventory system for products",
			},
			expectedWt: taskdomain.WorkTypeDevelopment,
		},
		{
			name: "develop webhook integration",
			task: &taskdomain.Task{
				Key:         "TEST-4",
				Summary:     "Develop webhook for notifications",
				Description: "Build new webhook API for real-time updates",
			},
			expectedWt: taskdomain.WorkTypeDevelopment,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For new API/inventory tasks, no asset repository calls should be made
			workType, err := classifier.ClassifyTask(tt.task)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedWt, workType)

			// No mock expectations to assert since no calls should be made
		})
	}
}

func TestBusinessRulesClassifier_ClassifyTask_BugFixPastRollout(t *testing.T) {
	assetRepo := new(MockAssetRepository)
	classifier := NewBusinessRulesClassifier(assetRepo)

	// Create test asset that was rolled out more than 6 months ago
	now := time.Now()
	oldAsset := &assetdomain.Asset{
		Name:           "legacy-asset",
		Keywords:       []string{"search"},
		LaunchDate:     now.AddDate(0, -12, 0), // 12 months ago
		IsRolledOut100: true,
	}

	tests := []struct {
		name       string
		task       *taskdomain.Task
		assets     []*assetdomain.Asset
		expectedWt taskdomain.WorkType
	}{
		{
			name: "bug fix for old asset",
			task: &taskdomain.Task{
				Key:         "TEST-1",
				Summary:     "Fix search bug",
				Description: "Resolve issue with search functionality",
				Type:        taskdomain.TaskTypeBug,
			},
			assets:     []*assetdomain.Asset{oldAsset},
			expectedWt: taskdomain.WorkTypeMaintenance,
		},
		{
			name: "hotfix for old asset",
			task: &taskdomain.Task{
				Key:         "TEST-2",
				Summary:     "Hotfix search performance",
				Description: "Fix performance problem in search service",
				Type:        taskdomain.TaskTypeTask,
			},
			assets:     []*assetdomain.Asset{oldAsset},
			expectedWt: taskdomain.WorkTypeMaintenance,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assetRepo.On("FindAll").Return(tt.assets, nil).Once()

			workType, err := classifier.ClassifyTask(tt.task)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedWt, workType)

			assetRepo.AssertExpectations(t)
		})
	}
}

func TestBusinessRulesClassifier_ClassifyTask_ContentFallback(t *testing.T) {
	assetRepo := new(MockAssetRepository)
	classifier := NewBusinessRulesClassifier(assetRepo)

	tests := []struct {
		name       string
		task       *taskdomain.Task
		expectedWt taskdomain.WorkType
	}{
		{
			name: "research content",
			task: &taskdomain.Task{
				Key:         "TEST-1",
				Summary:     "Study user behavior",
				Description: "Analyze and understand user patterns",
			},
			expectedWt: taskdomain.WorkTypeDiscovery,
		},
		{
			name: "implementation content",
			task: &taskdomain.Task{
				Key:         "TEST-2",
				Summary:     "Build new feature",
				Description: "Implement user dashboard functionality",
			},
			expectedWt: taskdomain.WorkTypeDevelopment,
		},
		{
			name: "maintenance content",
			task: &taskdomain.Task{
				Key:         "TEST-3",
				Summary:     "Fix broken functionality",
				Description: "Repair issue with login system",
			},
			expectedWt: taskdomain.WorkTypeMaintenance,
		},
		{
			name: "default case",
			task: &taskdomain.Task{
				Key:         "TEST-4",
				Summary:     "Regular task",
				Description: "Some work to be done",
			},
			expectedWt: taskdomain.WorkTypeDevelopment,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For content fallback tests, asset lookup will be attempted but no assets found
			assetRepo.On("FindAll").Return([]*assetdomain.Asset{}, nil).Once()

			workType, err := classifier.ClassifyTask(tt.task)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedWt, workType)

			assetRepo.AssertExpectations(t)
		})
	}
}

func TestBusinessRulesClassifier_ClassifyTasks(t *testing.T) {
	assetRepo := new(MockAssetRepository)
	classifier := NewBusinessRulesClassifier(assetRepo)

	tasks := []*taskdomain.Task{
		{
			Key:     "TEST-1",
			Summary: "Spike: Research new framework",
			Labels:  []string{"spike"},
		},
		{
			Key:         "TEST-2",
			Summary:     "Add new API endpoint",
			Description: "Create REST API for user management",
		},
		{
			Key:         "TEST-3",
			Summary:     "Fix login bug",
			Description: "Resolve authentication issue",
			Type:        taskdomain.TaskTypeBug,
		},
	}

	// Mock no assets found - only called for tasks that need asset lookup (TEST-3)
	assetRepo.On("FindAll").Return([]*assetdomain.Asset{}, nil).Once()

	workTypes, err := classifier.ClassifyTasks(tasks)
	assert.NoError(t, err)
	assert.Len(t, workTypes, 3)

	assert.Equal(t, taskdomain.WorkTypeDiscovery, workTypes["TEST-1"])
	assert.Equal(t, taskdomain.WorkTypeDevelopment, workTypes["TEST-2"])
	assert.Equal(t, taskdomain.WorkTypeMaintenance, workTypes["TEST-3"])

	assetRepo.AssertExpectations(t)
}

func TestBusinessRulesClassifier_IsSpikeOrResearch(t *testing.T) {
	assetRepo := new(MockAssetRepository)
	classifier := NewBusinessRulesClassifier(assetRepo)

	tests := []struct {
		name     string
		task     *taskdomain.Task
		expected bool
	}{
		{
			name: "spike in labels",
			task: &taskdomain.Task{
				Labels: []string{"spike", "frontend"},
			},
			expected: true,
		},
		{
			name: "research in content",
			task: &taskdomain.Task{
				Summary: "Research best practices",
			},
			expected: true,
		},
		{
			name: "investigation in labels",
			task: &taskdomain.Task{
				Labels: []string{"investigation"},
			},
			expected: true,
		},
		{
			name: "regular task",
			task: &taskdomain.Task{
				Summary: "Implement feature",
				Labels:  []string{"feature"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.isSpikeOrResearch(tt.task)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBusinessRulesClassifier_IsWithinDevelopmentPeriod(t *testing.T) {
	assetRepo := new(MockAssetRepository)
	classifier := NewBusinessRulesClassifier(assetRepo)

	now := time.Now()

	tests := []struct {
		name     string
		asset    *assetdomain.Asset
		expected bool
	}{
		{
			name:     "nil asset",
			asset:    nil,
			expected: false,
		},
		{
			name: "not rolled out",
			asset: &assetdomain.Asset{
				IsRolledOut100: false,
			},
			expected: true,
		},
		{
			name: "zero launch date",
			asset: &assetdomain.Asset{
				IsRolledOut100: true,
				LaunchDate:     time.Time{},
			},
			expected: true,
		},
		{
			name: "within 6 months",
			asset: &assetdomain.Asset{
				IsRolledOut100: true,
				LaunchDate:     now.AddDate(0, -3, 0),
			},
			expected: true,
		},
		{
			name: "older than 6 months",
			asset: &assetdomain.Asset{
				IsRolledOut100: true,
				LaunchDate:     now.AddDate(0, -12, 0),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.isWithinDevelopmentPeriod(tt.asset)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBusinessRulesClassifier_AddsNewAPIOrInventory(t *testing.T) {
	assetRepo := new(MockAssetRepository)
	classifier := NewBusinessRulesClassifier(assetRepo)

	tests := []struct {
		name     string
		task     *taskdomain.Task
		expected bool
	}{
		{
			name: "add new API",
			task: &taskdomain.Task{
				Summary: "Add new API endpoint",
			},
			expected: true,
		},
		{
			name: "create inventory",
			task: &taskdomain.Task{
				Description: "Create new inventory system",
			},
			expected: true,
		},
		{
			name: "implement service",
			task: &taskdomain.Task{
				Summary: "Implement microservice",
			},
			expected: true,
		},
		{
			name: "add new rule",
			task: &taskdomain.Task{
				Summary: "Add new business rule",
			},
			expected: true,
		},
		{
			name: "create insurance model",
			task: &taskdomain.Task{
				Description: "Create new insurance model",
			},
			expected: true,
		},
		{
			name: "implement policy system",
			task: &taskdomain.Task{
				Summary: "Implement new policy framework",
			},
			expected: true,
		},
		{
			name: "build coverage calculator",
			task: &taskdomain.Task{
				Description: "Build new coverage calculator",
			},
			expected: true,
		},
		{
			name: "develop eligibility service",
			task: &taskdomain.Task{
				Summary: "Develop eligibility checking service",
			},
			expected: true,
		},
		{
			name: "add benefit plan",
			task: &taskdomain.Task{
				Description: "Add new benefit plan type",
			},
			expected: true,
		},
		{
			name: "just API mention",
			task: &taskdomain.Task{
				Summary: "Use existing API",
			},
			expected: false,
		},
		{
			name: "just inventory mention",
			task: &taskdomain.Task{
				Summary: "Check inventory levels",
			},
			expected: false,
		},
		{
			name: "no API or inventory",
			task: &taskdomain.Task{
				Summary: "Update user interface",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.addsNewAPIOrInventory(tt.task)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBusinessRulesClassifier_IsBugOrFix(t *testing.T) {
	assetRepo := new(MockAssetRepository)
	classifier := NewBusinessRulesClassifier(assetRepo)

	tests := []struct {
		name     string
		task     *taskdomain.Task
		expected bool
	}{
		{
			name: "bug type",
			task: &taskdomain.Task{
				Type: taskdomain.TaskTypeBug,
			},
			expected: true,
		},
		{
			name: "fix in summary",
			task: &taskdomain.Task{
				Summary: "Fix login issue",
				Type:    taskdomain.TaskTypeTask,
			},
			expected: true,
		},
		{
			name: "error in description",
			task: &taskdomain.Task{
				Description: "Resolve error in payment processing",
				Type:        taskdomain.TaskTypeTask,
			},
			expected: true,
		},
		{
			name: "hotfix in label",
			task: &taskdomain.Task{
				Labels: []string{"hotfix"},
				Type:   taskdomain.TaskTypeTask,
			},
			expected: true,
		},
		{
			name: "regular task",
			task: &taskdomain.Task{
				Summary: "Implement new feature",
				Type:    taskdomain.TaskTypeTask,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.isBugOrFix(tt.task)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Additional comprehensive tests for better coverage
func TestBusinessRulesClassifier_TaskMatchesAsset_Comprehensive(t *testing.T) {
	assetRepo := new(MockAssetRepository)
	classifier := NewBusinessRulesClassifier(assetRepo)

	asset := &assetdomain.Asset{
		Name:     "Payment Gateway",
		Keywords: []string{"payment", "gateway", "transaction"},
	}

	tests := []struct {
		name     string
		task     *taskdomain.Task
		asset    *assetdomain.Asset
		expected bool
	}{
		{
			name: "matches asset name in summary",
			task: &taskdomain.Task{
				Summary: "Fix Payment Gateway issue",
			},
			asset:    asset,
			expected: true,
		},
		{
			name: "matches asset name in description",
			task: &taskdomain.Task{
				Description: "Update payment gateway configuration",
			},
			asset:    asset,
			expected: true,
		},
		{
			name: "matches asset keyword in summary",
			task: &taskdomain.Task{
				Summary: "Improve transaction processing",
			},
			asset:    asset,
			expected: true,
		},
		{
			name: "matches asset keyword in description",
			task: &taskdomain.Task{
				Description: "Handle gateway timeouts",
			},
			asset:    asset,
			expected: true,
		},
		{
			name: "matches asset name in label",
			task: &taskdomain.Task{
				Labels: []string{"payment-gateway", "backend"},
			},
			asset:    asset,
			expected: true,
		},
		{
			name: "matches asset keyword in label",
			task: &taskdomain.Task{
				Labels: []string{"transaction-fix", "urgent"},
			},
			asset:    asset,
			expected: true,
		},
		{
			name: "no match",
			task: &taskdomain.Task{
				Summary:     "Update user interface",
				Description: "Improve UI components",
				Labels:      []string{"frontend", "ui"},
			},
			asset:    asset,
			expected: false,
		},
		{
			name: "case insensitive match",
			task: &taskdomain.Task{
				Summary: "PAYMENT GATEWAY update",
			},
			asset:    asset,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.taskMatchesAsset(tt.task, tt.asset)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBusinessRulesClassifier_FindRelatedAsset_EdgeCases(t *testing.T) {
	assetRepo := new(MockAssetRepository)
	classifier := NewBusinessRulesClassifier(assetRepo)

	tests := []struct {
		name          string
		task          *taskdomain.Task
		assets        []*assetdomain.Asset
		repoError     error
		expectedAsset *assetdomain.Asset
		expectError   bool
	}{
		{
			name: "repository error",
			task: &taskdomain.Task{
				Summary: "Test task",
			},
			assets:        nil,
			repoError:     assert.AnError,
			expectedAsset: nil,
			expectError:   true,
		},
		{
			name: "no assets found",
			task: &taskdomain.Task{
				Summary: "Random task",
			},
			assets:        []*assetdomain.Asset{},
			repoError:     nil,
			expectedAsset: nil,
			expectError:   false,
		},
		{
			name: "no matching asset",
			task: &taskdomain.Task{
				Summary: "Unrelated task",
			},
			assets: []*assetdomain.Asset{
				{
					Name:     "Other Asset",
					Keywords: []string{"other", "different"},
				},
			},
			repoError:     nil,
			expectedAsset: nil,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assetRepo.On("FindAll").Return(tt.assets, tt.repoError).Once()

			asset, err := classifier.findRelatedAsset(tt.task)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedAsset, asset)

			assetRepo.AssertExpectations(t)
		})
	}
}

func TestBusinessRulesClassifier_ClassifyTask_EdgeCases(t *testing.T) {
	assetRepo := new(MockAssetRepository)
	classifier := NewBusinessRulesClassifier(assetRepo)

	tests := []struct {
		name        string
		task        *taskdomain.Task
		assets      []*assetdomain.Asset
		repoError   error
		expectedWt  taskdomain.WorkType
		expectError bool
	}{
		{
			name: "repository error - falls back to content classification",
			task: &taskdomain.Task{
				Key:     "TEST-1",
				Summary: "Regular task",
			},
			assets:      nil,
			repoError:   assert.AnError,
			expectedWt:  taskdomain.WorkTypeDevelopment, // fallback classification (default)
			expectError: false,
		},
		{
			name: "no matching asset - fallback to content",
			task: &taskdomain.Task{
				Key:         "TEST-2",
				Summary:     "Some regular task",
				Description: "Just a regular development task",
			},
			assets:      []*assetdomain.Asset{},
			repoError:   nil,
			expectedWt:  taskdomain.WorkTypeDevelopment, // fallback classification
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Only set up mock if not testing spike/research
			if !strings.Contains(strings.ToLower(tt.task.Summary+" "+tt.task.Description), "spike") &&
				!containsLabel(tt.task.Labels, []string{"spike", "research", "poc"}) {
				assetRepo.On("FindAll").Return(tt.assets, tt.repoError).Once()
			}

			workType, err := classifier.ClassifyTask(tt.task)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedWt, workType)
			}

			assetRepo.AssertExpectations(t)
		})
	}
}

func TestBusinessRulesClassifier_ClassifyTasks_EdgeCases(t *testing.T) {
	assetRepo := new(MockAssetRepository)
	classifier := NewBusinessRulesClassifier(assetRepo)

	tests := []struct {
		name        string
		tasks       []*taskdomain.Task
		expectError bool
	}{
		{
			name:        "empty task list",
			tasks:       []*taskdomain.Task{},
			expectError: false,
		},
		{
			name: "repository error handled gracefully",
			tasks: []*taskdomain.Task{
				{
					Key:     "ERROR-1",
					Summary: "Task that causes error",
				},
			},
			expectError: false, // ClassifyTask handles errors gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "repository error handled gracefully" {
				assetRepo.On("FindAll").Return(nil, assert.AnError).Once()
			}

			classifications, err := classifier.ClassifyTasks(tt.tasks)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, classifications)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, classifications)
				assert.Len(t, classifications, len(tt.tasks))
			}

			assetRepo.AssertExpectations(t)
		})
	}
}

func TestBusinessRulesClassifier_IsBugOrFixPastRollout_EdgeCases(t *testing.T) {
	assetRepo := new(MockAssetRepository)
	classifier := NewBusinessRulesClassifier(assetRepo)

	now := time.Now()

	tests := []struct {
		name     string
		task     *taskdomain.Task
		asset    *assetdomain.Asset
		expected bool
	}{
		{
			name: "not a bug - returns false",
			task: &taskdomain.Task{
				Summary: "Regular feature task",
				Type:    taskdomain.TaskTypeTask,
			},
			asset: &assetdomain.Asset{
				IsRolledOut100: true,
				LaunchDate:     now.AddDate(0, -12, 0),
			},
			expected: false,
		},
		{
			name: "bug but no asset - returns false",
			task: &taskdomain.Task{
				Summary: "Fix critical bug",
				Type:    taskdomain.TaskTypeBug,
			},
			asset:    nil,
			expected: false,
		},
		{
			name: "bug but asset not rolled out - returns false",
			task: &taskdomain.Task{
				Summary: "Fix bug in development",
				Type:    taskdomain.TaskTypeBug,
			},
			asset: &assetdomain.Asset{
				IsRolledOut100: false,
				LaunchDate:     now.AddDate(0, -12, 0),
			},
			expected: false,
		},
		{
			name: "bug but zero launch date - returns false",
			task: &taskdomain.Task{
				Summary: "Fix production bug",
				Type:    taskdomain.TaskTypeBug,
			},
			asset: &assetdomain.Asset{
				IsRolledOut100: true,
				LaunchDate:     time.Time{}, // zero time
			},
			expected: false,
		},
		{
			name: "bug and asset launched more than 6 months ago - returns true",
			task: &taskdomain.Task{
				Summary: "Fix legacy bug",
				Type:    taskdomain.TaskTypeBug,
			},
			asset: &assetdomain.Asset{
				IsRolledOut100: true,
				LaunchDate:     now.AddDate(0, -12, 0), // 12 months ago
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.isBugOrFixPastRollout(tt.task, tt.asset)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function for testing
func containsLabel(labels []string, searchLabels []string) bool {
	for _, label := range labels {
		labelLower := strings.ToLower(label)
		for _, search := range searchLabels {
			if strings.Contains(labelLower, search) {
				return true
			}
		}
	}
	return false
}

// Additional edge case tests for complete coverage
func TestBusinessRulesClassifier_TaskMatchesAsset_AdditionalEdgeCases(t *testing.T) {
	assetRepo := new(MockAssetRepository)
	classifier := NewBusinessRulesClassifier(assetRepo)

	tests := []struct {
		name     string
		task     *taskdomain.Task
		asset    *assetdomain.Asset
		expected bool
	}{
		{
			name: "nil asset",
			task: &taskdomain.Task{
				Summary: "Some task",
			},
			asset:    nil,
			expected: false,
		},
		{
			name: "asset with empty keywords",
			task: &taskdomain.Task{
				Summary: "Payment Gateway maintenance task",
			},
			asset: &assetdomain.Asset{
				Name:     "Payment Gateway",
				Keywords: []string{},
			},
			expected: true, // should match on name
		},
		{
			name: "task with empty fields",
			task: &taskdomain.Task{
				Summary:     "",
				Description: "",
				Labels:      []string{},
			},
			asset: &assetdomain.Asset{
				Name:     "Payment Gateway",
				Keywords: []string{"payment"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifier.taskMatchesAsset(tt.task, tt.asset)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBusinessRulesClassifier_ClassifyTasks_WithActualError(t *testing.T) {
	assetRepo := new(MockAssetRepository)
	classifier := NewBusinessRulesClassifier(assetRepo)

	// Create a task that will trigger the API/inventory rule first, avoiding asset lookup
	// Then create a task that will need asset lookup and fail
	tasks := []*taskdomain.Task{
		{
			Key:     "SUCCESS-1",
			Summary: "Add new API endpoint", // This will succeed via API rule
		},
		{
			Key:     "SUCCESS-2",
			Summary: "Regular task", // This will succeed via fallback even with repo error
		},
	}

	// Test successful case
	t.Run("all tasks succeed", func(t *testing.T) {
		// For the second task, it will need asset lookup for non-API/inventory tasks
		assetRepo.On("FindAll").Return([]*assetdomain.Asset{}, nil).Once()

		classifications, err := classifier.ClassifyTasks(tasks)

		assert.NoError(t, err)
		assert.NotNil(t, classifications)
		assert.Len(t, classifications, 2)
		assert.Equal(t, taskdomain.WorkTypeDevelopment, classifications["SUCCESS-1"])
		assert.Equal(t, taskdomain.WorkTypeDevelopment, classifications["SUCCESS-2"])

		assetRepo.AssertExpectations(t)
	})
}
