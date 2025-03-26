package confluence

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

func TestNewAdapter(t *testing.T) {
	config := &Config{
		BaseURL:    "https://test.atlassian.net",
		SpaceKey:   "TEST",
		Label:      "test-label",
		Token:      "test-token",
		Username:   "test@example.com",
		MaxResults: 25,
	}

	adapter := NewAdapter(config)

	if adapter.config != config {
		t.Error("config not set correctly")
	}
	if adapter.httpClient == nil {
		t.Error("httpClient not initialized")
	}
	if adapter.httpClient.Timeout != 30*time.Second {
		t.Errorf("httpClient timeout = %v, want %v", adapter.httpClient.Timeout, 30*time.Second)
	}
}

func TestGetSpaceID(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		statusCode     int
		expectedID     string
		expectError    bool
	}{
		{
			name: "successful space lookup",
			serverResponse: `{
				"results": [
					{
						"id": "test-space-id",
						"key": "TEST",
						"name": "Test Space"
					}
				]
			}`,
			statusCode:  http.StatusOK,
			expectedID:  "test-space-id",
			expectError: false,
		},
		{
			name:           "space not found",
			serverResponse: `{"results": []}`,
			statusCode:     http.StatusOK,
			expectedID:     "",
			expectError:    true,
		},
		{
			name:           "server error",
			serverResponse: `{"error": "internal server error"}`,
			statusCode:     http.StatusInternalServerError,
			expectedID:     "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			config := &Config{
				BaseURL:  server.URL,
				SpaceKey: "TEST",
				Token:    "test-token",
			}
			adapter := NewAdapter(config)

			id, err := adapter.getSpaceID(context.Background())

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tt.expectedID {
				t.Errorf("getSpaceID() = %v, want %v", id, tt.expectedID)
			}
		})
	}
}

func TestBuildSearchURL(t *testing.T) {
	config := &Config{
		BaseURL:    "https://test.atlassian.net",
		MaxResults: 25,
	}
	adapter := NewAdapter(config)

	spaceID := "test-space-id"
	url := adapter.buildSearchURL(spaceID)

	expectedURL := "https://test.atlassian.net/wiki/api/v2/pages?expand=body.storage%2Cversion%2Cmetadata.labels&limit=25&space-id=test-space-id"
	if url != expectedURL {
		t.Errorf("buildSearchURL() = %v, want %v", url, expectedURL)
	}
}

func TestFetchAssets(t *testing.T) {
	tests := []struct {
		name            string
		searchResponse  string
		contentResponse string
		statusCode      int
		expectError     bool
		expectedAssets  []*domain.Asset
	}{
		{
			name: "successful asset fetch",
			searchResponse: `{
				"results": [
					{
						"id": "test-id",
						"title": "Test Asset",
						"space": {"key": "TEST"},
						"version": {"number": 1},
						"_links": {"webui": "https://test.atlassian.net/wiki/spaces/TEST/pages/test-id"}
					}
				],
				"_links": {}
			}`,
			contentResponse: `{
				"id": "test-id",
				"title": "Test Asset",
				"space": {"key": "TEST"},
				"version": {"number": 1},
				"body": {"storage": {"value": "<table><tr><td><strong>Why are we doing this?</strong></td><td><p>Test description</p></td></tr><tr><td><strong>Pod</strong></td><td><p>Test Platform</p></td></tr><tr><td><strong>Status</strong></td><td><p>in development</p></td></tr><tr><td><strong>Launch date</strong></td><td><p>since 2022</p></td></tr></table><div class=\"labels\">{\"label\":\"cap-asset-test-asset\"}</div>"}},
				"_links": {"webui": "https://test.atlassian.net/wiki/spaces/TEST/pages/test-id"}
			}`,
			statusCode:  http.StatusOK,
			expectError: false,
			expectedAssets: []*domain.Asset{
				{
					ID:          "cap-asset-test-asset",
					Name:        "Test Asset",
					Description: "Test description",
					Version:     1,
					Platform:    "Test Platform",
					Status:      "in development",
					LaunchDate:  time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
					DocLink:     "https://test.atlassian.net/wiki/spaces/TEST/pages/test-id",
				},
			},
		},
		{
			name:           "no assets found with label",
			searchResponse: `{"results": [], "_links": {}}`,
			statusCode:     http.StatusOK,
			expectError:    true,
		},
		{
			name:           "server error",
			searchResponse: `{"error": "internal server error"}`,
			statusCode:     http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if strings.Contains(r.URL.Path, "/content/search") {
					w.Write([]byte(tt.searchResponse))
				} else if strings.Contains(r.URL.Path, "/content/") {
					w.Write([]byte(tt.contentResponse))
				} else {
					w.Write([]byte(tt.searchResponse))
				}
			}))
			defer server.Close()

			config := &Config{
				BaseURL:  server.URL,
				Label:    "test-label",
				SpaceKey: "TEST",
				Token:    "test-token",
				Username: "test@example.com",
			}
			adapter := NewAdapter(config)

			assets, err := adapter.FetchAssets(context.Background())

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				if tt.name == "no assets found with label" && !strings.Contains(err.Error(), "no assets found with label") {
					t.Errorf("expected error message to contain 'no assets found with label', got: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(assets) != len(tt.expectedAssets) {
				t.Fatalf("got %d assets, want %d", len(assets), len(tt.expectedAssets))
			}

			for i, asset := range assets {
				expected := tt.expectedAssets[i]
				if asset.ID != expected.ID {
					t.Errorf("asset[%d].ID = %v, want %v", i, asset.ID, expected.ID)
				}
				if asset.Name != expected.Name {
					t.Errorf("asset[%d].Name = %v, want %v", i, asset.Name, expected.Name)
				}
				if asset.Description != expected.Description {
					t.Errorf("asset[%d].Description = %v, want %v", i, asset.Description, expected.Description)
				}
				if asset.Version != expected.Version {
					t.Errorf("asset[%d].Version = %v, want %v", i, asset.Version, expected.Version)
				}
				if asset.Platform != expected.Platform {
					t.Errorf("asset[%d].Platform = %v, want %v", i, asset.Platform, expected.Platform)
				}
				if asset.Status != expected.Status {
					t.Errorf("asset[%d].Status = %v, want %v", i, asset.Status, expected.Status)
				}
				if !asset.LaunchDate.Equal(expected.LaunchDate) {
					t.Errorf("asset[%d].LaunchDate = %v, want %v", i, asset.LaunchDate, expected.LaunchDate)
				}
				if asset.DocLink != expected.DocLink {
					t.Errorf("asset[%d].DocLink = %v, want %v", i, asset.DocLink, expected.DocLink)
				}
			}
		})
	}
}

