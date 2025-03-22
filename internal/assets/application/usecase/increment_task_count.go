package usecase

import (
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// IncrementTaskCountUseCase handles incrementing an asset's task count
type IncrementTaskCountUseCase struct {
	*AssetUseCase
}

// NewIncrementTaskCountUseCase creates a new increment task count use case
func NewIncrementTaskCountUseCase(repo ports.AssetRepository) *IncrementTaskCountUseCase {
	return &IncrementTaskCountUseCase{
		AssetUseCase: NewAssetUseCase(repo),
	}
}

// Execute increments the task count for an asset
func (uc *IncrementTaskCountUseCase) Execute(name string) error {
	asset, err := uc.repo.FindByName(name)
	if err != nil {
		return fmt.Errorf("asset not found")
	}

	if err := asset.IncrementTaskCount(); err != nil {
		return err
	}
	return uc.repo.Save(asset)
}
