package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/assetcap/action"
)

const testAssetsFile = "test_assets.json"

func cleanupTestAssets() {
	os.Remove(filepath.Join("testdata", testAssetsFile))
}

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
	// Clean up test assets before and after tests
	cleanupTestAssets()
	defer cleanupTestAssets()

	am := action.NewAssetManager("testdata", testAssetsFile)

	t.Run("CreateAsset", func(t *testing.T) {
		// Test successful creation
		err := am.CreateAsset("test-asset", "Test description")
		if err != nil {
			t.Errorf("CreateAsset failed: %v", err)
		}

		// Test duplicate creation
		err = am.CreateAsset("test-asset", "Another description")
		if err == nil {
			t.Error("Expected error for duplicate asset creation, got nil")
		}
		if err != nil && err.Error() != "asset test-asset already exists" {
			t.Errorf("Expected 'already exists' error, got: %v", err)
		}

		// Test empty name
		err = am.CreateAsset("", "Description")
		if err == nil {
			t.Error("Expected error for empty name, got nil")
		}

		// Test empty description
		err = am.CreateAsset("new-asset", "")
		if err == nil {
			t.Error("Expected error for empty description, got nil")
		}
	})

	t.Run("AddContributionType", func(t *testing.T) {
		// Test adding to non-existent asset
		err := am.AddContributionType("non-existent", "development")
		if err == nil {
			t.Error("Expected error for non-existent asset, got nil")
		}
		if err != nil && err.Error() != "asset non-existent not found" {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}

		// Test adding valid contribution type
		err = am.AddContributionType("test-asset", "development")
		if err != nil {
			t.Errorf("AddContributionType failed: %v", err)
		}

		// Test adding invalid contribution type
		err = am.AddContributionType("test-asset", "invalid-type")
		if err == nil {
			t.Error("Expected error for invalid contribution type, got nil")
		}

		// Test adding duplicate contribution type
		err = am.AddContributionType("test-asset", "development")
		if err == nil {
			t.Error("Expected error for duplicate contribution type, got nil")
		}
	})

	t.Run("ListAssets", func(t *testing.T) {
		assets := am.ListAssets()
		found := false
		for _, name := range assets {
			if name == "test-asset" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find 'test-asset' in list")
		}
	})

	t.Run("GetAsset", func(t *testing.T) {
		// Test getting existing asset
		asset, err := am.GetAsset("test-asset")
		if err != nil {
			t.Errorf("GetAsset failed: %v", err)
		}
		if asset == nil {
			t.Error("Expected non-nil asset")
		}
		if asset.Name != "test-asset" {
			t.Errorf("Expected asset name 'test-asset', got: %s", asset.Name)
		}

		// Test getting non-existent asset
		asset, err = am.GetAsset("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent asset, got nil")
		}
		if asset != nil {
			t.Error("Expected nil asset")
		}
	})

	t.Run("UpdateDocumentation", func(t *testing.T) {
		// Test updating non-existent asset
		err := am.UpdateDocumentation("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent asset, got nil")
		}

		// Test updating existing asset
		err = am.UpdateDocumentation("test-asset")
		if err != nil {
			t.Errorf("UpdateDocumentation failed: %v", err)
		}

		// Verify update
		asset, err := am.GetAsset("test-asset")
		if err != nil {
			t.Errorf("GetAsset failed: %v", err)
		}
		if asset == nil {
			t.Error("Expected non-nil asset")
		}
	})

	t.Run("TaskCountOperations", func(t *testing.T) {
		// Test incrementing non-existent asset
		err := am.IncrementTaskCount("non-existent")
		if err == nil {
			t.Error("Expected error for non-existent asset, got nil")
		}

		// Test incrementing existing asset
		err = am.IncrementTaskCount("test-asset")
		if err != nil {
			t.Errorf("IncrementTaskCount failed: %v", err)
		}

		// Verify increment
		asset, err := am.GetAsset("test-asset")
		if err != nil {
			t.Errorf("GetAsset failed: %v", err)
		}
		if asset.AssociatedTaskCount != 1 {
			t.Errorf("Expected task count 1, got: %d", asset.AssociatedTaskCount)
		}

		// Test decrementing
		err = am.DecrementTaskCount("test-asset")
		if err != nil {
			t.Errorf("DecrementTaskCount failed: %v", err)
		}

		// Verify decrement
		asset, err = am.GetAsset("test-asset")
		if err != nil {
			t.Errorf("GetAsset failed: %v", err)
		}
		if asset.AssociatedTaskCount != 0 {
			t.Errorf("Expected task count 0, got: %d", asset.AssociatedTaskCount)
		}

		// Test decrementing below zero
		err = am.DecrementTaskCount("test-asset")
		if err != nil {
			t.Errorf("DecrementTaskCount failed: %v", err)
		}
		asset, err = am.GetAsset("test-asset")
		if err != nil {
			t.Errorf("GetAsset failed: %v", err)
		}
		if asset.AssociatedTaskCount != 0 {
			t.Errorf("Expected task count 0, got: %d", asset.AssociatedTaskCount)
		}
	})
}

func TestAssetManagerConcurrent(t *testing.T) {
	// Clean up test assets before and after tests
	cleanupTestAssets()
	defer cleanupTestAssets()

	am := action.NewAssetManager("testdata", testAssetsFile)

	// Create a test asset
	err := am.CreateAsset("concurrent-asset", "Test description")
	if err != nil {
		t.Fatalf("CreateAsset failed: %v", err)
	}

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
		if err != nil {
			t.Fatalf("GetAsset failed: %v", err)
		}
		if asset.AssociatedTaskCount != concurrentOps {
			t.Errorf("Expected task count %d, got: %d", concurrentOps, asset.AssociatedTaskCount)
		}
		found := false
		for _, ct := range asset.ContributionTypes {
			if ct == "development" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected to find 'development' in contribution types")
		}
	})
}
