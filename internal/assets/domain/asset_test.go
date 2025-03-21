package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				require.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.assetName, asset.Name)
			assert.Equal(t, tt.description, asset.Description)
			assert.NotEmpty(t, asset.ID, "Expected non-empty ID")
			assert.Equal(t, 1, asset.Version)
		})
	}
}

func TestUpdateDescription(t *testing.T) {
	asset, err := NewAsset("test-asset", "Initial description")
	require.NoError(t, err)

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
				require.Error(t, err)
				assert.Equal(t, tt.errMsg, err.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.description, asset.Description)
			assert.Equal(t, 2, asset.Version)
		})
	}
}

func TestUpdateDocumentation(t *testing.T) {
	asset, err := NewAsset("test-asset", "Test description")
	require.NoError(t, err)

	// Store initial time
	initialTime := asset.LastDocUpdateAt

	// Wait a bit to ensure time difference
	time.Sleep(time.Millisecond)

	// Update documentation
	asset.UpdateDocumentation()

	// Verify update
	assert.True(t, asset.LastDocUpdateAt.After(initialTime), "LastDocUpdateAt should be after initial time")
	assert.Equal(t, 2, asset.Version)
}

func TestTaskCountOperations(t *testing.T) {
	asset, err := NewAsset("test-asset", "Test description")
	require.NoError(t, err)

	// Test increment
	asset.IncrementTaskCount()
	assert.Equal(t, 1, asset.AssociatedTaskCount)
	assert.Equal(t, 2, asset.Version)

	// Test decrement
	asset.DecrementTaskCount()
	assert.Equal(t, 0, asset.AssociatedTaskCount)
	assert.Equal(t, 3, asset.Version)

	// Test decrement below zero
	asset.DecrementTaskCount()
	assert.Equal(t, 0, asset.AssociatedTaskCount)
	assert.Equal(t, 3, asset.Version)
}

func TestGenerateID(t *testing.T) {
	// Test that IDs are unique
	id1 := generateID("test-asset")
	id2 := generateID("test-asset")
	assert.NotEqual(t, id1, id2, "Generated IDs should be unique")

	// Test ID length
	assert.Len(t, id1, 16, "Expected ID length 16")
}
