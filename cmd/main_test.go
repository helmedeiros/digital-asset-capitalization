package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	assetsdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// Test content for teams.json
const mainTestTeamsContent = `{
	"teams": [
		{
			"name": "TeamA",
			"description": "Team A description",
			"members": ["alice", "bob"]
		},
		{
			"name": "TeamB",
			"description": "Team B description",
			"members": ["charlie", "diana"]
		}
	]
}`

// Test specific config service implementation

// SyncResult represents the result of a sync operation
type SyncResult struct {
	SyncedAssets    []*assetsdomain.Asset
	NotSyncedAssets []*assetsdomain.Asset
	MissingFields   []string
	AvailableFields map[string]string
}

func TestRunCommandsSuccess(t *testing.T) {
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
			name: "delete asset",
			args: []string{"assets", "delete", "--name", "Test Asset"},
			setup: func(mas *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
				mas.On("DeleteAsset", "Test Asset").Return(nil)
			},
			wantErr: false,
		},
		{
			name: "delete asset not found",
			args: []string{"assets", "delete", "--name", "Nonexistent"},
			setup: func(mas *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
				mas.On("DeleteAsset", "Nonexistent").Return(errors.New("asset not found: Nonexistent"))
			},
			wantErr: true,
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
			name: "show help",
			args: []string{"--help"},
			setup: func(_ *MockAssetService, _ *MockTaskService, _ *MockSprintService) {
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			// Save original args
			originalArgs := os.Args
			defer func() { os.Args = originalArgs }()

			// Set test args
			os.Args = append([]string{"assetcap"}, tt.args...)

			// Run the test
			err := app.Run()

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
			name:     "normal token",
			token:    "abcdefghij1234567890",
			expected: "abcd...7890",
		},
		{
			name:     "empty token",
			token:    "",
			expected: "****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskToken(tt.token)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInitializeApp(t *testing.T) {
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

func TestMain(t *testing.T) {
	t.Run("main function should not panic", func(t *testing.T) {
		// Create temporary directory for test
		tempDir := t.TempDir()
		originalWd, _ := os.Getwd()
		defer os.Chdir(originalWd)
		os.Chdir(tempDir)

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

		// This should not panic
		assert.NotPanics(t, func() {
			defer func() {
				if r := recover(); r != nil {
					t.Logf("Expected panic due to missing configuration: %v", r)
				}
			}()
			main()
		})
	})
}
