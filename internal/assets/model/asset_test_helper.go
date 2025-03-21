package model

import "time"

// AssetMother provides helper functions for creating test assets
type AssetMother struct{}

// NewAssetMother creates a new AssetMother instance
func NewAssetMother() *AssetMother {
	return &AssetMother{}
}

// CreateValidAsset creates a valid asset with default test values
func (m *AssetMother) CreateValidAsset() *Asset {
	asset, _ := NewAsset("Test Asset", "Test Description")
	return asset
}

// CreateAssetWithCustomValues creates an asset with custom values
func (m *AssetMother) CreateAssetWithCustomValues(name, description string) *Asset {
	asset, _ := NewAsset(name, description)
	return asset
}

// CreateAssetWithContributionTypes creates an asset with predefined contribution types
func (m *AssetMother) CreateAssetWithContributionTypes(types ...string) *Asset {
	asset := m.CreateValidAsset()
	for _, t := range types {
		asset.AddContributionType(t)
	}
	return asset
}

// CreateAssetWithTaskCount creates an asset with a specific task count
func (m *AssetMother) CreateAssetWithTaskCount(count int) *Asset {
	asset := m.CreateValidAsset()
	for i := 0; i < count; i++ {
		asset.IncrementTaskCount()
	}
	return asset
}

// CreateAssetWithCustomTimestamps creates an asset with custom timestamps
func (m *AssetMother) CreateAssetWithCustomTimestamps(createdAt, updatedAt, lastDocUpdateAt time.Time) *Asset {
	asset := m.CreateValidAsset()
	asset.CreatedAt = createdAt
	asset.UpdatedAt = updatedAt
	asset.LastDocUpdateAt = lastDocUpdateAt
	return asset
}
