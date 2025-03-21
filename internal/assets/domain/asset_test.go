package domain

import (
	"testing"
	"time"
)

func TestNewAsset(t *testing.T) {
	tests := []struct {
		name        string
		assetName   string
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid asset",
			assetName:   "test-asset",
			description: "Test description",
			wantErr:     false,
		},
		{
			name:        "empty name",
			assetName:   "",
			description: "Test description",
			wantErr:     true,
			errMsg:      ErrEmptyName.Error(),
		},
		{
			name:        "empty description",
			assetName:   "test-asset",
			description: "",
			wantErr:     true,
			errMsg:      ErrEmptyDescription.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset, err := NewAsset(tt.assetName, tt.description)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if err.Error() != tt.errMsg {
					t.Errorf("Expected error %q, got %q", tt.errMsg, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if asset.Name != tt.assetName {
				t.Errorf("Expected name %q, got %q", tt.assetName, asset.Name)
			}
			if asset.Description != tt.description {
				t.Errorf("Expected description %q, got %q", tt.description, asset.Description)
			}
			if asset.ID == "" {
				t.Error("Expected non-empty ID")
			}
			if asset.Version != 1 {
				t.Errorf("Expected version 1, got %d", asset.Version)
			}
		})
	}
}

func TestUpdateDescription(t *testing.T) {
	asset, err := NewAsset("test-asset", "Initial description")
	if err != nil {
		t.Fatalf("Failed to create test asset: %v", err)
	}

	tests := []struct {
		name        string
		description string
		wantErr     bool
		errMsg      string
	}{
		{
			name:        "valid description",
			description: "Updated description",
			wantErr:     false,
		},
		{
			name:        "empty description",
			description: "",
			wantErr:     true,
			errMsg:      ErrEmptyDescription.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := asset.UpdateDescription(tt.description)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if err.Error() != tt.errMsg {
					t.Errorf("Expected error %q, got %q", tt.errMsg, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if asset.Description != tt.description {
				t.Errorf("Expected description %q, got %q", tt.description, asset.Description)
			}
			if asset.Version != 2 {
				t.Errorf("Expected version 2, got %d", asset.Version)
			}
		})
	}
}

func TestUpdateDocumentation(t *testing.T) {
	asset, err := NewAsset("test-asset", "Test description")
	if err != nil {
		t.Fatalf("Failed to create test asset: %v", err)
	}

	// Store initial time
	initialTime := asset.LastDocUpdateAt

	// Wait a bit to ensure time difference
	time.Sleep(time.Millisecond)

	// Update documentation
	asset.UpdateDocumentation()

	// Verify update
	if asset.LastDocUpdateAt.Before(initialTime) {
		t.Error("LastDocUpdateAt should be after initial time")
	}
	if asset.Version != 2 {
		t.Errorf("Expected version 2, got %d", asset.Version)
	}
}

func TestTaskCountOperations(t *testing.T) {
	asset, err := NewAsset("test-asset", "Test description")
	if err != nil {
		t.Fatalf("Failed to create test asset: %v", err)
	}

	// Test increment
	asset.IncrementTaskCount()
	if asset.AssociatedTaskCount != 1 {
		t.Errorf("Expected task count 1, got %d", asset.AssociatedTaskCount)
	}
	if asset.Version != 2 {
		t.Errorf("Expected version 2, got %d", asset.Version)
	}

	// Test decrement
	asset.DecrementTaskCount()
	if asset.AssociatedTaskCount != 0 {
		t.Errorf("Expected task count 0, got %d", asset.AssociatedTaskCount)
	}
	if asset.Version != 3 {
		t.Errorf("Expected version 3, got %d", asset.Version)
	}

	// Test decrement below zero
	asset.DecrementTaskCount()
	if asset.AssociatedTaskCount != 0 {
		t.Errorf("Expected task count 0, got %d", asset.AssociatedTaskCount)
	}
	if asset.Version != 3 {
		t.Errorf("Expected version 3, got %d", asset.Version)
	}
}

func TestGenerateID(t *testing.T) {
	// Test that IDs are unique
	id1 := generateID("test-asset")
	id2 := generateID("test-asset")
	if id1 == id2 {
		t.Error("Generated IDs should be unique")
	}

	// Test ID length
	if len(id1) != 16 {
		t.Errorf("Expected ID length 16, got %d", len(id1))
	}
}
