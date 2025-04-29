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
	sprintdomain "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	tasksdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	taskports "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

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
			setup: func(mas *MockAssetService, _ *MockTaskService, mss *MockSprintService) {
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
			setup: func(mas *MockAssetService, mts *MockTaskService, mss *MockSprintService) {
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
			setup: func(mas *MockAssetService, mts *MockTaskService, mss *MockSprintService) {
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
			setup: func(mas *MockAssetService, mts *MockTaskService, mss *MockSprintService) {
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
			setup: func(mas *MockAssetService, mts *MockTaskService, mss *MockSprintService) {
			},
			wantErr: false,
		},
		{
			name: "missing required flag",
			args: []string{"assets", "create"},
			setup: func(mas *MockAssetService, mts *MockTaskService, mss *MockSprintService) {
			},
			wantErr: true,
		},
		{
			name: "sprint allocate with required flags",
			args: []string{"sprint", "allocate", "--project", "TEST", "--sprint", "Sprint1"},
			setup: func(mas *MockAssetService, mts *MockTaskService, mss *MockSprintService) {
				mss.On("ProcessJiraIssues", "TEST", "Sprint1", "").Return("Allocation result", nil)
			},
			wantErr: false,
		},
		{
			name: "sprint allocate with override",
			args: []string{"sprint", "allocate", "--project", "TEST", "--sprint", "Sprint1", "--override", "{\"ISSUE-1\": 6}"},
			setup: func(mas *MockAssetService, mts *MockTaskService, mss *MockSprintService) {
				mss.On("ProcessJiraIssues", "TEST", "Sprint1", "{\"ISSUE-1\": 6}").Return("Allocation result", nil)
			},
			wantErr: false,
		},
		{
			name: "sprint allocate missing project",
			args: []string{"sprint", "allocate", "--sprint", "Sprint1", "--platform", "jira"},
			setup: func(mas *MockAssetService, mts *MockTaskService, mss *MockSprintService) {
			},
			wantErr: true,
		},
		{
			name: "sprint allocate missing sprint",
			args: []string{"sprint", "allocate", "--project", "TEST", "--platform", "jira"},
			setup: func(mas *MockAssetService, mts *MockTaskService, mss *MockSprintService) {
			},
			wantErr: true,
		},
		{
			name: "shell completion commands",
			args: []string{"completion", "bash"},
			setup: func(mas *MockAssetService, mts *MockTaskService, mss *MockSprintService) {
			},
			wantErr: false,
		},
		{
			name: "tasks classify with required flags",
			args: []string{"tasks", "classify", "--project", "TEST", "--sprint", "Sprint1", "--platform", "jira"},
			setup: func(mas *MockAssetService, mts *MockTaskService, mss *MockSprintService) {
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
			setup: func(mas *MockAssetService, mts *MockTaskService, mss *MockSprintService) {
			},
			wantErr: true,
		},
		{
			name: "tasks classify missing sprint",
			args: []string{"tasks", "classify", "--project", "TEST", "--platform", "jira"},
			setup: func(mas *MockAssetService, mts *MockTaskService, mss *MockSprintService) {
			},
			wantErr: true,
		},
		{
			name: "tasks classify missing platform",
			args: []string{"tasks", "classify", "--project", "TEST", "--sprint", "Sprint1"},
			setup: func(mas *MockAssetService, mts *MockTaskService, mss *MockSprintService) {
			},
			wantErr: true,
		},
		{
			name: "tasks show with asset option",
			args: []string{"tasks", "show", "--asset", "test"},
			setup: func(mas *MockAssetService, mts *MockTaskService, mss *MockSprintService) {
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
			setup: func(mas *MockAssetService, mts *MockTaskService, mss *MockSprintService) {
				mas.On("GetAsset", "nonexistent").Return(nil, fmt.Errorf("asset not found"))
			},
			wantErr: true,
		},
		{
			name: "show asset",
			args: []string{"assets", "show", "--name", "test"},
			setup: func(mas *MockAssetService, mts *MockTaskService, mss *MockSprintService) {
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
			setup: func(mas *MockAssetService, mts *MockTaskService, mss *MockSprintService) {
				mas.On("GetAsset", "nonexistent").Return(nil, fmt.Errorf("asset not found"))
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
