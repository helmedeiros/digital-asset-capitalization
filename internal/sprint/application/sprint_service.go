package application

import (
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
)

// SprintService defines the interface for sprint management operations
type SprintService interface {
	// ProcessSprint processes a sprint and its issues
	ProcessSprint(project string, sprint *domain.Sprint) error

	// ProcessTeamIssues processes issues for a team
	ProcessTeamIssues(team *domain.Team) error

	// ProcessJiraIssues processes Jira issues and returns CSV data
	ProcessJiraIssues(project, sprint, override string) (string, error)
}
