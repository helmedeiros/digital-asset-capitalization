package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/confluence"
)

// PublishToConfluenceInput represents the input for publishing an asset to Confluence
type PublishToConfluenceInput struct {
	AssetName string // Name of the asset to publish
	SpaceKey  string // Confluence space key to publish to
	DryRun    bool   // If true, only preview what would be created
	Debug     bool   // Enable debug output
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

// ConfluencePublisher defines the interface for publishing to Confluence
type ConfluencePublisher interface {
	CreatePage(ctx context.Context, title, spaceKey, content string) (*confluence.PagePublishResult, error)
	AddLabels(ctx context.Context, pageID string, labels []string) error
	PageExistsByTitle(ctx context.Context, spaceKey, title string) (bool, string, error)
}

// PublishToConfluenceUseCase handles publishing assets to Confluence
type PublishToConfluenceUseCase struct {
	repo        ports.AssetRepository
	publisher   ConfluencePublisher
	idGenerator ports.IDGenerator
}

// NewPublishToConfluenceUseCase creates a new publish to Confluence use case
func NewPublishToConfluenceUseCase(
	repo ports.AssetRepository,
	publisher ConfluencePublisher,
	idGenerator ports.IDGenerator,
) *PublishToConfluenceUseCase {
	return &PublishToConfluenceUseCase{
		repo:        repo,
		publisher:   publisher,
		idGenerator: idGenerator,
	}
}

// Execute publishes an asset to Confluence
func (uc *PublishToConfluenceUseCase) Execute(ctx context.Context, input PublishToConfluenceInput) (*PublishToConfluenceResult, error) {
	// Validate input
	if input.AssetName == "" {
		return nil, fmt.Errorf("asset name is required")
	}
	if input.SpaceKey == "" {
		return nil, fmt.Errorf("space key is required")
	}

	// Find the asset
	asset, err := uc.repo.FindByName(input.AssetName)
	if err != nil {
		return nil, fmt.Errorf("failed to find asset '%s': %w", input.AssetName, err)
	}
	if asset == nil {
		return nil, fmt.Errorf("asset '%s' not found", input.AssetName)
	}

	// Check if page already exists
	exists, existingPageID, err := uc.publisher.PageExistsByTitle(ctx, input.SpaceKey, asset.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check if page exists: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("a page with title '%s' already exists in space '%s' (page ID: %s)", asset.Name, input.SpaceKey, existingPageID)
	}

	// Generate page content
	pageContent := confluence.GeneratePageContent(asset)

	// Prepare labels
	assetLabel := uc.getAssetLabel(asset)
	labels := []string{"cap-asset", assetLabel}

	// If dry-run, return preview
	if input.DryRun {
		return &PublishToConfluenceResult{
			AssetName:    asset.Name,
			SpaceKey:     input.SpaceKey,
			Labels:       labels,
			Created:      false,
			Preview:      pageContent,
			DocLinkSaved: false,
		}, nil
	}

	// Create the page
	publishResult, err := uc.publisher.CreatePage(ctx, asset.Name, input.SpaceKey, pageContent)
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	// Add labels to the page
	labelsErr := uc.publisher.AddLabels(ctx, publishResult.PageID, labels)
	if labelsErr != nil {
		// Log warning but don't fail - page was created
		fmt.Printf("Warning: failed to add labels to page %s: %v\n", publishResult.PageID, labelsErr)
	}

	// Update asset with DocLink
	docLinkSaved := false
	if err := uc.updateAssetDocLink(asset, publishResult.PageURL); err != nil {
		fmt.Printf("Warning: failed to update asset DocLink: %v\n", err)
	} else {
		docLinkSaved = true
	}

	return &PublishToConfluenceResult{
		AssetName:    asset.Name,
		PageID:       publishResult.PageID,
		PageURL:      publishResult.PageURL,
		SpaceKey:     publishResult.SpaceKey,
		Labels:       labels,
		Created:      true,
		PublishedAt:  time.Now(),
		DocLinkSaved: docLinkSaved,
	}, nil
}

// getAssetLabel returns the cap-asset-* label for the asset
func (uc *PublishToConfluenceUseCase) getAssetLabel(asset *domain.Asset) string {
	// If the asset already has an ID in cap-asset-* format, use it
	if domain.IsValidAssetID(asset.ID) {
		return asset.ID
	}

	// Otherwise, generate one from the name
	return uc.idGenerator.GenerateID(asset.Name)
}

// updateAssetDocLink updates the asset's DocLink and saves it
func (uc *PublishToConfluenceUseCase) updateAssetDocLink(asset *domain.Asset, docLink string) error {
	asset.DocLink = docLink
	asset.UpdatedAt = time.Now()
	asset.Version++

	return uc.repo.Save(asset)
}
