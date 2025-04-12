package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/confluence"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAssetRepository is a mock implementation of AssetRepository
type MockAssetRepository struct {
	mock.Mock
}

func (m *MockAssetRepository) Save(asset *domain.Asset) error {
	args := m.Called(asset)
	return args.Error(0)
}

func (m *MockAssetRepository) FindByName(name string) (*domain.Asset, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Asset), args.Error(1)
}

func (m *MockAssetRepository) FindByID(id string) (*domain.Asset, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Asset), args.Error(1)
}

func (m *MockAssetRepository) FindAll() ([]*domain.Asset, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Asset), args.Error(1)
}

func (m *MockAssetRepository) Delete(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

// MockLlamaClient is a mock implementation of LlamaClient
type MockLlamaClient struct {
	mock.Mock
}

func (m *MockLlamaClient) EnrichContent(content, field string, asset *domain.Asset) (string, error) {
	args := m.Called(content, field, asset)
	return args.String(0), args.Error(1)
}

func (m *MockLlamaClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockConfluenceAdapter is a mock implementation of the Confluence adapter
type MockConfluenceAdapter struct {
	mock.Mock
}

func (m *MockConfluenceAdapter) FetchPage(ctx context.Context, pageID string) (*confluence.ConfluencePage, error) {
	args := m.Called(ctx, pageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*confluence.ConfluencePage), args.Error(1)
}

var _ ports.ConfluenceAdapter = (*MockConfluenceAdapter)(nil)

func TestCreateAsset(t *testing.T) {
	tests := []struct {
		name          string
		assetName     string
		description   string
		setupMock     func(*MockAssetRepository)
		expectedError error
		checkError    func(error) bool
	}{
		{
			name:        "successful creation",
			assetName:   "test-asset",
			description: "Test description",
			setupMock: func(m *MockAssetRepository) {
				m.On("FindByName", "test-asset").Return(nil, errors.New("not found"))
				m.On("FindByID", mock.AnythingOfType("string")).Return(nil, errors.New("not found"))
				m.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
			},
			expectedError: nil,
			checkError: func(err error) bool {
				return err == nil
			},
		},
		{
			name:        "existing asset",
			assetName:   "existing-asset",
			description: "Test description",
			setupMock: func(m *MockAssetRepository) {
				m.On("FindByName", "existing-asset").Return(&domain.Asset{
					Name:        "existing-asset",
					Description: "Existing description",
				}, nil)
			},
			expectedError: fmt.Errorf("asset with name 'existing-asset' already exists"),
			checkError: func(err error) bool {
				return err != nil && err.Error() == "asset with name 'existing-asset' already exists"
			},
		},
		{
			name:        "existing ID",
			assetName:   "existing-id",
			description: "Test description",
			setupMock: func(m *MockAssetRepository) {
				m.On("FindByName", "existing-id").Return(nil, errors.New("not found"))
				m.On("FindByID", "existing-id").Return(&domain.Asset{
					ID:          "existing-id",
					Name:        "some-asset",
					Description: "Some description",
				}, nil)
			},
			expectedError: fmt.Errorf("cannot create asset with name 'existing-id' as it matches an existing asset's ID"),
		},
		{
			name:        "duplicate ID",
			assetName:   "test-asset",
			description: "Test description",
			setupMock: func(m *MockAssetRepository) {
				m.On("FindByName", "test-asset").Return(nil, errors.New("not found"))
				m.On("FindByID", mock.AnythingOfType("string")).Return(&domain.Asset{
					ID:          "test-id",
					Name:        "test-asset",
					Description: "Test description",
					Status:      "active",
					DocLink:     "https://example.com",
				}, nil)
			},
			expectedError: fmt.Errorf("cannot create asset with name 'test-asset' as it matches an existing asset's ID"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAssetRepository)
			tt.setupMock(mockRepo)
			service := NewAssetService(mockRepo)

			err := service.CreateAsset(tt.assetName, tt.description)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				return
			}

			require.NoError(t, err)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestListAssets(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*MockAssetRepository)
		expectedError  error
		expectedAssets []*domain.Asset
	}{
		{
			name: "successful listing",
			setupMock: func(m *MockAssetRepository) {
				m.On("FindAll").Return([]*domain.Asset{
					{Name: "asset1", Description: "Description 1"},
					{Name: "asset2", Description: "Description 2"},
				}, nil)
			},
			expectedError: nil,
			expectedAssets: []*domain.Asset{
				{Name: "asset1", Description: "Description 1"},
				{Name: "asset2", Description: "Description 2"},
			},
		},
		{
			name: "empty list",
			setupMock: func(m *MockAssetRepository) {
				m.On("FindAll").Return([]*domain.Asset{}, nil)
			},
			expectedError:  nil,
			expectedAssets: []*domain.Asset{},
		},
		{
			name: "repository error",
			setupMock: func(m *MockAssetRepository) {
				m.On("FindAll").Return(nil, errors.New("repository error"))
			},
			expectedError:  errors.New("repository error"),
			expectedAssets: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAssetRepository)
			tt.setupMock(mockRepo)
			service := NewAssetService(mockRepo)

			assets, err := service.ListAssets()

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				return
			}

			require.NoError(t, err)
			assert.True(t, mockRepo.AssertCalled(t, "FindAll"), "FindAll was not called")
			assert.Len(t, assets, len(tt.expectedAssets), "unexpected number of assets")

			for i, asset := range assets {
				assert.Equal(t, tt.expectedAssets[i].Name, asset.Name, "unexpected asset name")
				assert.Equal(t, tt.expectedAssets[i].Description, asset.Description, "unexpected asset description")
			}
		})
	}
}

