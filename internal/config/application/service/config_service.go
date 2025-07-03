package service

import (
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
	configPorts "github.com/helmedeiros/digital-asset-capitalization/internal/config/domain/ports"
	sprintDomain "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
)

// ConfigService provides configuration data to the application
type ConfigService struct {
	repo configPorts.ConfigurationRepository
}

// NewConfigService creates a new configuration service
func NewConfigService(repo configPorts.ConfigurationRepository) *ConfigService {
	return &ConfigService{
		repo: repo,
	}
}

// GetJiraConfig returns the Jira configuration
func (s *ConfigService) GetJiraConfig() (*domain.JiraConfig, error) {
	config, err := s.repo.LoadJiraConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load Jira configuration: %w", err)
	}
	return config, nil
}

// GetTeamConfig returns the team configuration
func (s *ConfigService) GetTeamConfig() (*domain.TeamConfig, error) {
	config, err := s.repo.LoadTeamConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load team configuration: %w", err)
	}
	return config, nil
}

// GetTeamMapForSprint returns team data in the format expected by sprint module
// This provides backward compatibility while using our new infrastructure
func (s *ConfigService) GetTeamMapForSprint() (sprintDomain.TeamMap, error) {
	teamConfig, err := s.GetTeamConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get team configuration: %w", err)
	}

	// Convert from our clean domain format to sprint's expected format
	teamMap := make(sprintDomain.TeamMap)
	for _, project := range teamConfig.GetProjects() {
		members, exists := teamConfig.GetTeam(project)
		if exists {
			teamMap[project] = sprintDomain.Team{
				Team: members,
			}
		}
	}

	return teamMap, nil
}

// GetTeamForProject returns team members for a specific project
func (s *ConfigService) GetTeamForProject(project string) ([]string, error) {
	teamConfig, err := s.GetTeamConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get team configuration: %w", err)
	}

	members, exists := teamConfig.GetTeam(project)
	if !exists {
		return nil, fmt.Errorf("project %s not found in team configuration", project)
	}

	return members, nil
}

// IsTeamMember checks if a person is a member of a project team
func (s *ConfigService) IsTeamMember(project, member string) (bool, error) {
	teamConfig, err := s.GetTeamConfig()
	if err != nil {
		return false, fmt.Errorf("failed to get team configuration: %w", err)
	}

	return teamConfig.IsTeamMember(project, member), nil
}

// ConfigExists checks if configuration files exist
func (s *ConfigService) ConfigExists() (bool, error) {
	return s.repo.ConfigExists()
}

// InitializeConfigDirectory initializes the configuration directory
func (s *ConfigService) InitializeConfigDirectory() error {
	return s.repo.InitializeConfigDirectory()
}
