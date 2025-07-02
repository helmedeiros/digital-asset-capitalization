package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/application/usecase"
)

const testTeamsContent = `{"TEST": {"team": ["alice", "bob"]}}`

// MockConfigService is a mock implementation that wraps the InitializeConfig use case
type MockConfigService struct {
	mock.Mock
}

func (m *MockConfigService) InitializeConfig(interactive bool) (*usecase.InitializeConfigResult, error) {
	args := m.Called(interactive)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.InitializeConfigResult), args.Error(1)
}

func TestConfigCommands(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		setup   func(*MockAssetService, *MockTaskService, *MockSprintService, *MockConfigService)
		wantErr bool
	}{
		{
			name: "config init interactive mode",
			args: []string{"config", "init"},
			setup: func(_ *MockAssetService, _ *MockTaskService, _ *MockSprintService, mcs *MockConfigService) {
				result := &usecase.InitializeConfigResult{
					JiraConfigCreated: true,
					TeamConfigCreated: true,
					Message:           "Configuration initialized successfully",
				}
				mcs.On("InitializeConfig", true).Return(result, nil)
			},
			wantErr: false,
		},
		{
			name: "config init non-interactive mode",
			args: []string{"config", "init", "--non-interactive"},
			setup: func(_ *MockAssetService, _ *MockTaskService, _ *MockSprintService, mcs *MockConfigService) {
				result := &usecase.InitializeConfigResult{
					JiraConfigCreated: true,
					TeamConfigCreated: false,
					Message:           "Configuration initialized from environment variables",
				}
				mcs.On("InitializeConfig", false).Return(result, nil)
			},
			wantErr: false,
		},
		{
			name: "config init with jira flags (still interactive)",
			args: []string{"config", "init", "--jira-url", "https://test.atlassian.net", "--jira-email", "test@example.com", "--jira-token", "token123"},
			setup: func(_ *MockAssetService, _ *MockTaskService, _ *MockSprintService, mcs *MockConfigService) {
				result := &usecase.InitializeConfigResult{
					JiraConfigCreated: true,
					TeamConfigCreated: true,
					Message:           "Configuration initialized successfully",
				}
				mcs.On("InitializeConfig", true).Return(result, nil)
			},
			wantErr: false,
		},
		{
			name: "config init failure",
			args: []string{"config", "init"},
			setup: func(_ *MockAssetService, _ *MockTaskService, _ *MockSprintService, mcs *MockConfigService) {
				mcs.On("InitializeConfig", true).Return(nil, assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "config show",
			args: []string{"config", "show"},
			setup: func(_ *MockAssetService, _ *MockTaskService, _ *MockSprintService, _ *MockConfigService) {
				// For show command, we don't use the mock service - it directly reads environment/files
				// This is a simple command that doesn't require complex business logic
			},
			wantErr: false,
		},
		{
			name: "config validate",
			args: []string{"config", "validate"},
			setup: func(_ *MockAssetService, _ *MockTaskService, _ *MockSprintService, _ *MockConfigService) {
				// Set up environment variables for validation to succeed
				os.Setenv("JIRA_BASE_URL", "https://test.atlassian.net")
				os.Setenv("JIRA_EMAIL", "test@example.com")
				os.Setenv("JIRA_TOKEN", "test-token")

				// Create .assetcap directory and teams.json file
				assetcapDir := filepath.Join(".", ".assetcap")
				os.MkdirAll(assetcapDir, 0755)
				teamsPath := filepath.Join(assetcapDir, "teams.json")
				os.WriteFile(teamsPath, []byte(testTeamsContent), 0644)
			},
			wantErr: false,
		},
		{
			name: "config help",
			args: []string{"config", "--help"},
			setup: func(_ *MockAssetService, _ *MockTaskService, _ *MockSprintService, _ *MockConfigService) {
				// Help command doesn't need setup
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockAssetService := &MockAssetService{}
			mockTaskService := &MockTaskService{}
			mockSprintService := &MockSprintService{}
			mockConfigService := &MockConfigService{}

			// Store original env vars for config validate test cleanup
			var origVars map[string]string
			if tt.name == "config validate" {
				origVars = map[string]string{
					"JIRA_BASE_URL": os.Getenv("JIRA_BASE_URL"),
					"JIRA_EMAIL":    os.Getenv("JIRA_EMAIL"),
					"JIRA_TOKEN":    os.Getenv("JIRA_TOKEN"),
				}
			}

			// Setup mocks
			tt.setup(mockAssetService, mockTaskService, mockSprintService, mockConfigService)

			// Create app with config service for testing
			app := NewAppWithConfigService(mockAssetService, mockTaskService, mockSprintService, mockConfigService)

			// Capture output
			var buf bytes.Buffer
			originalOutput := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Run command
			originalArgs := os.Args
			os.Args = append([]string{"assetcap"}, tt.args...)

			err := app.Run()

			// Restore
			os.Args = originalArgs
			w.Close()
			os.Stdout = originalOutput

			// Read output
			buf.ReadFrom(r)
			output := buf.String()

			// Cleanup environment for config validate test
			if tt.name == "config validate" {
				// Restore original environment variables
				for key, value := range origVars {
					if value == "" {
						os.Unsetenv(key)
					} else {
						os.Setenv(key, value)
					}
				}
				// Clean up test files
				os.RemoveAll(".assetcap")
			}

			// Assertions
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify mock expectations
			mockConfigService.AssertExpectations(t)

			// Additional output assertions for specific commands
			switch tt.name {
			case "config init interactive mode":
				assert.Contains(t, output, "Configuration initialized successfully")
			case "config init non-interactive mode":
				assert.Contains(t, output, "Configuration initialized from environment variables")
			case "config help":
				assert.Contains(t, output, "config")
				assert.Contains(t, output, "init")
			}
		})
	}
}

func TestConfigInitFlags(t *testing.T) {
	t.Run("should parse jira flags correctly", func(t *testing.T) {
		// This test ensures our CLI flag parsing works correctly
		// We'll test this when we implement the actual command parsing
		assert.True(t, true) // Placeholder - will be implemented with actual command
	})
}

func TestConfigShowCommand(t *testing.T) {
	t.Run("should display current configuration", func(t *testing.T) {
		// Set up test environment
		cleanup := setupTestEnvironment(t)
		defer cleanup()

		// Set some environment variables for testing
		os.Setenv("JIRA_BASE_URL", "https://test.atlassian.net")
		os.Setenv("JIRA_EMAIL", "test@example.com")
		os.Setenv("JIRA_TOKEN", "hidden-token")

		// Create teams.json file in the correct location (.assetcap directory)
		assetcapDir := filepath.Join(".", ".assetcap")
		err := os.MkdirAll(assetcapDir, 0755)
		require.NoError(t, err)

		teamsPath := filepath.Join(assetcapDir, "teams.json")
		err = os.WriteFile(teamsPath, []byte(testTeamsContent), 0644)
		require.NoError(t, err)

		// Create app and test the actual config show command
		app := NewApp(nil, nil, nil) // For show command, we don't need services

		// Capture output
		var buf bytes.Buffer
		originalOutput := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run config show command
		originalArgs := os.Args
		os.Args = []string{"assetcap", "config", "show"}

		err = app.Run()

		// Restore
		os.Args = originalArgs
		w.Close()
		os.Stdout = originalOutput

		// Read output
		buf.ReadFrom(r)
		output := buf.String()

		// Verify the command executed successfully
		assert.NoError(t, err)
		assert.Contains(t, output, "Current Configuration:")
		assert.Contains(t, output, "JIRA_BASE_URL: https://test.atlassian.net")
		assert.Contains(t, output, "JIRA_EMAIL: test@example.com")
		assert.Contains(t, output, "JIRA_TOKEN: hidd...oken") // Masked token
		assert.Contains(t, output, ".assetcap/teams.json exists")
	})
}

func TestConfigValidateCommand(t *testing.T) {
	t.Run("should validate configuration successfully", func(t *testing.T) {
		cleanup := setupTestEnvironment(t)
		defer cleanup()

		// Set valid environment variables
		os.Setenv("JIRA_BASE_URL", "https://test.atlassian.net")
		os.Setenv("JIRA_EMAIL", "test@example.com")
		os.Setenv("JIRA_TOKEN", "valid-token")

		// Create valid teams.json file in the correct location (.assetcap directory)
		assetcapDir := filepath.Join(".", ".assetcap")
		err := os.MkdirAll(assetcapDir, 0755)
		require.NoError(t, err)

		teamsPath := filepath.Join(assetcapDir, "teams.json")
		err = os.WriteFile(teamsPath, []byte(testTeamsContent), 0644)
		require.NoError(t, err)

		// Create app and test the actual config validate command
		app := NewApp(nil, nil, nil)

		// Capture output
		var buf bytes.Buffer
		originalOutput := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run config validate command
		originalArgs := os.Args
		os.Args = []string{"assetcap", "config", "validate"}

		err = app.Run()

		// Restore
		os.Args = originalArgs
		w.Close()
		os.Stdout = originalOutput

		// Read output
		buf.ReadFrom(r)
		output := buf.String()

		// Verify successful validation
		assert.NoError(t, err)
		assert.Contains(t, output, "✅ Configuration is valid")
	})

	t.Run("should report validation errors", func(t *testing.T) {
		cleanup := setupTestEnvironment(t)
		defer cleanup()

		// Don't set environment variables to trigger validation errors
		// Clear any existing env vars
		os.Unsetenv("JIRA_BASE_URL")
		os.Unsetenv("JIRA_EMAIL")
		os.Unsetenv("JIRA_TOKEN")

		// Create app and test the actual config validate command
		app := NewApp(nil, nil, nil)

		// Capture output
		var buf bytes.Buffer
		originalOutput := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Run config validate command
		originalArgs := os.Args
		os.Args = []string{"assetcap", "config", "validate"}

		err := app.Run()

		// Restore
		os.Args = originalArgs
		w.Close()
		os.Stdout = originalOutput

		// Read output
		buf.ReadFrom(r)
		output := buf.String()

		// Verify validation failure
		assert.Error(t, err)
		assert.Contains(t, output, "❌ Configuration validation failed:")
		assert.Contains(t, output, "JIRA_BASE_URL is not set")
		assert.Contains(t, output, "JIRA_EMAIL is not set")
		assert.Contains(t, output, "JIRA_TOKEN is not set")
	})
}
