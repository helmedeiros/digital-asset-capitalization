package ports

import (
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// AssetRepository defines the interface for asset persistence
type AssetRepository interface {
	// Save saves an asset to the repository
	Save(asset *domain.Asset) error
	// FindByName finds an asset by its name
	FindByName(name string) (*domain.Asset, error)
	// FindByID finds an asset by its ID
	FindByID(id string) (*domain.Asset, error)
	// FindAll returns all assets
	FindAll() ([]*domain.Asset, error)
	// Delete deletes an asset by name
	Delete(name string) error
}
