package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/helmedeiros/jira-time-allocator/internal/assets/model"
)

// Storage defines the interface for asset persistence
type Storage interface {
	// Load loads all assets from storage
	Load() (map[string]*model.Asset, error)
	// Save saves all assets to storage
	Save(assets map[string]*model.Asset) error
}

// JSONStorage implements Storage interface using JSON files
type JSONStorage struct {
	dir  string
	file string
}

// NewJSONStorage creates a new JSON storage instance
func NewJSONStorage(dir, file string) *JSONStorage {
	return &JSONStorage{
		dir:  dir,
		file: file,
	}
}

// Load implements Storage.Load
func (s *JSONStorage) Load() (map[string]*model.Asset, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	filePath := filepath.Join(s.dir, s.file)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*model.Asset), nil // Return empty map if file doesn't exist
		}
		return nil, fmt.Errorf("failed to read storage file: %w", err)
	}

	var assets map[string]*model.Asset
	if err := json.Unmarshal(data, &assets); err != nil {
		return nil, fmt.Errorf("failed to unmarshal assets: %w", err)
	}

	return assets, nil
}

// Save implements Storage.Save
func (s *JSONStorage) Save(assets map[string]*model.Asset) error {
	data, err := json.MarshalIndent(assets, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal assets: %w", err)
	}

	filePath := filepath.Join(s.dir, s.file)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write storage file: %w", err)
	}

	return nil
}
