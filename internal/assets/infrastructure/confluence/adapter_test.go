package confluence

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/id"
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

	adapter := NewAdapter(config, id.NewHashIDGenerator())

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
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			config := &Config{
				BaseURL:  server.URL,
				SpaceKey: "TEST",
				Token:    "test-token",
			}
			adapter := NewAdapter(config, id.NewHashIDGenerator())

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
	adapter := NewAdapter(config, id.NewHashIDGenerator())

	spaceID := "test-space-id"
	url := adapter.buildSearchURL(spaceID)

	expectedURL := "https://test.atlassian.net/wiki/api/v2/pages?expand=version%2Cmetadata.labels&limit=25&space-id=test-space-id"
	if url != expectedURL {
		t.Errorf("buildSearchURL() = %v, want %v", url, expectedURL)
	}
}

func TestFetchAssets(t *testing.T) {
	tests := []struct {
		name            string
		config          *Config
		searchResponse  string
		contentResponse string
		statusCode      int
		expectError     bool
		expectedAssets  []*domain.Asset
		validateRequest func(*testing.T, *http.Request)
	}{
		{
			name: "successful asset fetch with default limit",
			config: &Config{
				Label:      "test-label",
				MaxResults: 200,
			},
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
			validateRequest: func(t *testing.T, r *http.Request) {
				if !strings.Contains(r.URL.RawQuery, "limit=200") {
					t.Errorf("URL does not contain expected limit parameter: got %v", r.URL.RawQuery)
				}
				if strings.Contains(r.URL.RawQuery, "body.storage") {
					t.Errorf("URL should not contain body.storage expansion: got %v", r.URL.RawQuery)
				}
			},
		},
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
				if tt.validateRequest != nil && strings.Contains(r.URL.Path, "/content/search") {
					tt.validateRequest(t, r)
				}
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

			if tt.config == nil {
				tt.config = &Config{}
			}
			tt.config.BaseURL = server.URL
			adapter := NewAdapter(tt.config, id.NewHashIDGenerator())

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
		page          Page
		config        *Config
		expectedAsset *domain.Asset
		expectError   bool
	}{
		{
			name: "successful conversion with full URL",
			page: Page{
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
			config: &Config{
				BaseURL: "https://test.atlassian.net",
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
			name: "successful conversion with relative URL",
			page: Page{
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
					WebUI: "/spaces/TEST/pages/test-id",
				},
			},
			config: &Config{
				BaseURL: "https://test.atlassian.net",
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
			name: "successful conversion with relative URL containing wiki",
			page: Page{
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
					WebUI: "/wiki/spaces/TEST/pages/test-id",
				},
			},
			config: &Config{
				BaseURL: "https://test.atlassian.net",
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
			page: Page{
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
			config:      &Config{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter := &Adapter{
				config: tt.config,
			}

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

func TestBuildCQLQuery(t *testing.T) {
	tests := []struct {
		name        string
		spaceKey    string
		label       string
		expectedCQL string
	}{
		{
			name:        "all spaces - empty space key",
			spaceKey:    "",
			label:       "cap-asset",
			expectedCQL: "type%3Dpage+AND+label%3D%22cap-asset%22",
		},
		{
			name:        "all spaces - wildcard",
			spaceKey:    "*",
			label:       "cap-asset",
			expectedCQL: "type%3Dpage+AND+label%3D%22cap-asset%22",
		},
		{
			name:        "single space",
			spaceKey:    "MZN",
			label:       "cap-asset",
			expectedCQL: "type%3Dpage+AND+label%3D%22cap-asset%22+AND+space%3D%22MZN%22",
		},
		{
			name:        "multiple spaces",
			spaceKey:    "MZN,CAP,DOC",
			label:       "cap-asset",
			expectedCQL: "type%3Dpage+AND+label%3D%22cap-asset%22+AND+space+in+%28%22MZN%22%2C+%22CAP%22%2C+%22DOC%22%29",
		},
		{
			name:        "multiple spaces with whitespace",
			spaceKey:    " MZN , CAP , DOC ",
			label:       "cap-asset",
			expectedCQL: "type%3Dpage+AND+label%3D%22cap-asset%22+AND+space+in+%28%22MZN%22%2C+%22CAP%22%2C+%22DOC%22%29",
		},
		{
			name:        "multiple spaces with empty values",
			spaceKey:    "MZN,,CAP, ,DOC",
			label:       "cap-asset",
			expectedCQL: "type%3Dpage+AND+label%3D%22cap-asset%22+AND+space+in+%28%22MZN%22%2C+%22CAP%22%2C+%22DOC%22%29",
		},
		{
			name:        "single space with whitespace",
			spaceKey:    " MZN ",
			label:       "test-label",
			expectedCQL: "type%3Dpage+AND+label%3D%22test-label%22+AND+space%3D%22MZN%22",
		},
		{
			name:        "empty spaces result in all spaces",
			spaceKey:    " , , ",
			label:       "cap-asset",
			expectedCQL: "type%3Dpage+AND+label%3D%22cap-asset%22",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				SpaceKey: tt.spaceKey,
				Label:    tt.label,
			}
			adapter := NewAdapter(config, id.NewHashIDGenerator())

			cql := adapter.buildCQLQuery()

			if cql != tt.expectedCQL {
				t.Errorf("buildCQLQuery() = %v, want %v", cql, tt.expectedCQL)
			}
		})
	}
}

func TestCreatePage(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		statusCode     int
		expectedPageID string
		expectError    bool
	}{
		{
			name: "successful page creation",
			serverResponse: `{
				"id": "12345",
				"title": "Test Asset",
				"space": {"key": "TEST"},
				"_links": {"webui": "/spaces/TEST/pages/12345/Test+Asset"}
			}`,
			statusCode:     http.StatusOK,
			expectedPageID: "12345",
			expectError:    false,
		},
		{
			name: "page creation with 201 status",
			serverResponse: `{
				"id": "67890",
				"title": "New Asset",
				"space": {"key": "TEST"},
				"_links": {"webui": "/spaces/TEST/pages/67890/New+Asset"}
			}`,
			statusCode:     http.StatusCreated,
			expectedPageID: "67890",
			expectError:    false,
		},
		{
			name:           "page already exists",
			serverResponse: `{"statusCode": 400, "message": "A page with this title already exists"}`,
			statusCode:     http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "unauthorized",
			serverResponse: `{"statusCode": 401, "message": "Unauthorized"}`,
			statusCode:     http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name:           "space not found",
			serverResponse: `{"statusCode": 404, "message": "Space not found"}`,
			statusCode:     http.StatusNotFound,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and path
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/wiki/rest/api/content") {
					t.Errorf("Unexpected path: %s", r.URL.Path)
				}
				// Verify content type
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			config := &Config{
				BaseURL:  server.URL,
				Username: "test@example.com",
				Token:    "test-token",
			}
			adapter := NewAdapter(config, id.NewHashIDGenerator())

			result, err := adapter.CreatePage(context.Background(), "Test Asset", "TEST", "<p>Content</p>")

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.PageID != tt.expectedPageID {
				t.Errorf("PageID = %v, want %v", result.PageID, tt.expectedPageID)
			}
			if result.SpaceKey != "TEST" {
				t.Errorf("SpaceKey = %v, want TEST", result.SpaceKey)
			}
			if !result.Created {
				t.Error("Expected Created to be true")
			}
		})
	}
}

func TestAddLabels(t *testing.T) {
	tests := []struct {
		name           string
		labels         []string
		serverResponse string
		statusCode     int
		expectError    bool
	}{
		{
			name:           "successful label addition",
			labels:         []string{"cap-asset", "cap-asset-test"},
			serverResponse: `{"results": [{"name": "cap-asset"}, {"name": "cap-asset-test"}]}`,
			statusCode:     http.StatusOK,
			expectError:    false,
		},
		{
			name:           "single label",
			labels:         []string{"cap-asset-test"},
			serverResponse: `{"results": [{"name": "cap-asset-test"}]}`,
			statusCode:     http.StatusOK,
			expectError:    false,
		},
		{
			name:        "empty labels",
			labels:      []string{},
			expectError: false,
		},
		{
			name:           "page not found",
			labels:         []string{"cap-asset"},
			serverResponse: `{"statusCode": 404, "message": "Page not found"}`,
			statusCode:     http.StatusNotFound,
			expectError:    true,
		},
		{
			name:           "unauthorized",
			labels:         []string{"cap-asset"},
			serverResponse: `{"statusCode": 401, "message": "Unauthorized"}`,
			statusCode:     http.StatusUnauthorized,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.labels) == 0 {
				// For empty labels, no server is needed
				config := &Config{
					BaseURL:  "https://test.atlassian.net",
					Username: "test@example.com",
					Token:    "test-token",
				}
				adapter := NewAdapter(config, id.NewHashIDGenerator())

				err := adapter.AddLabels(context.Background(), "12345", tt.labels)
				if err != nil {
					t.Fatalf("unexpected error for empty labels: %v", err)
				}
				return
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and path
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if !strings.Contains(r.URL.Path, "/label") {
					t.Errorf("Expected path to contain /label, got %s", r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			config := &Config{
				BaseURL:  server.URL,
				Username: "test@example.com",
				Token:    "test-token",
			}
			adapter := NewAdapter(config, id.NewHashIDGenerator())

			err := adapter.AddLabels(context.Background(), "12345", tt.labels)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestPageExistsByTitle(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		statusCode     int
		expectedExists bool
		expectedPageID string
		expectError    bool
	}{
		{
			name: "page exists",
			serverResponse: `{
				"results": [
					{
						"id": "12345",
						"title": "Test Asset"
					}
				]
			}`,
			statusCode:     http.StatusOK,
			expectedExists: true,
			expectedPageID: "12345",
			expectError:    false,
		},
		{
			name:           "page does not exist",
			serverResponse: `{"results": []}`,
			statusCode:     http.StatusOK,
			expectedExists: false,
			expectedPageID: "",
			expectError:    false,
		},
		{
			name:           "unauthorized",
			serverResponse: `{"statusCode": 401, "message": "Unauthorized"}`,
			statusCode:     http.StatusUnauthorized,
			expectError:    true,
		},
		{
			name:           "server error",
			serverResponse: `{"statusCode": 500, "message": "Internal Server Error"}`,
			statusCode:     http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				// Verify CQL query is in URL
				if !strings.Contains(r.URL.RawQuery, "cql=") {
					t.Errorf("Expected CQL query in URL, got %s", r.URL.RawQuery)
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			config := &Config{
				BaseURL:  server.URL,
				Username: "test@example.com",
				Token:    "test-token",
			}
			adapter := NewAdapter(config, id.NewHashIDGenerator())

			exists, pageID, err := adapter.PageExistsByTitle(context.Background(), "TEST", "Test Asset")

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if exists != tt.expectedExists {
				t.Errorf("exists = %v, want %v", exists, tt.expectedExists)
			}
			if pageID != tt.expectedPageID {
				t.Errorf("pageID = %v, want %v", pageID, tt.expectedPageID)
			}
		})
	}
}

func TestUpdatePage(t *testing.T) {
	tests := []struct {
		name             string
		fetchResponse    string
		fetchStatusCode  int
		updateResponse   string
		updateStatusCode int
		expectedPageID   string
		expectedSpace    string
		expectError      bool
	}{
		{
			name: "successful page update",
			fetchResponse: `{
				"id": "12345",
				"title": "Test Asset",
				"space": {"key": "TEST"},
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Old content</p>"}},
				"_links": {"webui": "/spaces/TEST/pages/12345/Test+Asset"}
			}`,
			fetchStatusCode: http.StatusOK,
			updateResponse: `{
				"id": "12345",
				"title": "Test Asset",
				"space": {"key": "TEST"},
				"version": {"number": 2},
				"_links": {"webui": "/spaces/TEST/pages/12345/Test+Asset"}
			}`,
			updateStatusCode: http.StatusOK,
			expectedPageID:   "12345",
			expectedSpace:    "TEST",
			expectError:      false,
		},
		{
			name: "page moved to different space",
			fetchResponse: `{
				"id": "12345",
				"title": "Test Asset",
				"space": {"key": "NEWSPACE"},
				"version": {"number": 3},
				"body": {"storage": {"value": "<p>Content</p>"}},
				"_links": {"webui": "/spaces/NEWSPACE/pages/12345/Test+Asset"}
			}`,
			fetchStatusCode: http.StatusOK,
			updateResponse: `{
				"id": "12345",
				"title": "Test Asset",
				"space": {"key": "NEWSPACE"},
				"version": {"number": 4},
				"_links": {"webui": "/spaces/NEWSPACE/pages/12345/Test+Asset"}
			}`,
			updateStatusCode: http.StatusOK,
			expectedPageID:   "12345",
			expectedSpace:    "NEWSPACE",
			expectError:      false,
		},
		{
			name: "page not found during fetch",
			fetchResponse: `{
				"statusCode": 404,
				"message": "Page not found"
			}`,
			fetchStatusCode: http.StatusNotFound,
			expectError:     true,
		},
		{
			name: "update fails with conflict",
			fetchResponse: `{
				"id": "12345",
				"title": "Test Asset",
				"space": {"key": "TEST"},
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Content</p>"}},
				"_links": {"webui": "/spaces/TEST/pages/12345/Test+Asset"}
			}`,
			fetchStatusCode:  http.StatusOK,
			updateResponse:   `{"statusCode": 409, "message": "Version conflict"}`,
			updateStatusCode: http.StatusConflict,
			expectError:      true,
		},
		{
			name: "unauthorized",
			fetchResponse: `{
				"statusCode": 401,
				"message": "Unauthorized"
			}`,
			fetchStatusCode: http.StatusUnauthorized,
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++

				// First request is GET to fetch page, second is PUT to update
				if r.Method == http.MethodGet {
					w.WriteHeader(tt.fetchStatusCode)
					w.Write([]byte(tt.fetchResponse))
					return
				}

				if r.Method == http.MethodPut {
					// Verify content type
					if r.Header.Get("Content-Type") != "application/json" {
						t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
					}

					w.WriteHeader(tt.updateStatusCode)
					w.Write([]byte(tt.updateResponse))
					return
				}

				t.Errorf("Unexpected request method: %s", r.Method)
			}))
			defer server.Close()

			config := &Config{
				BaseURL:  server.URL,
				Username: "test@example.com",
				Token:    "test-token",
			}
			adapter := NewAdapter(config, id.NewHashIDGenerator())

			result, err := adapter.UpdatePage(context.Background(), "12345", "Test Asset", "TEST", "<p>New content</p>")

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.PageID != tt.expectedPageID {
				t.Errorf("PageID = %v, want %v", result.PageID, tt.expectedPageID)
			}
			if result.SpaceKey != tt.expectedSpace {
				t.Errorf("SpaceKey = %v, want %v", result.SpaceKey, tt.expectedSpace)
			}
			if result.Created {
				t.Error("Expected Created to be false for update")
			}
		})
	}
}

func TestUpdatePage_DebugMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test Asset",
				"space": {"key": "NEWSPACE"},
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Content</p>"}},
				"_links": {"webui": "/spaces/NEWSPACE/pages/12345/Test+Asset"}
			}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "12345",
			"title": "Test Asset",
			"space": {"key": "NEWSPACE"},
			"version": {"number": 2},
			"_links": {"webui": "/spaces/NEWSPACE/pages/12345/Test+Asset"}
		}`))
	}))
	defer server.Close()

	config := &Config{
		BaseURL:  server.URL,
		Username: "test@example.com",
		Token:    "test-token",
		Debug:    true, // Enable debug mode
	}
	adapter := NewAdapter(config, id.NewHashIDGenerator())

	// This should work and log debug info (page was "moved" from TEST to NEWSPACE)
	result, err := adapter.UpdatePage(context.Background(), "12345", "Test Asset", "TEST", "<p>New content</p>")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The actual space key should be NEWSPACE (from the page), not TEST (from the parameter)
	if result.SpaceKey != "NEWSPACE" {
		t.Errorf("SpaceKey = %v, want NEWSPACE", result.SpaceKey)
	}
}

func TestUpdatePage_EmptySpaceKeyFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			// Return page with empty space key
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test Asset",
				"space": {"key": ""},
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Content</p>"}},
				"_links": {"webui": "/spaces/FALLBACK/pages/12345/Test+Asset"}
			}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "12345",
			"title": "Test Asset",
			"space": {"key": "FALLBACK"},
			"version": {"number": 2},
			"_links": {"webui": "/spaces/FALLBACK/pages/12345/Test+Asset"}
		}`))
	}))
	defer server.Close()

	config := &Config{
		BaseURL:  server.URL,
		Username: "test@example.com",
		Token:    "test-token",
	}
	adapter := NewAdapter(config, id.NewHashIDGenerator())

	// Should fallback to provided space key when page has empty space
	result, err := adapter.UpdatePage(context.Background(), "12345", "Test Asset", "FALLBACK", "<p>New content</p>")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should use the fallback space key
	if result.SpaceKey != "FALLBACK" {
		t.Errorf("SpaceKey = %v, want FALLBACK", result.SpaceKey)
	}
}

func TestUpdatePage_InvalidResponseJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "12345",
				"title": "Test Asset",
				"space": {"key": "TEST"},
				"version": {"number": 1},
				"body": {"storage": {"value": "<p>Content</p>"}},
				"_links": {"webui": "/spaces/TEST/pages/12345/Test+Asset"}
			}`))
			return
		}
		// Return invalid JSON for the update response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	config := &Config{
		BaseURL:  server.URL,
		Username: "test@example.com",
		Token:    "test-token",
	}
	adapter := NewAdapter(config, id.NewHashIDGenerator())

	_, err := adapter.UpdatePage(context.Background(), "12345", "Test Asset", "TEST", "<p>New content</p>")

	if err == nil {
		t.Error("Expected error for invalid JSON response")
	}
}
