package usecase

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

// MockTeamSyncPort is a mock implementation of ports.TeamSyncPort
type MockTeamSyncPort struct {
	mock.Mock
}

func (m *MockTeamSyncPort) GetProjectMembers(projectKey string) (*domain.ProjectTeamData, error) {
	args := m.Called(projectKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ProjectTeamData), args.Error(1)
}

func (m *MockTeamSyncPort) GetProjectRoles(projectKey string) (map[string][]domain.TeamMember, error) {
	args := m.Called(projectKey)
	return args.Get(0).(map[string][]domain.TeamMember), args.Error(1)
}

func (m *MockTeamSyncPort) GetAssignableUsers(projectKey string) ([]domain.TeamMember, error) {
	args := m.Called(projectKey)
	return args.Get(0).([]domain.TeamMember), args.Error(1)
}

func TestNewSyncTeamFromJira(t *testing.T) {
	mockTeamSyncPort := &MockTeamSyncPort{}
	mockConfigRepo := &MockConfigurationRepository{}

	useCase := NewSyncTeamFromJira(mockTeamSyncPort, mockConfigRepo)

	assert.NotNil(t, useCase)
	assert.Equal(t, mockTeamSyncPort, useCase.teamSyncPort)
	assert.Equal(t, mockConfigRepo, useCase.configRepo)
}

func TestSyncTeamFromJira_Execute_EmptyProjectKey(t *testing.T) {
	mockTeamSyncPort := &MockTeamSyncPort{}
	mockConfigRepo := &MockConfigurationRepository{}
	useCase := NewSyncTeamFromJira(mockTeamSyncPort, mockConfigRepo)

	result, err := useCase.Execute("")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "project key is required")
}

func TestSyncTeamFromJira_Execute_SuccessfulSync(t *testing.T) {
	// Setup mocks
	mockTeamSyncPort := &MockTeamSyncPort{}
	mockConfigRepo := &MockConfigurationRepository{}

	// Create existing team config
	existingTeams := map[string][]string{
		"TEST": {"Old Member"},
	}
	existingConfig, _ := domain.NewTeamConfig(existingTeams)

	// Setup mock expectations
	mockConfigRepo.On("LoadTeamConfig").Return(existingConfig, nil)

	projectTeamData := &domain.ProjectTeamData{
		ProjectKey: "TEST",
		Members: []domain.TeamMember{
			{Name: "John Doe", DisplayName: "John Doe", Email: "john@example.com"},
			{Name: "Jane Smith", DisplayName: "Jane Smith", Email: "jane@example.com"},
		},
	}
	mockTeamSyncPort.On("GetProjectMembers", "TEST").Return(projectTeamData, nil)
	mockConfigRepo.On("SaveTeamConfig", mock.AnythingOfType("*domain.TeamConfig")).Return(nil)

	useCase := NewSyncTeamFromJira(mockTeamSyncPort, mockConfigRepo)

	// Execute
	result, err := useCase.Execute("TEST")

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "TEST", result.ProjectKey)
	assert.Equal(t, "jira", result.Source)
	assert.Equal(t, 2, result.TotalMembers)
	assert.Len(t, result.Members, 2)
	assert.False(t, result.HasErrors())

	// Check added/removed members
	assert.Contains(t, result.AddedMembers, "John Doe")
	assert.Contains(t, result.AddedMembers, "Jane Smith")
	assert.Contains(t, result.RemovedMembers, "Old Member")

	mockTeamSyncPort.AssertExpectations(t)
	mockConfigRepo.AssertExpectations(t)
}

func TestSyncTeamFromJira_Execute_NoExistingConfig(t *testing.T) {
	// Setup mocks
	mockTeamSyncPort := &MockTeamSyncPort{}
	mockConfigRepo := &MockConfigurationRepository{}

	// Setup mock expectations - config doesn't exist
	mockConfigRepo.On("LoadTeamConfig").Return(nil, errors.New("config not found"))

	projectTeamData := &domain.ProjectTeamData{
		ProjectKey: "TEST",
		Members: []domain.TeamMember{
			{Name: "John Doe", DisplayName: "John Doe", Email: "john@example.com"},
		},
	}
	mockTeamSyncPort.On("GetProjectMembers", "TEST").Return(projectTeamData, nil)
	mockConfigRepo.On("SaveTeamConfig", mock.AnythingOfType("*domain.TeamConfig")).Return(nil)

	useCase := NewSyncTeamFromJira(mockTeamSyncPort, mockConfigRepo)

	// Execute
	result, err := useCase.Execute("TEST")

	// Verify
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "TEST", result.ProjectKey)
	assert.Equal(t, 1, result.TotalMembers)
	assert.False(t, result.HasErrors())

	// Should have added one member, removed none
	assert.Contains(t, result.AddedMembers, "John Doe")
	assert.Empty(t, result.RemovedMembers)

	mockTeamSyncPort.AssertExpectations(t)
	mockConfigRepo.AssertExpectations(t)
}

