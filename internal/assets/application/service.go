package application

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/confluence"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/keywords"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/llama"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/application/service"
)

// AssetServiceImpl implements the AssetService interface
type AssetServiceImpl struct {
	repo          ports.AssetRepository
	llama         LlamaClient
	confluence    ConfluenceAdapter
	configService *service.ConfigService
	idGenerator   ports.IDGenerator
}

// NewAssetService creates a new AssetService instance using shared configuration
func NewAssetService(repo ports.AssetRepository, configService *service.ConfigService, idGenerator ports.IDGenerator) AssetService {
	// Initialize LLaMA client
	llamaConfig := llama.DefaultConfig()
	llamaClient, err := llama.NewClient(llamaConfig)
	if err != nil {
		// Log the error but don't fail initialization
		fmt.Printf("Warning: Failed to initialize LLaMA client: %v\n", err)
	}

	// Create Confluence adapter with shared configuration
	confluenceAdapter, err := createConfluenceAdapter(configService, idGenerator)
	if err != nil {
		// Log the error but don't fail initialization
		fmt.Printf("Warning: Failed to initialize Confluence adapter: %v\n", err)
	}

	return &AssetServiceImpl{
		repo:          repo,
		llama:         llamaClient,
		confluence:    confluenceAdapter,
		configService: configService,
		idGenerator:   idGenerator,
	}
}

// NewAssetServiceLegacy creates a new AssetService instance using legacy environment variables
// Deprecated: Use NewAssetService with ConfigService instead
func NewAssetServiceLegacy(repo ports.AssetRepository, idGenerator ports.IDGenerator) AssetService {
	// Initialize LLaMA client
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
	confluenceAdapter := confluence.NewAdapter(config, idGenerator)

	return &AssetServiceImpl{
		repo:        repo,
		llama:       llamaClient,
		confluence:  confluenceAdapter,
		idGenerator: idGenerator,
	}
}

// createConfluenceAdapter creates a Confluence adapter using shared configuration
func createConfluenceAdapter(configService *service.ConfigService, idGenerator ports.IDGenerator) (ConfluenceAdapter, error) {
	if configService == nil {
		return nil, fmt.Errorf("config service is required")
	}

	jiraConfig, err := configService.GetJiraConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get Jira configuration: %w", err)
	}

	config := confluence.DefaultConfig()
	config.BaseURL = jiraConfig.BaseURL()
	config.Username = jiraConfig.Email()
	config.Token = jiraConfig.Token()

	return confluence.NewAdapter(config, idGenerator), nil
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
	id := s.idGenerator.GenerateID(name)
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
	// Use shared configuration if available, fallback to environment variables
	config := confluence.DefaultConfig()

	if s.configService != nil {
		jiraConfig, err := s.configService.GetJiraConfig()
		if err == nil {
			config.BaseURL = jiraConfig.BaseURL()
			config.Username = jiraConfig.Email()
			config.Token = jiraConfig.Token()
		} else {
			// Fallback to environment variables
			config.BaseURL = os.Getenv("JIRA_BASE_URL")
			config.Token = os.Getenv("JIRA_TOKEN")
		}
	} else {
		// Fallback to environment variables
		config.BaseURL = os.Getenv("JIRA_BASE_URL")
		config.Token = os.Getenv("JIRA_TOKEN")
	}

	// Configure space key with proper validation and normalization
	config.SpaceKey = s.normalizeSpaceKey(spaceKey)
	config.Label = label
	config.Debug = debug

	if config.BaseURL == "" {
		return nil, fmt.Errorf("Jira base URL is not configured. Please run 'assetcap config init' or set JIRA_BASE_URL environment variable")
	}
	if config.Token == "" {
		return nil, fmt.Errorf("Jira token is not configured. Please run 'assetcap config init' or set JIRA_TOKEN environment variable")
	}

	adapter := confluence.NewAdapter(config, s.idGenerator)
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

// AssignTeam assigns owning and contributing teams to an asset
func (s *AssetServiceImpl) AssignTeam(assetName, owningTeam string, contributingTeams []string) error {
	// Find the asset
	asset, err := s.repo.FindByName(assetName)
	if err != nil {
		return fmt.Errorf("failed to find asset: %w", err)
	}

	// Update owning team if provided
	if owningTeam != "" {
		if err := asset.SetOwningTeam(owningTeam); err != nil {
			return fmt.Errorf("failed to set owning team: %w", err)
		}
	}

	// Update contributing teams if provided
	if len(contributingTeams) > 0 {
		if err := asset.SetContributingTeams(contributingTeams); err != nil {
			return fmt.Errorf("failed to set contributing teams: %w", err)
		}
	}

	// Save the updated asset
	if err := s.repo.Save(asset); err != nil {
		return fmt.Errorf("failed to save asset: %w", err)
	}

	return nil
}

