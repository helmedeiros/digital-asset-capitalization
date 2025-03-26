package infrastructure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// JSONRepository implements AssetRepository using JSON files
type JSONRepository struct {
	dir  string
	file string
}

// RepositoryConfig holds configuration for the JSON repository
type RepositoryConfig struct {
	// Directory where assets will be stored
	Directory string
	// Filename for the assets JSON file
	Filename string
	// File permissions for the JSON file
	FileMode os.FileMode
	// Directory permissions for the storage directory
	DirMode os.FileMode
}

// DefaultConfig returns a default configuration for the repository
func DefaultConfig() RepositoryConfig {
	return RepositoryConfig{
		Directory: "data",
		Filename:  "assets.json",
		FileMode:  0644,
		DirMode:   0755,
	}
}

// NewJSONRepository creates a new JSON repository with the given configuration
func NewJSONRepository(config RepositoryConfig) ports.AssetRepository {
	return &JSONRepository{
		dir:  config.Directory,
		file: config.Filename,
	}
}

// Save saves an asset to the repository
func (r *JSONRepository) Save(asset *domain.Asset) error {
	if asset == nil {
		return fmt.Errorf("cannot save nil asset")
	}

	// Load existing assets
	assets, err := r.loadAssets()
	if err != nil {
		return fmt.Errorf("failed to load assets: %w", err)
	}

	// Update or add the asset
	assets[asset.Name] = asset

	// Save back to file
	return r.saveAssets(assets)
}

// FindByName finds an asset by its name
func (r *JSONRepository) FindByName(name string) (*domain.Asset, error) {
	if name == "" {
		return nil, fmt.Errorf("asset name cannot be empty")
	}

	assets, err := r.loadAssets()
	if err != nil {
		return nil, fmt.Errorf("failed to load assets: %w", err)
	}

	asset, exists := assets[name]
	if !exists {
		return nil, fmt.Errorf("asset %s not found", name)
	}

	return asset, nil
}

// FindAll returns all assets
func (r *JSONRepository) FindAll() ([]*domain.Asset, error) {
	assets, err := r.loadAssets()
	if err != nil {
		return nil, fmt.Errorf("failed to load assets: %w", err)
	}

	var result []*domain.Asset
	for _, asset := range assets {
		result = append(result, asset)
	}

	return result, nil
}

// Delete deletes an asset by name
func (r *JSONRepository) Delete(name string) error {
	if name == "" {
		return fmt.Errorf("asset name cannot be empty")
	}

	assets, err := r.loadAssets()
	if err != nil {
		return fmt.Errorf("failed to load assets: %w", err)
	}

	if _, exists := assets[name]; !exists {
		return fmt.Errorf("asset %s not found", name)
	}

	delete(assets, name)
	return r.saveAssets(assets)
}

// FindByID finds an asset by its ID
func (r *JSONRepository) FindByID(id string) (*domain.Asset, error) {
	if id == "" {
		return nil, fmt.Errorf("asset ID cannot be empty")
	}

	assets, err := r.loadAssets()
	if err != nil {
		return nil, fmt.Errorf("failed to load assets: %w", err)
	}

	// Search through all assets to find one with matching ID
	for _, asset := range assets {
		if asset.ID == id {
			return asset, nil
		}
	}

	return nil, fmt.Errorf("asset with ID %s not found", id)
}

// loadAssets loads all assets from the JSON file
func (r *JSONRepository) loadAssets() (map[string]*domain.Asset, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(r.dir, DefaultConfig().DirMode); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	filePath := filepath.Join(r.dir, r.file)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*domain.Asset), nil
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var assets map[string]*domain.Asset
	if err := json.Unmarshal(data, &assets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal assets: %w", err)
	}

	return assets, nil
}

// saveAssets saves all assets to the JSON file
func (r *JSONRepository) saveAssets(assets map[string]*domain.Asset) error {
	data, err := json.MarshalIndent(assets, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal assets: %w", err)
	}

	filePath := filepath.Join(r.dir, r.file)
	if err := os.WriteFile(filePath, data, DefaultConfig().FileMode); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
