package usecase

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

// MockConfigurationRepository is a mock implementation of ConfigurationRepository
type MockConfigurationRepository struct {
	mock.Mock
}

func (m *MockConfigurationRepository) LoadJiraConfig() (*domain.JiraConfig, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.JiraConfig), args.Error(1)
}

func (m *MockConfigurationRepository) SaveJiraConfig(config *domain.JiraConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockConfigurationRepository) LoadTeamConfig() (*domain.TeamConfig, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TeamConfig), args.Error(1)
}

func (m *MockConfigurationRepository) SaveTeamConfig(config *domain.TeamConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockConfigurationRepository) ConfigExists() (bool, error) {
	args := m.Called()
	return args.Bool(0), args.Error(1)
}

func (m *MockConfigurationRepository) InitializeConfigDirectory() error {
	args := m.Called()
	return args.Error(0)
}

// MockEnvironmentProvider is a mock implementation of EnvironmentProvider
type MockEnvironmentProvider struct {
	mock.Mock
}

func (m *MockEnvironmentProvider) GetJiraBaseURL() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockEnvironmentProvider) GetJiraEmail() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockEnvironmentProvider) GetJiraToken() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockEnvironmentProvider) SetJiraBaseURL(value string) error {
	args := m.Called(value)
	return args.Error(0)
}

func (m *MockEnvironmentProvider) SetJiraEmail(value string) error {
	args := m.Called(value)
	return args.Error(0)
}

func (m *MockEnvironmentProvider) SetJiraToken(value string) error {
	args := m.Called(value)
	return args.Error(0)
}

func (m *MockEnvironmentProvider) IsConfigured() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockEnvironmentProvider) GetMissingVars() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

// MockUserInteraction is a mock implementation of UserInteraction
type MockUserInteraction struct {
	mock.Mock
}

func (m *MockUserInteraction) PromptString(message string) (string, error) {
	args := m.Called(message)
	return args.String(0), args.Error(1)
}

func (m *MockUserInteraction) PromptStringWithDefault(message, defaultValue string) (string, error) {
	args := m.Called(message, defaultValue)
	return args.String(0), args.Error(1)
}

func (m *MockUserInteraction) PromptPassword(message string) (string, error) {
	args := m.Called(message)
	return args.String(0), args.Error(1)
}

func (m *MockUserInteraction) PromptConfirm(message string) (bool, error) {
	args := m.Called(message)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserInteraction) PromptSelect(message string, options []string) (string, error) {
	args := m.Called(message, options)
	return args.String(0), args.Error(1)
}

func (m *MockUserInteraction) PromptMultiSelect(message string, options []string) ([]string, error) {
	args := m.Called(message, options)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockUserInteraction) DisplayMessage(message string) {
	m.Called(message)
}

func (m *MockUserInteraction) DisplayError(message string) {
	m.Called(message)
}

func (m *MockUserInteraction) DisplaySuccess(message string) {
	m.Called(message)
}

func (m *MockUserInteraction) DisplayWarning(message string) {
	m.Called(message)
}

func TestNewInitializeConfig(t *testing.T) {
	repo := &MockConfigurationRepository{}
	envProvider := &MockEnvironmentProvider{}
	ui := &MockUserInteraction{}

	useCase := NewInitializeConfig(repo, envProvider, ui)

	require.NotNil(t, useCase)
	assert.Equal(t, repo, useCase.repo)
	assert.Equal(t, envProvider, useCase.envProvider)
	assert.Equal(t, ui, useCase.ui)
}

