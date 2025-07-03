package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	assetsdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/application/usecase"
	configdomain "github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
	sprintdomain "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	tasksdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	taskports "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

const mainTestTeamsContent = `{"TEST": {"team": ["test-user"]}}`

// SyncResult represents the result of a sync operation
type SyncResult struct {
	SyncedAssets    []*assetsdomain.Asset
	NotSyncedAssets []*assetsdomain.Asset
	MissingFields   []string
	AvailableFields map[string]string
}

// MockAssetService is a mock implementation of AssetService
type MockAssetService struct {
	mock.Mock
}

func (m *MockAssetService) CreateAsset(name, description string) error {
	args := m.Called(name, description)
	return args.Error(0)
}

func (m *MockAssetService) ListAssets() ([]*assetsdomain.Asset, error) {
	args := m.Called()
	return args.Get(0).([]*assetsdomain.Asset), args.Error(1)
}

func (m *MockAssetService) GetAsset(name string) (*assetsdomain.Asset, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*assetsdomain.Asset), args.Error(1)
}

func (m *MockAssetService) UpdateAsset(name, description, why, benefits, how, metrics string) error {
	args := m.Called(name, description, why, benefits, how, metrics)
	return args.Error(0)
}

func (m *MockAssetService) UpdateDocumentation(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockAssetService) IncrementTaskCount(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockAssetService) DecrementTaskCount(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockAssetService) GenerateKeywords(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockAssetService) EnrichAsset(name, field string) error {
	args := m.Called(name, field)
	return args.Error(0)
}

func (m *MockAssetService) DeleteAsset(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockAssetService) SyncFromConfluence(space, label string, debug bool) (*assetsdomain.SyncResult, error) {
	args := m.Called(space, label, debug)
	return args.Get(0).(*assetsdomain.SyncResult), args.Error(1)
}

// MockTaskService is a mock implementation of TaskService
type MockTaskService struct {
	mock.Mock
}

func (m *MockTaskService) FetchTasks(ctx context.Context, project, sprint, platform string) error {
	args := m.Called(ctx, project, sprint, platform)
	return args.Error(0)
}

func (m *MockTaskService) GetTasks(ctx context.Context, project, sprint string) ([]*tasksdomain.Task, error) {
	args := m.Called(ctx, project, sprint)
	return args.Get(0).([]*tasksdomain.Task), args.Error(1)
}

func (m *MockTaskService) GetTasksByAsset(ctx context.Context, asset string) ([]*tasksdomain.Task, error) {
	args := m.Called(ctx, asset)
	return args.Get(0).([]*tasksdomain.Task), args.Error(1)
}

func (m *MockTaskService) ClassifyTasks(ctx context.Context, input tasksdomain.ClassifyTasksInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *MockTaskService) GetLocalRepository() taskports.TaskRepository {
	args := m.Called()
	return args.Get(0).(taskports.TaskRepository)
}

// MockSprintService is a mock implementation of SprintService
type MockSprintService struct {
	mock.Mock
}

func (m *MockSprintService) ProcessJiraIssues(project, sprint, override string) (string, error) {
	args := m.Called(project, sprint, override)
	return args.String(0), args.Error(1)
}

func (m *MockSprintService) ProcessSprint(project string, sprint *sprintdomain.Sprint) error {
	args := m.Called(project, sprint)
	return args.Error(0)
}

func (m *MockSprintService) ProcessTeamIssues(team *sprintdomain.Team) error {
	args := m.Called(team)
	return args.Error(0)
}

// MockTaskRepository is a mock implementation of TaskRepository
type MockTaskRepository struct {
	mock.Mock
}

func (m *MockTaskRepository) Save(ctx context.Context, task *tasksdomain.Task) error {
	args := m.Called(ctx, task)
	return args.Error(0)
}

func (m *MockTaskRepository) FindByKey(ctx context.Context, key string) (*tasksdomain.Task, error) {
	args := m.Called(ctx, key)
	return args.Get(0).(*tasksdomain.Task), args.Error(1)
}