// GetAssetTeams returns team assignments for all assets
func (s *AssetServiceImpl) GetAssetTeams() ([]AssetTeamInfo, error) {
	assets, err := s.repo.FindAll()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve assets: %w", err)
	}

	var result []AssetTeamInfo
	for _, asset := range assets {
		// Only include assets that have team assignments
		if asset.GetOwningTeam() != "" || len(asset.GetContributingTeams()) > 0 {
			info := AssetTeamInfo{
				AssetName:         asset.Name,
				OwningTeam:        asset.GetOwningTeam(),
				ContributingTeams: asset.GetContributingTeams(),
			}
			result = append(result, info)
		}
	}

	return result, nil
}

// GetAssetTeamInfo returns team assignments for a specific asset
func (s *AssetServiceImpl) GetAssetTeamInfo(assetName string) (*AssetTeamInfo, error) {
	if assetName == "" {
		return nil, fmt.Errorf("asset name is required")
	}

	asset, err := s.repo.FindByName(assetName)
	if err != nil {
		return nil, fmt.Errorf("failed to find asset: %w", err)
	}

	info := &AssetTeamInfo{
		AssetName:         asset.Name,
		OwningTeam:        asset.GetOwningTeam(),
		ContributingTeams: asset.GetContributingTeams(),
	}

	return info, nil
}

// AddContributingTeam adds a contributing team to an asset
func (s *AssetServiceImpl) AddContributingTeam(assetName, teamName string) error {
	if assetName == "" {
		return fmt.Errorf("asset name is required")
	}
	if teamName == "" {
		return fmt.Errorf("team name is required")
	}

	// Find the asset
	asset, err := s.repo.FindByName(assetName)
	if err != nil {
		return fmt.Errorf("failed to find asset: %w", err)
	}

	// Add the contributing team
	if err := asset.AddContributingTeam(teamName); err != nil {
		return fmt.Errorf("failed to add contributing team: %w", err)
	}

	// Save the updated asset
	if err := s.repo.Save(asset); err != nil {
		return fmt.Errorf("failed to save asset: %w", err)
	}

	return nil
}

// RemoveContributingTeam removes a contributing team from an asset
func (s *AssetServiceImpl) RemoveContributingTeam(assetName, teamName string) error {
	if assetName == "" {
		return fmt.Errorf("asset name is required")
	}
	if teamName == "" {
		return fmt.Errorf("team name is required")
	}

	// Find the asset
	asset, err := s.repo.FindByName(assetName)
	if err != nil {
		return fmt.Errorf("failed to find asset: %w", err)
	}

	// Remove the contributing team
	if err := asset.RemoveContributingTeam(teamName); err != nil {
		return fmt.Errorf("failed to remove contributing team: %w", err)
	}

	// Save the updated asset
	if err := s.repo.Save(asset); err != nil {
		return fmt.Errorf("failed to save asset: %w", err)
	}

	return nil
}

// normalizeSpaceKey validates and normalizes the space key parameter
func (s *AssetServiceImpl) normalizeSpaceKey(spaceKey string) string {
	// Handle empty or wildcard as "all spaces"
	if spaceKey == "" || spaceKey == "*" {
		return ""
	}

	// Handle single space key
	if !strings.Contains(spaceKey, ",") {
		return strings.TrimSpace(spaceKey)
	}

	// Handle comma-separated space keys
	spaces := strings.Split(spaceKey, ",")
	var validSpaces []string
	spaceSet := make(map[string]bool)

	for _, space := range spaces {
		trimmedSpace := strings.TrimSpace(space)
		if trimmedSpace != "" && !spaceSet[trimmedSpace] {
			validSpaces = append(validSpaces, trimmedSpace)
			spaceSet[trimmedSpace] = true
		}
	}

	// Return empty string if no valid spaces (equivalent to "all spaces")
	if len(validSpaces) == 0 {
		return ""
	}

	// Return single space or comma-separated list
	return strings.Join(validSpaces, ",")
}