func TestInitializeConfig_Execute(t *testing.T) {
	tests := []struct {
		name           string
		setupMocks     func(*MockConfigurationRepository, *MockEnvironmentProvider, *MockUserInteraction)
		interactive    bool
		expectedResult *InitializeConfigResult
		expectError    bool
		errorMessage   string
	}{
		{
			name: "successful interactive initialization",
			setupMocks: func(repo *MockConfigurationRepository, env *MockEnvironmentProvider, ui *MockUserInteraction) {
				// Environment not configured
				env.On("IsConfigured").Return(false)

				// User provides configuration
				ui.On("PromptString", "Enter Jira Base URL (e.g., https://company.atlassian.net):").Return("https://test.atlassian.net", nil)
				ui.On("PromptString", "Enter Jira Email:").Return("test@company.com", nil)
				ui.On("PromptPassword", "Enter Jira API Token:").Return("test-token", nil)

				// Repository operations
				repo.On("InitializeConfigDirectory").Return(nil)
				repo.On("SaveJiraConfig", mock.AnythingOfType("*domain.JiraConfig")).Return(nil)

				// Team configuration
				ui.On("PromptConfirm", "Would you like to configure team members now? (y/n):").Return(true, nil)
				ui.On("PromptString", "Enter project key (e.g., FN):").Return("FN", nil)
				ui.On("PromptString", "Enter team member username:").Return("test.user", nil)
				ui.On("PromptConfirm", "Add another team member? (y/n):").Return(false, nil)
				ui.On("PromptConfirm", "Add another project? (y/n):").Return(false, nil)
				repo.On("SaveTeamConfig", mock.AnythingOfType("*domain.TeamConfig")).Return(nil)

				// Success messages
				ui.On("DisplaySuccess", mock.AnythingOfType("string")).Return()
			},
			interactive: true,
			expectedResult: &InitializeConfigResult{
				JiraConfigCreated: true,
				TeamConfigCreated: true,
				Message:           "Configuration initialized successfully",
			},
			expectError: false,
		},
		{
			name: "environment already configured",
			setupMocks: func(repo *MockConfigurationRepository, env *MockEnvironmentProvider, ui *MockUserInteraction) {
				env.On("IsConfigured").Return(true)
				env.On("GetJiraBaseURL").Return("https://existing.atlassian.net")
				env.On("GetJiraEmail").Return("existing@company.com")
				env.On("GetJiraToken").Return("existing-token")

				// Try to create Jira config from environment
				repo.On("SaveJiraConfig", mock.AnythingOfType("*domain.JiraConfig")).Return(nil)

				repo.On("InitializeConfigDirectory").Return(nil)

				// Team configuration prompt
				ui.On("PromptConfirm", "Would you like to configure team members now? (y/n):").Return(false, nil)

				ui.On("DisplaySuccess", mock.AnythingOfType("string")).Return()
			},
			interactive: true,
			expectedResult: &InitializeConfigResult{
				JiraConfigCreated: true,
				TeamConfigCreated: false,
				Message:           "Configuration initialized successfully",
			},
			expectError: false,
		},
		{
			name: "repository initialization error",
			setupMocks: func(repo *MockConfigurationRepository, env *MockEnvironmentProvider, _ *MockUserInteraction) {
				env.On("IsConfigured").Return(false)
				repo.On("InitializeConfigDirectory").Return(errors.New("permission denied"))
			},
			interactive:  true,
			expectError:  true,
			errorMessage: "failed to initialize configuration directory",
		},
		{
			name: "invalid user input",
			setupMocks: func(repo *MockConfigurationRepository, env *MockEnvironmentProvider, ui *MockUserInteraction) {
				env.On("IsConfigured").Return(false)

				ui.On("PromptString", "Enter Jira Base URL (e.g., https://company.atlassian.net):").Return("", nil)

				repo.On("InitializeConfigDirectory").Return(nil)
			},
			interactive:  true,
			expectError:  true,
			errorMessage: "base URL is required",
		},
		{
			name: "non-interactive with environment configured",
			setupMocks: func(repo *MockConfigurationRepository, env *MockEnvironmentProvider, _ *MockUserInteraction) {
				env.On("IsConfigured").Return(true)
				env.On("GetJiraBaseURL").Return("https://test.atlassian.net")
				env.On("GetJiraEmail").Return("test@company.com")
				env.On("GetJiraToken").Return("test-token")

				repo.On("InitializeConfigDirectory").Return(nil)
				repo.On("SaveJiraConfig", mock.AnythingOfType("*domain.JiraConfig")).Return(nil)
			},
			interactive: false,
			expectedResult: &InitializeConfigResult{
				JiraConfigCreated: true,
				TeamConfigCreated: false,
				Message:           "Configuration initialized from environment variables",
			},
			expectError: false,
		},
		{
			name: "non-interactive without environment configured",
			setupMocks: func(_ *MockConfigurationRepository, env *MockEnvironmentProvider, _ *MockUserInteraction) {
				env.On("IsConfigured").Return(false)
				env.On("GetMissingVars").Return([]string{"JIRA_BASE_URL", "JIRA_EMAIL", "JIRA_TOKEN"})
			},
			interactive:  false,
			expectError:  true,
			errorMessage: "environment variables not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockConfigurationRepository{}
			envProvider := &MockEnvironmentProvider{}
			ui := &MockUserInteraction{}

			tt.setupMocks(repo, envProvider, ui)

			useCase := NewInitializeConfig(repo, envProvider, ui)
			result, err := useCase.Execute(tt.interactive)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedResult.JiraConfigCreated, result.JiraConfigCreated)
				assert.Equal(t, tt.expectedResult.TeamConfigCreated, result.TeamConfigCreated)
				assert.Contains(t, result.Message, "Configuration initialized")
			}

			repo.AssertExpectations(t)
			envProvider.AssertExpectations(t)
			ui.AssertExpectations(t)
		})
	}
}
