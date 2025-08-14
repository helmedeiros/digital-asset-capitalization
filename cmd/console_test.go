package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/urfave/cli/v2"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	investmentdomain "github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain"
	tasksdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

// Mock services for testing
// Mocks are now defined in mocks_test.go

type MockInvestmentService struct {
	mock.Mock
}

func (m *MockInvestmentService) CalculateAssetInvestment(ctx context.Context, asset, project string, sprints []string) (*investmentdomain.Investment, error) {
	args := m.Called(ctx, asset, project, sprints)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*investmentdomain.Investment), args.Error(1)
}

func (m *MockInvestmentService) ListInvestments(ctx context.Context, project string) ([]*investmentdomain.Investment, error) {
	args := m.Called(ctx, project)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*investmentdomain.Investment), args.Error(1)
}

func (m *MockInvestmentService) GetCostModel(ctx context.Context, project string) (*investmentdomain.CostModel, error) {
	args := m.Called(ctx, project)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*investmentdomain.CostModel), args.Error(1)
}

// MockConfigService is now defined in mocks_test.go

func createMockApp() *App {
	return &App{
		assetService:      &MockAssetService{},
		taskService:       &MockTaskService{},
		sprintService:     &MockSprintService{},
		investmentService: nil, // Not needed for these tests
		configService:     nil, // Not needed for these tests
	}
}

func TestCreateConsoleCommand(t *testing.T) {
	app := createMockApp()
	command := app.createConsoleCommand()

	assert.Equal(t, "console", command.Name)
	assert.Equal(t, []string{"c"}, command.Aliases)
	assert.Equal(t, "Start an interactive AI-powered console for AssetCap", command.Usage)
	assert.NotNil(t, command.Action)
	assert.Len(t, command.Flags, 4)

	// Test flags
	flagNames := []string{"ollama-url", "model", "max-sessions", "debug"}
	for _, flagName := range flagNames {
		found := false
		for _, flag := range command.Flags {
			if flag.Names()[0] == flagName {
				found = true
				break
			}
		}
		assert.True(t, found, "Flag %s not found", flagName)
	}
}

