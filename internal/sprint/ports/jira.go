package ports

import (
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
)

// JiraIssue represents a Jira issue
type JiraIssue struct {
	Key         string
	Summary     string
	Assignee    string
	Status      string
	StoryPoints *float64
}

// JiraPort defines the interface for Jira integration
type JiraPort interface {
	// GetIssuesForSprint retrieves all issues for a given sprint
	GetIssuesForSprint(sprintID string) ([]JiraIssue, error)
	// GetIssuesForTeamMember retrieves all issues assigned to a team member
	GetIssuesForTeamMember(member string) ([]JiraIssue, error)
	// GetSprintIssues retrieves all issues in a sprint
	GetSprintIssues(sprint *domain.Sprint) ([]JiraIssue, error)
	// GetTeamIssues retrieves all issues for a team
	GetTeamIssues(team *domain.Team) ([]JiraIssue, error)
}
