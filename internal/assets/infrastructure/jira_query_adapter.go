package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/application/usecase"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

// JiraQueryAdapter implements the JiraQueryPort for searching JIRA tasks
type JiraQueryAdapter struct {
	configService ConfigServiceInterface
	httpClient    *http.Client
}

// ConfigServiceInterface defines the interface for config service
type ConfigServiceInterface interface {
	GetJiraConfig() (*domain.JiraConfig, error)
}

// JiraSearchResponse represents the response from JIRA search API
type JiraSearchResponse struct {
	Issues []JiraIssue `json:"issues"`
	Total  int         `json:"total"`
}

// JiraIssue represents a JIRA issue in search results
type JiraIssue struct {
	Key    string          `json:"key"`
	Fields JiraIssueFields `json:"fields"`
}

// JiraIssueFields represents the fields of a JIRA issue
type JiraIssueFields struct {
	Assignee *JiraUser `json:"assignee"`
	Reporter *JiraUser `json:"reporter"`
	Labels   []string  `json:"labels"`
}

// JiraUser represents a user in JIRA
type JiraUser struct {
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
	AccountID    string `json:"accountId"`
}

// NewJiraQueryAdapter creates a new JIRA query adapter
func NewJiraQueryAdapter(configService ConfigServiceInterface) (*JiraQueryAdapter, error) {
	if configService == nil {
		return nil, fmt.Errorf("config service is required")
	}

	return &JiraQueryAdapter{
		configService: configService,
		httpClient:    &http.Client{},
	}, nil
}

// SearchTasksByLabelPrefix searches for tasks with labels starting with the given prefix
func (a *JiraQueryAdapter) SearchTasksByLabelPrefix(ctx context.Context, labelPrefix string, maxResults int) ([]usecase.JiraTaskInfo, error) {
	jiraConfig, err := a.configService.GetJiraConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get JIRA config: %w", err)
	}

	// Build JQL query to search for tasks with labels starting with the prefix
	jql := fmt.Sprintf("labels ~ \"%s*\"", labelPrefix)

	// Add ordering to get most recent tasks first
	jql += " ORDER BY updated DESC"

	// URL encode the JQL
	encodedJQL := url.QueryEscape(jql)

	// Build the search URL
	searchURL := fmt.Sprintf("%s/rest/api/3/search?jql=%s&maxResults=%d&fields=assignee,reporter,labels",
		jiraConfig.BaseURL(), encodedJQL, maxResults)

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", jiraConfig.AuthHeader())
	req.Header.Set("Accept", "application/json")

	// Execute the request
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("JIRA API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var searchResponse JiraSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our format
	tasks := make([]usecase.JiraTaskInfo, 0, len(searchResponse.Issues))
	for _, issue := range searchResponse.Issues {
		task := usecase.JiraTaskInfo{
			Key:    issue.Key,
			Labels: issue.Fields.Labels,
		}

		// Extract assignee information
		if issue.Fields.Assignee != nil {
			task.Assignee = a.extractUserIdentifier(issue.Fields.Assignee)
		}

		// Extract reporter information
		if issue.Fields.Reporter != nil {
			task.Reporter = a.extractUserIdentifier(issue.Fields.Reporter)
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// extractUserIdentifier extracts a user identifier from JIRA user data
// Prefers email address, falls back to display name
func (a *JiraQueryAdapter) extractUserIdentifier(user *JiraUser) string {
	if user.EmailAddress != "" {
		return user.EmailAddress
	}
	if user.DisplayName != "" {
		return user.DisplayName
	}
	return user.AccountID
}
