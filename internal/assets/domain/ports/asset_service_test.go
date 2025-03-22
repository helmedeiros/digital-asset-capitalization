package ports

import (
	"errors"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/stretchr/testify/assert"
)

// MockAssetService is a mock implementation of AssetService for testing
type MockAssetService struct {
	assets map[string]*domain.Asset
}

func NewMockAssetService() *MockAssetService {
	return &MockAssetService{
		assets: make(map[string]*domain.Asset),
	}
}

func (m *MockAssetService) CreateAsset(name, description string) error {
	if _, exists := m.assets[name]; exists {
		return errors.New("asset already exists")
	}
	m.assets[name] = &domain.Asset{
		Name:        name,
		Description: description,
	}
	return nil
}

func (m *MockAssetService) ListAssets() ([]*domain.Asset, error) {
	assets := make([]*domain.Asset, 0, len(m.assets))
	for _, asset := range m.assets {
		assets = append(assets, asset)
	}
	return assets, nil
}

func (m *MockAssetService) GetAsset(name string) (*domain.Asset, error) {
	if asset, exists := m.assets[name]; exists {
		return asset, nil
	}
	return nil, errors.New("asset not found")
}

func (m *MockAssetService) DeleteAsset(name string) error {
	if _, exists := m.assets[name]; !exists {
		return errors.New("asset not found")
	}
	delete(m.assets, name)
	return nil
}

func (m *MockAssetService) UpdateAsset(name, description string) error {
	if _, exists := m.assets[name]; !exists {
		return errors.New("asset not found")
	}
	m.assets[name].Description = description
	return nil
}

func (m *MockAssetService) UpdateDocumentation(assetName string) error {
	if _, exists := m.assets[assetName]; !exists {
		return errors.New("asset not found")
	}
	return nil
}

func (m *MockAssetService) IncrementTaskCount(name string) error {
	if asset, exists := m.assets[name]; exists {
		asset.AssociatedTaskCount++
		return nil
	}
	return errors.New("asset not found")
}

func (m *MockAssetService) DecrementTaskCount(name string) error {
	if asset, exists := m.assets[name]; exists {
		if asset.AssociatedTaskCount > 0 {
			asset.AssociatedTaskCount--
		}
		return nil
	}
	return errors.New("asset not found")
}

func TestAssetService(t *testing.T) {
	service := NewMockAssetService()

	t.Run("CreateAsset", func(t *testing.T) {
		err := service.CreateAsset("test-asset", "Test Description")
		assert.NoError(t, err)

		// Test duplicate creation
		err = service.CreateAsset("test-asset", "Duplicate")
		assert.Error(t, err)
		assert.Equal(t, "asset already exists", err.Error())
	})

	t.Run("ListAssets", func(t *testing.T) {
		assets, err := service.ListAssets()
		assert.NoError(t, err)
		assert.Len(t, assets, 1)
		assert.Equal(t, "test-asset", assets[0].Name)
	})

	t.Run("GetAsset", func(t *testing.T) {
		asset, err := service.GetAsset("test-asset")
		assert.NoError(t, err)
		assert.Equal(t, "test-asset", asset.Name)

		// Test non-existent asset
		asset, err = service.GetAsset("non-existent")
		assert.Error(t, err)
		assert.Equal(t, "asset not found", err.Error())
	})

	t.Run("UpdateAsset", func(t *testing.T) {
		err := service.UpdateAsset("test-asset", "Updated Description")
		assert.NoError(t, err)

		asset, _ := service.GetAsset("test-asset")
		assert.Equal(t, "Updated Description", asset.Description)

		// Test updating non-existent asset
		err = service.UpdateAsset("non-existent", "New Description")
		assert.Error(t, err)
		assert.Equal(t, "asset not found", err.Error())
	})

	t.Run("DeleteAsset", func(t *testing.T) {
		err := service.DeleteAsset("test-asset")
		assert.NoError(t, err)

		// Verify asset is deleted
		_, err = service.GetAsset("test-asset")
		assert.Error(t, err)

		// Test deleting non-existent asset
		err = service.DeleteAsset("non-existent")
		assert.Error(t, err)
		assert.Equal(t, "asset not found", err.Error())
	})

	t.Run("TaskCount Operations", func(t *testing.T) {
		// Create a new asset for task count tests
		service.CreateAsset("task-asset", "Task Test")

		// Test increment
		err := service.IncrementTaskCount("task-asset")
		assert.NoError(t, err)

		asset, _ := service.GetAsset("task-asset")
		assert.Equal(t, 1, asset.AssociatedTaskCount)

		// Test decrement
		err = service.DecrementTaskCount("task-asset")
		assert.NoError(t, err)

		asset, _ = service.GetAsset("task-asset")
		assert.Equal(t, 0, asset.AssociatedTaskCount)

		// Test decrement below zero
		err = service.DecrementTaskCount("task-asset")
		assert.NoError(t, err)
		asset, _ = service.GetAsset("task-asset")
		assert.Equal(t, 0, asset.AssociatedTaskCount)
	})

	t.Run("UpdateDocumentation", func(t *testing.T) {
		err := service.UpdateDocumentation("task-asset")
		assert.NoError(t, err)

		// Test non-existent asset
		err = service.UpdateDocumentation("non-existent")
		assert.Error(t, err)
		assert.Equal(t, "asset not found", err.Error())
	})
}