func TestCheckOllamaAvailability(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		serverFunc  func() *httptest.Server
		expectError bool
	}{
		{
			name:        "empty URL",
			baseURL:     "",
			expectError: true,
		},
		{
			name:        "invalid URL",
			baseURL:     "http://invalid-host:99999",
			expectError: true,
		},
		{
			name:    "valid response",
			baseURL: "", // Will be set by server
			serverFunc: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			},
			expectError: false,
		},
		{
			name:    "server error",
			baseURL: "", // Will be set by server
			serverFunc: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
			expectError: false, // checkOllamaAvailability doesn't check status code, just connectivity
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *httptest.Server
			if tt.serverFunc != nil {
				server = tt.serverFunc()
				defer server.Close()
				tt.baseURL = server.URL
			}

			err := checkOllamaAvailability(tt.baseURL)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test AssetServiceAdapter
func TestAssetServiceAdapter_ListAssets(t *testing.T) {
	mockService := &MockAssetService{}
	adapter := &AssetServiceAdapter{service: mockService}
	ctx := context.Background()

	tests := []struct {
		name        string
		setupMock   func()
		expectError bool
	}{
		{
			name: "successful list with assets",
			setupMock: func() {
				asset := &domain.Asset{
					ID:          "1",
					Name:        "Test Asset",
					Description: "Test Description",
					Status:      "active",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
				}
				mockService.On("ListAssets").Return([]*domain.Asset{asset}, nil)
			},
			expectError: false,
		},
		{
			name: "empty asset list",
			setupMock: func() {
				mockService.On("ListAssets").Return([]*domain.Asset{}, nil)
			},
			expectError: false,
		},
		{
			name: "service error",
			setupMock: func() {
				mockService.On("ListAssets").Return(nil, errors.New("service error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.ExpectedCalls = nil
			tt.setupMock()

			result, err := adapter.ListAssets(ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestAssetServiceAdapter_CreateAsset(t *testing.T) {
	mockService := &MockAssetService{}
	adapter := &AssetServiceAdapter{service: mockService}
	ctx := context.Background()

	tests := []struct {
		name        string
		assetName   string
		description string
		setupMock   func()
		expectError bool
	}{
		{
			name:        "empty name",
			assetName:   "",
			description: "Test Description",
			expectError: true,
		},
		{
			name:        "empty description",
			assetName:   "Test Asset",
			description: "",
			expectError: true,
		},
		{
			name:        "successful creation",
			assetName:   "Test Asset",
			description: "Test Description",
			setupMock: func() {
				mockService.On("CreateAsset", "Test Asset", "Test Description").Return(nil)
				asset := &domain.Asset{
					ID:          "1",
					Name:        "Test Asset",
					Description: "Test Description",
					CreatedAt:   time.Now(),
				}
				mockService.On("GetAsset", "Test Asset").Return(asset, nil)
			},
			expectError: false,
		},
		{
			name:        "creation error",
			assetName:   "Test Asset",
			description: "Test Description",
			setupMock: func() {
				mockService.On("CreateAsset", "Test Asset", "Test Description").Return(errors.New("creation failed"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.ExpectedCalls = nil
			if tt.setupMock != nil {
				tt.setupMock()
			}

			result, err := adapter.CreateAsset(ctx, tt.assetName, tt.description)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			if tt.setupMock != nil {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestAssetServiceAdapter_GetAsset(t *testing.T) {
	mockService := &MockAssetService{}
	adapter := &AssetServiceAdapter{service: mockService}
	ctx := context.Background()

	tests := []struct {
		name        string
		assetName   string
		setupMock   func()
		expectError bool
	}{
		{
			name:        "empty name",
			assetName:   "",
			expectError: true,
		},
		{
			name:      "successful get",
			assetName: "Test Asset",
			setupMock: func() {
				asset := &domain.Asset{
					ID:              "1",
					Name:            "Test Asset",
					Description:     "Test Description",
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
					LaunchDate:      time.Now(),
					LastDocUpdateAt: time.Now(),
				}
				mockService.On("GetAsset", "Test Asset").Return(asset, nil)
			},
			expectError: false,
		},
		{
			name:      "asset not found",
			assetName: "Nonexistent Asset",
			setupMock: func() {
				mockService.On("GetAsset", "Nonexistent Asset").Return(nil, errors.New("not found"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.ExpectedCalls = nil
			if tt.setupMock != nil {
				tt.setupMock()
			}

			result, err := adapter.GetAsset(ctx, tt.assetName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			if tt.setupMock != nil {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestAssetServiceAdapter_UpdateAsset(t *testing.T) {
	mockService := &MockAssetService{}
	adapter := &AssetServiceAdapter{service: mockService}
	ctx := context.Background()

	tests := []struct {
		name        string
		assetName   string
		description string
		setupMock   func()
		expectError bool
	}{
		{
			name:        "empty name",
			assetName:   "",
			description: "Test Description",
			expectError: true,
		},
		{
			name:        "empty description",
			assetName:   "Test Asset",
			description: "",
			expectError: true,
		},
		{
			name:        "successful update",
			assetName:   "Test Asset",
			description: "Updated Description",
			setupMock: func() {
				mockService.On("UpdateAsset", "Test Asset", "Updated Description", "", "", "", "").Return(nil)
				asset := &domain.Asset{
					ID:          "1",
					Name:        "Test Asset",
					Description: "Updated Description",
					UpdatedAt:   time.Now(),
				}
				mockService.On("GetAsset", "Test Asset").Return(asset, nil)
			},
			expectError: false,
		},
		{
			name:        "update error",
			assetName:   "Test Asset",
			description: "Updated Description",
			setupMock: func() {
				mockService.On("UpdateAsset", "Test Asset", "Updated Description", "", "", "", "").Return(errors.New("update failed"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.ExpectedCalls = nil
			if tt.setupMock != nil {
				tt.setupMock()
			}

			result, err := adapter.UpdateAsset(ctx, tt.assetName, tt.description)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			if tt.setupMock != nil {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestAssetServiceAdapter_DeleteAsset(t *testing.T) {
	mockService := &MockAssetService{}
	adapter := &AssetServiceAdapter{service: mockService}
	ctx := context.Background()

	tests := []struct {
		name        string
		assetName   string
		setupMock   func()
		expectError bool
	}{
		{
			name:        "empty name",
			assetName:   "",
			expectError: true,
		},
		{
			name:      "successful delete",
			assetName: "Test Asset",
			setupMock: func() {
				asset := &domain.Asset{Name: "Test Asset"}
				mockService.On("GetAsset", "Test Asset").Return(asset, nil)
				mockService.On("DeleteAsset", "Test Asset").Return(nil)
			},
			expectError: false,
		},
		{
			name:      "asset not found",
			assetName: "Nonexistent Asset",
			setupMock: func() {
				mockService.On("GetAsset", "Nonexistent Asset").Return(nil, errors.New("not found"))
			},
			expectError: true,
		},
		{
			name:      "delete error",
			assetName: "Test Asset",
			setupMock: func() {
				asset := &domain.Asset{Name: "Test Asset"}
				mockService.On("GetAsset", "Test Asset").Return(asset, nil)
				mockService.On("DeleteAsset", "Test Asset").Return(errors.New("delete failed"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.ExpectedCalls = nil
			if tt.setupMock != nil {
				tt.setupMock()
			}

			err := adapter.DeleteAsset(ctx, tt.assetName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.setupMock != nil {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestAssetServiceAdapter_SyncAssets(t *testing.T) {
	mockService := &MockAssetService{}
	adapter := &AssetServiceAdapter{service: mockService}
	ctx := context.Background()

	tests := []struct {
		name        string
		space       string
		label       string
		setupMock   func()
		expectError bool
	}{
		{
			name:        "empty label",
			space:       "TEST",
			label:       "",
			expectError: true,
		},
		{
			name:  "successful sync",
			space: "TEST",
			label: "cap-asset",
			setupMock: func() {
				syncResult := &domain.SyncResult{
					SyncedAssets: []*domain.Asset{
						{Name: "Asset 1"},
						{Name: "Asset 2"},
					},
					NotSyncedAssets: []*domain.NotSyncedAsset{
						{Name: "Asset 3", MissingFields: []string{"description"}},
					},
				}
				mockService.On("SyncFromConfluence", "TEST", "cap-asset", false).Return(syncResult, nil)
			},
			expectError: false,
		},
		{
			name:  "sync error",
			space: "TEST",
			label: "cap-asset",
			setupMock: func() {
				mockService.On("SyncFromConfluence", "TEST", "cap-asset", false).Return(nil, errors.New("sync failed"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.ExpectedCalls = nil
			if tt.setupMock != nil {
				tt.setupMock()
			}

			result, err := adapter.SyncAssets(ctx, tt.space, tt.label)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			if tt.setupMock != nil {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestAssetServiceAdapter_EnrichAsset(t *testing.T) {
	mockService := &MockAssetService{}
	adapter := &AssetServiceAdapter{service: mockService}
	ctx := context.Background()

	tests := []struct {
		name        string
		assetName   string
		field       string
		setupMock   func()
		expectError bool
	}{
		{
			name:        "empty name",
			assetName:   "",
			field:       "description",
			expectError: true,
		},
		{
			name:        "empty field",
			assetName:   "Test Asset",
			field:       "",
			expectError: true,
		},
		{
			name:        "unsupported field",
			assetName:   "Test Asset",
			field:       "unsupported",
			expectError: true,
		},
		{
			name:      "successful enrichment",
			assetName: "Test Asset",
			field:     "description",
			setupMock: func() {
				mockService.On("EnrichAsset", "Test Asset", "description").Return(nil)
				asset := &domain.Asset{
					Name:        "Test Asset",
					Description: "Enriched description",
				}
				mockService.On("GetAsset", "Test Asset").Return(asset, nil)
			},
			expectError: false,
		},
		{
			name:      "enrichment error",
			assetName: "Test Asset",
			field:     "description",
			setupMock: func() {
				mockService.On("EnrichAsset", "Test Asset", "description").Return(errors.New("enrichment failed"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.ExpectedCalls = nil
			if tt.setupMock != nil {
				tt.setupMock()
			}

			result, err := adapter.EnrichAsset(ctx, tt.assetName, tt.field)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			if tt.setupMock != nil {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestAssetServiceAdapter_GenerateKeywords(t *testing.T) {
	mockService := &MockAssetService{}
	adapter := &AssetServiceAdapter{service: mockService}
	ctx := context.Background()

	tests := []struct {
		name        string
		assetName   string
		setupMock   func()
		expectError bool
	}{
		{
			name:        "empty name",
			assetName:   "",
			expectError: true,
		},
		{
			name:      "successful keyword generation",
			assetName: "Test Asset",
			setupMock: func() {
				mockService.On("GenerateKeywords", "Test Asset").Return(nil)
				asset := &domain.Asset{
					Name:     "Test Asset",
					Keywords: []string{"test", "asset", "keyword"},
				}
				mockService.On("GetAsset", "Test Asset").Return(asset, nil)
			},
			expectError: false,
		},
		{
			name:      "keyword generation error",
			assetName: "Test Asset",
			setupMock: func() {
				mockService.On("GenerateKeywords", "Test Asset").Return(errors.New("generation failed"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.ExpectedCalls = nil
			if tt.setupMock != nil {
				tt.setupMock()
			}

			result, err := adapter.GenerateKeywords(ctx, tt.assetName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			if tt.setupMock != nil {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestRemoveDuplicates(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "no duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "all same",
			input:    []string{"a", "a", "a"},
			expected: []string{"a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeDuplicates(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test runConsole method with mocked CLI context
func TestApp_runConsole_OllamaUnavailable(t *testing.T) {
	app := createMockApp()

	// Create a CLI context that will fail the Ollama check
	flagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	flagSet.String("ollama-url", "http://invalid-host:99999", "")
	flagSet.String("model", "llama3", "")
	flagSet.Int("max-sessions", 10, "")
	flagSet.Bool("debug", false, "")

	c := cli.NewContext(nil, flagSet, nil)

	err := app.runConsole(c)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Ollama is not available")
}

// Test TaskServiceAdapter methods
func TestTaskServiceAdapter_FetchTasks(t *testing.T) {
	mockService := &MockTaskService{}
	adapter := &TaskServiceAdapter{service: mockService}
	ctx := context.Background()

	tests := []struct {
		name        string
		project     string
		sprint      string
		setupMock   func()
		expectError bool
	}{
		{
			name:        "empty project",
			project:     "",
			sprint:      "Sprint1",
			expectError: true,
		},
		{
			name:        "empty sprint",
			project:     "PROJ",
			sprint:      "",
			expectError: true,
		},
		{
			name:    "successful fetch",
			project: "PROJ",
			sprint:  "Sprint1",
			setupMock: func() {
				mockService.On("FetchTasks", ctx, "PROJ", "Sprint1", "jira").Return(nil)
			},
			expectError: false,
		},
		{
			name:    "service error",
			project: "PROJ",
			sprint:  "Sprint1",
			setupMock: func() {
				mockService.On("FetchTasks", ctx, "PROJ", "Sprint1", "jira").Return(errors.New("fetch failed"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.ExpectedCalls = nil
			if tt.setupMock != nil {
				tt.setupMock()
			}

			result, err := adapter.FetchTasks(ctx, tt.project, tt.sprint)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			if tt.setupMock != nil {
				mockService.AssertExpectations(t)
			}
		})
	}
}

func TestTaskServiceAdapter_ShowTasks(t *testing.T) {
	mockService := &MockTaskService{}
	adapter := &TaskServiceAdapter{service: mockService}
	ctx := context.Background()

	tests := []struct {
		name        string
		project     string
		sprint      string
		setupMock   func()
		expectError bool
	}{
		{
			name:        "empty project",
			project:     "",
			sprint:      "Sprint1",
			expectError: true,
		},
		{
			name:        "empty sprint",
			project:     "PROJ",
			sprint:      "",
			expectError: true,
		},
		{
			name:    "successful show with tasks",
			project: "PROJ",
			sprint:  "Sprint1",
			setupMock: func() {
				tasks := []*tasksdomain.Task{
					{Key: "PROJ-1", Summary: "Task 1"},
					{Key: "PROJ-2", Summary: "Task 2"},
				}
				mockService.On("GetTasks", ctx, "PROJ", "Sprint1").Return(tasks, nil)
			},
			expectError: false,
		},
		{
			name:    "no tasks found",
			project: "PROJ",
			sprint:  "Sprint1",
			setupMock: func() {
				mockService.On("GetTasks", ctx, "PROJ", "Sprint1").Return([]*tasksdomain.Task{}, nil)
			},
			expectError: false,
		},
		{
			name:    "service error",
			project: "PROJ",
			sprint:  "Sprint1",
			setupMock: func() {
				mockService.On("GetTasks", ctx, "PROJ", "Sprint1").Return(nil, errors.New("get tasks failed"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService.ExpectedCalls = nil
			if tt.setupMock != nil {
				tt.setupMock()
			}

			result, err := adapter.ShowTasks(ctx, tt.project, tt.sprint)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			if tt.setupMock != nil {
				mockService.AssertExpectations(t)
			}
		})
	}
}
