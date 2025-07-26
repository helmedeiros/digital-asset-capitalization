package ports

import "github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"

// TeamSyncPort defines the contract for extracting team data from external systems
type TeamSyncPort interface {
	// GetProjectMembers retrieves team members for a specific project from JIRA
	GetProjectMembers(projectKey string) (*domain.ProjectTeamData, error)

	// GetProjectRoles retrieves project roles and their members
	GetProjectRoles(projectKey string) (map[string][]domain.TeamMember, error)

	// GetAssignableUsers retrieves users who can be assigned to issues in the project
	GetAssignableUsers(projectKey string) ([]domain.TeamMember, error)
}
