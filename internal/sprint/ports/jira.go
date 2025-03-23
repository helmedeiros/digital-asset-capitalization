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
	Changelog   JiraChangelog
}

// JiraChangelog represents the changelog of a Jira issue
type JiraChangelog struct {
	Histories []JiraChangeHistory
}

// JiraChangeHistory represents a historical change in a Jira issue
type JiraChangeHistory struct {
	Created string
	Items   []JiraChangeItem
}

// JiraChangeItem represents a single change in a Jira issue's history
type JiraChangeItem struct {
	Field      string
	FromString string
	ToString   string
}

// JiraPort defines the interface for Jira integration
type JiraPort interface {
	// GetIssuesForSprint retrieves all issues for a given sprint
	GetIssuesForSprint(project, sprintID string) ([]JiraIssue, error)
	// GetIssuesForTeamMember retrieves all issues assigned to a team member
	GetIssuesForTeamMember(member string) ([]JiraIssue, error)
	// GetSprintIssues retrieves all issues in a sprint
	GetSprintIssues(sprint *domain.Sprint) ([]JiraIssue, error)
	// GetTeamIssues retrieves all issues for a team
	GetTeamIssues(team *domain.Team) ([]JiraIssue, error)
}
