package application

import (
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/application/usecase"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// assetService implements AssetService by composing use cases
type assetService struct {
	createAsset         *usecase.CreateAssetUseCase
	listAssets          *usecase.ListAssetsUseCase
	getAsset            *usecase.GetAssetUseCase
	deleteAsset         *usecase.DeleteAssetUseCase
	updateAsset         *usecase.UpdateAssetUseCase
	updateDocumentation *usecase.UpdateDocumentationUseCase
	incrementTaskCount  *usecase.IncrementTaskCountUseCase
	decrementTaskCount  *usecase.DecrementTaskCountUseCase
}

// NewAssetService creates a new asset service with all use cases
func NewAssetService(repo ports.AssetRepository) ports.AssetService {
	return &assetService{
		createAsset:         usecase.NewCreateAssetUseCase(repo),
		listAssets:          usecase.NewListAssetsUseCase(repo),
		getAsset:            usecase.NewGetAssetUseCase(repo),
		deleteAsset:         usecase.NewDeleteAssetUseCase(repo),
		updateAsset:         usecase.NewUpdateAssetUseCase(repo),
		updateDocumentation: usecase.NewUpdateDocumentationUseCase(repo),
		incrementTaskCount:  usecase.NewIncrementTaskCountUseCase(repo),
		decrementTaskCount:  usecase.NewDecrementTaskCountUseCase(repo),
	}
}

// CreateAsset creates a new asset
func (s *assetService) CreateAsset(name, description string) error {
	return s.createAsset.Execute(name, description)
}

// ListAssets returns a list of all assets
func (s *assetService) ListAssets() ([]*domain.Asset, error) {
	return s.listAssets.Execute()
}

// GetAsset returns an asset by name
func (s *assetService) GetAsset(name string) (*domain.Asset, error) {
	return s.getAsset.Execute(name)
}

// DeleteAsset deletes an asset by name
func (s *assetService) DeleteAsset(name string) error {
	return s.deleteAsset.Execute(name)
}

// UpdateAsset updates an asset's description
func (s *assetService) UpdateAsset(name, description string) error {
	return s.updateAsset.Execute(name, description)
}

// UpdateDocumentation marks the documentation for an asset as updated
func (s *assetService) UpdateDocumentation(assetName string) error {
	return s.updateDocumentation.Execute(assetName)
}

// IncrementTaskCount increments the task count for an asset
func (s *assetService) IncrementTaskCount(name string) error {
	return s.incrementTaskCount.Execute(name)
}

// DecrementTaskCount decrements the task count for an asset
func (s *assetService) DecrementTaskCount(name string) error {
	return s.decrementTaskCount.Execute(name)
}
