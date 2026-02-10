package infrastructure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

// MockConfigRepository is a mock implementation of ConfigurationRepository
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

func (m *MockConfigRepository) LoadJiraConfig() (*domain.JiraConfig, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.JiraConfig), args.Error(1)
}

func (m *MockConfigRepository) SaveJiraConfig(config *domain.JiraConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockConfigRepository) LoadTeamConfig() (*domain.TeamConfig, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TeamConfig), args.Error(1)
}

func (m *MockConfigRepository) SaveTeamConfig(config *domain.TeamConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func TestTeamConfigAdapter_GetTeamForUser(t *testing.T) {
	tests := []struct {
		name           string
		userIdentifier string
		teamConfig     map[string][]string
		expectedTeam   string
		expectError    bool
	}{
		{
			name:           "exact email match",
			userIdentifier: "john.doe@example.com",
			teamConfig: map[string][]string{
				"FN": {"john.doe@example.com", "jane.smith@example.com"},
			},
			expectedTeam: "FN",
			expectError:  false,
		},
		{
			name:           "display name match",
			userIdentifier: "John Doe",
			teamConfig: map[string][]string{
				"Backend": {"john.doe", "jane.smith"},
			},
			expectedTeam: "Backend",
			expectError:  false,
		},
		{
			name:           "partial email match",
			userIdentifier: "john.doe@example.com",
			teamConfig: map[string][]string{
				"DevOps": {"john.doe", "jane.smith"},
			},
			expectedTeam: "DevOps",
			expectError:  false,
		},
		{
			name:           "user not found",
			userIdentifier: "unknown@example.com",
			teamConfig: map[string][]string{
				"FN": {"john.doe@example.com"},
			},
			expectedTeam: "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockConfigRepository)
			adapter := NewTeamConfigAdapter(mockRepo)

			teamConfig, err := domain.NewTeamConfig(tt.teamConfig)
			assert.NoError(t, err)

			mockRepo.On("LoadTeamConfig").Return(teamConfig, nil)

			result, err := adapter.GetTeamForUser(tt.userIdentifier)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedTeam, result)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestTeamConfigAdapter_GetAllTeams(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	adapter := NewTeamConfigAdapter(mockRepo)

	expectedTeams := map[string][]string{
		"FN":      {"user1@example.com", "user2@example.com"},
		"Backend": {"user3@example.com", "user4@example.com"},
	}

	teamConfig, err := domain.NewTeamConfig(expectedTeams)
	assert.NoError(t, err)

	mockRepo.On("LoadTeamConfig").Return(teamConfig, nil)

	result, err := adapter.GetAllTeams()

	assert.NoError(t, err)
	assert.Equal(t, expectedTeams, result)

	mockRepo.AssertExpectations(t)
}

func TestTeamConfigAdapter_MatchesUser(t *testing.T) {
	adapter := &TeamConfigAdapter{}

	tests := []struct {
		name           string
		userIdentifier string
		teamMember     string
		expected       bool
	}{
		{
			name:           "exact match",
			userIdentifier: "john.doe@example.com",
			teamMember:     "john.doe@example.com",
			expected:       true,
		},
		{
			name:           "email username match",
			userIdentifier: "john.doe@example.com",
			teamMember:     "john.doe",
			expected:       true,
		},
		{
			name:           "display name to email match",
			userIdentifier: "John Doe",
			teamMember:     "john.doe@example.com",
			expected:       true,
		},
		{
			name:           "normalized match",
			userIdentifier: "John Doe",
			teamMember:     "john.doe",
			expected:       true,
		},
		{
			name:           "no match",
			userIdentifier: "alice@example.com",
			teamMember:     "bob@example.com",
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adapter.matchesUser(tt.userIdentifier, tt.teamMember)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTeamConfigAdapter_GetTribeForTeam(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	adapter := NewTeamConfigAdapter(mockRepo)

	teams := map[string][]string{
		"FN":  {"Alice", "Bob"},
		"COP": {"Charlie"},
	}
	nicknames := map[string][]string{}
	tribes := map[string]string{
		"FN":  "Engineering",
		"COP": "Platform",
	}

	teamConfig, err := domain.NewTeamConfigWithTribes(teams, nicknames, tribes)
	assert.NoError(t, err)

	mockRepo.On("LoadTeamConfig").Return(teamConfig, nil)

	t.Run("returns tribe for existing team", func(t *testing.T) {
		tribe, err := adapter.GetTribeForTeam("FN")
		assert.NoError(t, err)
		assert.Equal(t, "Engineering", tribe)
	})

	t.Run("returns tribe for another team", func(t *testing.T) {
		tribe, err := adapter.GetTribeForTeam("COP")
		assert.NoError(t, err)
		assert.Equal(t, "Platform", tribe)
	})

	t.Run("returns empty for team without tribe", func(t *testing.T) {
		tribe, err := adapter.GetTribeForTeam("NONEXISTENT")
		assert.NoError(t, err)
		assert.Equal(t, "", tribe)
	})

	mockRepo.AssertExpectations(t)
}

func TestTeamConfigAdapter_GetTribeForTeam_Error(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	adapter := NewTeamConfigAdapter(mockRepo)

	mockRepo.On("LoadTeamConfig").Return(nil, assert.AnError)

	tribe, err := adapter.GetTribeForTeam("FN")

	assert.Error(t, err)
	assert.Equal(t, "", tribe)
	mockRepo.AssertExpectations(t)
}

func TestTeamConfigAdapter_GetCompanyForTeam(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	adapter := NewTeamConfigAdapter(mockRepo)

	teams := map[string][]string{
		"FN":  {"Alice", "Bob"},
		"COP": {"Charlie"},
	}
	nicknames := map[string][]string{}
	tribes := map[string]string{}
	companies := map[string]string{
		"FN":  "TechCorp",
		"COP": "PartnerInc",
	}

	teamConfig, err := domain.NewTeamConfigFull(teams, nicknames, tribes, companies)
	assert.NoError(t, err)

	mockRepo.On("LoadTeamConfig").Return(teamConfig, nil)

	t.Run("returns company for existing team", func(t *testing.T) {
		company, err := adapter.GetCompanyForTeam("FN")
		assert.NoError(t, err)
		assert.Equal(t, "TechCorp", company)
	})

	t.Run("returns company for another team", func(t *testing.T) {
		company, err := adapter.GetCompanyForTeam("COP")
		assert.NoError(t, err)
		assert.Equal(t, "PartnerInc", company)
	})

	t.Run("returns empty for team without company", func(t *testing.T) {
		company, err := adapter.GetCompanyForTeam("NONEXISTENT")
		assert.NoError(t, err)
		assert.Equal(t, "", company)
	})

	mockRepo.AssertExpectations(t)
}

func TestTeamConfigAdapter_GetCompanyForTeam_Error(t *testing.T) {
	mockRepo := new(MockConfigRepository)
	adapter := NewTeamConfigAdapter(mockRepo)

	mockRepo.On("LoadTeamConfig").Return(nil, assert.AnError)

	company, err := adapter.GetCompanyForTeam("FN")

	assert.Error(t, err)
	assert.Equal(t, "", company)
	mockRepo.AssertExpectations(t)
}
