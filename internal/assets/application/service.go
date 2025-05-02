package application

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/common"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/confluence"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/keywords"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/llama"
)

// AssetServiceImpl implements the AssetService interface
type AssetServiceImpl struct {
	repo       ports.AssetRepository
	llama      LlamaClient
	confluence ConfluenceAdapter
}

// NewAssetService creates a new AssetService instance
func NewAssetService(repo ports.AssetRepository) AssetService {
	llamaConfig := llama.DefaultConfig()
	llamaClient, err := llama.NewClient(llamaConfig)
	if err != nil {
		// Log the error but don't fail initialization
		fmt.Printf("Warning: Failed to initialize LLaMA client: %v\n", err)
	}

	// Create Confluence adapter with default config
	config := confluence.DefaultConfig()
	config.BaseURL = os.Getenv("JIRA_BASE_URL")
	config.Token = os.Getenv("JIRA_TOKEN")
	confluenceAdapter := confluence.NewAdapter(config)

	return &AssetServiceImpl{
		repo:       repo,
		llama:      llamaClient,
		confluence: confluenceAdapter,
	}
}

// CreateAsset creates a new asset with the given name and description
func (s *AssetServiceImpl) CreateAsset(name, description string) error {
	// Check if asset already exists by name
	if _, err := s.repo.FindByName(name); err == nil {
		return fmt.Errorf("asset with name '%s' already exists", name)
	}

	// Check if name matches any existing asset's ID
	if _, err := s.repo.FindByID(name); err == nil {
		return fmt.Errorf("cannot create asset with name '%s' as it matches an existing asset's ID", name)
	}

	// Generate ID and check if it already exists
	id := common.GenerateID(name)
	if _, err := s.repo.FindByID(id); err == nil {
		return fmt.Errorf("asset with ID '%s' already exists", id)
	}

	now := time.Now()
	asset := &domain.Asset{
		ID:              id,
		Name:            name,
		Description:     description,
		CreatedAt:       now,
		UpdatedAt:       now,
		LastDocUpdateAt: now,
		Version:         1,
	}
	return s.repo.Save(asset)
}

// ListAssets returns all assets in the repository
func (s *AssetServiceImpl) ListAssets() ([]*domain.Asset, error) {
	return s.repo.FindAll()
}

// GetAsset returns an asset by name or ID
func (s *AssetServiceImpl) GetAsset(identifier string) (*domain.Asset, error) {
	// First try to find by name
	asset, err := s.repo.FindByName(identifier)
	if err == nil {
		return asset, nil
	}

	// If not found by name, try to find by ID
	asset, err = s.repo.FindByID(identifier)
	if err != nil {
		return nil, fmt.Errorf("asset not found by name or ID: %s", identifier)
	}
	return asset, nil
}

// DeleteAsset deletes an asset by name
func (s *AssetServiceImpl) DeleteAsset(name string) error {
	return s.repo.Delete(name)
}

// UpdateAsset updates an asset's description
func (s *AssetServiceImpl) UpdateAsset(name, description, why, benefits, how, metrics string) error {
	if description == "" {
		return fmt.Errorf("asset description cannot be empty")
	}

	asset, err := s.repo.FindByName(name)
	if err != nil {
		return fmt.Errorf("asset not found")
	}
	asset.Description = description
	asset.Why = why
	asset.Benefits = benefits
	asset.How = how
	asset.Metrics = metrics
	asset.UpdatedAt = time.Now()
	asset.Version++
	return s.repo.Save(asset)
}

// UpdateDocumentation marks the documentation for an asset as updated
func (s *AssetServiceImpl) UpdateDocumentation(assetName string) error {
	asset, err := s.repo.FindByName(assetName)
	if err != nil {
		return fmt.Errorf("asset not found")
	}
	asset.LastDocUpdateAt = time.Now()
	asset.Version++
	return s.repo.Save(asset)
}

// IncrementTaskCount increments the task count for an asset
func (s *AssetServiceImpl) IncrementTaskCount(name string) error {
	asset, err := s.repo.FindByName(name)
	if err != nil {
		return fmt.Errorf("asset not found")
	}
	asset.AssociatedTaskCount++
	asset.UpdatedAt = time.Now()
	asset.Version++
	return s.repo.Save(asset)
}

// DecrementTaskCount decrements the task count for an asset
func (s *AssetServiceImpl) DecrementTaskCount(name string) error {
	asset, err := s.repo.FindByName(name)
	if err != nil {
		return fmt.Errorf("asset not found")
	}
	if asset.AssociatedTaskCount > 0 {
		asset.AssociatedTaskCount--
		asset.UpdatedAt = time.Now()
		asset.Version++
		return s.repo.Save(asset)
	}
	return fmt.Errorf("task count cannot be negative")
}

