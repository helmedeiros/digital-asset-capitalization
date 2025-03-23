package infrastructure

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/config"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/ports"
)

// JiraAdapter implements the JiraPort interface
type JiraAdapter struct {
	config     *config.JiraConfig
	teams      domain.TeamMap
	httpClient *HTTPClient
}

// NewJiraAdapter creates a new Jira adapter
func NewJiraAdapter() (*JiraAdapter, error) {
	// Load Jira configuration
	jiraConfig, err := config.NewJiraConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load Jira configuration: %w", err)
	}

	// Load teams data
	data, err := ioutil.ReadFile("teams.json")
	if err != nil {
		return nil, fmt.Errorf("error reading teams.json: %w", err)
	}

	var teams domain.TeamMap
	if err := json.Unmarshal(data, &teams); err != nil {
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
func (a *JiraAdapter) GetIssuesForSprint(sprintID string) ([]ports.JiraIssue, error) {
	query := fmt.Sprintf("sprint = '%s'", sprintID)
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
	return a.GetIssuesForSprint(sprint.ID)
}

// GetTeamIssues retrieves all issues for a team
func (a *JiraAdapter) GetTeamIssues(team *domain.Team) ([]ports.JiraIssue, error) {
	var allIssues []ports.JiraIssue
	for _, member := range team.Members {
		issues, err := a.GetIssuesForTeamMember(member)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch issues for team member %s: %w", member, err)
		}
		allIssues = append(allIssues, issues...)
	}
	return allIssues, nil
}

// convertToPortIssues converts domain JiraIssue to port JiraIssue
func (a *JiraAdapter) convertToPortIssues(issues []domain.JiraIssue) []ports.JiraIssue {
	var portIssues []ports.JiraIssue
	for _, issue := range issues {
		portIssues = append(portIssues, ports.JiraIssue{
			Key:         issue.Key,
			Summary:     issue.Fields.Summary,
			Assignee:    issue.Fields.Assignee.DisplayName,
			Status:      issue.Fields.Status.Name,
			StoryPoints: issue.Fields.StoryPoints,
		})
	}
	return portIssues
}

// Ensure JiraAdapter implements JiraPort
var _ ports.JiraPort = (*JiraAdapter)(nil)
