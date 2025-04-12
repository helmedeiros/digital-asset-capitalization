package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	assetsapp "github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure"
	tasksapp "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/application/usecase/testutil"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

const testAssetsFile = "assets.json"

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

	// Create teams.json in .assetcap directory
	teamsFilePath := filepath.Join(assetcapDir, "teams.json")
	teamsData := []byte(`{
		"TEST": {
			"team": ["Test User 1", "Test User 2"]
		},
		"FN": {
			"team": ["helio.medeiros", "julio.medeiros"]
		}
	}`)

	err = os.WriteFile(teamsFilePath, teamsData, 0644)
	require.NoError(t, err, "Failed to write teams.json")

	// Change working directory to test directory
	err = os.Chdir(testDir)
	require.NoError(t, err, "Failed to change working directory")

	// Initialize repositories
	config := infrastructure.RepositoryConfig{
		Directory: assetsDir,
		Filename:  assetsFile,
		FileMode:  0644,
		DirMode:   0755,
	}
	assetRepo := infrastructure.NewJSONRepository(config)
	assetService = assetsapp.NewAssetService(assetRepo)

	// Initialize task service with mock dependencies
	jiraRepo := testutil.NewMockTaskRepository()
	localRepo := testutil.NewMockTaskRepository()
	classifier := testutil.NewMockTaskClassifier()
	userInput := testutil.NewMockUserInput()

	// Set up mock behavior for task classification
	classifier.SetClassifyTasksFunc(func(tasks []*domain.Task) (map[string]domain.WorkType, error) {
		workTypes := make(map[string]domain.WorkType)
		for _, task := range tasks {
			workTypes[task.Key] = domain.WorkTypeDevelopment
		}
		return workTypes, nil
	})

	// Set up mock behavior for user input
	userInput.SetConfirmFunc(func(prompt string, args ...interface{}) (bool, error) {
		return true, nil
	})

	// Set up mock behavior for repositories
	jiraRepo.SetFindByProjectAndSprintFunc(func(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
		return []*domain.Task{
			{
				Key:     "TEST-1",
				Type:    "Story",
				Summary: "Test Task",
				Status:  "In Progress",
				Sprint:  "Sprint 1",
			},
		}, nil
	})

	localRepo.SetSaveFunc(func(ctx context.Context, task *domain.Task) error {
		return nil
	})

	taskService = tasksapp.NewTasksService(jiraRepo, localRepo, classifier, userInput)

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
		err := f()
		w.Close()
		errCh <- err
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
		wantErr bool
		wantOut string
		setup   func() error
	}{
		{
			name:    "create asset",
			args:    []string{"assetcap", "assets", "create", "--name", "test-asset", "--description", "Test description"},
			wantErr: false,
			wantOut: "Created asset: test-asset\n",
		},
		{
			name:    "list empty assets",
			args:    []string{"assetcap", "assets", "list"},
			wantErr: false,
			wantOut: "No assets found\n",
		},
		{
			name:    "list assets after creation",
			args:    []string{"assetcap", "assets", "list"},
			wantErr: false,
			wantOut: "Assets:\n- test-asset:\n  Description: Test description\n  Why: \n  Benefits: \n  How: \n  Metrics: \n\n",
			setup: func() error {
				return assetService.CreateAsset("test-asset", "Test description")
			},
		},
		{
			name:    "update documentation",
			args:    []string{"assetcap", "assets", "documentation", "update", "--asset", "test-asset"},
			wantErr: false,
			wantOut: "Marked documentation as updated for asset test-asset\n",
			setup: func() error {
				return assetService.CreateAsset("test-asset", "Test description")
			},
		},
		{
			name:    "increment task count",
			args:    []string{"assetcap", "assets", "tasks", "increment", "--asset", "test-asset"},
			wantErr: false,
			wantOut: "Incremented task count for asset test-asset\n",
			setup: func() error {
				return assetService.CreateAsset("test-asset", "Test description")
			},
		},
		{
			name:    "decrement task count",
			args:    []string{"assetcap", "assets", "tasks", "decrement", "--asset", "test-asset"},
			wantErr: false,
			wantOut: "Decremented task count for asset test-asset\n",
			setup: func() error {
				if err := assetService.CreateAsset("test-asset", "Test description"); err != nil {
					return err
				}
				return assetService.IncrementTaskCount("test-asset")
			},
		},
		{
			name:    "show help",
			args:    []string{"assetcap", "--help"},
			wantErr: false,
		},
		{
			name:    "missing required flag",
			args:    []string{"assetcap", "assets", "create", "--name", "test-asset"},
			wantErr: true,
			wantOut: "",
		},
		{
			name:    "sprint allocate with required flags",
			args:    []string{"assetcap", "sprint", "allocate", "--project", "FN", "--sprint", "Sprint 1"},
			wantErr: false,
		},
		{
			name:    "sprint allocate with override",
			args:    []string{"assetcap", "sprint", "allocate", "--project", "FN", "--sprint", "Sprint 1", "--override", "{\"ISSUE-1\": 6}"},
			wantErr: false,
		},
		{
			name:    "sprint allocate missing project",
			args:    []string{"assetcap", "sprint", "allocate", "--sprint", "Sprint 1"},
			wantErr: true,
		},
		{
			name:    "sprint allocate missing sprint",
			args:    []string{"assetcap", "sprint", "allocate", "--project", "FN"},
			wantErr: true,
		},
		{
			name:    "shell completion commands",
			args:    []string{"assetcap", "completion", "bash"},
			wantErr: false,
		},
		{
			name:    "tasks classify with required flags",
			args:    []string{"assetcap", "tasks", "classify", "--project", "FN", "--sprint", "Sprint 1", "--platform", "jira"},
			wantErr: false,
			wantOut: "Successfully classified tasks for project FN, sprint Sprint 1 from jira\n",
		},
		{
			name:    "tasks classify missing project",
			args:    []string{"assetcap", "tasks", "classify", "--sprint", "Sprint 1", "--platform", "jira"},
			wantErr: true,
		},
		{
			name:    "tasks classify missing sprint",
			args:    []string{"assetcap", "tasks", "classify", "--project", "FN", "--platform", "jira"},
			wantErr: true,
		},
		{
			name:    "tasks classify missing platform",
			args:    []string{"assetcap", "tasks", "classify", "--project", "FN", "--sprint", "Sprint 1"},
			wantErr: true,
		},
		{
			name:    "tasks show with asset option",
			args:    []string{"assetcap", "tasks", "show", "--asset", "test-asset"},
			wantErr: false,
			wantOut: "Tasks for asset test-asset:\n" +
				"----------------------------------------\n" +
				"No tasks found\n",
			setup: func() error {
				if err := assetService.CreateAsset("test-asset", "Test description"); err != nil {
					return err
				}
				task := &domain.Task{
					Key:     "TEST-1",
					Type:    "Story",
					Summary: "Test Task",
					Status:  "In Progress",
					Labels:  []string{"cap-asset-test-asset"},
				}
				return taskService.GetLocalRepository().Save(context.Background(), task)
			},
		},
		{
			name:    "tasks show with non-existent asset",
			args:    []string{"assetcap", "tasks", "show", "--asset", "non-existent"},
			wantErr: true,
			wantOut: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTestEnvironment(t)
			defer cleanup()

			if tt.setup != nil {
				err := tt.setup()
				require.NoError(t, err, "Setup failed")
			}

			os.Args = tt.args
			out, err := captureOutput(Run)

			if tt.wantErr {
				assert.Error(t, err, "Expected error but got none")
			} else {
				assert.NoError(t, err, "Unexpected error")
			}

			if tt.wantOut != "" {
				assert.Equal(t, tt.wantOut, out, "Output mismatch")
			}
		})
	}

	t.Run("show asset", func(t *testing.T) {
		cleanup := setupTestEnvironment(t)
		defer cleanup()

		// Create a test asset first
		output, err := captureOutput(func() error {
			os.Args = []string{"assetcap", "assets", "create", "--name", "test-asset", "--description", "Test description"}
			return Run()
		})
		require.NoError(t, err, "Failed to create test asset")
		assert.Contains(t, output, "Created asset: test-asset", "Expected output to contain 'Created asset: test-asset'")

		// Test showing the asset
		output, err = captureOutput(func() error {
			os.Args = []string{"assetcap", "assets", "show", "--name", "test-asset"}
			return Run()
		})
		require.NoError(t, err, "Failed to show asset")
		expectedStrings := []string{
			"Asset: test-asset",
			"Description: Test description",
			"Task Count: 0",
			"Created:",
			"Updated:",
		}
		for _, expected := range expectedStrings {
			assert.Contains(t, output, expected, "Expected output to contain %q", expected)
		}
	})

	t.Run("show non-existent asset", func(t *testing.T) {
		cleanup := setupTestEnvironment(t)
		defer cleanup()

		output, err := captureOutput(func() error {
			os.Args = []string{"assetcap", "assets", "show", "--name", "non-existent"}
			return Run()
		})
		assert.Error(t, err, "Expected error for non-existent asset")
		assert.Empty(t, output, "Expected empty output for non-existent asset")
	})
}
