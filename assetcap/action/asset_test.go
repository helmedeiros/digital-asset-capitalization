package action

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAssetsFile = "test_assets.json"
const testAssetsDir = "testdata"

func cleanupTestAssets() {
	os.Remove(filepath.Join(testAssetsDir, testAssetsFile))
}

func TestAssetManager(t *testing.T) {
	// Clean up test assets before and after tests
	cleanupTestAssets()
	defer cleanupTestAssets()

	am := NewAssetManager(testAssetsDir, testAssetsFile)

	t.Run("CreateAsset", func(t *testing.T) {
		// Test successful creation
		err := am.CreateAsset("test-asset", "Test description")
		require.NoError(t, err, "CreateAsset should succeed")

		// Test duplicate creation
		err = am.CreateAsset("test-asset", "Another description")
		assert.Error(t, err, "Expected error for duplicate asset creation")
		assert.Equal(t, "asset test-asset already exists", err.Error(), "Expected 'already exists' error")

		// Test empty name
		err = am.CreateAsset("", "Description")
		assert.Error(t, err, "Expected error for empty name")

		// Test empty description
		err = am.CreateAsset("new-asset", "")
		assert.Error(t, err, "Expected error for empty description")
	})

	t.Run("AddContributionType", func(t *testing.T) {
		// Test adding to non-existent asset
		err := am.AddContributionType("non-existent", "development")
		assert.Error(t, err, "Expected error for non-existent asset")
		assert.Equal(t, "asset non-existent not found", err.Error(), "Expected 'not found' error")

		// Test adding valid contribution type
		err = am.AddContributionType("test-asset", "development")
		require.NoError(t, err, "AddContributionType should succeed")

		// Test adding invalid contribution type
		err = am.AddContributionType("test-asset", "invalid-type")
		assert.Error(t, err, "Expected error for invalid contribution type")

		// Test adding duplicate contribution type
		err = am.AddContributionType("test-asset", "development")
		assert.Error(t, err, "Expected error for duplicate contribution type")
	})

	t.Run("ListAssets", func(t *testing.T) {
		assets := am.ListAssets()
		assert.Contains(t, assets, "test-asset", "Expected to find 'test-asset' in list")
	})

	t.Run("GetAsset", func(t *testing.T) {
		// Test getting existing asset
		asset, err := am.GetAsset("test-asset")
		require.NoError(t, err, "GetAsset should succeed")
		require.NotNil(t, asset, "Expected non-nil asset")
		assert.Equal(t, "test-asset", asset.Name, "Expected asset name 'test-asset'")

		// Test getting non-existent asset
		asset, err = am.GetAsset("non-existent")
		assert.Error(t, err, "Expected error for non-existent asset")
		assert.Nil(t, asset, "Expected nil asset")
	})

	t.Run("UpdateDocumentation", func(t *testing.T) {
		// Test updating non-existent asset
		err := am.UpdateDocumentation("non-existent")
		assert.Error(t, err, "Expected error for non-existent asset")

		// Test updating existing asset
		err = am.UpdateDocumentation("test-asset")
		require.NoError(t, err, "UpdateDocumentation should succeed")

		// Verify update
		asset, err := am.GetAsset("test-asset")
		require.NoError(t, err, "GetAsset should succeed")
		require.NotNil(t, asset, "Expected non-nil asset")
	})

	t.Run("TaskCountOperations", func(t *testing.T) {
		// Test incrementing non-existent asset
		err := am.IncrementTaskCount("non-existent")
		assert.Error(t, err, "Expected error for non-existent asset")

		// Test incrementing existing asset
		err = am.IncrementTaskCount("test-asset")
		require.NoError(t, err, "IncrementTaskCount should succeed")

		// Verify increment
		asset, err := am.GetAsset("test-asset")
		require.NoError(t, err, "GetAsset should succeed")
		assert.Equal(t, 1, asset.AssociatedTaskCount, "Expected task count 1")

		// Test decrementing
		err = am.DecrementTaskCount("test-asset")
		require.NoError(t, err, "DecrementTaskCount should succeed")

		// Verify decrement
		asset, err = am.GetAsset("test-asset")
		require.NoError(t, err, "GetAsset should succeed")
		assert.Equal(t, 0, asset.AssociatedTaskCount, "Expected task count 0")

		// Test decrementing below zero
		err = am.DecrementTaskCount("test-asset")
		require.NoError(t, err, "DecrementTaskCount should succeed")
		asset, err = am.GetAsset("test-asset")
		require.NoError(t, err, "GetAsset should succeed")
		assert.Equal(t, 0, asset.AssociatedTaskCount, "Expected task count 0")
	})

	t.Run("FormatAssetList", func(t *testing.T) {
		// Test empty list
		emptyList := FormatAssetList([]string{})
		assert.Equal(t, "No assets found", emptyList, "Expected 'No assets found'")

		// Test with assets
		assets := []string{"asset1", "asset2"}
		list := FormatAssetList(assets)
		assert.Contains(t, list, "asset1", "Expected list to contain 'asset1'")
		assert.Contains(t, list, "asset2", "Expected list to contain 'asset2'")
	})
}

func TestAssetManagerConcurrent(t *testing.T) {
	// Clean up test assets before and after tests
	cleanupTestAssets()
	defer cleanupTestAssets()

	am := NewAssetManager(testAssetsDir, testAssetsFile)

	// Create a test asset
	err := am.CreateAsset("concurrent-asset", "Test description")
	require.NoError(t, err, "CreateAsset should succeed")

	// Test concurrent operations
	t.Run("ConcurrentOperations", func(t *testing.T) {
		done := make(chan bool)
		concurrentOps := 100

		// Launch concurrent goroutines
		for i := 0; i < concurrentOps; i++ {
			go func() {
				// Mix of operations
				am.IncrementTaskCount("concurrent-asset")
				am.AddContributionType("concurrent-asset", "development")
				am.UpdateDocumentation("concurrent-asset")
				done <- true
			}()
		}

		// Wait for all operations to complete
		for i := 0; i < concurrentOps; i++ {
			<-done
		}

		// Verify final state
		asset, err := am.GetAsset("concurrent-asset")
		require.NoError(t, err, "GetAsset should succeed")
		require.NotNil(t, asset, "Expected non-nil asset")
		assert.Greater(t, asset.AssociatedTaskCount, 0, "Task count should be greater than 0")
	})
}
