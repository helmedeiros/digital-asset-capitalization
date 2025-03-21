package infrastructure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
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
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	repo := NewJSONRepository(dir, testFile).(*JSONRepository)

	return &testHelper{
		repo: repo,
		dir:  dir,
	}
}

func (h *testHelper) cleanup(t *testing.T) {
	t.Helper()
	if err := os.RemoveAll(h.dir); err != nil {
		t.Errorf("Failed to cleanup test directory: %v", err)
	}
}

func (h *testHelper) createTestAsset(name, description string) *domain.Asset {
	asset := &domain.Asset{
		Name:        name,
		Description: description,
	}
	return asset
}

func TestJSONRepository_Save(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should save new asset", func(t *testing.T) {
		// Create and save a test asset
		asset := h.createTestAsset("test-asset", "Test Description")
		err := h.repo.Save(asset)
		if err != nil {
			t.Fatalf("Failed to save asset: %v", err)
		}

		// Verify the asset was saved
		saved, err := h.repo.FindByName("test-asset")
		if err != nil {
			t.Fatalf("Failed to find saved asset: %v", err)
		}

		if saved.Name != asset.Name {
			t.Errorf("Expected asset name %s, got %s", asset.Name, saved.Name)
		}
		if saved.Description != asset.Description {
			t.Errorf("Expected asset description %s, got %s", asset.Description, saved.Description)
		}
	})

	t.Run("should update existing asset", func(t *testing.T) {
		// Create and save initial asset
		asset := h.createTestAsset("test-asset", "Initial Description")
		err := h.repo.Save(asset)
		if err != nil {
			t.Fatalf("Failed to save initial asset: %v", err)
		}

		// Update the asset
		asset.Description = "Updated Description"
		err = h.repo.Save(asset)
		if err != nil {
			t.Fatalf("Failed to update asset: %v", err)
		}

		// Verify the update
		updated, err := h.repo.FindByName("test-asset")
		if err != nil {
			t.Fatalf("Failed to find updated asset: %v", err)
		}

		if updated.Description != "Updated Description" {
			t.Errorf("Expected updated description %s, got %s", "Updated Description", updated.Description)
		}
	})
}

func TestJSONRepository_FindByName(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should find existing asset", func(t *testing.T) {
		// Create and save a test asset
		asset := h.createTestAsset("test-asset", "Test Description")
		err := h.repo.Save(asset)
		if err != nil {
			t.Fatalf("Failed to save asset: %v", err)
		}

		// Find the asset
		found, err := h.repo.FindByName("test-asset")
		if err != nil {
			t.Fatalf("Failed to find asset: %v", err)
		}

		if found.Name != asset.Name {
			t.Errorf("Expected asset name %s, got %s", asset.Name, found.Name)
		}
	})

	t.Run("should return error for non-existent asset", func(t *testing.T) {
		_, err := h.repo.FindByName("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent asset, got nil")
		}
	})
}

func TestJSONRepository_FindAll(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should return empty list when no assets exist", func(t *testing.T) {
		assets, err := h.repo.FindAll()
		if err != nil {
			t.Fatalf("Failed to find assets: %v", err)
		}

		if len(assets) != 0 {
			t.Errorf("Expected empty asset list, got %d assets", len(assets))
		}
	})

	t.Run("should return all saved assets", func(t *testing.T) {
		// Create and save multiple assets
		asset1 := h.createTestAsset("asset1", "Description 1")
		asset2 := h.createTestAsset("asset2", "Description 2")

		err := h.repo.Save(asset1)
		if err != nil {
			t.Fatalf("Failed to save asset1: %v", err)
		}

		err = h.repo.Save(asset2)
		if err != nil {
			t.Fatalf("Failed to save asset2: %v", err)
		}

		// Find all assets
		assets, err := h.repo.FindAll()
		if err != nil {
			t.Fatalf("Failed to find all assets: %v", err)
		}

		if len(assets) != 2 {
			t.Errorf("Expected 2 assets, got %d", len(assets))
		}
	})
}

func TestJSONRepository_Delete(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should delete existing asset", func(t *testing.T) {
		// Create and save a test asset
		asset := h.createTestAsset("test-asset", "Test Description")
		err := h.repo.Save(asset)
		if err != nil {
			t.Fatalf("Failed to save asset: %v", err)
		}

		// Delete the asset
		err = h.repo.Delete("test-asset")
		if err != nil {
			t.Fatalf("Failed to delete asset: %v", err)
		}

		// Verify the asset was deleted
		_, err = h.repo.FindByName("test-asset")
		if err == nil {
			t.Error("Expected error finding deleted asset, got nil")
		}
	})

	t.Run("should return error when deleting non-existent asset", func(t *testing.T) {
		err := h.repo.Delete("non-existent")
		if err == nil {
			t.Error("Expected error deleting non-existent asset, got nil")
		}
	})
}

