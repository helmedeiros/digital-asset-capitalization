package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/model"
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
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	storage := NewJSONStorage(dir, testFile)

	return &testHelper{
		storage: storage,
		dir:     dir,
	}
}

func (h *testHelper) cleanup(t *testing.T) {
	t.Helper()
	if err := os.RemoveAll(h.dir); err != nil {
		t.Errorf("Failed to cleanup test directory: %v", err)
	}
}

func (h *testHelper) createTestAsset(name, description string) *model.Asset {
	asset, _ := model.NewAsset(name, description)
	return asset
}

func TestJSONStorage_Load(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should return empty map when file doesn't exist", func(t *testing.T) {
		assets, err := h.storage.Load()
		if err != nil {
			t.Fatalf("Failed to load assets: %v", err)
		}

		if len(assets) != 0 {
			t.Errorf("Expected empty map, got map with %d items", len(assets))
		}
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
		if err := h.storage.Save(assets); err != nil {
			t.Fatalf("Failed to save assets: %v", err)
		}

		// Load assets
		loaded, err := h.storage.Load()
		if err != nil {
			t.Fatalf("Failed to load assets: %v", err)
		}

		if len(loaded) != 2 {
			t.Errorf("Expected 2 assets, got %d", len(loaded))
		}

		// Verify asset contents
		if loaded["asset1"].Description != "Description 1" {
			t.Errorf("Expected description 'Description 1', got %s", loaded["asset1"].Description)
		}
		if loaded["asset2"].Description != "Description 2" {
			t.Errorf("Expected description 'Description 2', got %s", loaded["asset2"].Description)
		}
	})

	t.Run("should handle invalid JSON file", func(t *testing.T) {
		// Write invalid JSON to file
		filePath := filepath.Join(h.dir, testFile)
		if err := os.WriteFile(filePath, []byte("invalid json"), 0644); err != nil {
			t.Fatalf("Failed to write invalid JSON: %v", err)
		}

		// Try to load assets
		_, err := h.storage.Load()
		if err == nil {
			t.Error("Expected error loading invalid JSON")
		}
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
		if err := h.storage.Save(assets); err != nil {
			t.Fatalf("Failed to save assets: %v", err)
		}

		// Verify file exists
		filePath := filepath.Join(h.dir, testFile)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Error("Expected file to exist")
		}

		// Load and verify contents
		loaded, err := h.storage.Load()
		if err != nil {
			t.Fatalf("Failed to load assets: %v", err)
		}

		if len(loaded) != 2 {
			t.Errorf("Expected 2 assets, got %d", len(loaded))
		}
	})

	t.Run("should handle directory creation error", func(t *testing.T) {
		// Create a file where the directory should be
		tmpDir := filepath.Join(os.TempDir(), "storage_test")
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create a file that will block directory creation
		filePath := filepath.Join(tmpDir, "blocking_file")
		if err := os.WriteFile(filePath, []byte("not a directory"), 0444); err != nil {
			t.Fatalf("Failed to create blocking file: %v", err)
		}

		// Try to use this location for storage
		storage := NewJSONStorage(filePath, testFile)
		assets := map[string]*model.Asset{
			"test": h.createTestAsset("test", "Test Description"),
		}

		err := storage.Save(assets)
		if err == nil {
			t.Error("Expected error when directory can't be created")
		}
	})

	t.Run("should handle write errors", func(t *testing.T) {
		// Create a read-only directory
		readOnlyDir := filepath.Join(h.dir, "readonly")
		if err := os.MkdirAll(readOnlyDir, 0444); err != nil {
			t.Fatalf("Failed to create read-only directory: %v", err)
		}

		storage := NewJSONStorage(readOnlyDir, testFile)
		assets := map[string]*model.Asset{
			"test": h.createTestAsset("test", "Test Description"),
		}

		err := storage.Save(assets)
		if err == nil {
			t.Error("Expected error when saving to read-only directory")
		}
	})
}

func TestJSONStorage_EdgeCases(t *testing.T) {
	h := setupTest(t)
	defer h.cleanup(t)

	t.Run("should handle empty asset map", func(t *testing.T) {
		// Save empty map
		err := h.storage.Save(make(map[string]*model.Asset))
		if err != nil {
			t.Fatalf("Failed to save empty map: %v", err)
		}

		// Load and verify
		loaded, err := h.storage.Load()
		if err != nil {
			t.Fatalf("Failed to load empty map: %v", err)
		}

		if len(loaded) != 0 {
			t.Errorf("Expected empty map, got map with %d items", len(loaded))
		}
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
		if err != nil {
			t.Fatalf("Failed to save many assets: %v", err)
		}

		// Load and verify
		loaded, err := h.storage.Load()
		if err != nil {
			t.Fatalf("Failed to load many assets: %v", err)
		}

		if len(loaded) != 1000 {
			t.Errorf("Expected 1000 assets, got %d", len(loaded))
		}
	})
}
