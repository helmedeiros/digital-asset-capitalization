package testutil

import (
	"errors"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

func TestMockAssetService_DefaultBehavior(t *testing.T) {
	mock := NewMockAssetService()

	// Test default behavior (should not panic and return sensible defaults)
	err := mock.CreateAsset("test", "description")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	assets, err := mock.ListAssets()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(assets) != 0 {
		t.Errorf("Expected empty assets list, got %d assets", len(assets))
	}

	asset, err := mock.GetAsset("test")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if asset.Name != "test" {
		t.Errorf("Expected asset name to be 'test', got %s", asset.Name)
	}

	err = mock.DeleteAsset("test")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = mock.UpdateAsset("test", "desc", "why", "benefits", "how", "metrics")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = mock.UpdateDocumentation("test")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = mock.IncrementTaskCount("test")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = mock.DecrementTaskCount("test")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	result, err := mock.SyncFromConfluence("space", "label", false)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result == nil {
		t.Error("Expected sync result, got nil")
	}

	err = mock.EnrichAsset("test", "field")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	err = mock.GenerateKeywords("test")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestMockAssetService_CustomBehavior(t *testing.T) {
	mock := NewMockAssetService()

	// Test custom behavior through function setters
	expectedError := errors.New("custom error")
	customAssets := []*domain.Asset{
		{Name: "asset1"},
		{Name: "asset2"},
	}

	mock.SetListAssetsFunc(func() ([]*domain.Asset, error) {
		return customAssets, expectedError
	})

	assets, err := mock.ListAssets()
	if err != expectedError {
		t.Errorf("Expected custom error %v, got %v", expectedError, err)
	}
	if len(assets) != 2 {
		t.Errorf("Expected 2 assets, got %d", len(assets))
	}

	customAsset := &domain.Asset{Name: "custom"}
	mock.SetGetAssetFunc(func(identifier string) (*domain.Asset, error) {
		if identifier == "custom" {
			return customAsset, nil
		}
		return nil, expectedError
	})

	asset, err := mock.GetAsset("custom")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if asset.Name != "custom" {
		t.Errorf("Expected asset name to be 'custom', got %s", asset.Name)
	}

	asset, err = mock.GetAsset("other")
	if err != expectedError {
		t.Errorf("Expected custom error %v, got %v", expectedError, err)
	}

	customResult := &domain.SyncResult{}
	mock.SetSyncFromConfluenceFunc(func(spaceKey, label string, debug bool) (*domain.SyncResult, error) {
		if spaceKey == "test" && label == "test" && debug {
			return customResult, nil
		}
		return nil, expectedError
	})

	result, err := mock.SyncFromConfluence("test", "test", true)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != customResult {
		t.Error("Expected custom result")
	}

	result, err = mock.SyncFromConfluence("other", "other", false)
	if err != expectedError {
		t.Errorf("Expected custom error %v, got %v", expectedError, err)
	}
}

func TestMockAssetService_Reset(t *testing.T) {
	mock := NewMockAssetService()

	// Set custom functions
	mock.SetListAssetsFunc(func() ([]*domain.Asset, error) {
		return []*domain.Asset{}, errors.New("should be reset")
	})
	mock.SetGetAssetFunc(func(identifier string) (*domain.Asset, error) {
		return nil, errors.New("should be reset")
	})
	mock.SetSyncFromConfluenceFunc(func(spaceKey, label string, debug bool) (*domain.SyncResult, error) {
		return nil, errors.New("should be reset")
	})

	// Reset
	mock.Reset()

	// Test that behavior is back to defaults
	assets, err := mock.ListAssets()
	if err != nil {
		t.Errorf("Expected no error after reset, got %v", err)
	}
	if len(assets) != 0 {
		t.Errorf("Expected empty assets list after reset, got %d assets", len(assets))
	}

	asset, err := mock.GetAsset("test")
	if err != nil {
		t.Errorf("Expected no error after reset, got %v", err)
	}
	if asset.Name != "test" {
		t.Errorf("Expected asset name to be 'test' after reset, got %s", asset.Name)
	}

	result, err := mock.SyncFromConfluence("space", "label", false)
	if err != nil {
		t.Errorf("Expected no error after reset, got %v", err)
	}
	if result == nil {
		t.Error("Expected sync result after reset, got nil")
	}
}
