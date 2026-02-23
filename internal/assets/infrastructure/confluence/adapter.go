package confluence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
)

// Page represents a page in Confluence
type Page struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Space struct {
		Key string `json:"key"`
	} `json:"space"`
	Version struct {
		Number int `json:"number"`
	} `json:"version"`
	Body struct {
		Storage struct {
			Value string `json:"value"`
		} `json:"storage"`
	} `json:"body"`
	Links struct {
		WebUI string `json:"webui"`
	} `json:"_links"`
	Metadata struct {
		Labels struct {
			Results []struct {
				Name string `json:"name"`
			} `json:"results"`
		} `json:"labels"`
	} `json:"metadata"`
}

// Response represents the response from the Confluence API
type Response struct {
	Results []Page `json:"results"`
	Links   struct {
		Next string `json:"next"`
	} `json:"_links"`
}

// Space represents a space in Confluence
type Space struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

// SpaceResponse represents the response from the Confluence API for spaces
type SpaceResponse struct {
	Results []Space `json:"results"`
}

// Adapter handles communication with Confluence API
type Adapter struct {
	config      *Config
	httpClient  *http.Client
	idGenerator ports.IDGenerator
}

// NewAdapter creates a new Confluence adapter
func NewAdapter(config *Config, idGenerator ports.IDGenerator) *Adapter {
	return &Adapter{
		config:      config,
		idGenerator: idGenerator,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (a *Adapter) getSpaceID(ctx context.Context) (string, error) {
	baseURL := strings.TrimRight(a.config.BaseURL, "/")
	url := fmt.Sprintf("%s/wiki/api/v2/spaces?keys=%s", baseURL, a.config.SpaceKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Set authentication header using Basic auth
	req.SetBasicAuth(a.config.Username, a.config.Token)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if a.config.Debug {
		fmt.Printf("Space response status: %d\nResponse body: %s\n", resp.StatusCode, string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var result SpaceResponse
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	if len(result.Results) == 0 {
		return "", fmt.Errorf("space not found: %s", a.config.SpaceKey)
	}

	return result.Results[0].ID, nil
}

func (a *Adapter) buildSearchURL(spaceID string) string {
	baseURL := strings.TrimRight(a.config.BaseURL, "/")
	searchURL := baseURL + "/wiki/api/v2/pages"

	query := url.Values{}
	query.Add("space-id", spaceID)
	query.Add("expand", "version,metadata.labels")
	query.Add("limit", fmt.Sprintf("%d", a.config.MaxResults))

	return searchURL + "?" + query.Encode()
}

// buildCQLQuery constructs the CQL query with proper space filtering
func (a *Adapter) buildCQLQuery() string {
	// Base query for pages with the specified label
	cqlParts := []string{
		"type=page",
		fmt.Sprintf("label=\"%s\"", a.config.Label),
	}

	// Add space filtering based on SpaceKey configuration
	if a.config.SpaceKey != "" && a.config.SpaceKey != "*" {
		// Check if it's multiple spaces (comma-separated)
		if strings.Contains(a.config.SpaceKey, ",") {
			// Multiple spaces: use IN operator
			spaces := strings.Split(a.config.SpaceKey, ",")
			var quotedSpaces []string
			for _, space := range spaces {
				trimmedSpace := strings.TrimSpace(space)
				if trimmedSpace != "" {
					quotedSpaces = append(quotedSpaces, fmt.Sprintf("\"%s\"", trimmedSpace))
				}
			}
			if len(quotedSpaces) > 0 {
				cqlParts = append(cqlParts, fmt.Sprintf("space in (%s)", strings.Join(quotedSpaces, ", ")))
			}
		} else {
			// Single space
			cqlParts = append(cqlParts, fmt.Sprintf("space=\"%s\"", strings.TrimSpace(a.config.SpaceKey)))
		}
	}
	// If SpaceKey is empty or "*", no space filter is added (search all spaces)

	// Join parts with AND and URL encode
	cqlQuery := strings.Join(cqlParts, " AND ")
	return url.QueryEscape(cqlQuery)
}

// FetchAssets retrieves assets from Confluence
func (a *Adapter) FetchAssets(ctx context.Context) ([]*domain.Asset, error) {
	baseURL := strings.TrimRight(a.config.BaseURL, "/")

	// Build CQL query with proper space filtering
	cqlQuery := a.buildCQLQuery()
	url := fmt.Sprintf("%s/wiki/rest/api/content/search?cql=%s&expand=version,metadata.labels&limit=%d",
		baseURL, cqlQuery, a.config.MaxResults)
	if a.config.Debug {
		fmt.Printf("Fetching pages from URL: %s\n", url)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set authentication header using Basic auth
	req.SetBasicAuth(a.config.Username, a.config.Token)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if a.config.Debug {
		fmt.Printf("Response status: %d\nResponse body: %s\n", resp.StatusCode, string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var result Response
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if len(result.Results) == 0 {
		return nil, fmt.Errorf("no assets found with label '%s' in space '%s'", a.config.Label, a.config.SpaceKey)
	}

	// Convert pages to assets
	var assets = make([]*domain.Asset, 0, len(result.Results))
	for _, page := range result.Results {
		// Fetch page content
		contentURL := fmt.Sprintf("%s/wiki/rest/api/content/%s?expand=body.storage,version,metadata.labels",
			baseURL, page.ID)
		contentReq, err := http.NewRequestWithContext(ctx, "GET", contentURL, nil)
		if err != nil {
			if a.config.Debug {
				fmt.Printf("Warning: failed to create request for page %s: %v\n", page.Title, err)
			}
			continue
		}

		contentReq.SetBasicAuth(a.config.Username, a.config.Token)
		contentReq.Header.Set("Accept", "application/json")

		contentResp, err := client.Do(contentReq)
		if err != nil {
			if a.config.Debug {
				fmt.Printf("Warning: failed to fetch content for page %s: %v\n", page.Title, err)
			}
			continue
		}
		defer contentResp.Body.Close()

		contentBody, _ := io.ReadAll(contentResp.Body)
		if a.config.Debug {
			fmt.Printf("Content response for page %s: %s\n", page.Title, string(contentBody))
		}

		if contentResp.StatusCode != http.StatusOK {
			if a.config.Debug {
				fmt.Printf("Warning: failed to fetch content for page %s: status %d\n", page.Title, contentResp.StatusCode)
			}
			continue
		}

		var contentPage Page
		var decodeErr error
		if decodeErr = json.NewDecoder(bytes.NewReader(contentBody)).Decode(&contentPage); decodeErr != nil {
			return nil, fmt.Errorf("failed to decode content page: %w", decodeErr)
		}

		if a.config.Debug {
			fmt.Printf("Labels for page %s:\n", contentPage.Title)
			for _, label := range contentPage.Metadata.Labels.Results {
				fmt.Printf("  - %s\n", label.Name)
			}
		}

		asset, err := a.convertPageToAsset(contentPage)
		if err != nil {
			if a.config.Debug {
				fmt.Printf("Warning: failed to convert page %s to asset: %v\n", page.Title, err)
			}
			continue
		}
		assets = append(assets, asset)
	}

	return assets, nil
}

// convertPageToAsset converts a Confluence page to an Asset
func (a *Adapter) convertPageToAsset(page Page) (*domain.Asset, error) {
	metadata, err := a.extractMetadata(page.Body.Storage.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata: %w", err)
	}

	// First try to get the identifier from the page's metadata labels
	for _, label := range page.Metadata.Labels.Results {
		if strings.HasPrefix(label.Name, "cap-asset-") {
			metadata.Identifier = label.Name
			break
		}
	}

	// If no identifier was found in the metadata labels, try to get it from the content
	if metadata.Identifier == "" {
		metadata.Identifier = extractAssetIdentifier(page.Body.Storage.Value)
	}

	// If still no identifier, generate one
	if metadata.Identifier == "" {
		metadata.Identifier = a.idGenerator.GenerateID(page.Title)
	}

	// Ensure we have the full URL for DocLink
	docLink := page.Links.WebUI
	if !strings.HasPrefix(docLink, "http") {
		baseURL := strings.TrimRight(a.config.BaseURL, "/")
		// Add /wiki if it's not already in the path
		if !strings.Contains(docLink, "/wiki/") {
			docLink = "/wiki" + docLink
		}
		docLink = baseURL + docLink
	}

	now := time.Now()
	asset := &domain.Asset{
		ID:               metadata.Identifier,
		Name:             page.Title,
		Description:      metadata.Description,
		Why:              metadata.Why,
		Benefits:         metadata.Benefits,
		How:              metadata.How,
		Metrics:          metadata.Metrics,
		CreatedAt:        now,
		UpdatedAt:        now,
		LastDocUpdateAt:  now,
		Version:          1,
		Platform:         metadata.Platform,
		Status:           metadata.Status,
		LaunchDate:       metadata.LaunchDate,
		IsRolledOut100:   metadata.IsRolledOut100,
		Keywords:         metadata.Keywords,
		DocLink:          docLink,
		ConfluencePageID: page.ID,
	}

	return asset, nil
}

// FetchPage retrieves a single page from Confluence by its ID
func (a *Adapter) FetchPage(ctx context.Context, pageID string) (*Page, error) {
	baseURL := strings.TrimRight(a.config.BaseURL, "/")
	url := fmt.Sprintf("%s/wiki/rest/api/content/%s?expand=body.storage,version,metadata.labels",
		baseURL, pageID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Set authentication header using Basic auth
	req.SetBasicAuth(a.config.Username, a.config.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var page Page
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &page, nil
}

// CreatePage creates a new page in Confluence, optionally under a parent page
func (a *Adapter) CreatePage(ctx context.Context, title, spaceKey, content, parentPageID string) (*PagePublishResult, error) {
	baseURL := strings.TrimRight(a.config.BaseURL, "/")
	apiURL := fmt.Sprintf("%s/wiki/rest/api/content", baseURL)

	// Create request body
	reqBody := CreatePageRequest{
		Type:  "page",
		Title: title,
		Space: CreatePageSpace{Key: spaceKey},
		Body: CreatePageBody{
			Storage: CreatePageStorage{
				Value:          content,
				Representation: "storage",
			},
		},
	}

	if parentPageID != "" {
		reqBody.Ancestors = []CreatePageAncestor{{ID: parentPageID}}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %v", err)
	}

	if a.config.Debug {
		fmt.Printf("Creating page in space %s with title: %s\n", spaceKey, title)
		fmt.Printf("Request URL: %s\n", apiURL)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.SetBasicAuth(a.config.Username, a.config.Token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if a.config.Debug {
		fmt.Printf("Create page response status: %d\nResponse body: %s\n", resp.StatusCode, string(body))
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create page: status %d, body: %s", resp.StatusCode, string(body))
	}

	var page Page
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&page); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// Build full URL
	pageURL := page.Links.WebUI
	if !strings.HasPrefix(pageURL, "http") {
		pageURL = baseURL + "/wiki" + pageURL
	}

	return &PagePublishResult{
		PageID:   page.ID,
		PageURL:  pageURL,
		SpaceKey: spaceKey,
		Title:    page.Title,
		Created:  true,
	}, nil
}

// AddLabels adds labels to a Confluence page
func (a *Adapter) AddLabels(ctx context.Context, pageID string, labels []string) error {
	if len(labels) == 0 {
		return nil
	}

	baseURL := strings.TrimRight(a.config.BaseURL, "/")
	apiURL := fmt.Sprintf("%s/wiki/rest/api/content/%s/label", baseURL, pageID)

	// Create request body as array of label objects
	labelRequests := make([]LabelRequest, 0, len(labels))
	for _, label := range labels {
		labelRequests = append(labelRequests, LabelRequest{
			Prefix: "global",
			Name:   label,
		})
	}

	jsonBody, err := json.Marshal(labelRequests)
	if err != nil {
		return fmt.Errorf("failed to marshal label request: %v", err)
	}

	if a.config.Debug {
		fmt.Printf("Adding labels to page %s: %v\n", pageID, labels)
		fmt.Printf("Request URL: %s\n", apiURL)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.SetBasicAuth(a.config.Username, a.config.Token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if a.config.Debug {
		fmt.Printf("Add labels response status: %d\nResponse body: %s\n", resp.StatusCode, string(body))
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to add labels: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdatePage updates an existing page in Confluence
func (a *Adapter) UpdatePage(ctx context.Context, pageID, title, spaceKey, content string) (*PagePublishResult, error) {
	// First, get the current page to retrieve its version and actual space key
	page, err := a.FetchPage(ctx, pageID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page for update: %v", err)
	}

	// Use the actual space key from the page (in case it was moved)
	actualSpaceKey := page.Space.Key
	if actualSpaceKey == "" {
		actualSpaceKey = spaceKey // Fallback to provided space key
	}

	baseURL := strings.TrimRight(a.config.BaseURL, "/")
	apiURL := fmt.Sprintf("%s/wiki/rest/api/content/%s", baseURL, pageID)

	// Create update request body with incremented version
	reqBody := map[string]interface{}{
		"type":  "page",
		"title": title,
		"space": map[string]string{"key": actualSpaceKey},
		"body": map[string]interface{}{
			"storage": map[string]string{
				"value":          content,
				"representation": "storage",
			},
		},
		"version": map[string]int{
			"number": page.Version.Number + 1,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %v", err)
	}

	if a.config.Debug {
		fmt.Printf("Updating page %s in space %s with title: %s\n", pageID, actualSpaceKey, title)
		if actualSpaceKey != spaceKey {
			fmt.Printf("Note: Page was moved from space '%s' to '%s'\n", spaceKey, actualSpaceKey)
		}
		fmt.Printf("Request URL: %s\n", apiURL)
		fmt.Printf("Current version: %d, New version: %d\n", page.Version.Number, page.Version.Number+1)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.SetBasicAuth(a.config.Username, a.config.Token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if a.config.Debug {
		fmt.Printf("Update page response status: %d\nResponse body: %s\n", resp.StatusCode, string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to update page: status %d, body: %s", resp.StatusCode, string(body))
	}

	var updatedPage Page
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&updatedPage); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// Build full URL
	pageURL := updatedPage.Links.WebUI
	if !strings.HasPrefix(pageURL, "http") {
		pageURL = baseURL + "/wiki" + pageURL
	}

	return &PagePublishResult{
		PageID:   updatedPage.ID,
		PageURL:  pageURL,
		SpaceKey: actualSpaceKey,
		Title:    updatedPage.Title,
		Created:  false, // This is an update, not a create
	}, nil
}

// DeletePage deletes a page from Confluence by its ID
func (a *Adapter) DeletePage(ctx context.Context, pageID string) error {
	baseURL := strings.TrimRight(a.config.BaseURL, "/")
	apiURL := fmt.Sprintf("%s/wiki/rest/api/content/%s", baseURL, pageID)

	if a.config.Debug {
		fmt.Printf("Deleting Confluence page: %s\n", pageID)
		fmt.Printf("Request URL: %s\n", apiURL)
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.SetBasicAuth(a.config.Username, a.config.Token)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("confluence page not found: %s", pageID)
	}

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete page: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// PageExistsByTitle checks if a page with the given title exists in the space
// Returns (exists, pageID, error)
func (a *Adapter) PageExistsByTitle(ctx context.Context, spaceKey, title string) (bool, string, error) {
	baseURL := strings.TrimRight(a.config.BaseURL, "/")

	// Build CQL query to search for page by title in space
	cql := fmt.Sprintf(`type=page AND space="%s" AND title="%s"`, spaceKey, title)
	encodedCQL := url.QueryEscape(cql)
	apiURL := fmt.Sprintf("%s/wiki/rest/api/content/search?cql=%s&limit=1", baseURL, encodedCQL)

	if a.config.Debug {
		fmt.Printf("Checking if page exists: space=%s, title=%s\n", spaceKey, title)
		fmt.Printf("Request URL: %s\n", apiURL)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return false, "", fmt.Errorf("failed to create request: %v", err)
	}

	req.SetBasicAuth(a.config.Username, a.config.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if a.config.Debug {
		fmt.Printf("Page exists check response status: %d\nResponse body: %s\n", resp.StatusCode, string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return false, "", fmt.Errorf("failed to search for page: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result Response
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&result); err != nil {
		return false, "", fmt.Errorf("failed to decode response: %v", err)
	}

	if len(result.Results) > 0 {
		return true, result.Results[0].ID, nil
	}

	return false, "", nil
}
