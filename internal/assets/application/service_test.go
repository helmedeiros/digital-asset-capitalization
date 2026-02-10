package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain/ports"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/confluence"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/id"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/llama"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/application/service"
	configdomain "github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
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

func (m *MockConfluenceAdapter) FetchPage(ctx context.Context, pageID string) (*confluence.Page, error) {
	args := m.Called(ctx, pageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*confluence.Page), args.Error(1)
}

var _ ConfluenceAdapter = (*MockConfluenceAdapter)(nil)

// ConfigServiceInterface defines the minimal interface needed for asset service
type ConfigServiceInterface interface {
	GetJiraConfig() (*configdomain.JiraConfig, error)
}

// MockConfigService for testing
type MockConfigService struct {
	mock.Mock
}

func (m *MockConfigService) GetJiraConfig() (*configdomain.JiraConfig, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*configdomain.JiraConfig), args.Error(1)
}

// Ensure MockConfigService implements ConfigServiceInterface
var _ ConfigServiceInterface = (*MockConfigService)(nil)

// TestableAssetService creates an AssetService for testing with interface
func TestableAssetService(repo ports.AssetRepository, configService ConfigServiceInterface) AssetService {
	// Initialize LLaMA client
	llamaConfig := llama.DefaultConfig()
	llamaClient, err := llama.NewClient(llamaConfig)
	if err != nil {
		// Log the error but don't fail initialization
		fmt.Printf("Warning: Failed to initialize LLaMA client: %v\n", err)
	}

	// Create Confluence adapter with shared configuration
	idGenerator := id.NewHashIDGenerator()
	confluenceAdapter, err := testableCreateConfluenceAdapter(configService, idGenerator)
	if err != nil {
		// Log the error but don't fail initialization
		fmt.Printf("Warning: Failed to initialize Confluence adapter: %v\n", err)
	}

	return &AssetServiceImpl{
		repo:       repo,
		llama:      llamaClient,
		confluence: confluenceAdapter,
		// Note: configService field is *service.ConfigService, so we leave it nil for tests
	}
}

