package infrastructure

import (
	"fmt"
	"strings"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain/ports"
)

// TeamConfigAdapter implements the TeamConfigPort using the existing team configuration
type TeamConfigAdapter struct {
	configRepo ports.ConfigurationRepository
}

// NewTeamConfigAdapter creates a new team configuration adapter
func NewTeamConfigAdapter(configRepo ports.ConfigurationRepository) *TeamConfigAdapter {
	return &TeamConfigAdapter{
		configRepo: configRepo,
	}
}

// GetTeamForUser returns the team name for a given user (by email or display name)
func (a *TeamConfigAdapter) GetTeamForUser(userIdentifier string) (string, error) {
	// Load team configuration
	teamConfig, err := a.configRepo.LoadTeamConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load team config: %w", err)
	}

	// Search through all teams for the user
	teams := teamConfig.ToMap()
	for teamName, members := range teams {
		for _, member := range members {
			if a.matchesUser(userIdentifier, member) {
				return teamName, nil
			}
		}
	}

	return "", fmt.Errorf("user not found in any team: %s", userIdentifier)
}

// GetAllTeams returns all configured teams
func (a *TeamConfigAdapter) GetAllTeams() (map[string][]string, error) {
	teamConfig, err := a.configRepo.LoadTeamConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load team config: %w", err)
	}

	return teamConfig.ToMap(), nil
}

// GetTribeForTeam returns the tribe for a given team
func (a *TeamConfigAdapter) GetTribeForTeam(teamName string) (string, error) {
	teamConfig, err := a.configRepo.LoadTeamConfig()
	if err != nil {
		return "", fmt.Errorf("failed to load team config: %w", err)
	}

	tribe := teamConfig.GetTribe(teamName)
	return tribe, nil
}

// matchesUser checks if a user identifier matches a team member
// Supports matching by email address, display name, or partial matching
func (a *TeamConfigAdapter) matchesUser(userIdentifier, teamMember string) bool {
	userIdentifier = strings.ToLower(strings.TrimSpace(userIdentifier))
	teamMember = strings.ToLower(strings.TrimSpace(teamMember))

	// Exact match
	if userIdentifier == teamMember {
		return true
	}

	// Check if userIdentifier is an email and teamMember contains the username part
	if strings.Contains(userIdentifier, "@") {
		emailParts := strings.Split(userIdentifier, "@")
		if len(emailParts) >= 1 {
			username := emailParts[0]
			if strings.Contains(teamMember, username) {
				return true
			}
		}
	}

	// Check if teamMember is an email and userIdentifier contains the username part
	if strings.Contains(teamMember, "@") {
		emailParts := strings.Split(teamMember, "@")
		if len(emailParts) >= 1 {
			username := emailParts[0]
			if strings.Contains(userIdentifier, username) {
				return true
			}
		}
	}

	// Check for partial matches in display names
	// This handles cases where JIRA display name is "John Doe" and team member is "john.doe"
	if strings.Contains(teamMember, strings.ReplaceAll(userIdentifier, " ", ".")) {
		return true
	}
	if strings.Contains(userIdentifier, strings.ReplaceAll(teamMember, " ", ".")) {
		return true
	}

	// Check for partial matches with spaces and dots
	userNormalized := strings.ReplaceAll(strings.ReplaceAll(userIdentifier, " ", ""), ".", "")
	memberNormalized := strings.ReplaceAll(strings.ReplaceAll(teamMember, " ", ""), ".", "")

	return userNormalized == memberNormalized
}
