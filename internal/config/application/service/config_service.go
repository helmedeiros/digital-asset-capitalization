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
				Team:               members,
				ExcludedIssueTypes: teamConfig.GetExcludedIssueTypes(project),
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

// SaveTeamConfig saves the team configuration
func (s *ConfigService) SaveTeamConfig(config *domain.TeamConfig) error {
	return s.repo.SaveTeamConfig(config)
}

// SetTribeForProject sets the tribe for a specific project and saves the configuration
func (s *ConfigService) SetTribeForProject(project, tribe string) error {
	config, err := s.GetTeamConfig()
	if err != nil {
		return fmt.Errorf("failed to load team configuration: %w", err)
	}

	if err := config.SetTribe(project, tribe); err != nil {
		return fmt.Errorf("failed to set tribe: %w", err)
	}

	if err := s.SaveTeamConfig(config); err != nil {
		return fmt.Errorf("failed to save team configuration: %w", err)
	}

	return nil
}

// GetTribeForProject returns the tribe for a specific project
func (s *ConfigService) GetTribeForProject(project string) (string, error) {
	config, err := s.GetTeamConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load team configuration: %w", err)
	}

	return config.GetTribe(project), nil
}

// SetCompanyForProject sets the company for a specific project and saves the configuration
func (s *ConfigService) SetCompanyForProject(project, company string) error {
	config, err := s.GetTeamConfig()
	if err != nil {
		return fmt.Errorf("failed to load team configuration: %w", err)
	}

	if err := config.SetCompany(project, company); err != nil {
		return fmt.Errorf("failed to set company: %w", err)
	}

	if err := s.SaveTeamConfig(config); err != nil {
		return fmt.Errorf("failed to save team configuration: %w", err)
	}

	return nil
}

// GetCompanyForProject returns the company for a specific project
func (s *ConfigService) GetCompanyForProject(project string) (string, error) {
	config, err := s.GetTeamConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load team configuration: %w", err)
	}

	return config.GetCompany(project), nil
}

// SetExcludedIssueTypesForProject sets the excluded issue types for a project and saves
func (s *ConfigService) SetExcludedIssueTypesForProject(project string, types []string) error {
	config, err := s.GetTeamConfig()
	if err != nil {
		return fmt.Errorf("failed to load team configuration: %w", err)
	}

	if err := config.SetExcludedIssueTypes(project, types); err != nil {
		return fmt.Errorf("failed to set excluded issue types: %w", err)
	}

	if err := s.SaveTeamConfig(config); err != nil {
		return fmt.Errorf("failed to save team configuration: %w", err)
	}

	return nil
}

// GetExcludedIssueTypesForProject returns the excluded issue types for a project
func (s *ConfigService) GetExcludedIssueTypesForProject(project string) ([]string, error) {
	config, err := s.GetTeamConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load team configuration: %w", err)
	}

	return config.GetExcludedIssueTypes(project), nil
}

// GetBoardWorkStream returns the work stream for a specific board in a project
func (s *ConfigService) GetBoardWorkStream(project string, boardID int) (string, error) {
	config, err := s.GetTeamConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load team configuration: %w", err)
	}

	return config.GetBoardWorkStream(project, boardID), nil
}

// GetBoardsForWorkStream returns board IDs for a given work stream in a project
func (s *ConfigService) GetBoardsForWorkStream(project, workStream string) ([]int, error) {
	config, err := s.GetTeamConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load team configuration: %w", err)
	}

	return config.GetBoardsForWorkStream(project, workStream), nil
}

// SetBoardWorkStreams sets the board-to-workstream mapping for a project and saves
func (s *ConfigService) SetBoardWorkStreams(project string, mapping map[int]string) error {
	config, err := s.GetTeamConfig()
	if err != nil {
		return fmt.Errorf("failed to load team configuration: %w", err)
	}

	if err := config.SetBoardWorkStreams(project, mapping); err != nil {
		return fmt.Errorf("failed to set board work streams: %w", err)
	}

	if err := s.SaveTeamConfig(config); err != nil {
		return fmt.Errorf("failed to save team configuration: %w", err)
	}

	return nil
}

// SetBoardWorkStream sets a single board-to-workstream mapping for a project and saves
func (s *ConfigService) SetBoardWorkStream(project string, boardID int, workStream string) error {
	config, err := s.GetTeamConfig()
	if err != nil {
		return fmt.Errorf("failed to load team configuration: %w", err)
	}

	existing := config.GetBoardWorkStreams(project)
	if existing == nil {
		existing = make(map[int]string)
	}
	existing[boardID] = workStream

	if err := config.SetBoardWorkStreams(project, existing); err != nil {
		return fmt.Errorf("failed to set board work stream: %w", err)
	}

	if err := s.SaveTeamConfig(config); err != nil {
		return fmt.Errorf("failed to save team configuration: %w", err)
	}

	return nil
}

// InitializeConfigDirectory initializes the configuration directory
func (s *ConfigService) InitializeConfigDirectory() error {
	return s.repo.InitializeConfigDirectory()
}