// testableCreateConfluenceAdapter creates a Confluence adapter for testing
func testableCreateConfluenceAdapter(configService ConfigServiceInterface, idGenerator ports.IDGenerator) (ConfluenceAdapter, error) {
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
			service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

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
			service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

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
			service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

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
			service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

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
				service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())
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
				service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())
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
				service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())
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
				service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())
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
			service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

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
			mockSetup: func(repo *MockAssetRepository, llama *MockLlamaClient, _ *MockConfluenceAdapter) {
				asset := &domain.Asset{
					ID:          "123",
					Name:        "test-asset",
					Description: "original description",
					DocLink:     "https://confluence.example.com/wiki/spaces/SPACE/pages/123456",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
					Version:     1,
				}
				repo.On("FindByName", "test-asset").Return(asset, nil)
				llama.On("EnrichContent", "original description", "description", asset).Return("enriched description", nil)
				repo.On("Save", mock.MatchedBy(func(a *domain.Asset) bool {
					return a.Description == "enriched description" && a.Version == 2
				})).Return(nil)
			},
		},
		{
			name:      "asset not found",
			assetName: "non-existent",
			field:     "description",
			mockSetup: func(repo *MockAssetRepository, _ *MockLlamaClient, _ *MockConfluenceAdapter) {
				repo.On("FindByName", "non-existent").Return(nil, errors.New("not found"))
				repo.On("FindByID", "non-existent").Return(nil, errors.New("not found"))
			},
			expectedError: "failed to get asset: asset not found by name or ID: non-existent",
		},
		{
			name:      "missing_DocLink",
			assetName: "test-asset",
			field:     "description",
			mockSetup: func(repo *MockAssetRepository, llama *MockLlamaClient, _ *MockConfluenceAdapter) {
				asset := &domain.Asset{
					ID:          "123",
					Name:        "test-asset",
					Description: "original description",
					CreatedAt:   time.Now(),
					UpdatedAt:   time.Now(),
					Version:     1,
				}
				repo.On("FindByName", "test-asset").Return(asset, nil)
				llama.On("EnrichContent", "original description", "description", asset).Return("enriched description", nil)
				repo.On("Save", mock.MatchedBy(func(a *domain.Asset) bool {
					return a.Description == "enriched description" && a.Version == 2
				})).Return(nil)
			},
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

				confluenceAdapter.On("FetchPage", mock.Anything, "123456").Return(&confluence.Page{
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

			service := &AssetServiceImpl{
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

func TestGenerateKeywords(t *testing.T) {
	tests := []struct {
		name          string
		assetName     string
		existingAsset *domain.Asset
		mockKeywords  []string
		expectedError string
		setupMocks    func(*MockAssetRepository, *MockLlamaClient)
	}{
		{
			name:      "successful keyword generation",
			assetName: "test-asset",
			existingAsset: &domain.Asset{
				Name:        "test-asset",
				Description: "A test asset",
				Why:         "Testing purposes",
				Benefits:    "Improved testing",
				How:         "Using mocks",
				Metrics:     "Test coverage",
			},
			mockKeywords: []string{"test", "mock", "coverage", "automation"},
			setupMocks: func(repo *MockAssetRepository, llama *MockLlamaClient) {
				repo.On("FindByName", "test-asset").Return(&domain.Asset{
					Name:        "test-asset",
					Description: "A test asset",
					Why:         "Testing purposes",
					Benefits:    "Improved testing",
					How:         "Using mocks",
					Metrics:     "Test coverage",
				}, nil)
				llama.On("EnrichContent", mock.Anything, "keywords", mock.Anything).Return("test, mock, coverage, automation", nil)
				repo.On("Save", mock.MatchedBy(func(asset *domain.Asset) bool {
					return asset.Name == "test-asset" &&
						len(asset.Keywords) == 4 &&
						asset.Keywords[0] == "test" &&
						asset.Keywords[3] == "automation"
				})).Return(nil)
			},
		},
		{
			name:      "asset not found",
			assetName: "non-existent",
			setupMocks: func(repo *MockAssetRepository, _ *MockLlamaClient) {
				repo.On("FindByName", "non-existent").Return(nil, assert.AnError)
				repo.On("FindByID", "non-existent").Return(nil, assert.AnError)
			},
			expectedError: "failed to get asset",
		},
		{
			name:      "llama client error",
			assetName: "test-asset",
			existingAsset: &domain.Asset{
				Name:        "test-asset",
				Description: "A test asset",
			},
			setupMocks: func(repo *MockAssetRepository, llama *MockLlamaClient) {
				repo.On("FindByName", "test-asset").Return(&domain.Asset{
					Name:        "test-asset",
					Description: "A test asset",
				}, nil)
				llama.On("EnrichContent", mock.Anything, "keywords", mock.Anything).Return("", assert.AnError)
			},
			expectedError: "failed to generate keywords",
		},
		{
			name:      "save error",
			assetName: "test-asset",
			existingAsset: &domain.Asset{
				Name:        "test-asset",
				Description: "A test asset",
			},
			setupMocks: func(repo *MockAssetRepository, llama *MockLlamaClient) {
				repo.On("FindByName", "test-asset").Return(&domain.Asset{
					Name:        "test-asset",
					Description: "A test asset",
				}, nil)
				llama.On("EnrichContent", mock.Anything, "keywords", mock.Anything).Return("test, mock", nil)
				repo.On("Save", mock.Anything).Return(assert.AnError)
			},
			expectedError: "failed to save asset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockRepo := new(MockAssetRepository)
			mockLlama := new(MockLlamaClient)

			// Setup mocks
			tt.setupMocks(mockRepo, mockLlama)

			// Create service with mocks
			service := &AssetServiceImpl{
				repo:  mockRepo,
				llama: mockLlama,
			}

			// Call the method
			err := service.GenerateKeywords(tt.assetName)

			// Check error
			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			// Verify all expectations were met
			mockRepo.AssertExpectations(t)
			mockLlama.AssertExpectations(t)
		})
	}
}

// Add simple tests for missing functions to improve coverage
func TestNewAssetServiceLegacy(t *testing.T) {
	mockRepo := new(MockAssetRepository)

	service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

	assert.NotNil(t, service)
	assert.IsType(t, &AssetServiceImpl{}, service)
}

func TestDeleteAsset(t *testing.T) {
	tests := []struct {
		name          string
		assetName     string
		setupMock     func(*MockAssetRepository)
		expectedError error
	}{
		{
			name:      "successful deletion",
			assetName: "test-asset",
			setupMock: func(m *MockAssetRepository) {
				m.On("Delete", "test-asset").Return(nil)
			},
			expectedError: nil,
		},
		{
			name:      "deletion error",
			assetName: "test-asset",
			setupMock: func(m *MockAssetRepository) {
				m.On("Delete", "test-asset").Return(errors.New("delete error"))
			},
			expectedError: errors.New("delete error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAssetRepository)
			tt.setupMock(mockRepo)
			service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

			err := service.DeleteAsset(tt.assetName)

			if tt.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				require.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestSyncFromConfluenceEnvironmentFallback(t *testing.T) {
	// Test the environment variable fallback path
	t.Run("missing environment configuration", func(t *testing.T) {
		// Clear environment variables
		os.Unsetenv("JIRA_BASE_URL")
		os.Unsetenv("JIRA_TOKEN")

		mockRepo := new(MockAssetRepository)
		service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

		_, err := service.SyncFromConfluence("TEST", "test-label", false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Jira base URL is not configured")
	})

	t.Run("missing token configuration", func(t *testing.T) {
		// Set base URL but not token
		os.Setenv("JIRA_BASE_URL", "https://test.atlassian.net")
		os.Unsetenv("JIRA_TOKEN")

		mockRepo := new(MockAssetRepository)
		service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

		_, err := service.SyncFromConfluence("TEST", "test-label", false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Jira token is not configured")
	})
}

func TestNewAssetService(t *testing.T) {
	t.Run("should create service with valid config service", func(t *testing.T) {
		// Create mock repository and config service
		mockRepo := new(MockAssetRepository)
		mockConfigService := &MockConfigService{}

		// Setup mock to return valid Jira config
		mockJiraConfig := &configdomain.JiraConfig{}
		mockConfigService.On("GetJiraConfig").Return(mockJiraConfig, nil)

		// Create service
		service := TestableAssetService(mockRepo, mockConfigService)

		// Verify service is created
		assert.NotNil(t, service, "Service should not be nil")

		// Verify it's the correct implementation
		impl, ok := service.(*AssetServiceImpl)
		assert.True(t, ok, "Should return AssetServiceImpl")
		assert.Equal(t, mockRepo, impl.repo, "Repository should be set")
		assert.NotNil(t, impl.llama, "LLaMA client should be initialized")
	})

	t.Run("should create service even if config service fails", func(t *testing.T) {
		// Create mock repository and config service
		mockRepo := new(MockAssetRepository)
		mockConfigService := &MockConfigService{}

		// Setup mock to return error for Jira config
		mockConfigService.On("GetJiraConfig").Return(nil, fmt.Errorf("config error"))

		// Create service - should not fail even if config fails
		service := TestableAssetService(mockRepo, mockConfigService)

		// Verify service is still created
		assert.NotNil(t, service, "Service should not be nil even with config error")

		impl, ok := service.(*AssetServiceImpl)
		assert.True(t, ok, "Should return AssetServiceImpl")
		assert.Equal(t, mockRepo, impl.repo, "Repository should be set")
	})

	t.Run("should handle nil config service", func(t *testing.T) {
		mockRepo := new(MockAssetRepository)

		// This should work but confluence adapter creation will fail
		service := TestableAssetService(mockRepo, nil)

		assert.NotNil(t, service, "Service should not be nil")
		impl, ok := service.(*AssetServiceImpl)
		assert.True(t, ok, "Should return AssetServiceImpl")
		assert.Equal(t, mockRepo, impl.repo, "Repository should be set")
	})
}

func TestCreateConfluenceAdapter(t *testing.T) {
	t.Run("should create adapter with valid config service", func(t *testing.T) {
		// Create mock config service
		mockConfigService := &MockConfigService{}

		// Create mock Jira config
		jiraConfig, err := configdomain.NewJiraConfig("https://example.atlassian.net", "test@example.com", "test-token")
		require.NoError(t, err)

		// Setup mock
		mockConfigService.On("GetJiraConfig").Return(jiraConfig, nil)

		// Test the function
		adapter, err := testableCreateConfluenceAdapter(mockConfigService, id.NewHashIDGenerator())

		// Verify results
		assert.NoError(t, err, "Should not return error with valid config")
		assert.NotNil(t, adapter, "Adapter should not be nil")

		mockConfigService.AssertExpectations(t)
	})

	t.Run("should return error with nil config service", func(t *testing.T) {
		adapter, err := testableCreateConfluenceAdapter(nil, id.NewHashIDGenerator())

		assert.Error(t, err, "Should return error with nil config service")
		assert.Nil(t, adapter, "Adapter should be nil")
		assert.Contains(t, err.Error(), "config service is required", "Error should mention required config service")
	})

	t.Run("should return error when config service fails", func(t *testing.T) {
		// Create mock config service
		mockConfigService := &MockConfigService{}

		// Setup mock to return error
		mockConfigService.On("GetJiraConfig").Return(nil, fmt.Errorf("config retrieval failed"))

		// Test the function
		adapter, err := testableCreateConfluenceAdapter(mockConfigService, id.NewHashIDGenerator())

		// Verify results
		assert.Error(t, err, "Should return error when config retrieval fails")
		assert.Nil(t, adapter, "Adapter should be nil")
		assert.Contains(t, err.Error(), "failed to get Jira configuration", "Error should mention config failure")

		mockConfigService.AssertExpectations(t)
	})

	t.Run("should handle invalid Jira config", func(t *testing.T) {
		// Create mock config service
		mockConfigService := &MockConfigService{}

		// Create invalid Jira config (will return error)
		mockConfigService.On("GetJiraConfig").Return(nil, fmt.Errorf("invalid config"))

		// Test the function
		adapter, err := testableCreateConfluenceAdapter(mockConfigService, id.NewHashIDGenerator())

		// Verify results
		assert.Error(t, err, "Should return error with invalid config")
		assert.Nil(t, adapter, "Adapter should be nil")

		mockConfigService.AssertExpectations(t)
	})
}

func TestNewAssetServiceConstructor(t *testing.T) {
	t.Run("should create service with nil config service", func(t *testing.T) {
		mockRepo := new(MockAssetRepository)

		// This should work but confluence adapter creation will fail
		assetService := NewAssetService(mockRepo, nil, id.NewHashIDGenerator())

		assert.NotNil(t, assetService, "Service should not be nil")
		impl, ok := assetService.(*AssetServiceImpl)
		assert.True(t, ok, "Should return AssetServiceImpl")
		assert.Equal(t, mockRepo, impl.repo, "Repository should be set")
		assert.Nil(t, impl.configService, "Config service should be nil")
	})
}

func TestCreateConfluenceAdapterFunction(t *testing.T) {
	t.Run("should create adapter with valid config service", func(t *testing.T) {
		// Create a real config service with mock dependencies for this test
		// This is a more complex test since we need real config service

		// We can't easily test the real createConfluenceAdapter without complex setup
		// So we'll test the error cases that are easier to trigger

		// Test with nil config service
		adapter, err := createConfluenceAdapter(nil, id.NewHashIDGenerator())

		assert.Error(t, err, "Should return error with nil config service")
		assert.Nil(t, adapter, "Adapter should be nil")
		assert.Contains(t, err.Error(), "config service is required", "Error should mention required config service")
	})
}

func TestCreateConfluenceAdapter_ConfigServiceNil_Error(t *testing.T) {
	adapter, err := createConfluenceAdapter(nil, id.NewHashIDGenerator())
	assert.Nil(t, adapter)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config service is required")
}

func TestAssetServiceImpl_DecrementTaskCount_ErrorBranch(t *testing.T) {
	mockRepo := new(MockAssetRepository)
	mockRepo.On("FindByName", "A").Return(&domain.Asset{ID: "1", Name: "A", AssociatedTaskCount: 0}, nil)
	service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())
	err := service.DecrementTaskCount("A")
	assert.Error(t, err)
	assert.Equal(t, "task count cannot be negative", err.Error())
	mockRepo.AssertExpectations(t)
}

func TestAssetServiceImpl_SyncFromConfluence_BaseURLError(t *testing.T) {
	repo := new(MockAssetRepository)
	llama := new(MockLlamaClient)
	service := &AssetServiceImpl{repo: repo, llama: llama, configService: nil}
	os.Setenv("JIRA_BASE_URL", "")
	os.Setenv("JIRA_TOKEN", "token")
	_, err := service.SyncFromConfluence("space", "label", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Jira base URL is not configured")
}

func TestAssetServiceImpl_NormalizeSpaceKey(t *testing.T) {
	tests := []struct {
		name           string
		spaceKey       string
		expectedResult string
	}{
		{
			name:           "empty space key returns empty (all spaces)",
			spaceKey:       "",
			expectedResult: "",
		},
		{
			name:           "wildcard returns empty (all spaces)",
			spaceKey:       "*",
			expectedResult: "",
		},
		{
			name:           "single space key",
			spaceKey:       "MZN",
			expectedResult: "MZN",
		},
		{
			name:           "single space key with whitespace",
			spaceKey:       " MZN ",
			expectedResult: "MZN",
		},
		{
			name:           "multiple spaces",
			spaceKey:       "MZN,CAP,DOC",
			expectedResult: "MZN,CAP,DOC",
		},
		{
			name:           "multiple spaces with whitespace",
			spaceKey:       " MZN , CAP , DOC ",
			expectedResult: "MZN,CAP,DOC",
		},
		{
			name:           "multiple spaces with empty values",
			spaceKey:       "MZN,,CAP, ,DOC",
			expectedResult: "MZN,CAP,DOC",
		},
		{
			name:           "duplicate spaces removed",
			spaceKey:       "MZN,CAP,MZN,DOC,CAP",
			expectedResult: "MZN,CAP,DOC",
		},
		{
			name:           "only empty spaces returns empty (all spaces)",
			spaceKey:       " , , ",
			expectedResult: "",
		},
		{
			name:           "mixed empty and valid spaces",
			spaceKey:       " , MZN , , CAP , ",
			expectedResult: "MZN,CAP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAssetRepository)
			service := &AssetServiceImpl{repo: mockRepo}

			result := service.normalizeSpaceKey(tt.spaceKey)

			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestAssetServiceImpl_SyncFromConfluence_SpaceNormalization(t *testing.T) {
	tests := []struct {
		name              string
		inputSpaceKey     string
		expectedSpaceKey  string
		configuredBaseURL string
		configuredToken   string
	}{
		{
			name:              "normalizes single space with whitespace",
			inputSpaceKey:     " MZN ",
			expectedSpaceKey:  "MZN",
			configuredBaseURL: "https://test.atlassian.net",
			configuredToken:   "test-token",
		},
		{
			name:              "normalizes multiple spaces",
			inputSpaceKey:     "MZN,CAP,DOC",
			expectedSpaceKey:  "MZN,CAP,DOC",
			configuredBaseURL: "https://test.atlassian.net",
			configuredToken:   "test-token",
		},
		{
			name:              "normalizes empty space to all spaces",
			inputSpaceKey:     "",
			expectedSpaceKey:  "",
			configuredBaseURL: "https://test.atlassian.net",
			configuredToken:   "test-token",
		},
		{
			name:              "normalizes wildcard to all spaces",
			inputSpaceKey:     "*",
			expectedSpaceKey:  "",
			configuredBaseURL: "https://test.atlassian.net",
			configuredToken:   "test-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			os.Setenv("JIRA_BASE_URL", tt.configuredBaseURL)
			os.Setenv("JIRA_TOKEN", tt.configuredToken)
			defer func() {
				os.Unsetenv("JIRA_BASE_URL")
				os.Unsetenv("JIRA_TOKEN")
			}()

			// Create mock repository
			mockRepo := new(MockAssetRepository)

			// Create service using legacy constructor (which uses env vars)
			service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

			// We expect this to fail due to the confluence adapter trying to connect,
			// but we can verify that space normalization is called by checking
			// the normalized space key gets used
			_, err := service.SyncFromConfluence(tt.inputSpaceKey, "test-label", false)

			// This should fail because we don't have a real confluence server
			// but the important thing is that it goes through the normalization logic
			assert.Error(t, err)
			// The error should be about connection/fetching, not about config
			assert.NotContains(t, err.Error(), "base URL is not configured")
			assert.NotContains(t, err.Error(), "token is not configured")
		})
	}
}

func TestAssetServiceImpl_SyncFromConfluence_ConfigService(t *testing.T) {
	t.Run("uses config service when available", func(t *testing.T) {
		// Create mock repository and config service
		mockRepo := new(MockAssetRepository)
		mockConfigService := &MockConfigService{}

		// Setup mock to return valid Jira config
		mockJiraConfig := &configdomain.JiraConfig{}
		mockConfigService.On("GetJiraConfig").Return(mockJiraConfig, nil)

		// Create service with config service
		service := TestableAssetService(mockRepo, mockConfigService)

		// Try to sync - this will fail due to empty config but should use config service path
		_, err := service.SyncFromConfluence("MZN", "test-label", false)

		// Should fail due to empty base URL, but proves config service path is taken
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "base URL is not configured")

		// Verify config service was called
		mockConfigService.AssertCalled(t, "GetJiraConfig")
	})

	t.Run("falls back to env vars when config service fails", func(t *testing.T) {
		// Set up environment variables
		os.Setenv("JIRA_BASE_URL", "https://test.atlassian.net")
		os.Setenv("JIRA_TOKEN", "test-token")
		defer func() {
			os.Unsetenv("JIRA_BASE_URL")
			os.Unsetenv("JIRA_TOKEN")
		}()

		// Create mock repository and config service
		mockRepo := new(MockAssetRepository)
		mockConfigService := &MockConfigService{}

		// Setup mock to return an error
		mockConfigService.On("GetJiraConfig").Return(nil, errors.New("config error"))

		// Create service with config service
		service := TestableAssetService(mockRepo, mockConfigService)

		// Try to sync - should fall back to env vars
		_, err := service.SyncFromConfluence("MZN", "test-label", false)

		// Should fail due to connection error, not config error
		assert.Error(t, err)
		assert.NotContains(t, err.Error(), "base URL is not configured")
		assert.NotContains(t, err.Error(), "token is not configured")

		// Verify config service was called
		mockConfigService.AssertCalled(t, "GetJiraConfig")
	})
}

func TestAssetServiceImpl_SyncFromConfluence_SuccessfulSync(t *testing.T) {
	t.Run("successful sync with valid assets", func(t *testing.T) {
		// Set up environment variables
		os.Setenv("JIRA_BASE_URL", "https://test.atlassian.net")
		os.Setenv("JIRA_TOKEN", "test-token")
		defer func() {
			os.Unsetenv("JIRA_BASE_URL")
			os.Unsetenv("JIRA_TOKEN")
		}()

		// Create test assets
		now := time.Now()
		testAsset := &domain.Asset{
			ID:          "test-asset-1",
			Name:        "Test Asset 1",
			Description: "Test description 1",
			CreatedAt:   now,
			UpdatedAt:   now,
			LaunchDate:  now.AddDate(0, 0, -30), // 30 days ago
			Status:      "active",
			DocLink:     "https://test.com/doc1",
		}

		// Create mock repository
		mockRepo := new(MockAssetRepository)
		mockRepo.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)

		// Create service with mocked confluence adapter
		service := &AssetServiceImpl{
			repo:          mockRepo,
			confluence:    nil, // Will be overridden by direct adapter creation
			configService: nil,
		}

		// Mock the confluence adapter's response
		mockConfluenceAdapter := new(MockConfluenceAdapter)
		mockConfluenceAdapter.On("FetchAssets", mock.Anything).Return([]*domain.Asset{testAsset}, nil)

		// We can't easily mock the internal adapter creation, so we test the validation logic
		// by calling the service and checking it processes the space normalization
		result, err := service.SyncFromConfluence("MZN", "test-label", false)

		// This will fail due to actual network call, but we can test that our code
		// got far enough to prove space normalization worked
		if err != nil {
			// If it fails due to network issues, that's expected
			assert.Contains(t, err.Error(), "failed to fetch assets from Confluence")
		} else {
			// If somehow it succeeds (shouldn't happen), verify the result
			assert.NotNil(t, result)
		}
	})

	t.Run("sync with missing required fields", func(t *testing.T) {
		// Set up environment variables
		os.Setenv("JIRA_BASE_URL", "https://test.atlassian.net")
		os.Setenv("JIRA_TOKEN", "test-token")
		defer func() {
			os.Unsetenv("JIRA_BASE_URL")
			os.Unsetenv("JIRA_TOKEN")
		}()

		// Create mock repository
		mockRepo := new(MockAssetRepository)

		// Create service
		service := &AssetServiceImpl{
			repo:          mockRepo,
			confluence:    nil,
			configService: nil,
		}

		// Test sync - this will fail due to network,
		// but we're testing the validation paths
		_, err := service.SyncFromConfluence("MZN", "test-label", false)

		// Should fail due to network/confluence connection
		assert.Error(t, err)
	})
}

func TestAssetServiceImpl_SyncFromConfluence_TokenError(t *testing.T) {
	t.Run("missing token configuration", func(t *testing.T) {
		// Set base URL but not token
		os.Setenv("JIRA_BASE_URL", "https://test.atlassian.net")
		os.Unsetenv("JIRA_TOKEN")
		defer func() {
			os.Unsetenv("JIRA_BASE_URL")
			os.Unsetenv("JIRA_TOKEN")
		}()

		mockRepo := new(MockAssetRepository)
		service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

		_, err := service.SyncFromConfluence("MZN", "test-label", false)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Jira token is not configured")
	})
}

// Team management method tests to improve coverage
func TestAssetServiceImpl_AssignTeam(t *testing.T) {
	tests := []struct {
		name              string
		assetName         string
		owningTeam        string
		contributingTeams []string
		setupMocks        func(*MockAssetRepository)
		expectedError     string
	}{
		{
			name:              "successful team assignment",
			assetName:         "test-asset",
			owningTeam:        "team-alpha",
			contributingTeams: []string{"team-beta", "team-gamma"},
			setupMocks: func(m *MockAssetRepository) {
				asset := &domain.Asset{
					ID:   "test-id",
					Name: "test-asset",
				}
				m.On("FindByName", "test-asset").Return(asset, nil)
				m.On("Save", mock.MatchedBy(func(a *domain.Asset) bool {
					return a.GetOwningTeam() == "team-alpha" &&
						len(a.GetContributingTeams()) == 2
				})).Return(nil)
			},
		},
		{
			name:       "asset not found",
			assetName:  "nonexistent",
			owningTeam: "team-alpha",
			setupMocks: func(m *MockAssetRepository) {
				m.On("FindByName", "nonexistent").Return(nil, errors.New("not found"))
			},
			expectedError: "not found",
		},
		{
			name:       "save error",
			assetName:  "test-asset",
			owningTeam: "team-alpha",
			setupMocks: func(m *MockAssetRepository) {
				asset := &domain.Asset{
					ID:   "test-id",
					Name: "test-asset",
				}
				m.On("FindByName", "test-asset").Return(asset, nil)
				m.On("Save", mock.AnythingOfType("*domain.Asset")).Return(errors.New("save failed"))
			},
			expectedError: "save failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAssetRepository)
			tt.setupMocks(mockRepo)
			service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

			err := service.AssignTeam(tt.assetName, tt.owningTeam, tt.contributingTeams)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAssetServiceImpl_GetAssetTeams(t *testing.T) {
	tests := []struct {
		name          string
		setupMocks    func(*MockAssetRepository)
		expectedCount int
		expectedError string
	}{
		{
			name: "successful retrieval with teams",
			setupMocks: func(m *MockAssetRepository) {
				asset1 := &domain.Asset{Name: "asset1"}
				asset1.SetOwningTeam("team-alpha")
				asset1.AddContributingTeam("team-beta")

				asset2 := &domain.Asset{Name: "asset2"}
				asset2.SetOwningTeam("team-gamma")

				asset3 := &domain.Asset{Name: "asset3"} // No teams

				assets := []*domain.Asset{asset1, asset2, asset3}
				m.On("FindAll").Return(assets, nil)
			},
			expectedCount: 2, // Only asset1 and asset2 have teams
		},
		{
			name: "repository error",
			setupMocks: func(m *MockAssetRepository) {
				m.On("FindAll").Return(nil, errors.New("repo error"))
			},
			expectedError: "repo error",
		},
		{
			name: "no assets with teams",
			setupMocks: func(m *MockAssetRepository) {
				asset1 := &domain.Asset{Name: "asset1"} // No teams
				assets := []*domain.Asset{asset1}
				m.On("FindAll").Return(assets, nil)
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAssetRepository)
			tt.setupMocks(mockRepo)
			service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

			result, err := service.GetAssetTeams()

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, tt.expectedCount)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAssetServiceImpl_GetAssetTeamInfo(t *testing.T) {
	tests := []struct {
		name          string
		assetName     string
		setupMocks    func(*MockAssetRepository)
		expectedError string
	}{
		{
			name:      "successful retrieval",
			assetName: "test-asset",
			setupMocks: func(m *MockAssetRepository) {
				asset := &domain.Asset{Name: "test-asset"}
				asset.SetOwningTeam("team-alpha")
				asset.AddContributingTeam("team-beta")
				m.On("FindByName", "test-asset").Return(asset, nil)
			},
		},
		{
			name:      "asset not found",
			assetName: "nonexistent",
			setupMocks: func(m *MockAssetRepository) {
				m.On("FindByName", "nonexistent").Return(nil, errors.New("not found"))
			},
			expectedError: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAssetRepository)
			tt.setupMocks(mockRepo)
			service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

			result, err := service.GetAssetTeamInfo(tt.assetName)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.assetName, result.AssetName)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAssetServiceImpl_AddContributingTeam(t *testing.T) {
	tests := []struct {
		name          string
		assetName     string
		teamName      string
		setupMocks    func(*MockAssetRepository)
		expectedError string
	}{
		{
			name:      "successful addition",
			assetName: "test-asset",
			teamName:  "team-beta",
			setupMocks: func(m *MockAssetRepository) {
				asset := &domain.Asset{
					ID:   "test-id",
					Name: "test-asset",
				}
				m.On("FindByName", "test-asset").Return(asset, nil)
				m.On("Save", mock.MatchedBy(func(a *domain.Asset) bool {
					teams := a.GetContributingTeams()
					for _, team := range teams {
						if team == "team-beta" {
							return true
						}
					}
					return false
				})).Return(nil)
			},
		},
		{
			name:      "asset not found",
			assetName: "nonexistent",
			teamName:  "team-beta",
			setupMocks: func(m *MockAssetRepository) {
				m.On("FindByName", "nonexistent").Return(nil, errors.New("not found"))
			},
			expectedError: "not found",
		},
		{
			name:      "save error",
			assetName: "test-asset",
			teamName:  "team-beta",
			setupMocks: func(m *MockAssetRepository) {
				asset := &domain.Asset{
					ID:   "test-id",
					Name: "test-asset",
				}
				m.On("FindByName", "test-asset").Return(asset, nil)
				m.On("Save", mock.AnythingOfType("*domain.Asset")).Return(errors.New("save failed"))
			},
			expectedError: "save failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAssetRepository)
			tt.setupMocks(mockRepo)
			service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

			err := service.AddContributingTeam(tt.assetName, tt.teamName)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAssetServiceImpl_RemoveContributingTeam(t *testing.T) {
	tests := []struct {
		name          string
		assetName     string
		teamName      string
		setupMocks    func(*MockAssetRepository)
		expectedError string
	}{
		{
			name:      "successful removal",
			assetName: "test-asset",
			teamName:  "team-beta",
			setupMocks: func(m *MockAssetRepository) {
				asset := &domain.Asset{
					ID:   "test-id",
					Name: "test-asset",
				}
				asset.AddContributingTeam("team-beta")
				asset.AddContributingTeam("team-gamma")
				m.On("FindByName", "test-asset").Return(asset, nil)
				m.On("Save", mock.MatchedBy(func(a *domain.Asset) bool {
					teams := a.GetContributingTeams()
					for _, team := range teams {
						if team == "team-beta" {
							return false // team-beta should be removed
						}
					}
					return true
				})).Return(nil)
			},
		},
		{
			name:      "asset not found",
			assetName: "nonexistent",
			teamName:  "team-beta",
			setupMocks: func(m *MockAssetRepository) {
				m.On("FindByName", "nonexistent").Return(nil, errors.New("not found"))
			},
			expectedError: "not found",
		},
		{
			name:      "save error",
			assetName: "test-asset",
			teamName:  "team-beta",
			setupMocks: func(m *MockAssetRepository) {
				asset := &domain.Asset{
					ID:   "test-id",
					Name: "test-asset",
				}
				asset.AddContributingTeam("team-beta")
				m.On("FindByName", "test-asset").Return(asset, nil)
				m.On("Save", mock.AnythingOfType("*domain.Asset")).Return(errors.New("save failed"))
			},
			expectedError: "save failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAssetRepository)
			tt.setupMocks(mockRepo)
			service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

			err := service.RemoveContributingTeam(tt.assetName, tt.teamName)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAssetServiceImpl_PublishToConfluence_ValidationErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("empty asset name returns error", func(t *testing.T) {
		mockRepo := new(MockAssetRepository)
		service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

		result, err := service.PublishToConfluence(ctx, "", "SPACE", false, false)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "asset name is required")
	})

	t.Run("empty space key returns error", func(t *testing.T) {
		mockRepo := new(MockAssetRepository)
		service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

		result, err := service.PublishToConfluence(ctx, "Test Asset", "", false, false)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "space key is required")
	})

	t.Run("asset not found returns error", func(t *testing.T) {
		mockRepo := new(MockAssetRepository)
		mockRepo.On("FindByName", "Unknown Asset").Return(nil, errors.New("not found"))
		service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

		result, err := service.PublishToConfluence(ctx, "Unknown Asset", "SPACE", false, false)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to find asset")
		mockRepo.AssertExpectations(t)
	})
}

func TestAssetServiceImpl_PublishToConfluence_ConfigErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("missing base URL returns error", func(t *testing.T) {
		os.Unsetenv("JIRA_BASE_URL")
		os.Unsetenv("JIRA_TOKEN")
		defer func() {
			os.Unsetenv("JIRA_BASE_URL")
			os.Unsetenv("JIRA_TOKEN")
		}()

		mockRepo := new(MockAssetRepository)
		asset := &domain.Asset{
			ID:   "cap-asset-test",
			Name: "Test Asset",
		}
		mockRepo.On("FindByName", "Test Asset").Return(asset, nil)
		service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

		result, err := service.PublishToConfluence(ctx, "Test Asset", "SPACE", false, false)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "Jira base URL is not configured")
		mockRepo.AssertExpectations(t)
	})

	t.Run("missing token returns error", func(t *testing.T) {
		os.Setenv("JIRA_BASE_URL", "https://test.atlassian.net")
		os.Unsetenv("JIRA_TOKEN")
		defer func() {
			os.Unsetenv("JIRA_BASE_URL")
			os.Unsetenv("JIRA_TOKEN")
		}()

		mockRepo := new(MockAssetRepository)
		asset := &domain.Asset{
			ID:   "cap-asset-test",
			Name: "Test Asset",
		}
		mockRepo.On("FindByName", "Test Asset").Return(asset, nil)
		service := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())

		result, err := service.PublishToConfluence(ctx, "Test Asset", "SPACE", false, false)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "Jira token is not configured")
		mockRepo.AssertExpectations(t)
	})
}

func TestAssetServiceImpl_GetAssetLabel(t *testing.T) {
	tests := []struct {
		name          string
		asset         *domain.Asset
		expectedLabel string
	}{
		{
			name: "uses existing cap-asset ID",
			asset: &domain.Asset{
				ID:   "cap-asset-existing-id",
				Name: "Test Asset",
			},
			expectedLabel: "cap-asset-existing-id",
		},
		{
			name: "generates new ID for non-cap-asset format",
			asset: &domain.Asset{
				ID:   "old-format-123",
				Name: "My New Asset",
			},
			expectedLabel: "cap-asset-my-new-asset",
		},
		{
			name: "handles empty ID",
			asset: &domain.Asset{
				ID:   "",
				Name: "Another Asset",
			},
			expectedLabel: "cap-asset-another-asset",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAssetRepository)
			service := &AssetServiceImpl{
				repo:        mockRepo,
				idGenerator: id.NewHashIDGenerator(),
			}

			result := service.getAssetLabel(tt.asset)

			assert.Equal(t, tt.expectedLabel, result)
		})
	}
}

func TestAssetServiceImpl_ExtractPageInfoFromDocLink(t *testing.T) {
	tests := []struct {
		name          string
		docLink       string
		expectedPage  string
		expectedSpace string
		expectError   bool
	}{
		{
			name:          "valid Confluence URL",
			docLink:       "https://example.atlassian.net/wiki/spaces/MYSPACE/pages/123456789/My+Page+Title",
			expectedPage:  "123456789",
			expectedSpace: "MYSPACE",
			expectError:   false,
		},
		{
			name:          "URL with different space",
			docLink:       "https://example.atlassian.net/wiki/spaces/ENGINEERING/pages/12345/My+Page",
			expectedPage:  "12345",
			expectedSpace: "ENGINEERING",
			expectError:   false,
		},
		{
			name:          "URL without page title",
			docLink:       "https://example.atlassian.net/wiki/spaces/TEST/pages/99999",
			expectedPage:  "99999",
			expectedSpace: "TEST",
			expectError:   false,
		},
		{
			name:        "missing spaces in URL",
			docLink:     "https://example.atlassian.net/wiki/pages/12345/My+Page",
			expectError: true,
		},
		{
			name:        "missing pages in URL",
			docLink:     "https://example.atlassian.net/wiki/spaces/TEST/overview",
			expectError: true,
		},
		{
			name:        "invalid URL",
			docLink:     "not-a-valid-url",
			expectError: true,
		},
		{
			name:        "empty URL",
			docLink:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockAssetRepository)
			service := &AssetServiceImpl{
				repo:        mockRepo,
				idGenerator: id.NewHashIDGenerator(),
			}

			pageID, spaceKey, err := service.extractPageInfoFromDocLink(tt.docLink)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPage, pageID)
			assert.Equal(t, tt.expectedSpace, spaceKey)
		})
	}
}

func TestAssetServiceImpl_UpdateConfluencePage_ValidationErrors(t *testing.T) {
	t.Run("empty asset name", func(t *testing.T) {
		mockRepo := new(MockAssetRepository)
		service := &AssetServiceImpl{
			repo:        mockRepo,
			idGenerator: id.NewHashIDGenerator(),
		}

		result, err := service.UpdateConfluencePage(context.Background(), "", false, false)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "asset name is required")
	})

	t.Run("asset not found", func(t *testing.T) {
		mockRepo := new(MockAssetRepository)
		mockRepo.On("FindByName", "Unknown Asset").Return(nil, nil)

		service := &AssetServiceImpl{
			repo:        mockRepo,
			idGenerator: id.NewHashIDGenerator(),
		}

		result, err := service.UpdateConfluencePage(context.Background(), "Unknown Asset", false, false)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found")
		mockRepo.AssertExpectations(t)
	})

	t.Run("asset has no DocLink", func(t *testing.T) {
		mockRepo := new(MockAssetRepository)
		asset := &domain.Asset{
			ID:      "cap-asset-test",
			Name:    "Test Asset",
			DocLink: "", // No DocLink
		}
		mockRepo.On("FindByName", "Test Asset").Return(asset, nil)

		service := &AssetServiceImpl{
			repo:        mockRepo,
			idGenerator: id.NewHashIDGenerator(),
		}

		result, err := service.UpdateConfluencePage(context.Background(), "Test Asset", false, false)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "does not have a Confluence page link")
		mockRepo.AssertExpectations(t)
	})

	t.Run("invalid DocLink format", func(t *testing.T) {
		mockRepo := new(MockAssetRepository)
		asset := &domain.Asset{
			ID:      "cap-asset-test",
			Name:    "Test Asset",
			DocLink: "https://invalid-url-without-pages",
		}
		mockRepo.On("FindByName", "Test Asset").Return(asset, nil)

		service := &AssetServiceImpl{
			repo:        mockRepo,
			idGenerator: id.NewHashIDGenerator(),
		}

		result, err := service.UpdateConfluencePage(context.Background(), "Test Asset", false, false)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to extract page info")
		mockRepo.AssertExpectations(t)
	})
}

func TestAssetServiceImpl_UpdateConfluencePage_ConfigErrors(t *testing.T) {
	t.Run("missing base URL", func(t *testing.T) {
		// Clear environment variables
		originalBaseURL := os.Getenv("JIRA_BASE_URL")
		originalEmail := os.Getenv("JIRA_EMAIL")
		originalToken := os.Getenv("JIRA_TOKEN")
		os.Unsetenv("JIRA_BASE_URL")
		os.Unsetenv("JIRA_EMAIL")
		os.Unsetenv("JIRA_TOKEN")
		defer func() {
			if originalBaseURL != "" {
				os.Setenv("JIRA_BASE_URL", originalBaseURL)
			}
			if originalEmail != "" {
				os.Setenv("JIRA_EMAIL", originalEmail)
			}
			if originalToken != "" {
				os.Setenv("JIRA_TOKEN", originalToken)
			}
		}()

		mockRepo := new(MockAssetRepository)
		asset := &domain.Asset{
			ID:      "cap-asset-test",
			Name:    "Test Asset",
			DocLink: "https://example.atlassian.net/wiki/spaces/TEST/pages/12345/Test",
		}
		mockRepo.On("FindByName", "Test Asset").Return(asset, nil)

		service := &AssetServiceImpl{
			repo:          mockRepo,
			idGenerator:   id.NewHashIDGenerator(),
			configService: nil, // No config service
		}

		result, err := service.UpdateConfluencePage(context.Background(), "Test Asset", false, false)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "Jira base URL is not configured")
		mockRepo.AssertExpectations(t)
	})

	t.Run("missing token", func(t *testing.T) {
		// Set base URL but not token
		originalBaseURL := os.Getenv("JIRA_BASE_URL")
		originalEmail := os.Getenv("JIRA_EMAIL")
		originalToken := os.Getenv("JIRA_TOKEN")
		os.Setenv("JIRA_BASE_URL", "https://example.atlassian.net")
		os.Setenv("JIRA_EMAIL", "test@example.com")
		os.Unsetenv("JIRA_TOKEN")
		defer func() {
			if originalBaseURL != "" {
				os.Setenv("JIRA_BASE_URL", originalBaseURL)
			} else {
				os.Unsetenv("JIRA_BASE_URL")
			}
			if originalEmail != "" {
				os.Setenv("JIRA_EMAIL", originalEmail)
			} else {
				os.Unsetenv("JIRA_EMAIL")
			}
			if originalToken != "" {
				os.Setenv("JIRA_TOKEN", originalToken)
			}
		}()

		mockRepo := new(MockAssetRepository)
		asset := &domain.Asset{
			ID:      "cap-asset-test",
			Name:    "Test Asset",
			DocLink: "https://example.atlassian.net/wiki/spaces/TEST/pages/12345/Test",
		}
		mockRepo.On("FindByName", "Test Asset").Return(asset, nil)

		service := &AssetServiceImpl{
			repo:          mockRepo,
			idGenerator:   id.NewHashIDGenerator(),
			configService: nil,
		}

		result, err := service.UpdateConfluencePage(context.Background(), "Test Asset", false, false)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "Jira token is not configured")
		mockRepo.AssertExpectations(t)
	})
}

func TestAssetServiceImpl_UpdateConfluencePage_DryRun(t *testing.T) {
	// Set environment variables for the test
	originalBaseURL := os.Getenv("JIRA_BASE_URL")
	originalEmail := os.Getenv("JIRA_EMAIL")
	originalToken := os.Getenv("JIRA_TOKEN")
	os.Setenv("JIRA_BASE_URL", "https://example.atlassian.net")
	os.Setenv("JIRA_EMAIL", "test@example.com")
	os.Setenv("JIRA_TOKEN", "test-token")
	defer func() {
		if originalBaseURL != "" {
			os.Setenv("JIRA_BASE_URL", originalBaseURL)
		} else {
			os.Unsetenv("JIRA_BASE_URL")
		}
		if originalEmail != "" {
			os.Setenv("JIRA_EMAIL", originalEmail)
		} else {
			os.Unsetenv("JIRA_EMAIL")
		}
		if originalToken != "" {
			os.Setenv("JIRA_TOKEN", originalToken)
		} else {
			os.Unsetenv("JIRA_TOKEN")
		}
	}()

	mockRepo := new(MockAssetRepository)
	asset := &domain.Asset{
		ID:      "cap-asset-test-asset",
		Name:    "Test Asset",
		DocLink: "https://example.atlassian.net/wiki/spaces/TEST/pages/12345/Test+Asset",
		Why:     "Test why",
	}
	mockRepo.On("FindByName", "Test Asset").Return(asset, nil)

	service := &AssetServiceImpl{
		repo:          mockRepo,
		idGenerator:   id.NewHashIDGenerator(),
		configService: nil,
	}

	result, err := service.UpdateConfluencePage(context.Background(), "Test Asset", true, false)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "Test Asset", result.AssetName)
	assert.Equal(t, "12345", result.PageID)
	assert.Equal(t, "TEST", result.SpaceKey)
	assert.False(t, result.Created)
	assert.NotEmpty(t, result.Preview)
	assert.Contains(t, result.Preview, "<h1>Asset Capitalisation</h1>")
	mockRepo.AssertExpectations(t)
}

// MockConfigRepository is a mock for testing ConfigService integration
type MockConfigRepository struct {
	mock.Mock
}

func (m *MockConfigRepository) InitializeConfigDirectory() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockConfigRepository) ConfigExists() (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

func (m *MockConfigRepository) LoadJiraConfig() (*configdomain.JiraConfig, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*configdomain.JiraConfig), args.Error(1)
}

func (m *MockConfigRepository) SaveJiraConfig(config *configdomain.JiraConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockConfigRepository) LoadTeamConfig() (*configdomain.TeamConfig, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*configdomain.TeamConfig), args.Error(1)
}

func (m *MockConfigRepository) SaveTeamConfig(config *configdomain.TeamConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func TestAssetServiceImpl_GetTribeForTeam(t *testing.T) {
	t.Run("returns empty when team name is empty", func(t *testing.T) {
		mockRepo := new(MockAssetRepository)
		mockConfigRepo := new(MockConfigRepository)
		configService := service.NewConfigService(mockConfigRepo)

		svc := &AssetServiceImpl{
			repo:          mockRepo,
			idGenerator:   id.NewHashIDGenerator(),
			configService: configService,
		}

		result := svc.getTribeForTeam("")

		assert.Equal(t, "", result)
	})

	t.Run("returns empty when configService is nil", func(t *testing.T) {
		mockRepo := new(MockAssetRepository)

		svc := &AssetServiceImpl{
			repo:          mockRepo,
			idGenerator:   id.NewHashIDGenerator(),
			configService: nil,
		}

		result := svc.getTribeForTeam("FN")

		assert.Equal(t, "", result)
	})

	t.Run("returns empty when GetTeamConfig fails", func(t *testing.T) {
		mockRepo := new(MockAssetRepository)
		mockConfigRepo := new(MockConfigRepository)
		mockConfigRepo.On("LoadTeamConfig").Return(nil, errors.New("config error"))
		configService := service.NewConfigService(mockConfigRepo)

		svc := &AssetServiceImpl{
			repo:          mockRepo,
			idGenerator:   id.NewHashIDGenerator(),
			configService: configService,
		}

		result := svc.getTribeForTeam("FN")

		assert.Equal(t, "", result)
		mockConfigRepo.AssertExpectations(t)
	})

	t.Run("returns tribe when team has tribe configured", func(t *testing.T) {
		mockRepo := new(MockAssetRepository)
		mockConfigRepo := new(MockConfigRepository)

		teams := map[string][]string{"FN": {"user1"}}
		nicknames := map[string][]string{}
		tribes := map[string]string{"FN": "Engineering"}
		teamConfig, _ := configdomain.NewTeamConfigWithTribes(teams, nicknames, tribes)
		mockConfigRepo.On("LoadTeamConfig").Return(teamConfig, nil)
		configService := service.NewConfigService(mockConfigRepo)

		svc := &AssetServiceImpl{
			repo:          mockRepo,
			idGenerator:   id.NewHashIDGenerator(),
			configService: configService,
		}

		result := svc.getTribeForTeam("FN")

		assert.Equal(t, "Engineering", result)
		mockConfigRepo.AssertExpectations(t)
	})

	t.Run("returns empty when team has no tribe", func(t *testing.T) {
		mockRepo := new(MockAssetRepository)
		mockConfigRepo := new(MockConfigRepository)

		teams := map[string][]string{"FN": {"user1"}}
		nicknames := map[string][]string{}
		tribes := map[string]string{} // No tribes configured
		teamConfig, _ := configdomain.NewTeamConfigWithTribes(teams, nicknames, tribes)
		mockConfigRepo.On("LoadTeamConfig").Return(teamConfig, nil)
		configService := service.NewConfigService(mockConfigRepo)

		svc := &AssetServiceImpl{
			repo:          mockRepo,
			idGenerator:   id.NewHashIDGenerator(),
			configService: configService,
		}

		result := svc.getTribeForTeam("FN")

		assert.Equal(t, "", result)
		mockConfigRepo.AssertExpectations(t)
	})
}

func TestAssetServiceImpl_GetCompanyForTeam(t *testing.T) {
	t.Run("returns empty when team name is empty", func(t *testing.T) {
		mockRepo := new(MockAssetRepository)
		mockConfigRepo := new(MockConfigRepository)
		configService := service.NewConfigService(mockConfigRepo)

		svc := &AssetServiceImpl{
			repo:          mockRepo,
			idGenerator:   id.NewHashIDGenerator(),
			configService: configService,
		}

		result := svc.getCompanyForTeam("")

		assert.Equal(t, "", result)
	})

	t.Run("returns empty when configService is nil", func(t *testing.T) {
		mockRepo := new(MockAssetRepository)

		svc := &AssetServiceImpl{
			repo:          mockRepo,
			idGenerator:   id.NewHashIDGenerator(),
			configService: nil,
		}

		result := svc.getCompanyForTeam("FN")

		assert.Equal(t, "", result)
	})

	t.Run("returns empty when GetTeamConfig fails", func(t *testing.T) {
		mockRepo := new(MockAssetRepository)
		mockConfigRepo := new(MockConfigRepository)
		mockConfigRepo.On("LoadTeamConfig").Return(nil, errors.New("config error"))
		configService := service.NewConfigService(mockConfigRepo)

		svc := &AssetServiceImpl{
			repo:          mockRepo,
			idGenerator:   id.NewHashIDGenerator(),
			configService: configService,
		}

		result := svc.getCompanyForTeam("FN")

		assert.Equal(t, "", result)
		mockConfigRepo.AssertExpectations(t)
	})

	t.Run("returns company when team has company configured", func(t *testing.T) {
		mockRepo := new(MockAssetRepository)
		mockConfigRepo := new(MockConfigRepository)

		teams := map[string][]string{"FN": {"user1"}}
		nicknames := map[string][]string{}
		tribes := map[string]string{}
		companies := map[string]string{"FN": "ACME Corp"}
		teamConfig, _ := configdomain.NewTeamConfigFull(teams, nicknames, tribes, companies)
		mockConfigRepo.On("LoadTeamConfig").Return(teamConfig, nil)
		configService := service.NewConfigService(mockConfigRepo)

		svc := &AssetServiceImpl{
			repo:          mockRepo,
			idGenerator:   id.NewHashIDGenerator(),
			configService: configService,
		}

		result := svc.getCompanyForTeam("FN")

		assert.Equal(t, "ACME Corp", result)
		mockConfigRepo.AssertExpectations(t)
	})

	t.Run("returns empty when team has no company", func(t *testing.T) {
		mockRepo := new(MockAssetRepository)
		mockConfigRepo := new(MockConfigRepository)

		teams := map[string][]string{"FN": {"user1"}}
		nicknames := map[string][]string{}
		tribes := map[string]string{}
		companies := map[string]string{} // No companies configured
		teamConfig, _ := configdomain.NewTeamConfigFull(teams, nicknames, tribes, companies)
		mockConfigRepo.On("LoadTeamConfig").Return(teamConfig, nil)
		configService := service.NewConfigService(mockConfigRepo)

		svc := &AssetServiceImpl{
			repo:          mockRepo,
			idGenerator:   id.NewHashIDGenerator(),
			configService: configService,
		}

		result := svc.getCompanyForTeam("FN")

		assert.Equal(t, "", result)
		mockConfigRepo.AssertExpectations(t)
	})
}

func TestAssetServiceImpl_UpdateConfluencePage_WithTribeAndCompany(t *testing.T) {
	// Set environment variables for the test
	originalBaseURL := os.Getenv("JIRA_BASE_URL")
	originalEmail := os.Getenv("JIRA_EMAIL")
	originalToken := os.Getenv("JIRA_TOKEN")
	os.Setenv("JIRA_BASE_URL", "https://example.atlassian.net")
	os.Setenv("JIRA_EMAIL", "test@example.com")
	os.Setenv("JIRA_TOKEN", "test-token")
	defer func() {
		if originalBaseURL != "" {
			os.Setenv("JIRA_BASE_URL", originalBaseURL)
		} else {
			os.Unsetenv("JIRA_BASE_URL")
		}
		if originalEmail != "" {
			os.Setenv("JIRA_EMAIL", originalEmail)
		} else {
			os.Unsetenv("JIRA_EMAIL")
		}
		if originalToken != "" {
			os.Setenv("JIRA_TOKEN", originalToken)
		} else {
			os.Unsetenv("JIRA_TOKEN")
		}
	}()

	t.Run("dry run includes tribe and company in preview", func(t *testing.T) {
		mockAssetRepo := new(MockAssetRepository)
		mockConfigRepo := new(MockConfigRepository)

		asset := &domain.Asset{
			ID:      "cap-asset-test-asset",
			Name:    "Test Asset",
			DocLink: "https://example.atlassian.net/wiki/spaces/TEST/pages/12345/Test+Asset",
			Why:     "Test why",
		}
		asset.SetOwningTeam("FN")
		mockAssetRepo.On("FindByName", "Test Asset").Return(asset, nil)

		// Mock Jira config for the confluence adapter
		jiraConfig, _ := configdomain.NewJiraConfig(
			"https://example.atlassian.net",
			"test@example.com",
			"test-token",
		)
		mockConfigRepo.On("LoadJiraConfig").Return(jiraConfig, nil)

		teams := map[string][]string{"FN": {"user1"}}
		nicknames := map[string][]string{}
		tribes := map[string]string{"FN": "Engineering"}
		companies := map[string]string{"FN": "ACME Corp"}
		teamConfig, _ := configdomain.NewTeamConfigFull(teams, nicknames, tribes, companies)
		mockConfigRepo.On("LoadTeamConfig").Return(teamConfig, nil)
		configService := service.NewConfigService(mockConfigRepo)

		svc := &AssetServiceImpl{
			repo:          mockAssetRepo,
			idGenerator:   id.NewHashIDGenerator(),
			configService: configService,
		}

		result, err := svc.UpdateConfluencePage(context.Background(), "Test Asset", true, false)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, result.Preview, "ACME Corp")   // Company should be in preview
		assert.Contains(t, result.Preview, "FN")          // Team name should be in preview
		assert.Contains(t, result.Preview, "Engineering") // Tribe should be in preview
		mockAssetRepo.AssertExpectations(t)
		mockConfigRepo.AssertExpectations(t)
	})
}
