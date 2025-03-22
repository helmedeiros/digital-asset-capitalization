package usecase

import (
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// ListAssetsUseCase handles listing all assets
type ListAssetsUseCase struct {
	*AssetUseCase
}

// NewListAssetsUseCase creates a new list assets use case
func NewListAssetsUseCase(repo ports.AssetRepository) *ListAssetsUseCase {
	return &ListAssetsUseCase{
		AssetUseCase: NewAssetUseCase(repo),
	}
}

// Execute returns a list of all assets
func (uc *ListAssetsUseCase) Execute() ([]*domain.Asset, error) {
	return uc.repo.FindAll()
}
