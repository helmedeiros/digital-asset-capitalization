package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDir is a temporary directory for test files
const testDir = "testdata"
const testFile = "test_assets.json"

type testHelper struct {
	storage *JSONStorage
	dir     string
}

func setupTest(t *testing.T) *testHelper {
	t.Helper()

	// Create a unique test directory for each test
	dir := filepath.Join(testDir, t.Name())

	// Clean up any existing test data
	_ = os.RemoveAll(dir)

	// Create test directory
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err, "Failed to create test directory")

	storage := NewJSONStorage(dir, testFile)

	return &testHelper{
		storage: storage,
		dir:     dir,
	}
}

func (h *testHelper) cleanup(t *testing.T) {
	t.Helper()
	err := os.RemoveAll(h.dir)
	assert.NoError(t, err, "Failed to cleanup test directory")
}

func (h *testHelper) createTestAsset(name, description string) *model.Asset {
	asset, err := model.NewAsset(name, description)
	if err != nil {
		panic(fmt.Sprintf("Failed to create test asset: %v", err))
	}
	return asset
}

func TestJSONStorage_Load(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should return empty map when file doesn't exist", func(t *testing.T) {
		assets, err := h.storage.Load()
		require.NoError(t, err, "Failed to load assets")
		assert.Empty(t, assets, "Expected empty map")
	})

	t.Run("should load existing assets", func(t *testing.T) {
		// Create test assets
		asset1 := h.createTestAsset("asset1", "Description 1")
		asset2 := h.createTestAsset("asset2", "Description 2")

		// Save assets
		assets := map[string]*model.Asset{
			asset1.Name: asset1,
			asset2.Name: asset2,
		}
		err := h.storage.Save(assets)
		require.NoError(t, err, "Failed to save assets")

		// Load assets
		loaded, err := h.storage.Load()
		require.NoError(t, err, "Failed to load assets")
		assert.Len(t, loaded, 2, "Expected 2 assets")

		// Verify asset contents
		assert.Equal(t, "Description 1", loaded["asset1"].Description)
		assert.Equal(t, "Description 2", loaded["asset2"].Description)
	})

	t.Run("should handle invalid JSON file", func(t *testing.T) {
		// Write invalid JSON to file
		filePath := filepath.Join(h.dir, testFile)
		err := os.WriteFile(filePath, []byte("invalid json"), 0644)
		require.NoError(t, err, "Failed to write invalid JSON")

		// Try to load assets
		_, err = h.storage.Load()
		assert.Error(t, err, "Expected error loading invalid JSON")
	})
}

func TestJSONStorage_Save(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should save assets successfully", func(t *testing.T) {
		// Create test assets
		asset1 := h.createTestAsset("asset1", "Description 1")
		asset2 := h.createTestAsset("asset2", "Description 2")

		// Save assets
		assets := map[string]*model.Asset{
			asset1.Name: asset1,
			asset2.Name: asset2,
		}
		err := h.storage.Save(assets)
		require.NoError(t, err, "Failed to save assets")

		// Verify file exists
		filePath := filepath.Join(h.dir, testFile)
		_, err = os.Stat(filePath)
		require.NoError(t, err, "Expected file to exist")

		// Load and verify contents
		loaded, err := h.storage.Load()
		require.NoError(t, err, "Failed to load assets")
		assert.Len(t, loaded, 2, "Expected 2 assets")
	})

	t.Run("should handle directory creation error", func(t *testing.T) {
		// Create a file where the directory should be
		tmpDir := filepath.Join(os.TempDir(), "storage_test")
		err := os.MkdirAll(tmpDir, 0755)
		require.NoError(t, err, "Failed to create temp directory")
		defer os.RemoveAll(tmpDir)

		// Create a file that will block directory creation
		filePath := filepath.Join(tmpDir, "blocking_file")
		err = os.WriteFile(filePath, []byte("not a directory"), 0444)
		require.NoError(t, err, "Failed to create blocking file")

		// Try to use this location for storage
		storage := NewJSONStorage(filePath, testFile)
		assets := map[string]*model.Asset{
			"test": h.createTestAsset("test", "Test Description"),
		}

		err = storage.Save(assets)
		assert.Error(t, err, "Expected error when directory can't be created")
	})

	t.Run("should handle write errors", func(t *testing.T) {
		// Create a read-only directory
		readOnlyDir := filepath.Join(h.dir, "readonly")
		err := os.MkdirAll(readOnlyDir, 0444)
		require.NoError(t, err, "Failed to create read-only directory")

		storage := NewJSONStorage(readOnlyDir, testFile)
		assets := map[string]*model.Asset{
			"test": h.createTestAsset("test", "Test Description"),
		}

		err = storage.Save(assets)
		assert.Error(t, err, "Expected error when saving to read-only directory")
	})
}

func TestJSONStorage_EdgeCases(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should handle empty asset map", func(t *testing.T) {
		// Save empty map
		err := h.storage.Save(make(map[string]*model.Asset))
		require.NoError(t, err, "Failed to save empty map")

		// Load and verify
		loaded, err := h.storage.Load()
		require.NoError(t, err, "Failed to load empty map")
		assert.Empty(t, loaded, "Expected empty map")
	})

	t.Run("should handle large number of assets", func(t *testing.T) {
		// Create many assets
		assets := make(map[string]*model.Asset)
		for i := 0; i < 1000; i++ {
			asset := h.createTestAsset(
				fmt.Sprintf("asset%d", i),
				fmt.Sprintf("Description %d", i),
			)
			assets[asset.Name] = asset
		}

		// Save assets
		err := h.storage.Save(assets)
		require.NoError(t, err, "Failed to save many assets")

		// Load and verify
		loaded, err := h.storage.Load()
		require.NoError(t, err, "Failed to load many assets")
		assert.Len(t, loaded, 1000, "Expected 1000 assets")
	})
}
