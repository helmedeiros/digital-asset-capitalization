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

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/common"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
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
	config     *Config
	httpClient *http.Client
}

// NewAdapter creates a new Confluence adapter
func NewAdapter(config *Config) *Adapter {
	return &Adapter{
		config: config,
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

// FetchAssets retrieves assets from Confluence
func (a *Adapter) FetchAssets(ctx context.Context) ([]*domain.Asset, error) {
	baseURL := strings.TrimRight(a.config.BaseURL, "/")
	url := fmt.Sprintf("%s/wiki/rest/api/content/search?cql=type=page%%20AND%%20label=%%22%s%%22&expand=version,metadata.labels&limit=%d",
		baseURL, a.config.Label, a.config.MaxResults)
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
		metadata.Identifier = common.GenerateID(page.Title)
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
		ID:              metadata.Identifier,
		Name:            page.Title,
		Description:     metadata.Description,
		Why:             metadata.Why,
		Benefits:        metadata.Benefits,
		How:             metadata.How,
		Metrics:         metadata.Metrics,
		CreatedAt:       now,
		UpdatedAt:       now,
		LastDocUpdateAt: now,
		Version:         1,
		Platform:        metadata.Platform,
		Status:          metadata.Status,
		LaunchDate:      metadata.LaunchDate,
		IsRolledOut100:  metadata.IsRolledOut100,
		Keywords:        metadata.Keywords,
		DocLink:         docLink,
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
