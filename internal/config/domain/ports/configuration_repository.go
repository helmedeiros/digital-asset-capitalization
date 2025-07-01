package ports

import "github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"

// ConfigurationRepository defines the contract for configuration persistence
type ConfigurationRepository interface {
	// LoadJiraConfig loads Jira configuration
	LoadJiraConfig() (*domain.JiraConfig, error)

	// SaveJiraConfig saves Jira configuration
	SaveJiraConfig(config *domain.JiraConfig) error

	// LoadTeamConfig loads team configuration
	LoadTeamConfig() (*domain.TeamConfig, error)

	// SaveTeamConfig saves team configuration
	SaveTeamConfig(config *domain.TeamConfig) error

	// ConfigExists checks if configuration files exist
	ConfigExists() (bool, error)

	// InitializeConfigDirectory creates the configuration directory if needed
	InitializeConfigDirectory() error
}
