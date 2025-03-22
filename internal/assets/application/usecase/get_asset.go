package usecase

import (
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// GetAssetUseCase handles retrieving a single asset
type GetAssetUseCase struct {
	*AssetUseCase
}

// NewGetAssetUseCase creates a new get asset use case
func NewGetAssetUseCase(repo ports.AssetRepository) *GetAssetUseCase {
	return &GetAssetUseCase{
		AssetUseCase: NewAssetUseCase(repo),
	}
}

// Execute returns an asset by name
func (uc *GetAssetUseCase) Execute(name string) (*domain.Asset, error) {
	asset, err := uc.repo.FindByName(name)
	if err != nil {
		return nil, fmt.Errorf("asset not found")
	}
	return asset, nil
}
