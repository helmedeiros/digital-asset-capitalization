package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

// MockJiraQueryPort is a mock implementation of JiraQueryPort
type MockJiraQueryPort struct {
	mock.Mock
}

func (m *MockJiraQueryPort) SearchTasksByLabelPrefix(ctx context.Context, labelPrefix string, maxResults int) ([]JiraTaskInfo, error) {
	args := m.Called(ctx, labelPrefix, maxResults)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]JiraTaskInfo), args.Error(1)
}

func (m *MockJiraQueryPort) SearchTasksWithFilters(ctx context.Context, filters JiraSearchFilters) ([]JiraTaskInfo, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]JiraTaskInfo), args.Error(1)
}

// MockTeamConfigPort is a mock implementation of TeamConfigPort
type MockTeamConfigPort struct {
	mock.Mock
}

func (m *MockTeamConfigPort) GetTeamForUser(userIdentifier string) (string, error) {
	args := m.Called(userIdentifier)
	return args.String(0), args.Error(1)
}

func (m *MockTeamConfigPort) GetAllTeams() (map[string][]string, error) {
	args := m.Called()
	return args.Get(0).(map[string][]string), args.Error(1)
}

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
	return args.Get(0).([]*domain.Asset), args.Error(1)
}

func (m *MockAssetRepository) Delete(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func TestSyncAssetContributorsFromJiraUseCase_Execute(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		input          SyncContributorsInput
		setupMocks     func(*MockJiraQueryPort, *MockTeamConfigPort, *MockAssetRepository)
		expectedAssets int
		expectedTasks  int
		expectError    bool
	}{
		{
			name: "successful sync with basic input",
			input: SyncContributorsInput{
				DryRun:     true,
				MaxResults: 100,
			},
			setupMocks: func(jira *MockJiraQueryPort, team *MockTeamConfigPort, asset *MockAssetRepository) {
				tasks := []JiraTaskInfo{
					{
						Key:      "FN-123",
						Assignee: "user1@example.com",
						Reporter: "user2@example.com",
						Labels:   []string{"cap-asset-omio-flex", "cap-development"},
					},
				}
				jira.On("SearchTasksByLabelPrefix", mock.Anything, "cap-asset-", 100).Return(tasks, nil)
				team.On("GetTeamForUser", "user1@example.com").Return("FN", nil)
				team.On("GetTeamForUser", "user2@example.com").Return("FN", nil)

				testAsset := &domain.Asset{
					Name:              "Omio Flex",
					OwningTeam:        "FN",
					ContributingTeams: []string{},
				}
				asset.On("FindByName", "Omio Flex").Return(testAsset, nil)
			},
			expectedAssets: 1,
			expectedTasks:  1,
			expectError:    false,
		},
		{
			name: "sync with project filter",
			input: SyncContributorsInput{
				DryRun:     true,
				MaxResults: 100,
				ProjectKey: "FN",
			},
			setupMocks: func(jira *MockJiraQueryPort, team *MockTeamConfigPort, asset *MockAssetRepository) {
				tasks := []JiraTaskInfo{
					{
						Key:      "FN-456",
						Assignee: "user3@example.com",
						Labels:   []string{"cap-asset-service-fee"},
					},
				}
				filters := JiraSearchFilters{
					LabelPrefix: "cap-asset-",
					ProjectKey:  "FN",
					MaxResults:  100,
				}
				jira.On("SearchTasksWithFilters", mock.Anything, filters).Return(tasks, nil)
				team.On("GetTeamForUser", "user3@example.com").Return("Backend Team", nil)

				testAsset := &domain.Asset{
					Name:              "Service Fee",
					ContributingTeams: []string{},
				}
				asset.On("FindByName", "Service Fee").Return(testAsset, nil)
			},
			expectedAssets: 1,
			expectedTasks:  1,
			expectError:    false,
		},
		{
			name: "handle JIRA query error",
			input: SyncContributorsInput{
				DryRun:     true,
				MaxResults: 100,
			},
			setupMocks: func(jira *MockJiraQueryPort, _ *MockTeamConfigPort, _ *MockAssetRepository) {
				// Mock JIRA query error
				jira.On("SearchTasksByLabelPrefix", mock.Anything, "cap-asset-", 100).Return(nil, errors.New("JIRA API error"))
			},
			expectedAssets: 0,
			expectedTasks:  0,
			expectError:    true,
		},
		{
			name: "handle tasks with no matching labels",
			input: SyncContributorsInput{
				DryRun:     true,
				MaxResults: 100,
			},
			setupMocks: func(jira *MockJiraQueryPort, _ *MockTeamConfigPort, _ *MockAssetRepository) {
				// Mock JIRA query with non-matching labels
				tasks := []JiraTaskInfo{
					{
						Key:      "TEST-1",
						Labels:   []string{"other-label", "non-cap-asset"},
						Assignee: "user1@example.com",
						Reporter: "user2@example.com",
					},
				}
				jira.On("SearchTasksByLabelPrefix", mock.Anything, "cap-asset-", 100).Return(tasks, nil)
			},
			expectedAssets: 0,
			expectedTasks:  1,
			expectError:    false,
		},
		{
			name: "handle asset not found error",
			input: SyncContributorsInput{
				DryRun:     true,
				MaxResults: 100,
			},
			setupMocks: func(jira *MockJiraQueryPort, _ *MockTeamConfigPort, asset *MockAssetRepository) {
				// Mock JIRA query
				tasks := []JiraTaskInfo{
					{
						Key:      "TEST-1",
						Labels:   []string{"cap-asset-unknown-asset"},
						Assignee: "user1@example.com",
						Reporter: "user2@example.com",
					},
				}
				jira.On("SearchTasksByLabelPrefix", mock.Anything, "cap-asset-", 100).Return(tasks, nil)

				// Mock asset not found for both attempts (original and converted name)
				asset.On("FindByName", "Unknown Asset").Return(nil, errors.New("asset not found"))
				asset.On("FindByName", "Unknown Asset").Return(nil, errors.New("asset not found"))
			},
			expectedAssets: 1, // Asset result is still created even if asset not found
			expectedTasks:  1,
			expectError:    false, // Should continue processing despite individual asset errors
		},
		{
			name: "handle team lookup errors gracefully",
			input: SyncContributorsInput{
				DryRun:     true,
				MaxResults: 100,
			},
			setupMocks: func(jira *MockJiraQueryPort, team *MockTeamConfigPort, asset *MockAssetRepository) {
				// Mock JIRA query
				tasks := []JiraTaskInfo{
					{
						Key:      "TEST-1",
						Labels:   []string{"cap-asset-service-fee"},
						Assignee: "unknown@example.com",
						Reporter: "user2@example.com",
					},
				}
				jira.On("SearchTasksByLabelPrefix", mock.Anything, "cap-asset-", 100).Return(tasks, nil)

				// Mock team config - one user found, one not found
				team.On("GetTeamForUser", "unknown@example.com").Return("", errors.New("user not found"))
				team.On("GetTeamForUser", "user2@example.com").Return("Team2", nil)

				testAsset := &domain.Asset{
					Name:              "Service Fee",
					ContributingTeams: []string{},
				}
				asset.On("FindByName", "Service Fee").Return(testAsset, nil)
			},
			expectedAssets: 1,
			expectedTasks:  1,
			expectError:    false,
		},
		{
			name: "handle filters search error",
			input: SyncContributorsInput{
				DryRun:     true,
				MaxResults: 100,
				ProjectKey: "TEST",
			},
			setupMocks: func(jira *MockJiraQueryPort, _ *MockTeamConfigPort, _ *MockAssetRepository) {
				// Mock JIRA filters search error
				filters := JiraSearchFilters{
					LabelPrefix: "cap-asset-",
					ProjectKey:  "TEST",
					MaxResults:  100,
				}
				jira.On("SearchTasksWithFilters", mock.Anything, filters).Return(nil, errors.New("filters search error"))
			},
			expectedAssets: 0,
			expectedTasks:  0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mocks
			mockJira := new(MockJiraQueryPort)
			mockTeam := new(MockTeamConfigPort)
			mockAsset := new(MockAssetRepository)

			// Setup mock expectations
			tt.setupMocks(mockJira, mockTeam, mockAsset)

			// Create use case
			uc := NewSyncAssetContributorsFromJiraUseCase(mockAsset, mockJira, mockTeam)

			// Execute
			result, err := uc.Execute(context.Background(), tt.input)

			// Assertions
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedTasks, result.TotalTasks)
				assert.Equal(t, tt.expectedAssets, len(result.AssetsProcessed))
			}

			// Verify mock expectations
			mockJira.AssertExpectations(t)
			mockTeam.AssertExpectations(t)
			mockAsset.AssertExpectations(t)
		})
	}
}

