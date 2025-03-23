package infrastructure

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/config"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
)

// JiraAdapter implements the JiraPort interface
type JiraAdapter struct {
	config     *config.JiraConfig
	teams      domain.TeamMap
	httpClient *HTTPClient
}

// NewJiraAdapter creates a new Jira adapter
func NewJiraAdapter(teamsFilePath string) (*JiraAdapter, error) {
	// Load Jira configuration
	jiraConfig, err := config.NewJiraConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load Jira configuration: %w", err)
	}

	// Load teams data
	var teams domain.TeamMap
	var teamsData []byte

	// Try to read teams.json from the specified path
	teamsData, err = ioutil.ReadFile(teamsFilePath)
	if err != nil {
		// Try to use teams.json.template as fallback
		teamsData, err = ioutil.ReadFile(teamsFilePath + ".template")
		if err != nil {
			// Create a default teams.json file
			teamsData = []byte(`{
				"FN": {
					"team": ["helio.medeiros", "julio.medeiros"]
				}
			}`)
			err = ioutil.WriteFile(teamsFilePath, teamsData, 0644)
			if err != nil {
				return nil, fmt.Errorf("failed to create default teams.json: %w", err)
			}
		}
	}

	if err := json.Unmarshal(teamsData, &teams); err != nil {
		return nil, fmt.Errorf("error unmarshaling teams data: %w", err)
	}

	// Create HTTP client
	httpClient := NewHTTPClient(jiraConfig.GetBaseURL(), jiraConfig.GetAuthHeader())

	return &JiraAdapter{
		config:     jiraConfig,
		teams:      teams,
		httpClient: httpClient,
	}, nil
}

// GetIssuesForSprint retrieves all issues for a given sprint
func (a *JiraAdapter) GetIssuesForSprint(project, sprintID string) ([]ports.JiraIssue, error) {
	query := fmt.Sprintf("project = %s AND sprint = '%s'", project, sprintID)
	encodedQuery := url.QueryEscape(query)
	fields := "summary,assignee,status,changelog"
	jiraURL := fmt.Sprintf("%s/rest/api/3/search?jql=%s&expand=changelog&fields=%s",
		a.config.GetBaseURL(), encodedQuery, fields)

	issues, err := a.httpClient.GetJiraIssues(jiraURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sprint issues: %w", err)
	}

	return a.convertToPortIssues(issues), nil
}

// GetIssuesForTeamMember retrieves all issues assigned to a team member
func (a *JiraAdapter) GetIssuesForTeamMember(member string) ([]ports.JiraIssue, error) {
	query := fmt.Sprintf("assignee = '%s'", member)
	encodedQuery := url.QueryEscape(query)
	fields := "summary,assignee,status,changelog"
	jiraURL := fmt.Sprintf("%s/rest/api/3/search?jql=%s&expand=changelog&fields=%s",
		a.config.GetBaseURL(), encodedQuery, fields)

	issues, err := a.httpClient.GetJiraIssues(jiraURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch team member issues: %w", err)
	}

	return a.convertToPortIssues(issues), nil
}

// GetSprintIssues retrieves all issues in a sprint
func (a *JiraAdapter) GetSprintIssues(sprint *domain.Sprint) ([]ports.JiraIssue, error) {
	return a.GetIssuesForSprint(sprint.Project, sprint.ID)
}

// GetTeamIssues retrieves all issues for a team
func (a *JiraAdapter) GetTeamIssues(team *domain.Team) ([]ports.JiraIssue, error) {
	var allIssues []ports.JiraIssue

	// Get issues for each team member
	for _, member := range team.Team {
		issues, err := a.GetIssuesForTeamMember(member)
		if err != nil {
			return nil, fmt.Errorf("failed to get issues for team member %s: %w", member, err)
		}
		allIssues = append(allIssues, issues...)
	}

	return allIssues, nil
}

// convertToPortIssues converts domain JiraIssue to port JiraIssue
func (a *JiraAdapter) convertToPortIssues(issues []domain.JiraIssue) []ports.JiraIssue {
	var portIssues []ports.JiraIssue
	for _, issue := range issues {
		portIssue := ports.JiraIssue{
			Key:         issue.Key,
			Summary:     issue.Fields.Summary,
			Assignee:    issue.Fields.Assignee.DisplayName,
			Status:      issue.Fields.Status.Name,
			StoryPoints: issue.Fields.StoryPoints,
			Changelog: ports.JiraChangelog{
				Histories: make([]ports.JiraChangeHistory, len(issue.Changelog.Histories)),
			},
		}

		// Convert changelog histories
		for i, history := range issue.Changelog.Histories {
			portHistory := ports.JiraChangeHistory{
				Created: history.Created,
				Items:   make([]ports.JiraChangeItem, len(history.Items)),
			}

			// Convert changelog items
			for j, item := range history.Items {
				portHistory.Items[j] = ports.JiraChangeItem{
					Field:      item.Field,
					FromString: item.FromString,
					ToString:   item.ToString,
				}
			}

			portIssue.Changelog.Histories[i] = portHistory
		}

		portIssues = append(portIssues, portIssue)
	}
	return portIssues
}

// Ensure JiraAdapter implements JiraPort
var _ ports.JiraPort = (*JiraAdapter)(nil)
