package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	// Change working directory
	oldWd, err := os.Getwd()
	require.NoError(t, err, "Failed to get working directory")
	err = os.Chdir(testDir)
	require.NoError(t, err, "Failed to change working directory")

	// Initialize test asset service
	repo := infrastructure.NewJSONRepository(assetsDir, assetsFile)
	assetService = application.NewAssetService(repo)

	return func() {
		// Restore original stdout
		os.Stdout = oldStdout

		// Change back to original directory
		err := os.Chdir(oldWd)
		assert.NoError(t, err, "Failed to restore working directory")

		// Clean up test directory
		err = os.RemoveAll(testDir)
		assert.NoError(t, err, "Failed to clean up test directory")
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
			wantOut: "Assets:\n- test-asset: Test description\n",
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
			name:    "timeallocation-calc with required flags",
			args:    []string{"assetcap", "timeallocation-calc", "--project", "TEST", "--sprint", "Sprint 1"},
			wantErr: false,
		},
		{
			name:    "timeallocation-calc with override",
			args:    []string{"assetcap", "timeallocation-calc", "--project", "TEST", "--sprint", "Sprint 1", "--override", "{\"ISSUE-1\": 6}"},
			wantErr: false,
		},
		{
			name:    "timeallocation-calc missing project",
			args:    []string{"assetcap", "timeallocation-calc", "--sprint", "Sprint 1"},
			wantErr: true,
		},
		{
			name:    "shell completion commands",
			args:    []string{"assetcap", "completion", "bash"},
			wantErr: false,
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
