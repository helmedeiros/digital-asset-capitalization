package infrastructure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDir is a temporary directory for test files
const testDir = "testdata"
const testFile = "test_assets.json"

type testHelper struct {
	repo *JSONRepository
	dir  string
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

	config := RepositoryConfig{
		Directory: dir,
		Filename:  testFile,
		FileMode:  0644,
		DirMode:   0755,
	}
	repo := NewJSONRepository(config).(*JSONRepository)

	return &testHelper{
		repo: repo,
		dir:  dir,
	}
}

func (h *testHelper) cleanup(t *testing.T) {
	t.Helper()
	err := os.RemoveAll(h.dir)
	assert.NoError(t, err, "Failed to cleanup test directory")
}

func (h *testHelper) createTestAsset(name, description string) *domain.Asset {
	asset, err := domain.NewAsset(name, description)
	if err != nil {
		panic(err)
	}
	return asset
}

func TestJSONRepository_Save(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should save valid asset", func(t *testing.T) {
		asset := h.createTestAsset("test-asset", "Test Description")
		err := h.repo.Save(asset)
		require.NoError(t, err, "Failed to save asset")

		// Verify the asset was saved
		found, err := h.repo.FindByName("test-asset")
		require.NoError(t, err, "Failed to find saved asset")
		assert.Equal(t, asset.Name, found.Name, "Asset name mismatch")
		assert.Equal(t, asset.Description, found.Description, "Asset description mismatch")
	})

	t.Run("should not save nil asset", func(t *testing.T) {
		err := h.repo.Save(nil)
		assert.Error(t, err, "Expected error saving nil asset")
		assert.Contains(t, err.Error(), "cannot save nil asset", "Unexpected error message")
	})

	t.Run("should update existing asset", func(t *testing.T) {
		// Create and save initial asset
		asset := h.createTestAsset("test-asset", "Initial Description")
		err := h.repo.Save(asset)
		require.NoError(t, err, "Failed to save initial asset")

		// Update the asset
		asset.Description = "Updated Description"
		err = h.repo.Save(asset)
		require.NoError(t, err, "Failed to update asset")

		// Verify the update
		found, err := h.repo.FindByName("test-asset")
		require.NoError(t, err, "Failed to find updated asset")
		assert.Equal(t, "Updated Description", found.Description, "Asset description not updated")
	})
}

func TestJSONRepository_FindByName(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should find existing asset", func(t *testing.T) {
		// Create and save a test asset
		asset := h.createTestAsset("test-asset", "Test Description")
		err := h.repo.Save(asset)
		require.NoError(t, err, "Failed to save asset")

		// Find the asset
		found, err := h.repo.FindByName("test-asset")
		require.NoError(t, err, "Failed to find asset")

		assert.Equal(t, asset.Name, found.Name, "Asset name mismatch")
		assert.Equal(t, asset.Description, found.Description, "Asset description mismatch")
	})

	t.Run("should return error for non-existent asset", func(t *testing.T) {
		found, err := h.repo.FindByName("non-existent")
		assert.Error(t, err, "Expected error for non-existent asset")
		assert.Nil(t, found, "Expected nil asset for non-existent name")
		assert.Contains(t, err.Error(), "asset non-existent not found", "Unexpected error message")
	})

	t.Run("should return error for empty name", func(t *testing.T) {
		found, err := h.repo.FindByName("")
		assert.Error(t, err, "Expected error for empty name")
		assert.Nil(t, found, "Expected nil asset for empty name")
		assert.Contains(t, err.Error(), "asset name cannot be empty", "Unexpected error message")
	})
}

func TestJSONRepository_FindAll(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should return empty list when no assets exist", func(t *testing.T) {
		assets, err := h.repo.FindAll()
		require.NoError(t, err, "Failed to find assets")
		assert.Empty(t, assets, "Expected empty asset list")
	})

	t.Run("should return all saved assets", func(t *testing.T) {
		// Create and save multiple assets
		asset1 := h.createTestAsset("asset1", "Description 1")
		asset2 := h.createTestAsset("asset2", "Description 2")

		err := h.repo.Save(asset1)
		require.NoError(t, err, "Failed to save asset1")

		err = h.repo.Save(asset2)
		require.NoError(t, err, "Failed to save asset2")

		// Find all assets
		assets, err := h.repo.FindAll()
		require.NoError(t, err, "Failed to find all assets")
		assert.Len(t, assets, 2, "Expected 2 assets")

		// Verify asset details
		foundAssets := make(map[string]*domain.Asset)
		for _, asset := range assets {
			foundAssets[asset.Name] = asset
		}

		assert.Equal(t, asset1.Description, foundAssets["asset1"].Description, "Asset1 description mismatch")
		assert.Equal(t, asset2.Description, foundAssets["asset2"].Description, "Asset2 description mismatch")
	})
}

