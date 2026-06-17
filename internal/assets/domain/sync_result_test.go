package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSyncResult(t *testing.T) {
	t.Parallel()
	t.Run("should create valid SyncResult", func(t *testing.T) {
		result := NewSyncResult()

		assert.NotNil(t, result, "SyncResult should not be nil")
		assert.NotNil(t, result.SyncedAssets, "SyncedAssets should be initialized")
		assert.NotNil(t, result.NotSyncedAssets, "NotSyncedAssets should be initialized")
	})

	t.Run("should initialize empty slices", func(t *testing.T) {
		result := NewSyncResult()

		assert.Empty(t, result.SyncedAssets, "SyncedAssets should be empty initially")
		assert.Empty(t, result.NotSyncedAssets, "NotSyncedAssets should be empty initially")
		assert.Len(t, result.SyncedAssets, 0, "SyncedAssets length should be 0")
		assert.Len(t, result.NotSyncedAssets, 0, "NotSyncedAssets length should be 0")
	})

	t.Run("should allow appending to initialized slices", func(t *testing.T) {
		result := NewSyncResult()

		// Create a test asset
		asset, err := NewAsset("Test Asset", "Test Description")
		assert.NoError(t, err)

		// Create a test not synced asset
		notSyncedAsset := &NotSyncedAsset{
			Name:            "Not Synced Asset",
			MissingFields:   []string{"description"},
			AvailableFields: map[string]string{"name": "Not Synced Asset"},
		}

		// Should be able to append
		result.SyncedAssets = append(result.SyncedAssets, asset)
		result.NotSyncedAssets = append(result.NotSyncedAssets, notSyncedAsset)

		assert.Len(t, result.SyncedAssets, 1, "Should have 1 synced asset")
		assert.Len(t, result.NotSyncedAssets, 1, "Should have 1 not synced asset")
		assert.Equal(t, asset, result.SyncedAssets[0], "Synced asset should match")
		assert.Equal(t, notSyncedAsset, result.NotSyncedAssets[0], "Not synced asset should match")
	})

	t.Run("should create separate instances", func(t *testing.T) {
		result1 := NewSyncResult()
		result2 := NewSyncResult()

		// They should be different instances
		assert.NotSame(t, result1, result2, "Should create different instances")

		// Slices should be independent - modifying one shouldn't affect the other
		asset1, err := NewAsset("Asset 1", "Description 1")
		assert.NoError(t, err)

		result1.SyncedAssets = append(result1.SyncedAssets, asset1)

		// result2 should still be empty
		assert.Len(t, result1.SyncedAssets, 1, "result1 should have 1 asset")
		assert.Len(t, result2.SyncedAssets, 0, "result2 should still be empty")
	})
}