func TestSyncAssetContributorsFromJiraUseCase_ExtractAssetNameFromLabel(t *testing.T) {
	t.Parallel()
	uc := &SyncAssetContributorsFromJiraUseCase{}

	tests := []struct {
		input    string
		expected string
	}{
		{"cap-asset-omio-flex", "Omio Flex"},
		{"cap-asset-dynamic-markup", "Dynamic Markup"},
		{"cap-asset-service-fee", "Service Fee"},
		{"cap-asset-insurance-platform", "Insurance Platform"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := uc.extractAssetNameFromLabel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSyncAssetContributorsFromJiraUseCase_GroupTasksByAsset(t *testing.T) {
	t.Parallel()
	uc := &SyncAssetContributorsFromJiraUseCase{}

	tasks := []JiraTaskInfo{
		{
			Key:    "FN-123",
			Labels: []string{"cap-asset-omio-flex", "cap-development"},
		},
		{
			Key:    "FN-124",
			Labels: []string{"cap-asset-omio-flex", "cap-maintenance"},
		},
		{
			Key:    "FN-125",
			Labels: []string{"cap-asset-service-fee", "cap-development"},
		},
	}

	result := uc.groupTasksByAsset(tasks)

	assert.Len(t, result, 2)
	assert.Contains(t, result, "cap-asset-omio-flex")
	assert.Contains(t, result, "cap-asset-service-fee")
	assert.Len(t, result["cap-asset-omio-flex"], 2)
	assert.Len(t, result["cap-asset-service-fee"], 1)
}

func TestSyncAssetContributorsFromJiraUseCase_ExtractTeamsFromTasks(t *testing.T) {
	t.Parallel()
	mockTeam := new(MockTeamConfigPort)
	uc := &SyncAssetContributorsFromJiraUseCase{
		teamConfig: mockTeam,
	}

	tasks := []JiraTaskInfo{
		{
			Key:      "FN-123",
			Assignee: "user1@example.com",
			Reporter: "user2@example.com",
		},
		{
			Key:      "FN-124",
			Assignee: "user1@example.com", // Duplicate user
			Reporter: "user3@example.com",
		},
	}

	mockTeam.On("GetTeamForUser", "user1@example.com").Return("FN", nil)
	mockTeam.On("GetTeamForUser", "user2@example.com").Return("FN", nil)
	mockTeam.On("GetTeamForUser", "user3@example.com").Return("Backend Team", nil)

	result := uc.extractTeamsFromTasks(tasks)

	assert.Len(t, result, 2)
	assert.Contains(t, result, "FN")
	assert.Contains(t, result, "Backend Team")

	mockTeam.AssertExpectations(t)
}

func TestSyncAssetContributorsFromJiraUseCase_FindNewAndRemovedTeams(t *testing.T) {
	t.Parallel()
	uc := &SyncAssetContributorsFromJiraUseCase{}

	current := []string{"TeamA", "TeamB"}
	found := []string{"TeamB", "TeamC", "TeamD"}

	newTeams := uc.findNewTeams(current, found)
	removedTeams := uc.findRemovedTeams(current, found)

	assert.Equal(t, []string{"TeamC", "TeamD"}, newTeams)
	assert.Equal(t, []string{"TeamA"}, removedTeams)
}

func TestSyncAssetContributorsFromJiraUseCase_MergeTeams(t *testing.T) {
	t.Parallel()
	uc := &SyncAssetContributorsFromJiraUseCase{}

	foundTeams := []string{"TeamA", "TeamB", "OwningTeam", "TeamC"}
	owningTeam := "OwningTeam"

	result := uc.mergeTeams(foundTeams, owningTeam)

	assert.Len(t, result, 3)
	assert.Contains(t, result, "TeamA")
	assert.Contains(t, result, "TeamB")
	assert.Contains(t, result, "TeamC")
	assert.NotContains(t, result, "OwningTeam") // Should be excluded
}

func TestSyncAssetContributorsFromJiraUseCase_ConvertLabelToAssetName(t *testing.T) {
	t.Parallel()
	uc := &SyncAssetContributorsFromJiraUseCase{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard label conversion",
			input:    "cap-asset-my-service",
			expected: "My Service",
		},
		{
			name:     "omio special case",
			input:    "cap-asset-omio-flex",
			expected: "Omio Flex",
		},
		{
			name:     "multiple word label",
			input:    "cap-asset-payment-gateway-api",
			expected: "Payment Gateway Api",
		},
		{
			name:     "single word label",
			input:    "cap-asset-dashboard",
			expected: "Dashboard",
		},
		{
			name:     "empty suffix",
			input:    "cap-asset-",
			expected: "",
		},
		{
			name:     "complex omio case",
			input:    "cap-asset-omio-marketplace-platform",
			expected: "Omio Marketplace Platform",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := uc.convertLabelToAssetName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSyncAssetContributorsFromJiraUseCase_ProcessAssetContributors(t *testing.T) {
	t.Parallel()
	t.Run("should process asset contributors successfully", func(t *testing.T) {
		mockAsset := new(MockAssetRepository)
		mockTeam := new(MockTeamConfigPort)
		uc := &SyncAssetContributorsFromJiraUseCase{
			assetRepo:  mockAsset,
			teamConfig: mockTeam,
		}

		// Create a test asset
		asset, _ := domain.NewAsset("Test Asset", "Description")
		asset.SetOwningTeam("OwningTeam")
		asset.AddContributingTeam("ExistingTeam")

		tasks := []JiraTaskInfo{
			{
				Key:      "TEST-1",
				Assignee: "user1@example.com",
				Reporter: "user2@example.com",
			},
		}

		mockAsset.On("FindByName", "Test Asset").Return(asset, nil)
		mockAsset.On("Save", mock.AnythingOfType("*domain.Asset")).Return(nil)
		mockTeam.On("GetTeamForUser", "user1@example.com").Return("NewTeam", nil)
		mockTeam.On("GetTeamForUser", "user2@example.com").Return("AnotherTeam", nil)

		result := uc.processAssetContributors(context.Background(), "cap-asset-test-asset", tasks, false)

		assert.Equal(t, "Test Asset", result.AssetName)
		assert.Equal(t, 1, result.TasksAnalyzed)
		assert.True(t, result.Updated)
		assert.Empty(t, result.Error)

		mockAsset.AssertExpectations(t)
		mockTeam.AssertExpectations(t)
	})

	t.Run("should handle asset not found", func(t *testing.T) {
		mockAsset := new(MockAssetRepository)
		uc := &SyncAssetContributorsFromJiraUseCase{
			assetRepo: mockAsset,
		}

		tasks := []JiraTaskInfo{
			{Key: "TEST-1"},
		}

		mockAsset.On("FindByName", "Nonexistent Asset").Return(nil, assert.AnError)
		mockAsset.On("FindByName", "Nonexistent Asset").Return(nil, assert.AnError)

		result := uc.processAssetContributors(context.Background(), "cap-asset-nonexistent-asset", tasks, false)

		assert.Equal(t, "Nonexistent Asset", result.AssetName)
		assert.Contains(t, result.Error, "asset not found")

		mockAsset.AssertExpectations(t)
	})

	t.Run("should handle dry run mode", func(t *testing.T) {
		mockAsset := new(MockAssetRepository)
		mockTeam := new(MockTeamConfigPort)
		uc := &SyncAssetContributorsFromJiraUseCase{
			assetRepo:  mockAsset,
			teamConfig: mockTeam,
		}

		asset, _ := domain.NewAsset("Test Asset", "Description")
		tasks := []JiraTaskInfo{
			{
				Key:      "TEST-1",
				Assignee: "user@example.com",
			},
		}

		mockAsset.On("FindByName", "Test Asset").Return(asset, nil)
		mockTeam.On("GetTeamForUser", "user@example.com").Return("NewTeam", nil)
		// No Save expectation in dry run mode

		result := uc.processAssetContributors(context.Background(), "cap-asset-test-asset", tasks, true)

		assert.Equal(t, "Test Asset", result.AssetName)
		assert.False(t, result.Updated) // No update in dry run

		mockAsset.AssertExpectations(t)
		mockTeam.AssertExpectations(t)
	})
}
