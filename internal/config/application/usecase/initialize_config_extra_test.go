package usecase

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// All of these target the missing-from-the-table-driven-test branches in
// initialize_config.go: SaveJiraConfig + SaveTeamConfig wrap, the
// invalid-environment-non-interactive branch in handleJiraConfiguration,
// every PromptString / PromptPassword / PromptConfirm error wrap in
// promptForJiraConfig / handleTeamConfiguration, and the
// NewJiraConfig + NewTeamConfig wraps.

// ----- Execute-level error wraps -----

func TestInitializeConfig_Execute_SaveJiraConfigErrorWraps(t *testing.T) {
	t.Parallel()
	repo := &MockConfigurationRepository{}
	env := &MockEnvironmentProvider{}
	ui := &MockUserInteraction{}

	env.On("IsConfigured").Return(true)
	env.On("GetJiraBaseURL").Return("https://test.atlassian.net")
	env.On("GetJiraEmail").Return("test@company.com")
	env.On("GetJiraToken").Return("token")
	repo.On("InitializeConfigDirectory").Return(nil)
	repo.On("SaveJiraConfig", mock.AnythingOfType("*domain.JiraConfig")).
		Return(errors.New("disk full"))

	uc := NewInitializeConfig(repo, env, ui)
	_, err := uc.Execute(false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to save Jira configuration")
}

func TestInitializeConfig_Execute_SaveTeamConfigErrorWraps(t *testing.T) {
	t.Parallel()
	repo := &MockConfigurationRepository{}
	env := &MockEnvironmentProvider{}
	ui := &MockUserInteraction{}

	env.On("IsConfigured").Return(true)
	env.On("GetJiraBaseURL").Return("https://test.atlassian.net")
	env.On("GetJiraEmail").Return("test@company.com")
	env.On("GetJiraToken").Return("token")
	repo.On("InitializeConfigDirectory").Return(nil)
	repo.On("SaveJiraConfig", mock.AnythingOfType("*domain.JiraConfig")).Return(nil)

	// Interactive mode: configure team, but SaveTeamConfig fails.
	ui.On("PromptConfirm", "Would you like to configure team members now? (y/n):").Return(true, nil)
	ui.On("PromptString", "Enter project key (e.g., FN):").Return("FN", nil)
	ui.On("PromptString", "Enter team member username:").Return("test.user", nil)
	ui.On("PromptConfirm", "Add another team member? (y/n):").Return(false, nil)
	ui.On("PromptConfirm", "Add another project? (y/n):").Return(false, nil)
	repo.On("SaveTeamConfig", mock.AnythingOfType("*domain.TeamConfig")).
		Return(errors.New("disk full"))

	uc := NewInitializeConfig(repo, env, ui)
	_, err := uc.Execute(true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to save team configuration")
}

// ----- handleJiraConfiguration: env-configured-but-invalid branch -----

func TestInitializeConfig_Execute_InvalidEnvNonInteractiveWraps(t *testing.T) {
	t.Parallel()
	// envProvider.IsConfigured() returns true, but the actual values
	// fail NewJiraConfig validation (empty base URL). In non-interactive
	// mode this surfaces as "invalid environment configuration".
	repo := &MockConfigurationRepository{}
	env := &MockEnvironmentProvider{}
	ui := &MockUserInteraction{}

	env.On("IsConfigured").Return(true)
	env.On("GetJiraBaseURL").Return("") // triggers domain validation error
	env.On("GetJiraEmail").Return("test@company.com")
	env.On("GetJiraToken").Return("token")
	repo.On("InitializeConfigDirectory").Return(nil)

	uc := NewInitializeConfig(repo, env, ui)
	_, err := uc.Execute(false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid environment configuration")
}

func TestInitializeConfig_Execute_InvalidEnvInteractiveFallsBackToPrompt(t *testing.T) {
	t.Parallel()
	// In interactive mode, invalid env config emits a warning and
	// falls through to the prompt path. Drive the prompt path to
	// success to exercise the fallback.
	repo := &MockConfigurationRepository{}
	env := &MockEnvironmentProvider{}
	ui := &MockUserInteraction{}

	env.On("IsConfigured").Return(true)
	env.On("GetJiraBaseURL").Return("") // invalid
	env.On("GetJiraEmail").Return("test@company.com")
	env.On("GetJiraToken").Return("token")
	repo.On("InitializeConfigDirectory").Return(nil)
	ui.On("DisplayWarning", mock.AnythingOfType("string")).Return()
	ui.On("PromptString", "Enter Jira Base URL (e.g., https://company.atlassian.net):").
		Return("https://test.atlassian.net", nil)
	ui.On("PromptString", "Enter Jira Email:").Return("test@company.com", nil)
	ui.On("PromptPassword", "Enter Jira API Token:").Return("token", nil)
	repo.On("SaveJiraConfig", mock.AnythingOfType("*domain.JiraConfig")).Return(nil)
	ui.On("PromptConfirm", "Would you like to configure team members now? (y/n):").Return(false, nil)
	ui.On("DisplaySuccess", mock.AnythingOfType("string")).Return()

	uc := NewInitializeConfig(repo, env, ui)
	_, err := uc.Execute(true)
	require.NoError(t, err)
}

// ----- promptForJiraConfig: each prompt-error branch -----

func TestInitializeConfig_promptForJiraConfig_BaseURLPromptError(t *testing.T) {
	t.Parallel()
	repo := &MockConfigurationRepository{}
	env := &MockEnvironmentProvider{}
	ui := &MockUserInteraction{}

	env.On("IsConfigured").Return(false)
	repo.On("InitializeConfigDirectory").Return(nil)
	ui.On("PromptString", "Enter Jira Base URL (e.g., https://company.atlassian.net):").
		Return("", errors.New("stdin closed"))

	uc := NewInitializeConfig(repo, env, ui)
	_, err := uc.Execute(true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get base URL")
}

func TestInitializeConfig_promptForJiraConfig_EmailPromptError(t *testing.T) {
	t.Parallel()
	repo := &MockConfigurationRepository{}
	env := &MockEnvironmentProvider{}
	ui := &MockUserInteraction{}

	env.On("IsConfigured").Return(false)
	repo.On("InitializeConfigDirectory").Return(nil)
	ui.On("PromptString", "Enter Jira Base URL (e.g., https://company.atlassian.net):").
		Return("https://test.atlassian.net", nil)
	ui.On("PromptString", "Enter Jira Email:").Return("", errors.New("stdin closed"))

	uc := NewInitializeConfig(repo, env, ui)
	_, err := uc.Execute(true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get email")
}

func TestInitializeConfig_promptForJiraConfig_TokenPromptError(t *testing.T) {
	t.Parallel()
	repo := &MockConfigurationRepository{}
	env := &MockEnvironmentProvider{}
	ui := &MockUserInteraction{}

	env.On("IsConfigured").Return(false)
	repo.On("InitializeConfigDirectory").Return(nil)
	ui.On("PromptString", "Enter Jira Base URL (e.g., https://company.atlassian.net):").
		Return("https://test.atlassian.net", nil)
	ui.On("PromptString", "Enter Jira Email:").Return("test@company.com", nil)
	ui.On("PromptPassword", "Enter Jira API Token:").Return("", errors.New("stdin closed"))

	uc := NewInitializeConfig(repo, env, ui)
	_, err := uc.Execute(true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get token")
}

func TestInitializeConfig_promptForJiraConfig_NewJiraConfigValidationError(t *testing.T) {
	t.Parallel()
	// Prompts succeed with values that make NewJiraConfig reject
	// (invalid base URL — not "http(s)://...").
	repo := &MockConfigurationRepository{}
	env := &MockEnvironmentProvider{}
	ui := &MockUserInteraction{}

	env.On("IsConfigured").Return(false)
	repo.On("InitializeConfigDirectory").Return(nil)
	ui.On("PromptString", "Enter Jira Base URL (e.g., https://company.atlassian.net):").
		Return("not-a-url", nil)
	ui.On("PromptString", "Enter Jira Email:").Return("test@company.com", nil)
	ui.On("PromptPassword", "Enter Jira API Token:").Return("token", nil)

	uc := NewInitializeConfig(repo, env, ui)
	_, err := uc.Execute(true)
	require.Error(t, err)
}

// ----- handleTeamConfiguration: each prompt-error branch -----

func TestInitializeConfig_handleTeamConfiguration_ConfirmPromptError(t *testing.T) {
	t.Parallel()
	repo, env, ui := wireValidEnv()
	ui.On("PromptConfirm", "Would you like to configure team members now? (y/n):").
		Return(false, errors.New("stdin closed"))

	uc := NewInitializeConfig(repo, env, ui)
	_, err := uc.Execute(true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get team configuration choice")
}

func TestInitializeConfig_handleTeamConfiguration_ProjectKeyPromptError(t *testing.T) {
	t.Parallel()
	repo, env, ui := wireValidEnv()
	ui.On("PromptConfirm", "Would you like to configure team members now? (y/n):").Return(true, nil)
	ui.On("PromptString", "Enter project key (e.g., FN):").
		Return("", errors.New("stdin closed"))

	uc := NewInitializeConfig(repo, env, ui)
	_, err := uc.Execute(true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get project key")
}

func TestInitializeConfig_handleTeamConfiguration_MemberPromptError(t *testing.T) {
	t.Parallel()
	repo, env, ui := wireValidEnv()
	ui.On("PromptConfirm", "Would you like to configure team members now? (y/n):").Return(true, nil)
	ui.On("PromptString", "Enter project key (e.g., FN):").Return("FN", nil)
	ui.On("PromptString", "Enter team member username:").
		Return("", errors.New("stdin closed"))

	uc := NewInitializeConfig(repo, env, ui)
	_, err := uc.Execute(true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get team member")
}

func TestInitializeConfig_handleTeamConfiguration_AddMorePromptError(t *testing.T) {
	t.Parallel()
	repo, env, ui := wireValidEnv()
	ui.On("PromptConfirm", "Would you like to configure team members now? (y/n):").Return(true, nil)
	ui.On("PromptString", "Enter project key (e.g., FN):").Return("FN", nil)
	ui.On("PromptString", "Enter team member username:").Return("alice", nil)
	ui.On("PromptConfirm", "Add another team member? (y/n):").
		Return(false, errors.New("stdin closed"))

	uc := NewInitializeConfig(repo, env, ui)
	_, err := uc.Execute(true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get add more choice")
}

func TestInitializeConfig_handleTeamConfiguration_AddProjectPromptError(t *testing.T) {
	t.Parallel()
	repo, env, ui := wireValidEnv()
	ui.On("PromptConfirm", "Would you like to configure team members now? (y/n):").Return(true, nil)
	ui.On("PromptString", "Enter project key (e.g., FN):").Return("FN", nil)
	ui.On("PromptString", "Enter team member username:").Return("alice", nil)
	ui.On("PromptConfirm", "Add another team member? (y/n):").Return(false, nil)
	ui.On("PromptConfirm", "Add another project? (y/n):").
		Return(false, errors.New("stdin closed"))

	uc := NewInitializeConfig(repo, env, ui)
	_, err := uc.Execute(true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to get add project choice")
}

// wireValidEnv returns repo/env/ui mocks with the env-configured
// happy path pre-wired — so the test can focus on the
// handleTeamConfiguration branch without re-stating the Jira setup.
func wireValidEnv() (*MockConfigurationRepository, *MockEnvironmentProvider, *MockUserInteraction) {
	repo := &MockConfigurationRepository{}
	env := &MockEnvironmentProvider{}
	ui := &MockUserInteraction{}

	env.On("IsConfigured").Return(true)
	env.On("GetJiraBaseURL").Return("https://test.atlassian.net")
	env.On("GetJiraEmail").Return("test@company.com")
	env.On("GetJiraToken").Return("token")
	repo.On("InitializeConfigDirectory").Return(nil)
	repo.On("SaveJiraConfig", mock.AnythingOfType("*domain.JiraConfig")).Return(nil)
	return repo, env, ui
}
