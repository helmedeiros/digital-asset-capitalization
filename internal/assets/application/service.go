package application

import (
	"fmt"

	"github.com/helmedeiros/jira-time-allocator/internal/assets/domain"
)

// assetService implements AssetService
type assetService struct {
	repo AssetRepository
}

// NewAssetService creates a new asset service
func NewAssetService(repo AssetRepository) AssetService {
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
		return fmt.Errorf("failed to create asset: %w", err)
	}

	return s.repo.Save(asset)
}

// AddContributionType adds a contribution type to an asset
func (s *assetService) AddContributionType(assetName, contributionType string) error {
	asset, err := s.repo.FindByName(assetName)
	if err != nil {
		return fmt.Errorf("asset not found")
	}

	if err := asset.AddContributionType(contributionType); err != nil {
		if err == domain.ErrInvalidContributionType {
			return fmt.Errorf("invalid contribution type")
		}
		return fmt.Errorf("failed to add contribution type: %w", err)
	}

	return s.repo.Save(asset)
}

// ListAssets returns a list of all assets
func (s *assetService) ListAssets() []string {
	assets, err := s.repo.FindAll()
	if err != nil {
		return nil
	}

	var names []string
	for _, asset := range assets {
		names = append(names, asset.Name)
	}
	return names
}

// GetAsset returns an asset by name
func (s *assetService) GetAsset(name string) (*domain.Asset, error) {
	return s.repo.FindByName(name)
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

// IncrementTaskCount increments the task count for an asset
func (s *assetService) IncrementTaskCount(assetName string) error {
	asset, err := s.repo.FindByName(assetName)
	if err != nil {
		return fmt.Errorf("asset not found")
	}

	asset.IncrementTaskCount()
	return s.repo.Save(asset)
}

// DecrementTaskCount decrements the task count for an asset
func (s *assetService) DecrementTaskCount(assetName string) error {
	asset, err := s.repo.FindByName(assetName)
	if err != nil {
		return fmt.Errorf("asset not found")
	}

	if asset.AssociatedTaskCount == 0 {
		return fmt.Errorf("task count cannot be negative")
	}

	asset.DecrementTaskCount()
	return s.repo.Save(asset)
}
