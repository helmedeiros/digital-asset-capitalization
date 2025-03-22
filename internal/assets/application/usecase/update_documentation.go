package usecase

import (
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// UpdateDocumentationUseCase handles updating an asset's documentation
type UpdateDocumentationUseCase struct {
	*AssetUseCase
}

// NewUpdateDocumentationUseCase creates a new update documentation use case
func NewUpdateDocumentationUseCase(repo ports.AssetRepository) *UpdateDocumentationUseCase {
	return &UpdateDocumentationUseCase{
		AssetUseCase: NewAssetUseCase(repo),
	}
}

// Execute marks the documentation for an asset as updated
func (uc *UpdateDocumentationUseCase) Execute(assetName string) error {
	asset, err := uc.repo.FindByName(assetName)
	if err != nil {
		return fmt.Errorf("asset not found")
	}

	if err := asset.UpdateDocumentation(); err != nil {
		return err
	}
	return uc.repo.Save(asset)
}
