package testutil

import (
	"github.com/stretchr/testify/mock"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// MockAssetRepository is a mock implementation of AssetRepository for testing
type MockAssetRepository struct {
	mock.Mock
}

// NewMockAssetRepository creates a new mock repository
func NewMockAssetRepository() *MockAssetRepository {
	return &MockAssetRepository{}
}

// Save saves an asset to the repository
func (m *MockAssetRepository) Save(asset *domain.Asset) error {
	args := m.Called(asset)
	return args.Error(0)
}

// FindByName finds an asset by its name
func (m *MockAssetRepository) FindByName(name string) (*domain.Asset, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Asset), args.Error(1)
}

// FindAll returns all assets
func (m *MockAssetRepository) FindAll() ([]*domain.Asset, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Asset), args.Error(1)
}

// FindByID finds an asset by its ID
func (m *MockAssetRepository) FindByID(id string) (*domain.Asset, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Asset), args.Error(1)
}

// Delete deletes an asset by name
func (m *MockAssetRepository) Delete(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

// Ensure MockAssetRepository implements AssetRepository
var _ ports.AssetRepository = (*MockAssetRepository)(nil)
