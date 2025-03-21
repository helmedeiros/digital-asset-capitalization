package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/helmedeiros/jira-time-allocator/assetcap/action"
)

func captureOutput(f func() error) (string, error) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)

	return buf.String(), err
}

func TestMain(t *testing.T) {
	// Save original args and restore them after test
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Save original assetManager and restore it after test
	origAssetManager := assetManager
	defer func() { assetManager = origAssetManager }()

	tests := []struct {
		name       string
		args       []string
		wantErr    bool
		wantOutput string
	}{
		{
			name:       "no args",
			args:       []string{"assetcap"},
			wantErr:    false,
			wantOutput: "NAME:",
		},
		{
			name:       "help command",
			args:       []string{"assetcap", "--help"},
			wantErr:    false,
			wantOutput: "NAME:",
		},
		{
			name:       "assets list empty",
			args:       []string{"assetcap", "assets", "list"},
			wantErr:    false,
			wantOutput: "No assets found",
		},
		{
			name:       "assets create missing name flag",
			args:       []string{"assetcap", "assets", "create", "--description", "Test description"},
			wantErr:    true,
			wantOutput: "",
		},
		{
			name:       "assets create missing description flag",
			args:       []string{"assetcap", "assets", "create", "--name", "test-asset"},
			wantErr:    true,
			wantOutput: "",
		},
		{
			name:       "assets create with required flags",
			args:       []string{"assetcap", "assets", "create", "--name", "test-asset", "--description", "Test description"},
			wantErr:    false,
			wantOutput: "Created asset: test-asset",
		},
		{
			name:       "assets list after creation",
			args:       []string{"assetcap", "assets", "list"},
			wantErr:    false,
			wantOutput: "test-asset",
		},
		{
			name:       "assets contribution-type add missing asset flag",
			args:       []string{"assetcap", "assets", "contribution-type", "add", "--type", "development"},
			wantErr:    true,
			wantOutput: "",
		},
		{
			name:       "assets contribution-type add missing type flag",
			args:       []string{"assetcap", "assets", "contribution-type", "add", "--asset", "test-asset"},
			wantErr:    true,
			wantOutput: "",
		},
		{
			name:       "assets contribution-type add with required flags",
			args:       []string{"assetcap", "assets", "contribution-type", "add", "--asset", "test-asset", "--type", "development"},
			wantErr:    false,
			wantOutput: "Added contribution type development to asset test-asset",
		},
		{
			name:       "assets documentation update missing asset flag",
			args:       []string{"assetcap", "assets", "documentation", "update"},
			wantErr:    true,
			wantOutput: "",
		},
		{
			name:       "assets documentation update with required flag",
			args:       []string{"assetcap", "assets", "documentation", "update", "--asset", "test-asset"},
			wantErr:    false,
			wantOutput: "Marked documentation as updated for asset test-asset",
		},
		{
			name:       "assets tasks increment missing asset flag",
			args:       []string{"assetcap", "assets", "tasks", "increment"},
			wantErr:    true,
			wantOutput: "",
		},
		{
			name:       "assets tasks increment with required flag",
			args:       []string{"assetcap", "assets", "tasks", "increment", "--asset", "test-asset"},
			wantErr:    false,
			wantOutput: "Incremented task count for asset test-asset",
		},
		{
			name:       "assets tasks decrement missing asset flag",
			args:       []string{"assetcap", "assets", "tasks", "decrement"},
			wantErr:    true,
			wantOutput: "",
		},
		{
			name:       "assets tasks decrement with required flag",
			args:       []string{"assetcap", "assets", "tasks", "decrement", "--asset", "test-asset"},
			wantErr:    false,
			wantOutput: "Decremented task count for asset test-asset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset assetManager for each test
			if strings.HasPrefix(tt.name, "assets list empty") {
				assetManager = action.NewAssetManager()
			}

			// Set up test args
			os.Args = tt.args

			// Capture output
			output, err := captureOutput(Run)

			// Check error
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			// Check output
			if tt.wantOutput != "" && !strings.Contains(output, tt.wantOutput) {
				t.Errorf("expected output to contain %q, got %q", tt.wantOutput, output)
			}
		})
	}
}
