package usecase

import (
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// AssetUseCase represents the base structure for all asset use cases
type AssetUseCase struct {
	repo ports.AssetRepository
}

// NewAssetUseCase creates a new asset use case with the given repository
func NewAssetUseCase(repo ports.AssetRepository) *AssetUseCase {
	return &AssetUseCase{
		repo: repo,
	}
}
