package infrastructure

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

// JiraTeamSyncAdapter implements the TeamSyncPort for extracting team data from JIRA
type JiraTeamSyncAdapter struct {
	configService ConfigServiceInterface
	httpClient    *http.Client
}

// JiraUser represents a user in JIRA API responses
type JiraUser struct {
	AccountID    string `json:"accountId"`
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
	Name         string `json:"name,omitempty"` // May not be present in newer JIRA versions
	Active       bool   `json:"active"`
}

// JiraRole represents a role in JIRA API responses
type JiraRole struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// JiraProjectRole represents a project role with its members
type JiraProjectRole struct {
	Role   JiraRole   `json:"role"`
	Actors []JiraUser `json:"actors"`
}

// JiraAssignableUserResponse represents the response from the assignable users API
type JiraAssignableUserResponse []JiraUser

// JiraSearchResponse represents the response from the JIRA search API
type JiraSearchResponse struct {
	Issues []JiraIssue `json:"issues"`
	Total  int         `json:"total"`
}

// JiraIssue represents an issue in JIRA search API responses
type JiraIssue struct {
	Fields JiraIssueFields `json:"fields"`
}

// JiraIssueFields represents the fields of a JIRA issue
type JiraIssueFields struct {
	Assignee *JiraUser `json:"assignee"`
}

// ConfigServiceInterface defines the interface for config service
type ConfigServiceInterface interface {
	GetJiraConfig() (*domain.JiraConfig, error)
}

// NewJiraTeamSyncAdapter creates a new JIRA team sync adapter
func NewJiraTeamSyncAdapter(configService ConfigServiceInterface) (*JiraTeamSyncAdapter, error) {
	if configService == nil {
		return nil, fmt.Errorf("config service is required")
	}

	return &JiraTeamSyncAdapter{
		configService: configService,
		httpClient:    &http.Client{},
	}, nil
}

// GetProjectMembers retrieves team members for a specific project from JIRA
func (a *JiraTeamSyncAdapter) GetProjectMembers(projectKey string) (*domain.ProjectTeamData, error) {
	if projectKey == "" {
		return nil, fmt.Errorf("project key is required")
	}

	// Try multiple strategies to get team members
	members, err := a.getAssignableUsers(projectKey)
	if err != nil {
		// Fallback to project roles if assignable users fails
		roleMembers, roleErr := a.getProjectRoleMembers(projectKey)
		if roleErr != nil {
			return nil, fmt.Errorf("failed to get team members: assignable users error: %v, project roles error: %v", err, roleErr)
		}
		members = roleMembers
	}

	if len(members) == 0 {
		return nil, fmt.Errorf("no team members found for project %s", projectKey)
	}

	return &domain.ProjectTeamData{
		ProjectKey: projectKey,
		Members:    members,
	}, nil
}

// GetProjectRoles retrieves project roles and their members
func (a *JiraTeamSyncAdapter) GetProjectRoles(projectKey string) (map[string][]domain.TeamMember, error) {
	if projectKey == "" {
		return nil, fmt.Errorf("project key is required")
	}

	jiraConfig, err := a.configService.GetJiraConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get JIRA config: %v", err)
	}

	// Get all project roles
	rolesURL := fmt.Sprintf("%s/rest/api/3/project/%s/role", jiraConfig.BaseURL(), projectKey)
	roleIDs, err := a.getProjectRoleIDs(rolesURL, jiraConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get project roles: %v", err)
	}

	roles := make(map[string][]domain.TeamMember)
	for roleName, roleID := range roleIDs {
		roleURL := fmt.Sprintf("%s/rest/api/3/project/%s/role/%d", jiraConfig.BaseURL(), projectKey, roleID)
		roleMembers, err := a.getProjectRoleDetails(roleURL, jiraConfig)
		if err != nil {
			// Continue with other roles if one fails
			continue
		}
		roles[roleName] = roleMembers
	}

	return roles, nil
}

// GetAssignableUsers retrieves users who can be assigned to issues in the project
func (a *JiraTeamSyncAdapter) GetAssignableUsers(projectKey string) ([]domain.TeamMember, error) {
	return a.getAssignableUsers(projectKey)
}