// SyncFromConfluence fetches assets from Confluence and updates the local repository
func (s *AssetServiceImpl) SyncFromConfluence(spaceKey, label string, debug bool) (*domain.SyncResult, error) {
	config := confluence.DefaultConfig()

	// Get configuration from environment variables
	config.BaseURL = os.Getenv("JIRA_BASE_URL")
	config.SpaceKey = spaceKey
	config.Label = label
	config.Token = os.Getenv("JIRA_TOKEN")
	config.Debug = debug

	if config.BaseURL == "" {
		return nil, fmt.Errorf("JIRA_BASE_URL environment variable must be set")
	}
	if config.Token == "" {
		return nil, fmt.Errorf("JIRA_TOKEN environment variable must be set")
	}

	adapter := confluence.NewAdapter(config)
	assets, err := adapter.FetchAssets(context.Background())
	if err != nil {
		if strings.Contains(err.Error(), "no assets found with label") {
			return nil, err
		}
		return nil, fmt.Errorf("failed to fetch assets from Confluence: %v", err)
	}

	result := domain.NewSyncResult()

	// Update local repository with fetched assets
	for _, asset := range assets {
		missingFields := validateRequiredFields(asset)
		if len(missingFields) > 0 {
			notSynced := &domain.NotSyncedAsset{
				Name:          asset.Name,
				MissingFields: missingFields,
				AvailableFields: map[string]string{
					"ID":          asset.ID,
					"Name":        asset.Name,
					"Description": asset.Description,
					"LaunchDate":  asset.LaunchDate.Format("2006-01-02"),
					"Status":      asset.Status,
					"DocLink":     asset.DocLink,
					"Why":         asset.Why,
					"Benefits":    asset.Benefits,
					"How":         asset.How,
					"Metrics":     asset.Metrics,
				},
			}
			result.NotSyncedAssets = append(result.NotSyncedAssets, notSynced)
			continue
		}

		if err := s.repo.Save(asset); err != nil {
			return nil, fmt.Errorf("failed to save asset %s: %v", asset.Name, err)
		}
		result.SyncedAssets = append(result.SyncedAssets, asset)
	}

	return result, nil
}

// EnrichAsset enriches a specific field of an asset using LLaMA 3
func (s *AssetServiceImpl) EnrichAsset(name, field string) error {
	// Get the asset
	asset, err := s.GetAsset(name)
	if err != nil {
		return fmt.Errorf("failed to get asset: %w", err)
	}

	// Get the content to enrich based on the field
	var content string
	switch field {
	case "description":
		content = asset.Description
	case "why":
		content = asset.Why
	case "benefits":
		content = asset.Benefits
	case "how":
		content = asset.How
	case "metrics":
		content = asset.Metrics
	default:
		return fmt.Errorf("failed to enrich content: unsupported field for enrichment: %s", field)
	}

	// Enrich the content
	enrichedContent, err := s.llama.EnrichContent(content, field, asset)
	if err != nil {
		return fmt.Errorf("failed to enrich content: %w", err)
	}

	// Update the asset with the enriched content
	switch field {
	case "description":
		asset.Description = enrichedContent
	case "why":
		asset.Why = enrichedContent
	case "benefits":
		asset.Benefits = enrichedContent
	case "how":
		asset.How = enrichedContent
	case "metrics":
		asset.Metrics = enrichedContent
	}

	asset.UpdatedAt = time.Now()
	asset.Version++

	// Save the updated asset
	return s.repo.Save(asset)
}

// GenerateKeywords generates keywords for an asset using LLaMA
func (s *AssetServiceImpl) GenerateKeywords(name string) error {
	// Get the asset
	asset, err := s.GetAsset(name)
	if err != nil {
		return fmt.Errorf("failed to get asset: %w", err)
	}

	// Create keyword generator
	generator := keywords.NewGenerator(s.llama)

	// Generate keywords
	generatedKeywords, err := generator.GenerateKeywords(asset)
	if err != nil {
		return fmt.Errorf("failed to generate keywords: %w", err)
	}

	// Update asset with new keywords
	asset.Keywords = generatedKeywords
	asset.UpdatedAt = time.Now()
	asset.Version++

	// Save the updated asset
	if err := s.repo.Save(asset); err != nil {
		return fmt.Errorf("failed to save asset: %w", err)
	}
	return nil
}

// Helper function to validate required fields
func validateRequiredFields(asset *domain.Asset) []string {
	var missingFields []string

	if asset.Name == "" {
		missingFields = append(missingFields, "Name")
	}
	if asset.Description == "" {
		missingFields = append(missingFields, "Description")
	}
	if asset.ID == "" {
		missingFields = append(missingFields, "ID")
	}
	if asset.LaunchDate.IsZero() {
		missingFields = append(missingFields, "LaunchDate")
	}
	if asset.Status == "" {
		missingFields = append(missingFields, "Status")
	}
	if asset.DocLink == "" {
		missingFields = append(missingFields, "DocLink")
	}

	return missingFields
}

// Helper function to extract page ID from Confluence doc link
func extractPageIDFromDocLink(docLink string) string {
	parsedURL, err := url.Parse(docLink)
	if err != nil {
		return ""
	}

	// Extract page ID from URL path
	pathParts := strings.Split(parsedURL.Path, "/")
	for i, part := range pathParts {
		if part == "pages" && i+1 < len(pathParts) {
			return pathParts[i+1]
		}
	}

	return ""
}
