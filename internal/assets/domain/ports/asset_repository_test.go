package ports

import (
	"errors"
	"testing"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/stretchr/testify/assert"
)

// MockAssetRepository is a mock implementation of AssetRepository for testing
type MockAssetRepository struct {
	assets map[string]*domain.Asset
}

func NewMockAssetRepository() *MockAssetRepository {
	return &MockAssetRepository{
		assets: make(map[string]*domain.Asset),
	}
}

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

func (m *MockAssetRepository) FindByName(name string) (*domain.Asset, error) {
	if name == "" {
		return nil, errors.New("name cannot be empty")
	}
	if asset, exists := m.assets[name]; exists {
		return asset, nil
	}
	return nil, errors.New("asset not found")
}

func (m *MockAssetRepository) FindAll() ([]*domain.Asset, error) {
	assets := make([]*domain.Asset, 0, len(m.assets))
	for _, asset := range m.assets {
		assets = append(assets, asset)
	}
	return assets, nil
}

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

func TestAssetRepository(t *testing.T) {
	repo := NewMockAssetRepository()

	// Helper function to create a test asset
	createTestAsset := func(name, description string) *domain.Asset {
		now := time.Now()
		return &domain.Asset{
			ID:                  "test-id",
			Name:                name,
			Description:         description,
			CreatedAt:           now,
			UpdatedAt:           now,
			LastDocUpdateAt:     now,
			AssociatedTaskCount: 0,
			Version:             1,
		}
	}

	t.Run("Save", func(t *testing.T) {
		// Test successful save
		asset := createTestAsset("test-asset", "Test Description")
		err := repo.Save(asset)
		assert.NoError(t, err)

		// Test saving nil asset
		err = repo.Save(nil)
		assert.Error(t, err)
		assert.Equal(t, "asset cannot be nil", err.Error())

		// Test saving asset with empty name
		emptyAsset := createTestAsset("", "Description")
		err = repo.Save(emptyAsset)
		assert.Error(t, err)
		assert.Equal(t, "asset name cannot be empty", err.Error())
	})

	t.Run("FindByName", func(t *testing.T) {
		// Test finding existing asset
		asset, err := repo.FindByName("test-asset")
		assert.NoError(t, err)
		assert.Equal(t, "test-asset", asset.Name)
		assert.Equal(t, "Test Description", asset.Description)

		// Test finding non-existent asset
		asset, err = repo.FindByName("non-existent")
		assert.Error(t, err)
		assert.Equal(t, "asset not found", err.Error())

		// Test finding with empty name
		asset, err = repo.FindByName("")
		assert.Error(t, err)
		assert.Equal(t, "name cannot be empty", err.Error())
	})

	t.Run("FindAll", func(t *testing.T) {
		// Add another asset to test FindAll
		secondAsset := createTestAsset("second-asset", "Second Description")
		err := repo.Save(secondAsset)
		assert.NoError(t, err)

		// Test finding all assets
		assets, err := repo.FindAll()
		assert.NoError(t, err)
		assert.Len(t, assets, 2)

		// Verify asset contents
		assetMap := make(map[string]*domain.Asset)
		for _, asset := range assets {
			assetMap[asset.Name] = asset
		}

		assert.Contains(t, assetMap, "test-asset")
		assert.Contains(t, assetMap, "second-asset")
		assert.Equal(t, "Test Description", assetMap["test-asset"].Description)
		assert.Equal(t, "Second Description", assetMap["second-asset"].Description)
	})

	t.Run("Delete", func(t *testing.T) {
		// Test deleting existing asset
		err := repo.Delete("test-asset")
		assert.NoError(t, err)

		// Verify asset is deleted
		_, err = repo.FindByName("test-asset")
		assert.Error(t, err)
		assert.Equal(t, "asset not found", err.Error())

		// Test deleting non-existent asset
		err = repo.Delete("non-existent")
		assert.Error(t, err)
		assert.Equal(t, "asset not found", err.Error())

		// Test deleting with empty name
		err = repo.Delete("")
		assert.Error(t, err)
		assert.Equal(t, "name cannot be empty", err.Error())
	})
}
