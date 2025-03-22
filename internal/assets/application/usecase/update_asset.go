package usecase

import (
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// UpdateAssetUseCase handles updating an asset's description
type UpdateAssetUseCase struct {
	*AssetUseCase
}

// NewUpdateAssetUseCase creates a new update asset use case
func NewUpdateAssetUseCase(repo ports.AssetRepository) *UpdateAssetUseCase {
	return &UpdateAssetUseCase{
		AssetUseCase: NewAssetUseCase(repo),
	}
}

// Execute updates an asset's description
func (uc *UpdateAssetUseCase) Execute(name, description string) error {
	asset, err := uc.repo.FindByName(name)
	if err != nil {
		return fmt.Errorf("asset not found")
	}

	if err := asset.UpdateDescription(description); err != nil {
		return err
	}
	return uc.repo.Save(asset)
}