func TestUpdateAsset(t *testing.T) {
	tests := []struct {
		name          string
		assetName     string
		description   string
		why           string
		benefits      string
		how           string
		metrics       string
		setupMock     func(*MockAssetRepository)
		expectedError string
	}{
		{
			name:        "successful update",
			assetName:   "test-asset",
			description: "Updated description",
			setupMock: func(m *MockAssetRepository) {
				m.On("FindByName", "test-asset").Return(&domain.Asset{
					Name:        "test-asset",
					Description: "Original description",
				}, nil)
				m.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
			},
		},
		{
			name:        "asset not found",
			assetName:   "non-existent",
			description: "Updated description",
			setupMock: func(m *MockAssetRepository) {
				m.On("FindByName", "non-existent").Return(nil, errors.New("not found"))
			},
			expectedError: "asset not found",
		},
		{
			name:        "empty description",
			assetName:   "test-asset",
			description: "",
			setupMock: func(m *MockAssetRepository) {
				m.On("FindByName", "test-asset").Return(&domain.Asset{
					Name:        "test-asset",
					Description: "Original description",
				}, nil)
			},
			expectedError: "asset description cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAssetRepository)
			tt.setupMock(mockRepo)
			service := NewAssetService(mockRepo)

			err := service.UpdateAsset(tt.assetName, tt.description, tt.why, tt.benefits, tt.how, tt.metrics)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				return
			}

			assert.NoError(t, err)
			mockRepo.AssertCalled(t, "FindByName", tt.assetName)
			mockRepo.AssertCalled(t, "Save", mock.AnythingOfType("*domain.Asset"))
		})
	}
}

func TestUpdateDocumentation(t *testing.T) {
	tests := []struct {
		name          string
		assetName     string
		setupMock     func(*MockAssetRepository)
		expectedError error
	}{
		{
			name:      "successful update",
			assetName: "test-asset",
			setupMock: func(m *MockAssetRepository) {
				m.On("FindByName", "test-asset").Return(&domain.Asset{
					Name:        "test-asset",
					Description: "Test description",
				}, nil)
				m.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:      "asset not found",
			assetName: "non-existent",
			setupMock: func(m *MockAssetRepository) {
				m.On("FindByName", "non-existent").Return(nil, errors.New("not found"))
			},
			expectedError: fmt.Errorf("asset not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAssetRepository)
			tt.setupMock(mockRepo)
			service := NewAssetService(mockRepo)

			err := service.UpdateDocumentation(tt.assetName)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				return
			}

			require.NoError(t, err)
			mockRepo.AssertCalled(t, "FindByName", tt.assetName)
			mockRepo.AssertCalled(t, "Save", mock.AnythingOfType("*domain.Asset"))
		})
	}
}