// getAssignableUsers internal method to get active team members from recent issue assignments
func (a *JiraTeamSyncAdapter) getAssignableUsers(projectKey string) ([]domain.TeamMember, error) {
	jiraConfig, err := a.configService.GetJiraConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get JIRA config: %v", err)
	}

	// Use search API to get recent issues with assignees to identify active team members
	jql := fmt.Sprintf("project=%s AND assignee is not EMPTY", projectKey)
	encodedJQL := url.QueryEscape(jql)
	searchURL := fmt.Sprintf("%s/rest/api/3/search?jql=%s&maxResults=100&fields=assignee",
		jiraConfig.BaseURL(), encodedJQL)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", jiraConfig.AuthHeader())
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("JIRA API error (status %d): %s", resp.StatusCode, string(body))
	}

	var searchResponse JiraSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// Extract unique assignees from issues
	uniqueAssignees := make(map[string]*JiraUser)
	for _, issue := range searchResponse.Issues {
		if issue.Fields.Assignee != nil && issue.Fields.Assignee.Active {
			uniqueAssignees[issue.Fields.Assignee.AccountID] = issue.Fields.Assignee
		}
	}

	// Convert unique assignees to slice
	jiraUsers := make([]JiraUser, 0, len(uniqueAssignees))
	for _, user := range uniqueAssignees {
		jiraUsers = append(jiraUsers, *user)
	}

	return a.convertJiraUsersToTeamMembers(jiraUsers), nil
}

// getProjectRoleMembers gets team members from project roles
func (a *JiraTeamSyncAdapter) getProjectRoleMembers(projectKey string) ([]domain.TeamMember, error) {
	roles, err := a.GetProjectRoles(projectKey)
	if err != nil {
		return nil, err
	}

	// Aggregate all members from all roles
	memberMap := make(map[string]domain.TeamMember)
	for _, roleMembers := range roles {
		for _, member := range roleMembers {
			memberMap[member.AccountID] = member
		}
	}

	// Convert map to slice
	members := make([]domain.TeamMember, 0, len(memberMap))
	for _, member := range memberMap {
		members = append(members, member)
	}

	return members, nil
}

// getProjectRoleIDs gets all role IDs for a project
func (a *JiraTeamSyncAdapter) getProjectRoleIDs(url string, jiraConfig *domain.JiraConfig) (map[string]int64, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", jiraConfig.AuthHeader())
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("JIRA API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response as map[string]string (role name -> role URL)
	var rolesResponse map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&rolesResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	// Extract role IDs from URLs
	roleIDs := make(map[string]int64)
	for roleName, roleURL := range rolesResponse {
		// Extract role ID from URL like "https://domain.atlassian.net/rest/api/3/project/KEY/role/10100"
		parts := strings.Split(roleURL, "/")
		if len(parts) > 0 {
			roleIDStr := parts[len(parts)-1]
			var roleID int64
			if _, err := fmt.Sscanf(roleIDStr, "%d", &roleID); err == nil {
				roleIDs[roleName] = roleID
			}
		}
	}

	return roleIDs, nil
}

// getProjectRoleDetails gets the details of a specific project role
func (a *JiraTeamSyncAdapter) getProjectRoleDetails(url string, jiraConfig *domain.JiraConfig) ([]domain.TeamMember, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", jiraConfig.AuthHeader())
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("JIRA API error (status %d): %s", resp.StatusCode, string(body))
	}

	var roleDetails JiraProjectRole
	if err := json.NewDecoder(resp.Body).Decode(&roleDetails); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return a.convertJiraUsersToTeamMembers(roleDetails.Actors), nil
}

// convertJiraUsersToTeamMembers converts JIRA users to domain TeamMembers
func (a *JiraTeamSyncAdapter) convertJiraUsersToTeamMembers(jiraUsers []JiraUser) []domain.TeamMember {
	members := make([]domain.TeamMember, 0, len(jiraUsers))
	for _, user := range jiraUsers {
		// Only include active users
		if !user.Active {
			continue
		}

		member := domain.TeamMember{
			AccountID:   user.AccountID,
			DisplayName: user.DisplayName,
			Email:       user.EmailAddress,
			Name:        user.Name,
		}

		// Use display name as fallback if name is empty
		if member.Name == "" {
			member.Name = user.DisplayName
		}

		members = append(members, member)
	}
	return members
}
