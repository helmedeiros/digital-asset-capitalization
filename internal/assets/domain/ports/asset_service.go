package ports

import (
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

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
	// SyncFromConfluence fetches assets from Confluence and updates the local repository
	SyncFromConfluence(spaceKey, label string) error
}