func TestTaskCountOperations(t *testing.T) {
	tests := []struct {
		name          string
		assetName     string
		operation     func(*MockAssetRepository, string) error
		setupMock     func(*MockAssetRepository)
		expectedError error
	}{
		{
			name:      "increment success",
			assetName: "test-asset",
			operation: func(mockRepo *MockAssetRepository, name string) error {
				service := NewAssetService(mockRepo)
				return service.IncrementTaskCount(name)
			},
			setupMock: func(m *MockAssetRepository) {
				m.On("FindByName", "test-asset").Return(&domain.Asset{
					Name:        "test-asset",
					Description: "Test description",
				}, nil)
				m.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:      "decrement success",
			assetName: "test-asset",
			operation: func(mockRepo *MockAssetRepository, name string) error {
				service := NewAssetService(mockRepo)
				return service.DecrementTaskCount(name)
			},
			setupMock: func(m *MockAssetRepository) {
				m.On("FindByName", "test-asset").Return(&domain.Asset{
					Name:                "test-asset",
					Description:         "Test description",
					AssociatedTaskCount: 1,
				}, nil)
				m.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
			},
			expectedError: nil,
		},
		{
			name:      "decrement below zero",
			assetName: "test-asset",
			operation: func(mockRepo *MockAssetRepository, name string) error {
				service := NewAssetService(mockRepo)
				return service.DecrementTaskCount(name)
			},
			setupMock: func(m *MockAssetRepository) {
				m.On("FindByName", "test-asset").Return(&domain.Asset{
					Name:                "test-asset",
					Description:         "Test description",
					AssociatedTaskCount: 0,
				}, nil)
			},
			expectedError: fmt.Errorf("task count cannot be negative"),
		},
		{
			name:      "asset not found",
			assetName: "non-existent",
			operation: func(mockRepo *MockAssetRepository, name string) error {
				service := NewAssetService(mockRepo)
				return service.IncrementTaskCount(name)
			},
			setupMock: func(m *MockAssetRepository) {
				m.On("FindByName", "non-existent").Return(nil, errors.New("not found"))
			},
			expectedError: fmt.Errorf("asset not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAssetRepository)
			tt.setupMock(mockRepo)

			err := tt.operation(mockRepo, tt.assetName)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				return
			}

			require.NoError(t, err)
			mockRepo.AssertCalled(t, "FindByName", tt.assetName)
			mockRepo.AssertCalled(t, "Save", mock.AnythingOfType("*domain.Asset"))
		})
	}
}

func TestGetAsset(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name          string
		identifier    string
		setupMock     func(*MockAssetRepository)
		expectedAsset *domain.Asset
		expectedError error
	}{
		{
			name:       "find by name",
			identifier: "test-asset",
			setupMock: func(m *MockAssetRepository) {
				m.On("FindByName", "test-asset").Return(&domain.Asset{
					ID:          "test-id",
					Name:        "test-asset",
					Description: "Test description",
					LaunchDate:  fixedTime,
					Status:      "active",
					DocLink:     "https://example.com",
				}, nil)
			},
			expectedAsset: &domain.Asset{
				ID:          "test-id",
				Name:        "test-asset",
				Description: "Test description",
				LaunchDate:  fixedTime,
				Status:      "active",
				DocLink:     "https://example.com",
			},
			expectedError: nil,
		},
		{
			name:       "find by ID",
			identifier: "test-id",
			setupMock: func(m *MockAssetRepository) {
				m.On("FindByName", "test-id").Return(nil, errors.New("not found"))
				m.On("FindByID", "test-id").Return(&domain.Asset{
					ID:          "test-id",
					Name:        "test-asset",
					Description: "Test description",
					LaunchDate:  fixedTime,
					Status:      "active",
					DocLink:     "https://example.com",
				}, nil)
			},
			expectedAsset: &domain.Asset{
				ID:          "test-id",
				Name:        "test-asset",
				Description: "Test description",
				LaunchDate:  fixedTime,
				Status:      "active",
				DocLink:     "https://example.com",
			},
			expectedError: nil,
		},
		{
			name:       "not found",
			identifier: "non-existent",
			setupMock: func(m *MockAssetRepository) {
				m.On("FindByName", "non-existent").Return(nil, errors.New("not found"))
				m.On("FindByID", "non-existent").Return(nil, errors.New("not found"))
			},
			expectedAsset: nil,
			expectedError: fmt.Errorf("asset not found by name or ID: non-existent"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAssetRepository)
			tt.setupMock(mockRepo)
			service := NewAssetService(mockRepo)

			asset, err := service.GetAsset(tt.identifier)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedAsset, asset)
			mockRepo.AssertCalled(t, "FindByName", tt.identifier)
			if tt.name == "find by ID" {
				mockRepo.AssertCalled(t, "FindByID", tt.identifier)
			} else {
				mockRepo.AssertNotCalled(t, "FindByID", tt.identifier)
			}
		})
	}
}

