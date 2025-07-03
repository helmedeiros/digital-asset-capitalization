package usecase

import (
	"fmt"
	"strings"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain/ports"
)

// InitializeConfigResult represents the result of configuration initialization
type InitializeConfigResult struct {
	JiraConfigCreated bool
	TeamConfigCreated bool
	Message           string
}

// InitializeConfig handles the initialization of configuration
type InitializeConfig struct {
	repo        ports.ConfigurationRepository
	envProvider ports.EnvironmentProvider
	ui          ports.UserInteraction
}

// NewInitializeConfig creates a new InitializeConfig use case
func NewInitializeConfig(
	repo ports.ConfigurationRepository,
	envProvider ports.EnvironmentProvider,
	ui ports.UserInteraction,
) *InitializeConfig {
	return &InitializeConfig{
		repo:        repo,
		envProvider: envProvider,
		ui:          ui,
	}
}

// Execute initializes the configuration
func (uc *InitializeConfig) Execute(interactive bool) (*InitializeConfigResult, error) {
	// Check environment first for early exit on non-interactive mode
	envConfigured := uc.envProvider.IsConfigured()

	// For non-interactive mode, fail early if environment is not configured
	if !interactive && !envConfigured {
		missingVars := uc.envProvider.GetMissingVars()
		return nil, fmt.Errorf("environment variables not configured. Missing: %v", missingVars)
	}

	// Initialize configuration directory
	if err := uc.repo.InitializeConfigDirectory(); err != nil {
		return nil, fmt.Errorf("failed to initialize configuration directory: %w", err)
	}

	result := &InitializeConfigResult{}

	// Handle Jira configuration
	jiraConfig, err := uc.handleJiraConfiguration(interactive, envConfigured)
	if err != nil {
		return nil, err
	}

	if jiraConfig != nil {
		if err := uc.repo.SaveJiraConfig(jiraConfig); err != nil {
			return nil, fmt.Errorf("failed to save Jira configuration: %w", err)
		}
		result.JiraConfigCreated = true
	}

	// Handle team configuration
	if interactive {
		teamConfig, err := uc.handleTeamConfiguration()
		if err != nil {
			return nil, err
		}

		if teamConfig != nil {
			if err := uc.repo.SaveTeamConfig(teamConfig); err != nil {
				return nil, fmt.Errorf("failed to save team configuration: %w", err)
			}
			result.TeamConfigCreated = true
		}
	}

	// Set appropriate message
	if envConfigured && !interactive {
		result.Message = "Configuration initialized from environment variables"
	} else {
		result.Message = "Configuration initialized successfully"
	}

	if interactive {
		uc.ui.DisplaySuccess(result.Message)
	}

	return result, nil
}

// handleJiraConfiguration handles Jira configuration setup
func (uc *InitializeConfig) handleJiraConfiguration(interactive bool, envConfigured bool) (*domain.JiraConfig, error) {
	// Check if environment variables are configured
	if envConfigured {
		// Create config from environment variables
		baseURL := uc.envProvider.GetJiraBaseURL()
		email := uc.envProvider.GetJiraEmail()
		token := uc.envProvider.GetJiraToken()

		config, err := domain.NewJiraConfig(baseURL, email, token)
		if err != nil {
			if interactive {
				uc.ui.DisplayWarning("Environment variables are set but invalid. Please provide correct values.")
				return uc.promptForJiraConfig()
			}
			return nil, fmt.Errorf("invalid environment configuration: %w", err)
		}

		return config, nil
	}

	// Environment not configured - this should only happen in interactive mode
	// since we check this early in Execute()
	if !interactive {
		missingVars := uc.envProvider.GetMissingVars()
		return nil, fmt.Errorf("environment variables not configured. Missing: %v", missingVars)
	}

	// Interactive mode - prompt user
	return uc.promptForJiraConfig()
}

// promptForJiraConfig prompts user for Jira configuration
func (uc *InitializeConfig) promptForJiraConfig() (*domain.JiraConfig, error) {
	baseURL, err := uc.ui.PromptString("Enter Jira Base URL (e.g., https://company.atlassian.net):")
	if err != nil {
		return nil, fmt.Errorf("failed to get base URL: %w", err)
	}

	// Validate baseURL immediately to fail fast
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	email, err := uc.ui.PromptString("Enter Jira Email:")
	if err != nil {
		return nil, fmt.Errorf("failed to get email: %w", err)
	}

	token, err := uc.ui.PromptPassword("Enter Jira API Token:")
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	config, err := domain.NewJiraConfig(baseURL, email, token)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// handleTeamConfiguration handles team configuration setup
func (uc *InitializeConfig) handleTeamConfiguration() (*domain.TeamConfig, error) {
	configure, err := uc.ui.PromptConfirm("Would you like to configure team members now? (y/n):")
	if err != nil {
		return nil, fmt.Errorf("failed to get team configuration choice: %w", err)
	}

	if !configure {
		return nil, nil
	}

	teams := make(map[string][]string)

	for {
		project, err := uc.ui.PromptString("Enter project key (e.g., FN):")
		if err != nil {
			return nil, fmt.Errorf("failed to get project key: %w", err)
		}

		var members []string
		for {
			member, err := uc.ui.PromptString("Enter team member username:")
			if err != nil {
				return nil, fmt.Errorf("failed to get team member: %w", err)
			}

			members = append(members, member)

			addMore, err := uc.ui.PromptConfirm("Add another team member? (y/n):")
			if err != nil {
				return nil, fmt.Errorf("failed to get add more choice: %w", err)
			}

			if !addMore {
				break
			}
		}

		teams[project] = members

		addProject, err := uc.ui.PromptConfirm("Add another project? (y/n):")
		if err != nil {
			return nil, fmt.Errorf("failed to get add project choice: %w", err)
		}

		if !addProject {
			break
		}
	}

	config, err := domain.NewTeamConfig(teams)
	if err != nil {
		return nil, fmt.Errorf("failed to create team configuration: %w", err)
	}

	return config, nil
}
