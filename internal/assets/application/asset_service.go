package application

import (
	"context"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/confluence"
)

// AssetTeamInfo contains team information for an asset
type AssetTeamInfo struct {
	AssetName         string   `json:"asset_name"`
	OwningTeam        string   `json:"owning_team"`
	ContributingTeams []string `json:"contributing_teams"`
}

// PublishToConfluenceResult represents the result of the publish operation
type PublishToConfluenceResult struct {
	AssetName    string    // Name of the published asset
	PageID       string    // ID of the created page
	PageURL      string    // URL of the created page
	SpaceKey     string    // Space where page was created
	Labels       []string  // Labels added to the page
	Created      bool      // Whether the page was created (false for dry-run)
	Preview      string    // Preview of the content (for dry-run)
	PublishedAt  time.Time // When the page was published
	DocLinkSaved bool      // Whether the DocLink was saved to the asset
}

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
	// DeletePage deletes a page from Confluence by its ID
	DeletePage(ctx context.Context, pageID string) error
}

// AssetService defines the interface for asset management operations
type AssetService interface {
	// CreateAsset creates a new asset
	CreateAsset(name, description string) error
	// ListAssets returns a list of all assets
	ListAssets() ([]*domain.Asset, error)
	// GetAsset returns an asset by name
	GetAsset(identifier string) (*domain.Asset, error)
	// DeleteAsset deletes an asset by name, optionally deleting its Confluence page
	DeleteAsset(name string, deleteConfluencePage bool) error
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
	// GenerateKeywords generates keywords for an asset using LLaMA
	GenerateKeywords(name string) error
	// AssignTeam assigns owning and contributing teams to an asset
	AssignTeam(assetName, owningTeam string, contributingTeams []string) error
	// GetAssetTeams returns team assignments for all assets
	GetAssetTeams() ([]AssetTeamInfo, error)
	// GetAssetTeamInfo returns team assignments for a specific asset
	GetAssetTeamInfo(assetName string) (*AssetTeamInfo, error)
	// AddContributingTeam adds a contributing team to an asset
	AddContributingTeam(assetName, teamName string) error
	// RemoveContributingTeam removes a contributing team from an asset
	RemoveContributingTeam(assetName, teamName string) error
	// PublishToConfluence publishes an asset as a new page in Confluence
	PublishToConfluence(ctx context.Context, assetName, spaceKey, parentPageID string, dryRun, debug bool) (*PublishToConfluenceResult, error)
	// UpdateConfluencePage updates an existing Confluence page with the asset's current content
	UpdateConfluencePage(ctx context.Context, assetName string, dryRun, debug bool) (*PublishToConfluenceResult, error)
}
