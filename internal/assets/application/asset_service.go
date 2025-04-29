package application

import (
	"context"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/confluence"
)

// LlamaClient defines the interface for LLaMA operations
type LlamaClient interface {
	// EnrichContent enriches the given content for the specified field
	EnrichContent(content, field string, asset *domain.Asset) (string, error)
	// Close closes the client connection
	Close() error
}

// ConfluenceAdapter defines the interface for Confluence operations
type ConfluenceAdapter interface {
	// FetchPage fetches a page from Confluence
	FetchPage(ctx context.Context, pageID string) (*confluence.Page, error)
}

// AssetService defines the interface for asset management operations
type AssetService interface {
	// CreateAsset creates a new asset
	CreateAsset(name, description string) error
	// ListAssets returns a list of all assets
	ListAssets() ([]*domain.Asset, error)
	// GetAsset returns an asset by name
	GetAsset(identifier string) (*domain.Asset, error)
	// DeleteAsset deletes an asset by name
	DeleteAsset(name string) error
	// UpdateAsset updates an asset's name and description
	UpdateAsset(name, description, why, benefits, how, metrics string) error
	// UpdateDocumentation marks the documentation for an asset as updated
	UpdateDocumentation(assetName string) error
	// IncrementTaskCount increments the task count for an asset
	IncrementTaskCount(name string) error
	// DecrementTaskCount decrements the task count for an asset
	DecrementTaskCount(name string) error
	// SyncFromConfluence fetches assets from Confluence and updates the local repository
	SyncFromConfluence(spaceKey, label string, debug bool) (*domain.SyncResult, error)
	// EnrichAsset enriches a specific field of an asset using LLaMA 3
	EnrichAsset(name, field string) error
}
