package usecase

import (
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// DecrementTaskCountUseCase handles decrementing an asset's task count
type DecrementTaskCountUseCase struct {
	*AssetUseCase
}

// NewDecrementTaskCountUseCase creates a new decrement task count use case
func NewDecrementTaskCountUseCase(repo ports.AssetRepository) *DecrementTaskCountUseCase {
	return &DecrementTaskCountUseCase{
		AssetUseCase: NewAssetUseCase(repo),
	}
}

// Execute decrements the task count for an asset
func (uc *DecrementTaskCountUseCase) Execute(name string) error {
	asset, err := uc.repo.FindByName(name)
	if err != nil {
		return fmt.Errorf("asset not found")
	}

	if err := asset.DecrementTaskCount(); err != nil {
		return err
	}
	return uc.repo.Save(asset)
}