func TestJSONRepository_Delete(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should delete existing asset", func(t *testing.T) {
		// Create and save a test asset
		asset := h.createTestAsset("test-asset", "Test Description")
		err := h.repo.Save(asset)
		require.NoError(t, err, "Failed to save asset")

		// Delete the asset
		err = h.repo.Delete("test-asset")
		require.NoError(t, err, "Failed to delete asset")

		// Verify the asset was deleted
		found, err := h.repo.FindByName("test-asset")
		assert.Error(t, err, "Expected error finding deleted asset")
		assert.Nil(t, found, "Expected nil asset after deletion")
		assert.Contains(t, err.Error(), "asset test-asset not found", "Unexpected error message")
	})

	t.Run("should return error when deleting non-existent asset", func(t *testing.T) {
		err := h.repo.Delete("non-existent")
		assert.Error(t, err, "Expected error deleting non-existent asset")
		assert.Contains(t, err.Error(), "asset non-existent not found", "Unexpected error message")
	})

	t.Run("should return error when deleting with empty name", func(t *testing.T) {
		err := h.repo.Delete("")
		assert.Error(t, err, "Expected error deleting with empty name")
		assert.Contains(t, err.Error(), "asset name cannot be empty", "Unexpected error message")
	})
}

func TestJSONRepository_ErrorHandling(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should handle file permission errors", func(t *testing.T) {
		// Create a directory with no write permissions
		noWriteDir := filepath.Join(h.dir, "no_write")
		err := os.MkdirAll(noWriteDir, 0444)
		require.NoError(t, err, "Failed to create no-write directory")

		config := RepositoryConfig{
			Directory: noWriteDir,
			Filename:  testFile,
			FileMode:  0644,
			DirMode:   0755,
		}
		repo := NewJSONRepository(config).(*JSONRepository)
		asset := h.createTestAsset("test-asset", "Test Description")

		// Try to save (should fail due to permissions)
		err = repo.Save(asset)
		assert.Error(t, err, "Expected error saving to read-only directory")
		assert.Contains(t, err.Error(), "failed to load assets", "Unexpected error message")
	})

	t.Run("should handle marshal errors", func(t *testing.T) {
		// Create a file with invalid JSON that will cause unmarshal errors
		filePath := filepath.Join(h.dir, testFile)
		invalidJSON := `{"name": "test", "created_at": "invalid-date"}`
		err := os.WriteFile(filePath, []byte(invalidJSON), 0644)
		require.NoError(t, err, "Failed to write invalid JSON")

		// Try to load the assets (should fail due to invalid date format)
		_, err = h.repo.FindAll()
		assert.Error(t, err, "Expected error loading invalid JSON")
		assert.Contains(t, err.Error(), "failed to unmarshal assets", "Unexpected error message")
	})

	t.Run("should handle file read errors", func(t *testing.T) {
		// Create a file with invalid JSON
		filePath := filepath.Join(h.dir, testFile)
		err := os.WriteFile(filePath, []byte("test"), 0644)
		require.NoError(t, err, "Failed to create test file")

		// Try to read the file
		_, err = h.repo.FindAll()
		assert.Error(t, err, "Expected error reading invalid file")
		assert.Contains(t, err.Error(), "failed to unmarshal assets", "Unexpected error message")
	})
}

func TestJSONRepository_EdgeCases(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should handle directory creation error", func(t *testing.T) {
		// Create a file in a temporary location
		tmpDir := filepath.Join(os.TempDir(), "json_repo_test")
		err := os.MkdirAll(tmpDir, 0755)
		require.NoError(t, err, "Failed to create temp directory")
		defer os.RemoveAll(tmpDir)

		// Create a file that will block directory creation
		filePath := filepath.Join(tmpDir, "blocking_file")
		err = os.WriteFile(filePath, []byte("not a directory"), 0444)
		require.NoError(t, err, "Failed to create blocking file")

		// Try to use this location for the repository
		config := RepositoryConfig{
			Directory: filePath,
			Filename:  testFile,
			FileMode:  0644,
			DirMode:   0755,
		}
		repo := NewJSONRepository(config).(*JSONRepository)
		asset := h.createTestAsset("test-asset", "Test Description")

		// Try operations that require directory access
		err = repo.Save(asset)
		assert.Error(t, err, "Expected error when directory can't be created")
		assert.Contains(t, err.Error(), "failed to create directory", "Unexpected error message")

		_, err = repo.FindByName("test-asset")
		assert.Error(t, err, "Expected error when directory can't be accessed")
		assert.Contains(t, err.Error(), "failed to create directory", "Unexpected error message")

		_, err = repo.FindAll()
		assert.Error(t, err, "Expected error when directory can't be accessed")
		assert.Contains(t, err.Error(), "failed to create directory", "Unexpected error message")

		err = repo.Delete("test-asset")
		assert.Error(t, err, "Expected error when directory can't be accessed")
		assert.Contains(t, err.Error(), "failed to create directory", "Unexpected error message")
	})

	t.Run("should handle write errors", func(t *testing.T) {
		// Create a read-only directory
		readOnlyDir := filepath.Join(h.dir, "readonly")
		err := os.MkdirAll(readOnlyDir, 0444)
		require.NoError(t, err, "Failed to create read-only directory")

		config := RepositoryConfig{
			Directory: readOnlyDir,
			Filename:  testFile,
			FileMode:  0644,
			DirMode:   0755,
		}
		repo := NewJSONRepository(config).(*JSONRepository)
		asset := h.createTestAsset("test-asset", "Test Description")

		err = repo.Save(asset)
		assert.Error(t, err, "Expected error when saving to read-only directory")
		assert.Contains(t, err.Error(), "failed to load assets", "Unexpected error message")

		err = repo.Delete("test-asset")
		assert.Error(t, err, "Expected error when deleting in read-only directory")
		assert.Contains(t, err.Error(), "failed to load assets", "Unexpected error message")
	})
}
