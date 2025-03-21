package action

import (
	"fmt"
	"strings"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/model"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/storage"
)

// AssetManager handles asset-related operations
type AssetManager struct {
	assets  map[string]*model.Asset
	storage storage.Storage
}

// NewAssetManager creates a new asset manager and loads existing assets
func NewAssetManager(assetsDir, assetsFile string) *AssetManager {
	am := &AssetManager{
		storage: storage.NewJSONStorage(assetsDir, assetsFile),
	}

	// Load existing assets
	assets, err := am.storage.Load()
	if err != nil {
		fmt.Printf("Warning: Failed to load assets: %v\n", err)
		assets = make(map[string]*model.Asset)
	}

	am.assets = assets
	return am
}

// CreateAsset creates a new asset
func (am *AssetManager) CreateAsset(name, description string) error {
	if _, exists := am.assets[name]; exists {
		return fmt.Errorf("asset %s already exists", name)
	}

	asset, err := model.NewAsset(name, description)
	if err != nil {
		return fmt.Errorf("failed to create asset: %w", err)
	}

	am.assets[name] = asset
	return am.storage.Save(am.assets)
}

// AddContributionType adds a contribution type to an asset
func (am *AssetManager) AddContributionType(assetName, contributionType string) error {
	asset, exists := am.assets[assetName]
	if !exists {
		return fmt.Errorf("asset %s not found", assetName)
	}

	if err := asset.AddContributionType(contributionType); err != nil {
		return fmt.Errorf("failed to add contribution type: %w", err)
	}

	return am.storage.Save(am.assets)
}

// ListAssets returns a list of all assets
func (am *AssetManager) ListAssets() []string {
	var names []string
	for name := range am.assets {
		names = append(names, name)
	}
	return names
}

// GetAsset returns an asset by name
func (am *AssetManager) GetAsset(name string) (*model.Asset, error) {
	asset, exists := am.assets[name]
	if !exists {
		return nil, fmt.Errorf("asset %s not found", name)
	}
	return asset, nil
}

// UpdateDocumentation marks the documentation for an asset as updated
func (am *AssetManager) UpdateDocumentation(assetName string) error {
	asset, exists := am.assets[assetName]
	if !exists {
		return fmt.Errorf("asset %s not found", assetName)
	}

	asset.UpdateDocumentation()
	return am.storage.Save(am.assets)
}

// IncrementTaskCount increments the task count for an asset
func (am *AssetManager) IncrementTaskCount(assetName string) error {
	asset, exists := am.assets[assetName]
	if !exists {
		return fmt.Errorf("asset %s not found", assetName)
	}

	asset.IncrementTaskCount()
	return am.storage.Save(am.assets)
}

// DecrementTaskCount decrements the task count for an asset
func (am *AssetManager) DecrementTaskCount(assetName string) error {
	asset, exists := am.assets[assetName]
	if !exists {
		return fmt.Errorf("asset %s not found", assetName)
	}

	asset.DecrementTaskCount()
	return am.storage.Save(am.assets)
}

// FormatAssetList formats the list of assets for display
func FormatAssetList(assets []string) string {
	if len(assets) == 0 {
		return "No assets found"
	}

	var sb strings.Builder
	sb.WriteString("Assets:\n")
	for _, name := range assets {
		sb.WriteString(fmt.Sprintf("- %s\n", name))
	}
	return sb.String()
}
