package usecase

import (
	"fmt"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain/ports"
)

// SyncTeamFromJira represents the use case for synchronizing team data from JIRA
type SyncTeamFromJira struct {
	teamSyncPort ports.TeamSyncPort
	configRepo   ports.ConfigurationRepository
}

// NewSyncTeamFromJira creates a new instance of the SyncTeamFromJira use case
func NewSyncTeamFromJira(
	teamSyncPort ports.TeamSyncPort,
	configRepo ports.ConfigurationRepository,
) *SyncTeamFromJira {
	return &SyncTeamFromJira{
		teamSyncPort: teamSyncPort,
		configRepo:   configRepo,
	}
}

// Execute synchronizes team members from JIRA for the specified project
func (s *SyncTeamFromJira) Execute(projectKey string) (*domain.TeamSyncResult, error) {
	if projectKey == "" {
		return nil, fmt.Errorf("project key is required")
	}

	// Get current team configuration
	currentTeamConfig, err := s.configRepo.LoadTeamConfig()
	if err != nil {
		// If team config doesn't exist, create a new one
		currentTeamConfig, err = domain.NewTeamConfig(make(map[string][]string))
		if err != nil {
			result := domain.NewTeamSyncResult(projectKey, []domain.TeamMember{}, "jira")
			result.AddError(fmt.Sprintf("Failed to create team configuration: %v", err), "config")
			return result, nil
		}
	}

	// Extract team members from JIRA
	projectTeamData, err := s.teamSyncPort.GetProjectMembers(projectKey)
	if err != nil {
		result := domain.NewTeamSyncResult(projectKey, []domain.TeamMember{}, "jira")
		result.AddError(fmt.Sprintf("Failed to extract team members from JIRA: %v", err), "jira")
		return result, nil // Return result with error instead of failing completely
	}

	// Validate the extracted data
	if err := projectTeamData.Validate(); err != nil {
		result := domain.NewTeamSyncResult(projectKey, []domain.TeamMember{}, "jira")
		result.AddError(fmt.Sprintf("Invalid team data: %v", err), "validation")
		return result, nil
	}

	// Create sync result
	result := domain.NewTeamSyncResult(projectKey, projectTeamData.Members, "jira")

	// Calculate changes (added/removed members)
	currentMembers, _ := currentTeamConfig.GetTeam(projectKey)
	newMemberNames := result.GetMemberNames()

	result.AddedMembers = findAddedMembers(currentMembers, newMemberNames)
	result.RemovedMembers = findRemovedMembers(currentMembers, newMemberNames)

	// Update team configuration
	if err := currentTeamConfig.SetTeam(projectKey, newMemberNames); err != nil {
		result.AddError(fmt.Sprintf("Failed to update team configuration: %v", err), "config")
		return result, nil
	}

	// Save updated configuration
	if err := s.configRepo.SaveTeamConfig(currentTeamConfig); err != nil {
		result.AddError(fmt.Sprintf("Failed to save team configuration: %v", err), "persistence")
		return result, nil
	}

	// Validate final result
	if err := result.Validate(); err != nil {
		result.AddError(fmt.Sprintf("Result validation failed: %v", err), "validation")
	}

	return result, nil
}

// findAddedMembers returns members that are in newMembers but not in currentMembers
func findAddedMembers(currentMembers, newMembers []string) []string {
	currentSet := make(map[string]bool)
	for _, member := range currentMembers {
		currentSet[member] = true
	}

	var added []string
	for _, member := range newMembers {
		if !currentSet[member] {
			added = append(added, member)
		}
	}
	return added
}

// findRemovedMembers returns members that are in currentMembers but not in newMembers
func findRemovedMembers(currentMembers, newMembers []string) []string {
	newSet := make(map[string]bool)
	for _, member := range newMembers {
		newSet[member] = true
	}

	var removed []string
	for _, member := range currentMembers {
		if !newSet[member] {
			removed = append(removed, member)
		}
	}
	return removed
}
