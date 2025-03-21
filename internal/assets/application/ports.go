package application

import (
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// AssetRepository defines the interface for asset persistence
type AssetRepository interface {
	// Save saves an asset to the repository
	Save(asset *domain.Asset) error
	// FindByName finds an asset by its name
	FindByName(name string) (*domain.Asset, error)
	// FindAll returns all assets
	FindAll() ([]*domain.Asset, error)
	// Delete deletes an asset by name
	Delete(name string) error
}

// AssetService defines the application service for asset operations
type AssetService interface {
	// CreateAsset creates a new asset
	CreateAsset(name, description string) error
	// ListAssets returns a list of all assets
	ListAssets() ([]*domain.Asset, error)
	// GetAsset returns an asset by name
	GetAsset(name string) (*domain.Asset, error)
	// DeleteAsset deletes an asset by name
	DeleteAsset(name string) error
	// UpdateAsset updates an asset's name and description
	UpdateAsset(name, description string) error
	// UpdateDocumentation marks the documentation for an asset as updated
	UpdateDocumentation(assetName string) error
	// IncrementTaskCount increments the task count for an asset
	IncrementTaskCount(name string) error
	// DecrementTaskCount decrements the task count for an asset
	DecrementTaskCount(name string) error
}