func TestSyncTeamFromJira_Execute_JiraError(t *testing.T) {
	// Setup mocks
	mockTeamSyncPort := &MockTeamSyncPort{}
	mockConfigRepo := &MockConfigurationRepository{}

	existingConfig, _ := domain.NewTeamConfig(make(map[string][]string))
	mockConfigRepo.On("LoadTeamConfig").Return(existingConfig, nil)
	mockTeamSyncPort.On("GetProjectMembers", "TEST").Return(nil, errors.New("JIRA connection failed"))

	useCase := NewSyncTeamFromJira(mockTeamSyncPort, mockConfigRepo)

	// Execute
	result, err := useCase.Execute("TEST")

	// Verify - should return result with error, not fail completely
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, "jira", result.Errors[0].Type)
	assert.Contains(t, result.Errors[0].Message, "JIRA connection failed")

	mockTeamSyncPort.AssertExpectations(t)
	mockConfigRepo.AssertExpectations(t)
}

func TestSyncTeamFromJira_Execute_SaveConfigError(t *testing.T) {
	// Setup mocks
	mockTeamSyncPort := &MockTeamSyncPort{}
	mockConfigRepo := &MockConfigurationRepository{}

	existingConfig, _ := domain.NewTeamConfig(make(map[string][]string))
	mockConfigRepo.On("LoadTeamConfig").Return(existingConfig, nil)

	projectTeamData := &domain.ProjectTeamData{
		ProjectKey: "TEST",
		Members: []domain.TeamMember{
			{Name: "John Doe", DisplayName: "John Doe", Email: "john@example.com"},
		},
	}
	mockTeamSyncPort.On("GetProjectMembers", "TEST").Return(projectTeamData, nil)
	mockConfigRepo.On("SaveTeamConfig", mock.AnythingOfType("*domain.TeamConfig")).Return(errors.New("save failed"))

	useCase := NewSyncTeamFromJira(mockTeamSyncPort, mockConfigRepo)

	// Execute
	result, err := useCase.Execute("TEST")

	// Verify - should return result with error
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.HasErrors())
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, "persistence", result.Errors[0].Type)
	assert.Contains(t, result.Errors[0].Message, "save failed")

	mockTeamSyncPort.AssertExpectations(t)
	mockConfigRepo.AssertExpectations(t)
}

func TestFindAddedMembers(t *testing.T) {
	tests := []struct {
		name           string
		currentMembers []string
		newMembers     []string
		expected       []string
	}{
		{
			name:           "all new members",
			currentMembers: []string{},
			newMembers:     []string{"John", "Jane"},
			expected:       []string{"John", "Jane"},
		},
		{
			name:           "some new members",
			currentMembers: []string{"John"},
			newMembers:     []string{"John", "Jane", "Bob"},
			expected:       []string{"Jane", "Bob"},
		},
		{
			name:           "no new members",
			currentMembers: []string{"John", "Jane"},
			newMembers:     []string{"John", "Jane"},
			expected:       []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findAddedMembers(tt.currentMembers, tt.newMembers)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestFindRemovedMembers(t *testing.T) {
	tests := []struct {
		name           string
		currentMembers []string
		newMembers     []string
		expected       []string
	}{
		{
			name:           "all members removed",
			currentMembers: []string{"John", "Jane"},
			newMembers:     []string{},
			expected:       []string{"John", "Jane"},
		},
		{
			name:           "some members removed",
			currentMembers: []string{"John", "Jane", "Bob"},
			newMembers:     []string{"John"},
			expected:       []string{"Jane", "Bob"},
		},
		{
			name:           "no members removed",
			currentMembers: []string{"John", "Jane"},
			newMembers:     []string{"John", "Jane"},
			expected:       []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findRemovedMembers(tt.currentMembers, tt.newMembers)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}