// PublishToConfluence publishes an asset as a new page in Confluence
func (s *AssetServiceImpl) PublishToConfluence(ctx context.Context, assetName, spaceKey string, dryRun, debug bool) (*PublishToConfluenceResult, error) {
	// Validate input
	if assetName == "" {
		return nil, fmt.Errorf("asset name is required")
	}
	if spaceKey == "" {
		return nil, fmt.Errorf("space key is required")
	}

	// Find the asset
	asset, err := s.repo.FindByName(assetName)
	if err != nil {
		return nil, fmt.Errorf("failed to find asset '%s': %w", assetName, err)
	}
	if asset == nil {
		return nil, fmt.Errorf("asset '%s' not found", assetName)
	}

	// Create Confluence adapter with shared configuration
	config := confluence.DefaultConfig()

	if s.configService != nil {
		jiraConfig, err := s.configService.GetJiraConfig()
		if err == nil {
			config.BaseURL = jiraConfig.BaseURL()
			config.Username = jiraConfig.Email()
			config.Token = jiraConfig.Token()
		} else {
			// Fallback to environment variables
			config.BaseURL = os.Getenv("JIRA_BASE_URL")
			config.Username = os.Getenv("JIRA_EMAIL")
			config.Token = os.Getenv("JIRA_TOKEN")
		}
	} else {
		// Fallback to environment variables
		config.BaseURL = os.Getenv("JIRA_BASE_URL")
		config.Username = os.Getenv("JIRA_EMAIL")
		config.Token = os.Getenv("JIRA_TOKEN")
	}

	config.Debug = debug

	if config.BaseURL == "" {
		return nil, fmt.Errorf("Jira base URL is not configured. Please run 'assetcap config init' or set JIRA_BASE_URL environment variable")
	}
	if config.Token == "" {
		return nil, fmt.Errorf("Jira token is not configured. Please run 'assetcap config init' or set JIRA_TOKEN environment variable")
	}

	// Create Confluence adapter
	adapter := confluence.NewAdapter(config, s.idGenerator)

	// Check if page already exists
	exists, existingPageID, err := adapter.PageExistsByTitle(ctx, spaceKey, asset.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to check if page exists: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("a page with title '%s' already exists in space '%s' (page ID: %s)", asset.Name, spaceKey, existingPageID)
	}

	// Generate page content
	pageContent := confluence.GeneratePageContent(asset)

	// Prepare labels
	assetLabel := s.getAssetLabel(asset)
	labels := []string{"cap-asset", assetLabel}

	// If dry-run, return preview
	if dryRun {
		return &PublishToConfluenceResult{
			AssetName:    asset.Name,
			SpaceKey:     spaceKey,
			Labels:       labels,
			Created:      false,
			Preview:      pageContent,
			DocLinkSaved: false,
		}, nil
	}

	// Create the page
	publishResult, err := adapter.CreatePage(ctx, asset.Name, spaceKey, pageContent)
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	// Add labels to the page
	labelsErr := adapter.AddLabels(ctx, publishResult.PageID, labels)
	if labelsErr != nil {
		// Log warning but don't fail - page was created
		fmt.Printf("Warning: failed to add labels to page %s: %v\n", publishResult.PageID, labelsErr)
	}

	// Update asset with DocLink
	docLinkSaved := false
	asset.DocLink = publishResult.PageURL
	asset.UpdatedAt = time.Now()
	asset.Version++
	if err := s.repo.Save(asset); err != nil {
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
func (s *AssetServiceImpl) getAssetLabel(asset *domain.Asset) string {
	// If the asset already has an ID in cap-asset-* format, use it
	if domain.IsValidAssetID(asset.ID) {
		return asset.ID
	}

	// Otherwise, generate one from the name
	return s.idGenerator.GenerateID(asset.Name)
}

// UpdateConfluencePage updates an existing Confluence page with the asset's current content
func (s *AssetServiceImpl) UpdateConfluencePage(ctx context.Context, assetName string, dryRun, debug bool) (*PublishToConfluenceResult, error) {
	// Validate input
	if assetName == "" {
		return nil, fmt.Errorf("asset name is required")
	}

	// Find the asset
	asset, err := s.repo.FindByName(assetName)
	if err != nil {
		return nil, fmt.Errorf("failed to find asset '%s': %w", assetName, err)
	}
	if asset == nil {
		return nil, fmt.Errorf("asset '%s' not found", assetName)
	}

	// Check that the asset has a DocLink (Confluence page URL)
	if asset.DocLink == "" {
		return nil, fmt.Errorf("asset '%s' does not have a Confluence page link. Use 'publish' to create a new page first", assetName)
	}

	// Extract page ID and space key from DocLink
	pageID, spaceKey, err := s.extractPageInfoFromDocLink(asset.DocLink)
	if err != nil {
		return nil, fmt.Errorf("failed to extract page info from DocLink: %w", err)
	}

	// Create Confluence adapter with shared configuration
	config := confluence.DefaultConfig()

	if s.configService != nil {
		jiraConfig, err := s.configService.GetJiraConfig()
		if err == nil {
			config.BaseURL = jiraConfig.BaseURL()
			config.Username = jiraConfig.Email()
			config.Token = jiraConfig.Token()
		} else {
			// Fallback to environment variables
			config.BaseURL = os.Getenv("JIRA_BASE_URL")
			config.Username = os.Getenv("JIRA_EMAIL")
			config.Token = os.Getenv("JIRA_TOKEN")
		}
	} else {
		// Fallback to environment variables
		config.BaseURL = os.Getenv("JIRA_BASE_URL")
		config.Username = os.Getenv("JIRA_EMAIL")
		config.Token = os.Getenv("JIRA_TOKEN")
	}

	config.Debug = debug

	if config.BaseURL == "" {
		return nil, fmt.Errorf("Jira base URL is not configured. Please run 'assetcap config init' or set JIRA_BASE_URL environment variable")
	}
	if config.Token == "" {
		return nil, fmt.Errorf("Jira token is not configured. Please run 'assetcap config init' or set JIRA_TOKEN environment variable")
	}

	// Create Confluence adapter
	adapter := confluence.NewAdapter(config, s.idGenerator)

	// Generate page content
	pageContent := confluence.GeneratePageContent(asset)

	// Prepare labels
	assetLabel := s.getAssetLabel(asset)
	labels := []string{"cap-asset", assetLabel}

	// If dry-run, return preview
	if dryRun {
		return &PublishToConfluenceResult{
			AssetName:    asset.Name,
			PageID:       pageID,
			SpaceKey:     spaceKey,
			Labels:       labels,
			Created:      false,
			Preview:      pageContent,
			DocLinkSaved: false,
		}, nil
	}

	// Update the page
	updateResult, err := adapter.UpdatePage(ctx, pageID, asset.Name, spaceKey, pageContent)
	if err != nil {
		return nil, fmt.Errorf("failed to update page: %w", err)
	}

	// Update asset timestamp and DocLink if it changed (e.g., page was moved)
	asset.UpdatedAt = time.Now()
	asset.Version++
	if updateResult.PageURL != "" && updateResult.PageURL != asset.DocLink {
		if debug {
			fmt.Printf("Updating DocLink from '%s' to '%s'\n", asset.DocLink, updateResult.PageURL)
		}
		asset.DocLink = updateResult.PageURL
	}
	docLinkSaved := false
	if err := s.repo.Save(asset); err != nil {
		fmt.Printf("Warning: failed to update asset: %v\n", err)
	} else {
		docLinkSaved = true
	}

	return &PublishToConfluenceResult{
		AssetName:    asset.Name,
		PageID:       updateResult.PageID,
		PageURL:      updateResult.PageURL,
		SpaceKey:     updateResult.SpaceKey,
		Labels:       labels,
		Created:      false, // This is an update
		PublishedAt:  time.Now(),
		DocLinkSaved: docLinkSaved,
	}, nil
}

// extractPageInfoFromDocLink extracts the page ID and space key from a Confluence page URL
func (s *AssetServiceImpl) extractPageInfoFromDocLink(docLink string) (pageID, spaceKey string, err error) {
	// Parse the URL to extract page ID and space key
	// Example: https://example.atlassian.net/wiki/spaces/SPACE/pages/123456/Page+Title
	parsedURL, err := url.Parse(docLink)
	if err != nil {
		return "", "", fmt.Errorf("invalid DocLink URL: %w", err)
	}

	pathParts := strings.Split(parsedURL.Path, "/")
	// Find the "spaces" and "pages" indices
	var spacesIdx, pagesIdx int = -1, -1
	for i, part := range pathParts {
		if part == "spaces" && i+1 < len(pathParts) {
			spacesIdx = i
		}
		if part == "pages" && i+1 < len(pathParts) {
			pagesIdx = i
		}
	}

	if spacesIdx == -1 || spacesIdx+1 >= len(pathParts) {
		return "", "", fmt.Errorf("could not find space key in DocLink: %s", docLink)
	}
	if pagesIdx == -1 || pagesIdx+1 >= len(pathParts) {
		return "", "", fmt.Errorf("could not find page ID in DocLink: %s", docLink)
	}

	spaceKey = pathParts[spacesIdx+1]
	pageID = pathParts[pagesIdx+1]

	return pageID, spaceKey, nil
}