func TestValidateRequiredFields(t *testing.T) {
	tests := []struct {
		name     string
		asset    *domain.Asset
		expected []string
	}{
		{
			name: "all fields present",
			asset: &domain.Asset{
				ID:          "test-id",
				Name:        "test-asset",
				Description: "Test description",
				Why:         "Test why",
				Benefits:    "Test benefits",
				How:         "Test how",
				Metrics:     "Test metrics",
				Status:      "active",
				DocLink:     "https://example.com",
				LaunchDate:  time.Now(),
			},
			expected: nil,
		},
		{
			name: "missing launch date",
			asset: &domain.Asset{
				ID:          "test-id",
				Name:        "test-asset",
				Description: "Test description",
				Why:         "Test why",
				Benefits:    "Test benefits",
				How:         "Test how",
				Metrics:     "Test metrics",
				Status:      "active",
				DocLink:     "https://example.com",
			},
			expected: []string{"LaunchDate"},
		},
		{
			name: "missing multiple fields",
			asset: &domain.Asset{
				Name:        "test-asset",
				Description: "Test description",
				Why:         "Test why",
				Benefits:    "Test benefits",
				How:         "Test how",
				Metrics:     "Test metrics",
			},
			expected: []string{"ID", "LaunchDate", "Status", "DocLink"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateRequiredFields(tt.asset)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestEnrichAsset(t *testing.T) {
	// Save original env vars
	origBaseURL := os.Getenv("JIRA_BASE_URL")
	origToken := os.Getenv("JIRA_TOKEN")

	// Set test env vars
	os.Setenv("JIRA_BASE_URL", "https://confluence.example.com")
	os.Setenv("JIRA_TOKEN", "test-token")

	// Restore env vars after test
	defer func() {
		os.Setenv("JIRA_BASE_URL", origBaseURL)
		os.Setenv("JIRA_TOKEN", origToken)
	}()

	tests := []struct {
		name          string
		assetName     string
		field         string
		mockSetup     func(*MockAssetRepository, *MockLlamaClient, *MockConfluenceAdapter)
		expectedError string
	}{
		{
			name:      "successful enrichment",
			assetName: "test-asset",
			field:     "description",
			mockSetup: func(repo *MockAssetRepository, llama *MockLlamaClient, confluenceAdapter *MockConfluenceAdapter) {
				repo.On("FindByName", "test-asset").Return(&domain.Asset{
					ID:          "123",
					Name:        "test-asset",
					Description: "original description",
					DocLink:     "https://confluence.example.com/wiki/spaces/SPACE/pages/123456",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
					Version:     1,
				}, nil)

				confluenceAdapter.On("FetchPage", mock.Anything, "123456").Return(&confluence.ConfluencePage{
					ID:    "123456",
					Title: "Test Page",
					Space: struct {
						Key string `json:"key"`
					}{
						Key: "SPACE",
					},
					Version: struct {
						Number int `json:"number"`
					}{
						Number: 1,
					},
					Body: struct {
						Storage struct {
							Value string `json:"value"`
						} `json:"storage"`
					}{
						Storage: struct {
							Value string `json:"value"`
						}{
							Value: "test content",
						},
					},
					Links: struct {
						WebUI string `json:"webui"`
					}{
						WebUI: "https://confluence.example.com/wiki/spaces/SPACE/pages/123456",
					},
					Metadata: struct {
						Labels struct {
							Results []struct {
								Name string `json:"name"`
							} `json:"results"`
						} `json:"labels"`
					}{
						Labels: struct {
							Results []struct {
								Name string `json:"name"`
							} `json:"results"`
						}{
							Results: []struct {
								Name string `json:"name"`
							}{
								{Name: "test-label"},
							},
						},
					},
				}, nil)

				llama.On("EnrichContent", "test content", "description", mock.Anything).Return("enriched description", nil)
				repo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
			},
		},
		{
			name:      "asset not found",
			assetName: "non-existent",
			field:     "description",
			mockSetup: func(repo *MockAssetRepository, llama *MockLlamaClient, confluenceAdapter *MockConfluenceAdapter) {
				repo.On("FindByName", "non-existent").Return(nil, errors.New("not found"))
				repo.On("FindByID", "non-existent").Return(nil, errors.New("not found"))
			},
			expectedError: "failed to get asset: asset not found by name or ID: non-existent",
		},
		{
			name:      "missing DocLink",
			assetName: "test-asset",
			field:     "description",
			mockSetup: func(repo *MockAssetRepository, llama *MockLlamaClient, confluenceAdapter *MockConfluenceAdapter) {
				repo.On("FindByName", "test-asset").Return(&domain.Asset{
					ID:          "123",
					Name:        "test-asset",
					Description: "original description",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
					Version:     1,
				}, nil)
			},
			expectedError: "asset has no DocLink",
		},
		{
			name:      "unsupported field",
			assetName: "test-asset",
			field:     "unsupported",
			mockSetup: func(repo *MockAssetRepository, llama *MockLlamaClient, confluenceAdapter *MockConfluenceAdapter) {
				repo.On("FindByName", "test-asset").Return(&domain.Asset{
					ID:          "123",
					Name:        "test-asset",
					Description: "original description",
					DocLink:     "https://confluence.example.com/wiki/spaces/SPACE/pages/123456",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
					Version:     1,
				}, nil)

				confluenceAdapter.On("FetchPage", mock.Anything, "123456").Return(&confluence.ConfluencePage{
					ID:    "123456",
					Title: "Test Page",
					Space: struct {
						Key string `json:"key"`
					}{
						Key: "SPACE",
					},
					Version: struct {
						Number int `json:"number"`
					}{
						Number: 1,
					},
					Body: struct {
						Storage struct {
							Value string `json:"value"`
						} `json:"storage"`
					}{
						Storage: struct {
							Value string `json:"value"`
						}{
							Value: "test content",
						},
					},
					Links: struct {
						WebUI string `json:"webui"`
					}{
						WebUI: "https://confluence.example.com/wiki/spaces/SPACE/pages/123456",
					},
					Metadata: struct {
						Labels struct {
							Results []struct {
								Name string `json:"name"`
							} `json:"results"`
						} `json:"labels"`
					}{
						Labels: struct {
							Results []struct {
								Name string `json:"name"`
							} `json:"results"`
						}{
							Results: []struct {
								Name string `json:"name"`
							}{
								{Name: "test-label"},
							},
						},
					},
				}, nil)

				llama.On("EnrichContent", "test content", "unsupported", mock.Anything).Return("", fmt.Errorf("unsupported field for enrichment: unsupported"))
			},
			expectedError: "failed to enrich content: unsupported field for enrichment: unsupported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAssetRepository)
			mockLlama := new(MockLlamaClient)
			mockConfluence := new(MockConfluenceAdapter)

			if tt.mockSetup != nil {
				tt.mockSetup(mockRepo, mockLlama, mockConfluence)
			}

			service := &AssetService{
				repo:       mockRepo,
				llama:      mockLlama,
				confluence: mockConfluence,
			}

			err := service.EnrichAsset(tt.assetName, tt.field)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError, err.Error())
				return
			}

			assert.NoError(t, err)
			mockRepo.AssertExpectations(t)
			mockLlama.AssertExpectations(t)
			mockConfluence.AssertExpectations(t)
		})
	}
}

func TestExtractPageIDFromDocLink(t *testing.T) {
	tests := []struct {
		name     string
		docLink  string
		expected string
	}{
		{
			name:     "full URL with query parameters",
			docLink:  "https://confluence.example.com/wiki/spaces/SPACE/pages/123456?param=value",
			expected: "123456",
		},
		{
			name:     "full URL with fragment",
			docLink:  "https://confluence.example.com/wiki/spaces/SPACE/pages/123456#section",
			expected: "123456",
		},
		{
			name:     "relative URL",
			docLink:  "/wiki/spaces/SPACE/pages/123456",
			expected: "123456",
		},
		{
			name:     "short relative URL",
			docLink:  "/spaces/SPACE/pages/123456",
			expected: "123456",
		},
		{
			name:     "invalid URL",
			docLink:  "https://confluence.example.com/invalid",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPageIDFromDocLink(tt.docLink)
			assert.Equal(t, tt.expected, result)
		})
	}
}
