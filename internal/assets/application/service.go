package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/common"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/confluence"
)

// AssetService handles business logic for asset management
type AssetService struct {
	repo ports.AssetRepository
}

// NewAssetService creates a new AssetService instance
func NewAssetService(repo ports.AssetRepository) ports.AssetService {
	return &AssetService{repo: repo}
}

// CreateAsset creates a new asset with the given name and description
func (s *AssetService) CreateAsset(name, description string) error {
	// Check if asset already exists by name
	if _, err := s.repo.FindByName(name); err == nil {
		return fmt.Errorf("asset with name '%s' already exists", name)
	}

	// Check if name matches any existing asset's ID
	if _, err := s.repo.FindByID(name); err == nil {
		return fmt.Errorf("cannot create asset with name '%s' as it matches an existing asset's ID", name)
	}

	// Generate ID and check if it already exists
	id := common.GenerateID(name)
	if _, err := s.repo.FindByID(id); err == nil {
		return fmt.Errorf("asset with ID '%s' already exists", id)
	}

	now := time.Now()
	asset := &domain.Asset{
		ID:              id,
		Name:            name,
		Description:     description,
		CreatedAt:       now,
		UpdatedAt:       now,
		LastDocUpdateAt: now,
		Version:         1,
	}
	return s.repo.Save(asset)
}

// ListAssets returns all assets in the repository
func (s *AssetService) ListAssets() ([]*domain.Asset, error) {
	return s.repo.FindAll()
}

// GetAsset returns an asset by name or ID
func (s *AssetService) GetAsset(identifier string) (*domain.Asset, error) {
	// First try to find by name
	asset, err := s.repo.FindByName(identifier)
	if err == nil {
		return asset, nil
	}

	// If not found by name, try to find by ID
	asset, err = s.repo.FindByID(identifier)
	if err != nil {
		return nil, fmt.Errorf("asset not found by name or ID: %s", identifier)
	}
	return asset, nil
}

// DeleteAsset deletes an asset by name
func (s *AssetService) DeleteAsset(name string) error {
	return s.repo.Delete(name)
}

// UpdateAsset updates an asset's description
func (s *AssetService) UpdateAsset(name, description string) error {
	if description == "" {
		return fmt.Errorf("asset description cannot be empty")
	}

	asset, err := s.repo.FindByName(name)
	if err != nil {
		return fmt.Errorf("asset not found")
	}
	asset.Description = description
	asset.UpdatedAt = time.Now()
	asset.Version++
	return s.repo.Save(asset)
}

// UpdateDocumentation marks the documentation for an asset as updated
func (s *AssetService) UpdateDocumentation(assetName string) error {
	asset, err := s.repo.FindByName(assetName)
	if err != nil {
		return fmt.Errorf("asset not found")
	}
	asset.LastDocUpdateAt = time.Now()
	asset.Version++
	return s.repo.Save(asset)
}

// IncrementTaskCount increments the task count for an asset
func (s *AssetService) IncrementTaskCount(name string) error {
	asset, err := s.repo.FindByName(name)
	if err != nil {
		return fmt.Errorf("asset not found")
	}
	asset.AssociatedTaskCount++
	asset.UpdatedAt = time.Now()
	asset.Version++
	return s.repo.Save(asset)
}

// DecrementTaskCount decrements the task count for an asset
func (s *AssetService) DecrementTaskCount(name string) error {
	asset, err := s.repo.FindByName(name)
	if err != nil {
		return fmt.Errorf("asset not found")
	}
	if asset.AssociatedTaskCount > 0 {
		asset.AssociatedTaskCount--
		asset.UpdatedAt = time.Now()
		asset.Version++
		return s.repo.Save(asset)
	}
	return fmt.Errorf("task count cannot be negative")
}

// SyncFromConfluence fetches assets from Confluence and updates the local repository
func (s *AssetService) SyncFromConfluence(spaceKey, label string, debug bool) error {
	config := confluence.DefaultConfig()

	// Get configuration from environment variables
	config.BaseURL = os.Getenv("JIRA_BASE_URL")
	config.SpaceKey = spaceKey
	config.Label = label
	config.Token = os.Getenv("JIRA_TOKEN")
	config.Debug = debug

	if config.BaseURL == "" {
		return fmt.Errorf("JIRA_BASE_URL environment variable must be set")
	}
	if config.Token == "" {
		return fmt.Errorf("JIRA_TOKEN environment variable must be set")
	}

	adapter := confluence.NewAdapter(config)
	assets, err := adapter.FetchAssets(context.Background())
	if err != nil {
		if strings.Contains(err.Error(), "no assets found with label") {
			return err
		}
		return fmt.Errorf("failed to fetch assets from Confluence: %v", err)
	}

	// Update local repository with fetched assets
	for _, asset := range assets {
		if err := s.repo.Save(asset); err != nil {
			return fmt.Errorf("failed to save asset %s: %v", asset.Name, err)
		}
	}

	return nil
}

// generateID creates a unique ID for the asset based on its name and timestamp
func generateID(name string) string {
	hash := sha256.New()
	hash.Write([]byte(name))
	hash.Write([]byte(time.Now().String()))
	return hex.EncodeToString(hash.Sum(nil))[:16]
}