func TestConvertPageToAsset(t *testing.T) {
	tests := []struct {
		name          string
		page          ConfluencePage
		expectedAsset *domain.Asset
		expectError   bool
	}{
		{
			name: "successful conversion",
			page: ConfluencePage{
				ID:    "test-id",
				Title: "Test Asset",
				Space: struct {
					Key string `json:"key"`
				}{Key: "TEST"},
				Version: struct {
					Number int `json:"number"`
				}{Number: 1},
				Body: struct {
					Storage struct {
						Value string `json:"value"`
					} `json:"storage"`
				}{
					Storage: struct {
						Value string `json:"value"`
					}{
						Value: `<table>
							<tr><td><strong>Why are we doing this?</strong></td><td><p>Test description</p></td></tr>
							<tr><td><strong>Pod</strong></td><td><p>Test Platform</p></td></tr>
							<tr><td><strong>Status</strong></td><td><p>in development</p></td></tr>
							<tr><td><strong>Launch date</strong></td><td><p>since 2022</p></td></tr>
						</table>
						<div class="labels">{"label":"cap-asset-test-asset"}</div>`,
					},
				},
				Links: struct {
					WebUI string `json:"webui"`
				}{
					WebUI: "https://test.atlassian.net/wiki/spaces/TEST/pages/test-id",
				},
			},
			expectedAsset: &domain.Asset{
				ID:          "cap-asset-test-asset",
				Name:        "Test Asset",
				Description: "Test description",
				Version:     1,
				Platform:    "Test Platform",
				Status:      "in development",
				LaunchDate:  time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
				DocLink:     "https://test.atlassian.net/wiki/spaces/TEST/pages/test-id",
			},
			expectError: false,
		},
		{
			name: "invalid content",
			page: ConfluencePage{
				ID:    "test-id",
				Title: "Test Asset",
				Body: struct {
					Storage struct {
						Value string `json:"value"`
					} `json:"storage"`
				}{
					Storage: struct {
						Value string `json:"value"`
					}{
						Value: "invalid content",
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &Adapter{}

			asset, err := adapter.convertPageToAsset(tt.page)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if asset.ID != tt.expectedAsset.ID {
				t.Errorf("asset.ID = %v, want %v", asset.ID, tt.expectedAsset.ID)
			}
			if asset.Name != tt.expectedAsset.Name {
				t.Errorf("asset.Name = %v, want %v", asset.Name, tt.expectedAsset.Name)
			}
			if asset.Description != tt.expectedAsset.Description {
				t.Errorf("asset.Description = %v, want %v", asset.Description, tt.expectedAsset.Description)
			}
			if asset.Version != tt.expectedAsset.Version {
				t.Errorf("asset.Version = %v, want %v", asset.Version, tt.expectedAsset.Version)
			}
			if asset.Platform != tt.expectedAsset.Platform {
				t.Errorf("asset.Platform = %v, want %v", asset.Platform, tt.expectedAsset.Platform)
			}
			if asset.Status != tt.expectedAsset.Status {
				t.Errorf("asset.Status = %v, want %v", asset.Status, tt.expectedAsset.Status)
			}
			if !asset.LaunchDate.Equal(tt.expectedAsset.LaunchDate) {
				t.Errorf("asset.LaunchDate = %v, want %v", asset.LaunchDate, tt.expectedAsset.LaunchDate)
			}
			if asset.DocLink != tt.expectedAsset.DocLink {
				t.Errorf("asset.DocLink = %v, want %v", asset.DocLink, tt.expectedAsset.DocLink)
			}
		})
	}
}