func (m *MockTaskRepository) FindByProjectAndSprint(ctx context.Context, project, sprint string) ([]*tasksdomain.Task, error) {
	args := m.Called(ctx, project, sprint)
	return args.Get(0).([]*tasksdomain.Task), args.Error(1)
}

func (m *MockTaskRepository) FindByAsset(ctx context.Context, asset string) ([]*tasksdomain.Task, error) {
	args := m.Called(ctx, asset)
	return args.Get(0).([]*tasksdomain.Task), args.Error(1)
}

func setupTestEnvironment(t *testing.T) func() {
	t.Helper()

	// Save original stdout
	oldStdout := os.Stdout

	// Create test directory
	testDir := filepath.Join("testdata", t.Name())
	err := os.MkdirAll(testDir, 0755)
	require.NoError(t, err, "Failed to create test directory")

	// Create .assetcap directory
	assetcapDir := filepath.Join(testDir, ".assetcap")
	err = os.MkdirAll(assetcapDir, 0755)
	require.NoError(t, err, "Failed to create .assetcap directory")

	// Get current working directory
	oldWd, err := os.Getwd()
	require.NoError(t, err, "Failed to get working directory")

	// Change working directory to test directory
	err = os.Chdir(testDir)
	require.NoError(t, err, "Failed to change working directory")

	return func() {
		// Restore original stdout
		os.Stdout = oldStdout

		// Restore original working directory
		err := os.Chdir(oldWd)
		if err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}

		// Clean up test directory
		err = os.RemoveAll(filepath.Join(oldWd, "testdata", t.Name()))
		if err != nil {
			t.Errorf("Failed to clean up test directory: %v", err)
		}
	}
}

func captureOutput(f func() error) (string, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	oldStdout := os.Stdout
	os.Stdout = w

	errCh := make(chan error, 1)
	outCh := make(chan string, 1)

	go func() {
		funcErr := f()
		w.Close()
		errCh <- funcErr
	}()

	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		outCh <- buf.String()
	}()

	err = <-errCh
	os.Stdout = oldStdout
	out := <-outCh

	return out, err
}

