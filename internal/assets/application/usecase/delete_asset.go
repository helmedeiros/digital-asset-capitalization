package usecase

import (
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// DeleteAssetUseCase handles deleting an asset
type DeleteAssetUseCase struct {
	*AssetUseCase
}

// NewDeleteAssetUseCase creates a new delete asset use case
func NewDeleteAssetUseCase(repo ports.AssetRepository) *DeleteAssetUseCase {
	return &DeleteAssetUseCase{
		AssetUseCase: NewAssetUseCase(repo),
	}
}

// Execute deletes an asset by name
func (uc *DeleteAssetUseCase) Execute(name string) error {
	// Check if asset exists before deleting
	_, err := uc.repo.FindByName(name)
	if err != nil {
		return fmt.Errorf("asset not found")
	}
	return uc.repo.Delete(name)
}
