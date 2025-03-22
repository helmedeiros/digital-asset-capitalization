package application

import (
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// assetService implements AssetService
type assetService struct {
	repo ports.AssetRepository
}

// NewAssetService creates a new asset service
func NewAssetService(repo ports.AssetRepository) ports.AssetService {
	return &assetService{
		repo: repo,
	}
}

// CreateAsset creates a new asset
func (s *assetService) CreateAsset(name, description string) error {
	// Check if asset already exists
	existing, err := s.repo.FindByName(name)
	if err == nil && existing != nil {
		return fmt.Errorf("asset already exists")
	}

	asset, err := domain.NewAsset(name, description)
	if err != nil {
		return err
	}
	return s.repo.Save(asset)
}

// ListAssets returns a list of all assets
func (s *assetService) ListAssets() ([]*domain.Asset, error) {
	return s.repo.FindAll()
}

// GetAsset returns an asset by name
func (s *assetService) GetAsset(name string) (*domain.Asset, error) {
	asset, err := s.repo.FindByName(name)
	if err != nil {
		return nil, fmt.Errorf("asset not found")
	}
	return asset, nil
}

// DeleteAsset deletes an asset by name
func (s *assetService) DeleteAsset(name string) error {
	return s.repo.Delete(name)
}

// UpdateAsset updates an asset's description
func (s *assetService) UpdateAsset(name, description string) error {
	if description == "" {
		return fmt.Errorf("asset description cannot be empty")
	}

	asset, err := s.repo.FindByName(name)
	if err != nil {
		return fmt.Errorf("asset not found")
	}

	if err := asset.UpdateDescription(description); err != nil {
		return err
	}
	return s.repo.Save(asset)
}

// IncrementTaskCount increments the task count for an asset
func (s *assetService) IncrementTaskCount(name string) error {
	asset, err := s.repo.FindByName(name)
	if err != nil {
		return fmt.Errorf("asset not found")
	}
	asset.IncrementTaskCount()
	return s.repo.Save(asset)
}

// DecrementTaskCount decrements the task count for an asset
func (s *assetService) DecrementTaskCount(name string) error {
	asset, err := s.repo.FindByName(name)
	if err != nil {
		return fmt.Errorf("asset not found")
	}
	if asset.AssociatedTaskCount == 0 {
		return fmt.Errorf("task count cannot be negative")
	}
	asset.DecrementTaskCount()
	return s.repo.Save(asset)
}

// UpdateDocumentation marks the documentation for an asset as updated
func (s *assetService) UpdateDocumentation(assetName string) error {
	asset, err := s.repo.FindByName(assetName)
	if err != nil {
		return fmt.Errorf("asset not found")
	}

	asset.UpdateDocumentation()
	return s.repo.Save(asset)
}
