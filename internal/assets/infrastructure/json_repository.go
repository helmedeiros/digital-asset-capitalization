package infrastructure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/helmedeiros/jira-time-allocator/internal/assets/application"
	"github.com/helmedeiros/jira-time-allocator/internal/assets/domain"
)

// JSONRepository implements AssetRepository using JSON files
type JSONRepository struct {
	dir  string
	file string
}

// NewJSONRepository creates a new JSON repository
func NewJSONRepository(dir, file string) application.AssetRepository {
	return &JSONRepository{
		dir:  dir,
		file: file,
	}
}

// Save saves an asset to the repository
func (r *JSONRepository) Save(asset *domain.Asset) error {
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

// loadAssets loads all assets from the JSON file
func (r *JSONRepository) loadAssets() (map[string]*domain.Asset, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(r.dir, 0755); err != nil {
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
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
