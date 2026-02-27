package ports

import (
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
)

// Sprint represents a sprint in the ports layer
type Sprint struct {
	ID        string
	Name      string
	State     string
	StartDate string
	EndDate   string
	Goal      string
}

// JiraIssue represents a Jira issue in the ports layer
type JiraIssue struct {
	Key              string
	Summary          string
	Assignee         string
	Status           string
	StoryPoints      *float64
	IssueType        string
	Labels           []string
	Changelog        JiraChangelog
	TPDBusinessUnits []string
	EngineeringHours *float64
	WorkStream       string
	BoardWorkStream  string // derived from board-to-workstream config, used as fallback
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

// CustomFieldUpdate holds values to write back to JIRA custom fields
type CustomFieldUpdate struct {
	EngineeringHours *float64
	WorkStream       *string
	TPDBusinessUnits []string
}

// CustomFieldValues holds the current values of JIRA custom fields for an issue
type CustomFieldValues struct {
	EngineeringHours *float64
	WorkStream       string
	TPDBusinessUnits []string
}

// JiraPort defines the interface for Jira integration
type JiraPort interface {
	// GetSprintsForProject retrieves all sprints for a given project
	GetSprintsForProject(project string, states []string) ([]Sprint, error)
	// GetSprintsForProjectWithBoardInfo retrieves sprints with board information
	GetSprintsForProjectWithBoardInfo(project string, states []string) ([]Sprint, []BoardInfo, error)
	// GetIssuesForSprint retrieves all issues for a given sprint
	GetIssuesForSprint(project, sprintID string) ([]JiraIssue, error)
	// GetIssuesForTeamMember retrieves all issues assigned to a team member
	GetIssuesForTeamMember(member string) ([]JiraIssue, error)
	// GetSprintIssues retrieves all issues in a sprint
	GetSprintIssues(sprint *domain.Sprint) ([]JiraIssue, error)
	// GetTeamIssues retrieves all issues for a team
	GetTeamIssues(team *domain.Team) ([]JiraIssue, error)
	// GetSprintByName retrieves sprint details by project and sprint name
	GetSprintByName(project, sprintName string) (*Sprint, error)
	// GetIssuesForSprintOnBoard retrieves issues for a sprint filtered to a specific board
	GetIssuesForSprintOnBoard(project, sprintName string, boardID int) ([]JiraIssue, error)
	// UpdateCustomFields writes custom field values to a JIRA issue
	UpdateCustomFields(issueKey string, update CustomFieldUpdate) error
	// FetchCustomFields reads current custom field values from a JIRA issue
	FetchCustomFields(issueKey string) (*CustomFieldValues, error)
}

// BoardInfo represents information about a board
type BoardInfo struct {
	ID         int
	Name       string
	Type       string
	HasSprints bool
}
