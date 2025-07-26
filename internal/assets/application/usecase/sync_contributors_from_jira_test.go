package usecase

import (
	"context"
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
	return args.Get(0).([]JiraTaskInfo), args.Error(1)
}

func (m *MockJiraQueryPort) SearchTasksWithFilters(ctx context.Context, filters JiraSearchFilters) ([]JiraTaskInfo, error) {
	args := m.Called(ctx, filters)
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
	uc := &SyncAssetContributorsFromJiraUseCase{}

	current := []string{"TeamA", "TeamB"}
	found := []string{"TeamB", "TeamC", "TeamD"}

	newTeams := uc.findNewTeams(current, found)
	removedTeams := uc.findRemovedTeams(current, found)

	assert.Equal(t, []string{"TeamC", "TeamD"}, newTeams)
	assert.Equal(t, []string{"TeamA"}, removedTeams)
}

func TestSyncAssetContributorsFromJiraUseCase_MergeTeams(t *testing.T) {
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
