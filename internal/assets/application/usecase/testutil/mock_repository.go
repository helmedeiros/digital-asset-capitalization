package testutil

import (
	"errors"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// MockAssetRepository is a mock implementation of AssetRepository for testing
type MockAssetRepository struct {
	assets map[string]*domain.Asset
}

// NewMockAssetRepository creates a new mock repository
func NewMockAssetRepository() *MockAssetRepository {
	return &MockAssetRepository{
		assets: make(map[string]*domain.Asset),
	}
}

// Save saves an asset to the repository
func (m *MockAssetRepository) Save(asset *domain.Asset) error {
	if asset == nil {
		return errors.New("asset cannot be nil")
	}
	if asset.Name == "" {
		return errors.New("asset name cannot be empty")
	}
	m.assets[asset.Name] = asset
	return nil
}

// FindByName finds an asset by its name
func (m *MockAssetRepository) FindByName(name string) (*domain.Asset, error) {
	if name == "" {
		return nil, errors.New("name cannot be empty")
	}
	if asset, exists := m.assets[name]; exists {
		return asset, nil
	}
	return nil, errors.New("asset not found")
}

// FindAll returns all assets
func (m *MockAssetRepository) FindAll() ([]*domain.Asset, error) {
	assets := make([]*domain.Asset, 0, len(m.assets))
	for _, asset := range m.assets {
		assets = append(assets, asset)
	}
	return assets, nil
}

// FindByID finds an asset by its ID
func (m *MockAssetRepository) FindByID(id string) (*domain.Asset, error) {
	if id == "" {
		return nil, errors.New("id cannot be empty")
	}
	for _, asset := range m.assets {
		if asset.ID == id {
			return asset, nil
		}
	}
	return nil, errors.New("asset not found")
}

// Delete deletes an asset by name
func (m *MockAssetRepository) Delete(name string) error {
	if name == "" {
		return errors.New("name cannot be empty")
	}
	if _, exists := m.assets[name]; !exists {
		return errors.New("asset not found")
	}
	delete(m.assets, name)
	return nil
}

// Ensure MockAssetRepository implements AssetRepository
var _ ports.AssetRepository = (*MockAssetRepository)(nil)
