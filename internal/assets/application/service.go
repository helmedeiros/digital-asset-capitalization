package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/common"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/confluence"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/llama"
)

// AssetService handles business logic for asset management
type AssetService struct {
	repo       ports.AssetRepository
	llama      ports.LlamaClient
	confluence ports.ConfluenceAdapter
}

// NewAssetService creates a new AssetService instance
func NewAssetService(repo ports.AssetRepository) ports.AssetService {
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

	return &AssetService{
		repo:       repo,
		llama:      llamaClient,
		confluence: confluenceAdapter,
	}
}

// CreateAsset creates a new asset with the given name and description
func (s *AssetService) CreateAsset(name, description string) error {
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
func (s *AssetService) ListAssets() ([]*domain.Asset, error) {
	return s.repo.FindAll()
}

// GetAsset returns an asset by name or ID
func (s *AssetService) GetAsset(identifier string) (*domain.Asset, error) {
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
func (s *AssetService) DeleteAsset(name string) error {
	return s.repo.Delete(name)
}

// UpdateAsset updates an asset's description
func (s *AssetService) UpdateAsset(name, description, why, benefits, how, metrics string) error {
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
func (s *AssetService) UpdateDocumentation(assetName string) error {
	asset, err := s.repo.FindByName(assetName)
	if err != nil {
		return fmt.Errorf("asset not found")
	}
	asset.LastDocUpdateAt = time.Now()
	asset.Version++
	return s.repo.Save(asset)
}

// IncrementTaskCount increments the task count for an asset
func (s *AssetService) IncrementTaskCount(name string) error {
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
func (s *AssetService) DecrementTaskCount(name string) error {
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
func (s *AssetService) SyncFromConfluence(spaceKey, label string, debug bool) (*domain.SyncResult, error) {
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
					"Status":      string(asset.Status),
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

// validateRequiredFields checks if all required fields are present in the asset
func validateRequiredFields(asset *domain.Asset) []string {
	var missingFields []string

	if asset.ID == "" {
		missingFields = append(missingFields, "ID")
	}
	if asset.Name == "" {
		missingFields = append(missingFields, "Name")
	}
	if asset.Description == "" {
		missingFields = append(missingFields, "Description")
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
	if asset.Why == "" {
		missingFields = append(missingFields, "Why")
	}
	if asset.Benefits == "" {
		missingFields = append(missingFields, "Benefits")
	}
	if asset.How == "" {
		missingFields = append(missingFields, "How")
	}
	if asset.Metrics == "" {
		missingFields = append(missingFields, "Metrics")
	}

	return missingFields
}

// generateID creates a unique ID for the asset based on its name and timestamp
func generateID(name string) string {
	hash := sha256.New()
	hash.Write([]byte(name))
	hash.Write([]byte(time.Now().String()))
	return hex.EncodeToString(hash.Sum(nil))[:16]
}

// EnrichAsset enriches a specific field of an asset using LLaMA 3
func (s *AssetService) EnrichAsset(name, field string) error {
	asset, err := s.GetAsset(name)
	if err != nil {
		return fmt.Errorf("failed to get asset: %w", err)
	}

	// Get the content from the DocLink
	if asset.DocLink == "" {
		return fmt.Errorf("asset has no DocLink")
	}

	if s.llama == nil {
		return fmt.Errorf("LLaMA client not initialized")
	}

	// Extract page ID from DocLink
	pageID := extractPageIDFromDocLink(asset.DocLink)
	if pageID == "" {
		return fmt.Errorf("could not extract page ID from DocLink: %s", asset.DocLink)
	}

	// Fetch page content
	page, err := s.confluence.FetchPage(context.Background(), pageID)
	if err != nil {
		return fmt.Errorf("failed to fetch page content: %w", err)
	}

	// Extract content from the page
	content := page.Body.Storage.Value

	enrichedContent, err := s.llama.EnrichContent(content, field, asset)
	if err != nil {
		return fmt.Errorf("failed to enrich content: %w", err)
	}

	// Update the asset based on the field
	switch field {
	case "description":
		asset.Description = enrichedContent
	default:
		return fmt.Errorf("unsupported field for enrichment: %s", field)
	}

	asset.UpdatedAt = time.Now()
	asset.Version++
	return s.repo.Save(asset)
}

// extractPageIDFromDocLink extracts the page ID from a Confluence DocLink
func extractPageIDFromDocLink(docLink string) string {
	// Handle different URL formats:
	// 1. https://domain.com/wiki/spaces/SPACE/pages/123456
	// 2. /wiki/spaces/SPACE/pages/123456
	// 3. /spaces/SPACE/pages/123456
	// 4. https://goeuro.atlassian.net/wiki/spaces/MZN/pages/2876997643/Omio+Flex

	// First, try to parse the URL
	u, err := url.Parse(docLink)
	if err != nil {
		return ""
	}

	// Split the path into parts
	parts := strings.Split(u.Path, "/")

	// Find the index of "pages" in the path
	pagesIndex := -1
	for i, part := range parts {
		if part == "pages" {
			pagesIndex = i
			break
		}
	}

	// If we found "pages", the next part should be the page ID
	if pagesIndex >= 0 && pagesIndex+1 < len(parts) {
		// The page ID is the next part after "pages"
		pageID := parts[pagesIndex+1]
		// If there are more parts (like a title), only take the ID part
		if strings.Contains(pageID, "+") {
			pageID = strings.Split(pageID, "+")[0]
		}
		return pageID
	}

	return ""
}
