package usecase

import (
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// CreateAssetUseCase handles the creation of new assets
type CreateAssetUseCase struct {
	*AssetUseCase
}

// NewCreateAssetUseCase creates a new create asset use case
func NewCreateAssetUseCase(repo ports.AssetRepository) *CreateAssetUseCase {
	return &CreateAssetUseCase{
		AssetUseCase: NewAssetUseCase(repo),
	}
}

// Execute creates a new asset with the given name and description
func (uc *CreateAssetUseCase) Execute(name, description string) error {
	// Check if asset already exists
	existing, err := uc.repo.FindByName(name)
	if err == nil && existing != nil {
		return fmt.Errorf("asset already exists")
	}

	asset, err := domain.NewAsset(name, description)
	if err != nil {
		return err
	}
	return uc.repo.Save(asset)
}