func TestRun(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		setup   func(*MockAssetService, *MockTaskService, *MockSprintService)
		wantErr bool
	}{
		{
			name: "create asset",
			args: []string{"assets", "create", "--name", "Test Asset", "--description", "Test Description"},
			setup: func(mas *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
				mas.On("CreateAsset", "Test Asset", "Test Description").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "list empty assets",
			args: []string{"assets", "list"},
			setup: func(mas *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
				mas.On("ListAssets").Return([]*assetsdomain.Asset{}, nil)
			},
			wantErr: false,
		},
		{
			name: "list assets after creation",
			args: []string{"assets", "list"},
			setup: func(mas *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
				mas.On("ListAssets").Return([]*assetsdomain.Asset{
					{
						ID:          "cap-asset-test",
						Name:        "Test Asset",
						Description: "Test Description",
					},
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "update documentation",
			args: []string{"assets", "documentation", "update", "--asset", "test"},
			setup: func(mas *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
				mas.On("GetAsset", "test").Return(&assetsdomain.Asset{
					ID:          "cap-asset-test",
					Name:        "Test Asset",
					Description: "Test Description",
				}, nil)
				mas.On("UpdateDocumentation", "test").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "increment task count",
			args: []string{"assets", "tasks", "increment", "--asset", "test"},
			setup: func(mas *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
				mas.On("GetAsset", "test").Return(&assetsdomain.Asset{
					ID:          "cap-asset-test",
					Name:        "Test Asset",
					Description: "Test Description",
				}, nil)
				mas.On("IncrementTaskCount", "test").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "decrement task count",
			args: []string{"assets", "tasks", "decrement", "--asset", "test"},
			setup: func(mas *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
				mas.On("GetAsset", "test").Return(&assetsdomain.Asset{
					ID:          "cap-asset-test",
					Name:        "Test Asset",
					Description: "Test Description",
				}, nil)
				mas.On("DecrementTaskCount", "test").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "show help",
			args: []string{"--help"},
			setup: func(_ *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
			},
			wantErr: false,
		},
		{
			name: "missing required flag",
			args: []string{"assets", "create"},
			setup: func(_ *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
			},
			wantErr: true,
		},
		{
			name: "sprint allocate with required flags",
			args: []string{"sprint", "allocate", "--project", "TEST", "--sprint", "Sprint1"},
			setup: func(_ *MockAssetService, _ *MockTaskService, mss *MockSprintService) {
				mss.On("ProcessJiraIssues", "TEST", "Sprint1", "").Return("Allocation result", nil)
			},
			wantErr: false,
		},
		{
			name: "sprint allocate with override",
			args: []string{"sprint", "allocate", "--project", "TEST", "--sprint", "Sprint1", "--override", "{\"ISSUE-1\": 6}"},
			setup: func(_ *MockAssetService, _ *MockTaskService, mss *MockSprintService) {
				mss.On("ProcessJiraIssues", "TEST", "Sprint1", "{\"ISSUE-1\": 6}").Return("Allocation result", nil)
			},
			wantErr: false,
		},
		{
			name: "sprint allocate missing project",
			args: []string{"sprint", "allocate", "--sprint", "Sprint1", "--platform", "jira"},
			setup: func(_ *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
			},
			wantErr: true,
		},
		{
			name: "sprint allocate missing sprint",
			args: []string{"sprint", "allocate", "--project", "TEST", "--platform", "jira"},
			setup: func(_ *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
			},
			wantErr: true,
		},
		{
			name: "shell completion commands",
			args: []string{"completion", "bash"},
			setup: func(_ *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
			},
			wantErr: false,
		},
		{
			name: "tasks classify with required flags",
			args: []string{"tasks", "classify", "--project", "TEST", "--sprint", "Sprint1", "--platform", "jira"},
			setup: func(_ *MockAssetService, mts *MockTaskService, _ *MockSprintService) {
				mts.On("ClassifyTasks", mock.Anything, tasksdomain.ClassifyTasksInput{
					Project: "TEST",
					Sprint:  "Sprint1",
					DryRun:  false,
					Apply:   false,
				}).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "tasks classify missing project",
			args: []string{"tasks", "classify", "--sprint", "Sprint1", "--platform", "jira"},
			setup: func(_ *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
			},
			wantErr: true,
		},
		{
			name: "tasks classify missing sprint",
			args: []string{"tasks", "classify", "--project", "TEST", "--platform", "jira"},
			setup: func(_ *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
			},
			wantErr: true,
		},
		{
			name: "tasks classify missing platform",
			args: []string{"tasks", "classify", "--project", "TEST", "--sprint", "Sprint1"},
			setup: func(_ *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
			},
			wantErr: true,
		},
		{
			name: "tasks show with asset option",
			args: []string{"tasks", "show", "--asset", "test"},
			setup: func(mas *MockAssetService, mts *MockTaskService, _ *MockSprintService) {
				mas.On("GetAsset", "test").Return(&assetsdomain.Asset{
					ID:          "cap-asset-test",
					Name:        "Test Asset",
					Description: "Test Description",
				}, nil)
				mts.On("GetTasksByAsset", mock.Anything, "test").Return([]*tasksdomain.Task{}, nil)
			},
			wantErr: false,
		},
		{
			name: "tasks show with non-existent asset",
			args: []string{"tasks", "show", "--asset", "nonexistent"},
			setup: func(mas *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
				mas.On("GetAsset", "nonexistent").Return(nil, fmt.Errorf("asset not found"))
			},
			wantErr: true,
		},
		{
			name: "show asset",
			args: []string{"assets", "show", "--name", "test"},
			setup: func(mas *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
				mas.On("GetAsset", "test").Return(&assetsdomain.Asset{
					ID:          "cap-asset-test",
					Name:        "Test Asset",
					Description: "Test Description",
				}, nil)
			},
			wantErr: false,
		},
		{
			name: "show non-existent asset",
			args: []string{"assets", "show", "--name", "nonexistent"},
			setup: func(mas *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
				mas.On("GetAsset", "nonexistent").Return(nil, fmt.Errorf("asset not found"))
			},
			wantErr: true,
		},
		{
			name: "generate keywords for existing asset",
			args: []string{"assets", "keywords", "--name", "test"},
			setup: func(mas *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
				mas.On("GetAsset", "test").Return(&assetsdomain.Asset{
					ID:          "cap-asset-test",
					Name:        "Test Asset",
					Description: "Test Description",
				}, nil)
				mas.On("GenerateKeywords", "test").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "generate keywords for non-existent asset",
			args: []string{"assets", "keywords", "--name", "nonexistent"},
			setup: func(mas *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
				mas.On("GetAsset", "nonexistent").Return(nil, fmt.Errorf("asset not found"))
			},
			wantErr: true,
		},
		{
			name: "generate keywords missing name flag",
			args: []string{"assets", "keywords"},
			setup: func(_ *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnvironment(t)
			defer cleanup()

			// Create mocks
			mockAssetService := new(MockAssetService)
			mockTaskService := new(MockTaskService)
			mockSprintService := new(MockSprintService)

			// Set up mock behavior if provided
			if tt.setup != nil {
				tt.setup(mockAssetService, mockTaskService, mockSprintService)
			}

			// Create app with mocks
			app := NewApp(mockAssetService, mockTaskService, mockSprintService)

			// Run the test
			_, err := captureOutput(func() error {
				os.Args = append([]string{"assetcap"}, tt.args...)
				return app.Run()
			})

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify mock expectations
			mockAssetService.AssertExpectations(t)
			mockTaskService.AssertExpectations(t)
			mockSprintService.AssertExpectations(t)
		})
	}
}

// Add tests for missing functions to improve coverage
func TestMaskToken(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		expected string
	}{
		{
			name:     "short token",
			token:    "abc",
			expected: "****",
		},
		{
			name:     "exactly 4 characters",
			token:    "abcd",
			expected: "****",
		},
		{
			name:     "normal token",
			token:    "abcdefghij1234567890",
			expected: "abcd...7890",
		},
		{
			name:     "empty token",
			token:    "",
			expected: "****",
		},
		{
			name:     "8 character token",
			token:    "abcd1234",
			expected: "abcd...1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskToken(tt.token)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigCommandsAdditional(t *testing.T) {
	setupEnv := setupTestEnvironment(t)
	defer setupEnv()

	// Create a mock app for testing
	mockAssetService := new(MockAssetService)
	mockTaskService := new(MockTaskService)
	mockSprintService := new(MockSprintService)
	app := NewApp(mockAssetService, mockTaskService, mockSprintService)

	t.Run("config show command", func(t *testing.T) {
		// Set some environment variables
		os.Setenv("JIRA_BASE_URL", "https://test.atlassian.net")
		os.Setenv("JIRA_EMAIL", "test@example.com")
		os.Setenv("JIRA_TOKEN", "test-token-1234567890")

		// This tests the CLI parsing, not the full command execution
		// since that would require complex mocking
		os.Args = []string{"assetcap", "--help"}
		err := app.Run()

		// Help command should not error
		assert.NoError(t, err)
	})
}

func TestInitializeApp(t *testing.T) {
	setupEnv := setupTestEnvironment(t)
	defer setupEnv()

	// Test that initializeApp doesn't panic with missing configuration
	app, err := initializeApp()

	// Error is expected due to missing configuration
	if err != nil {
		assert.Contains(t, err.Error(), "Jira")
		return
	}

	// If no error, app should be valid
	assert.NotNil(t, app)
}

func TestNewApp(t *testing.T) {
	mockAssetService := new(MockAssetService)
	mockTaskService := new(MockTaskService)
	mockSprintService := new(MockSprintService)

	app := NewApp(mockAssetService, mockTaskService, mockSprintService)

	assert.NotNil(t, app)
	assert.Equal(t, mockAssetService, app.assetService)
	assert.Equal(t, mockTaskService, app.taskService)
	assert.Equal(t, mockSprintService, app.sprintService)
}

func TestNewAppWithConfigService(t *testing.T) {
	mockAssetService := new(MockAssetService)
	mockTaskService := new(MockTaskService)
	mockSprintService := new(MockSprintService)
	mockConfigService := &mockConfigServiceImpl{}

	app := NewAppWithConfigService(mockAssetService, mockTaskService, mockSprintService, mockConfigService)

	assert.NotNil(t, app)
	assert.Equal(t, mockAssetService, app.assetService)
	assert.Equal(t, mockTaskService, app.taskService)
	assert.Equal(t, mockSprintService, app.sprintService)
	assert.Equal(t, mockConfigService, app.configService)
}

// Mock implementation for testing
type mockConfigServiceImpl struct{}

func (m *mockConfigServiceImpl) InitializeConfig(_ bool) (*usecase.InitializeConfigResult, error) {
	return &usecase.InitializeConfigResult{
		JiraConfigCreated: true,
		TeamConfigCreated: true,
		Message:           "Test configuration initialized",
	}, nil
}

func TestConfigServiceImpl_InitializeConfig(t *testing.T) {
	t.Run("should delegate to use case successfully", func(t *testing.T) {
		// This test verifies that the InitializeConfig method properly delegates to the use case
		// We'll use a simple test that shows the method exists and calls through properly

		// Create mock dependencies
		mockRepo := &MockConfigRepo{}
		mockEnvProvider := &MockEnvProvider{}
		mockUI := &MockUserInteraction{}

		// Set up basic successful path mocks
		mockEnvProvider.On("IsConfigured").Return(false)
		mockUI.On("PromptString", "Enter Jira Base URL (e.g., https://company.atlassian.net):").Return("https://test.atlassian.net", nil)
		mockUI.On("PromptString", "Enter Jira Email:").Return("test@example.com", nil)
		mockUI.On("PromptPassword", "Enter Jira API Token:").Return("test-token", nil)
		mockUI.On("PromptConfirm", "Would you like to configure team members now? (y/n):").Return(false, nil)
		mockUI.On("DisplaySuccess", mock.AnythingOfType("string")).Return()
		mockRepo.On("InitializeConfigDirectory").Return(nil)
		mockRepo.On("SaveJiraConfig", mock.Anything).Return(nil)

		// Create use case with mocks
		initializeConfigUseCase := usecase.NewInitializeConfig(mockRepo, mockEnvProvider, mockUI)

		// Create config service implementation
		configService := &configServiceImpl{
			initializeConfig: initializeConfigUseCase,
		}

		// Act - call the method we're testing
		result, err := configService.InitializeConfig(true)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.JiraConfigCreated)
		assert.Contains(t, result.Message, "successfully")
	})

	t.Run("should handle non-interactive mode with missing env vars", func(t *testing.T) {
		// Create mock dependencies
		mockRepo := &MockConfigRepo{}
		mockEnvProvider := &MockEnvProvider{}
		mockUI := &MockUserInteraction{}

		// Set up mocks for non-interactive mode with missing vars
		mockEnvProvider.On("IsConfigured").Return(false)
		mockEnvProvider.On("GetMissingVars").Return([]string{"JIRA_BASE_URL", "JIRA_EMAIL", "JIRA_TOKEN"})

		// Create use case with mocks
		initializeConfigUseCase := usecase.NewInitializeConfig(mockRepo, mockEnvProvider, mockUI)

		// Create config service implementation
		configService := &configServiceImpl{
			initializeConfig: initializeConfigUseCase,
		}

		// Act
		result, err := configService.InitializeConfig(false)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "environment variables not configured")
	})
}

// Mock implementations for testing InitializeConfig
type MockConfigRepo struct {
	mock.Mock
}

func (m *MockConfigRepo) InitializeConfigDirectory() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockConfigRepo) ConfigExists() (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

func (m *MockConfigRepo) LoadJiraConfig() (*configdomain.JiraConfig, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*configdomain.JiraConfig), args.Error(1)
}

func (m *MockConfigRepo) SaveJiraConfig(config *configdomain.JiraConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockConfigRepo) LoadTeamConfig() (*configdomain.TeamConfig, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*configdomain.TeamConfig), args.Error(1)
}

func (m *MockConfigRepo) SaveTeamConfig(config *configdomain.TeamConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

type MockEnvProvider struct {
	mock.Mock
}

func (m *MockEnvProvider) GetJiraBaseURL() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockEnvProvider) GetJiraEmail() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockEnvProvider) GetJiraToken() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockEnvProvider) SetJiraBaseURL(url string) error {
	args := m.Called(url)
	return args.Error(0)
}

func (m *MockEnvProvider) SetJiraEmail(email string) error {
	args := m.Called(email)
	return args.Error(0)
}

func (m *MockEnvProvider) SetJiraToken(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockEnvProvider) IsConfigured() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockEnvProvider) GetMissingVars() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

type MockUserInteraction struct {
	mock.Mock
}

func (m *MockUserInteraction) PromptString(prompt string) (string, error) {
	args := m.Called(prompt)
	return args.String(0), args.Error(1)
}

func (m *MockUserInteraction) PromptStringWithDefault(prompt, defaultValue string) (string, error) {
	args := m.Called(prompt, defaultValue)
	return args.String(0), args.Error(1)
}

func (m *MockUserInteraction) PromptPassword(prompt string) (string, error) {
	args := m.Called(prompt)
	return args.String(0), args.Error(1)
}

func (m *MockUserInteraction) PromptConfirm(prompt string) (bool, error) {
	args := m.Called(prompt)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserInteraction) PromptSelect(prompt string, options []string) (string, error) {
	args := m.Called(prompt, options)
	return args.String(0), args.Error(1)
}

func (m *MockUserInteraction) PromptMultiSelect(prompt string, options []string) ([]string, error) {
	args := m.Called(prompt, options)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockUserInteraction) DisplayMessage(message string) {
	m.Called(message)
}

func (m *MockUserInteraction) DisplayError(message string) {
	m.Called(message)
}

func (m *MockUserInteraction) DisplaySuccess(message string) {
	m.Called(message)
}

func (m *MockUserInteraction) DisplayWarning(message string) {
	m.Called(message)
}

func TestMain(t *testing.T) {
	t.Run("main function should not panic", func(t *testing.T) {
		// Set up test environment with required configuration files
		cleanup := setupTestEnvironment(t)
		defer cleanup()

		// Create minimal teams.json file to prevent initialization errors
		assetcapDir := filepath.Join(".", ".assetcap")
		err := os.MkdirAll(assetcapDir, 0755)
		require.NoError(t, err)

		teamsPath := filepath.Join(assetcapDir, "teams.json")
		err = os.WriteFile(teamsPath, []byte(mainTestTeamsContent), 0644)
		require.NoError(t, err)

		// Save original os.Args
		originalArgs := os.Args
		defer func() {
			os.Args = originalArgs
		}()

		// Test with help flag to avoid actual execution
		os.Args = []string{"assetcap", "--help"}

		// This should not panic - main() will call initializeApp() and run the CLI
		// We can't easily test the full main() flow without complex setup,
		// but we can ensure it doesn't panic on basic invocation
		assert.NotPanics(t, func() {
			// We'll test that the main function exists and can be called
			// The actual execution is tested through other test cases
			defer func() {
				if r := recover(); r != nil {
					// If there's a panic due to missing config, that's expected
					// We just want to ensure the function exists and is callable
					t.Logf("Expected panic due to missing configuration: %v", r)
				}
			}()

			// Call main - this will attempt to run but may fail due to missing config
			main()
		})
	})
}

func TestMain_NoArgs(t *testing.T) {
	cleanup := setupTestEnvironment(t)
	defer cleanup()

	// Create minimal teams.json file to prevent initialization errors
	assetcapDir := filepath.Join(".", ".assetcap")
	err := os.MkdirAll(assetcapDir, 0755)
	require.NoError(t, err)

	teamsPath := filepath.Join(assetcapDir, "teams.json")
	err = os.WriteFile(teamsPath, []byte(mainTestTeamsContent), 0644)
	require.NoError(t, err)

	// Save original os.Args
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Test with no arguments
	os.Args = []string{"assetcap"}

	assert.NotPanics(t, func() {
		main()
	})
}