func TestJSONRepository_FileOperations(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should handle invalid JSON file", func(t *testing.T) {
		// Write invalid JSON to file
		filePath := filepath.Join(h.dir, testFile)
		err := os.WriteFile(filePath, []byte("invalid json"), 0644)
		if err != nil {
			t.Fatalf("Failed to write invalid JSON: %v", err)
		}

		// Try to load assets
		_, err = h.repo.loadAssets()
		if err == nil {
			t.Error("Expected error loading invalid JSON, got nil")
		}
	})

	t.Run("should create directory if it doesn't exist", func(t *testing.T) {
		// Remove test directory
		_ = os.RemoveAll(h.dir)

		// Try to save an asset (should create directory)
		asset := h.createTestAsset("test-asset", "Test Description")
		err := h.repo.Save(asset)
		if err != nil {
			t.Fatalf("Failed to save asset: %v", err)
		}

		// Verify directory was created
		if _, err := os.Stat(h.dir); os.IsNotExist(err) {
			t.Error("Expected directory to be created")
		}
	})
}

func TestJSONRepository_ErrorHandling(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should handle file permission errors", func(t *testing.T) {
		// Create a directory with no write permissions
		noWriteDir := filepath.Join(h.dir, "no_write")
		if err := os.MkdirAll(noWriteDir, 0444); err != nil {
			t.Fatalf("Failed to create no-write directory: %v", err)
		}

		repo := NewJSONRepository(noWriteDir, testFile).(*JSONRepository)
		asset := h.createTestAsset("test-asset", "Test Description")

		// Try to save (should fail due to permissions)
		err := repo.Save(asset)
		if err == nil {
			t.Error("Expected error saving to read-only directory")
		}
	})

	t.Run("should handle marshal errors", func(t *testing.T) {
		// Create a file with invalid JSON that will cause unmarshal errors
		filePath := filepath.Join(h.dir, testFile)
		invalidJSON := `{"name": "test", "created_at": "invalid-date"}`
		if err := os.WriteFile(filePath, []byte(invalidJSON), 0644); err != nil {
			t.Fatalf("Failed to write invalid JSON: %v", err)
		}

		// Try to load the assets (should fail due to invalid date format)
		_, err := h.repo.FindAll()
		if err == nil {
			t.Error("Expected error loading invalid JSON")
		}
	})

	t.Run("should handle file read errors", func(t *testing.T) {
		// Create a file that can't be read
		filePath := filepath.Join(h.dir, testFile)
		if err := os.WriteFile(filePath, []byte("test"), 0000); err != nil {
			t.Fatalf("Failed to create unreadable file: %v", err)
		}

		// Try to read the file
		_, err := h.repo.FindAll()
		if err == nil {
			t.Error("Expected error reading unreadable file")
		}
	})
}

func TestJSONRepository_EdgeCases(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should handle directory creation error", func(t *testing.T) {
		// Create a file in a temporary location
		tmpDir := filepath.Join(os.TempDir(), "json_repo_test")
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create a file that will block directory creation
		filePath := filepath.Join(tmpDir, "blocking_file")
		if err := os.WriteFile(filePath, []byte("not a directory"), 0444); err != nil {
			t.Fatalf("Failed to create blocking file: %v", err)
		}

		// Try to use this location for the repository
		repo := NewJSONRepository(filePath, testFile).(*JSONRepository)
		asset := h.createTestAsset("test-asset", "Test Description")

		// Try operations that require directory access
		err := repo.Save(asset)
		if err == nil {
			t.Error("Expected error when directory can't be created")
		}

		_, err = repo.FindByName("test-asset")
		if err == nil {
			t.Error("Expected error when directory can't be accessed")
		}

		_, err = repo.FindAll()
		if err == nil {
			t.Error("Expected error when directory can't be accessed")
		}

		err = repo.Delete("test-asset")
		if err == nil {
			t.Error("Expected error when directory can't be accessed")
		}
	})

	t.Run("should handle write errors", func(t *testing.T) {
		// Create a read-only directory
		readOnlyDir := filepath.Join(h.dir, "readonly")
		if err := os.MkdirAll(readOnlyDir, 0444); err != nil {
			t.Fatalf("Failed to create read-only directory: %v", err)
		}

		repo := NewJSONRepository(readOnlyDir, testFile).(*JSONRepository)
		asset := h.createTestAsset("test-asset", "Test Description")

		err := repo.Save(asset)
		if err == nil {
			t.Error("Expected error when saving to read-only directory")
		}

		err = repo.Delete("test-asset")
		if err == nil {
			t.Error("Expected error when deleting in read-only directory")
		}
	})
}
