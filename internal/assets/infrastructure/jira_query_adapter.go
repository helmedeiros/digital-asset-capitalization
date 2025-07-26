package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

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

	// Since JIRA has limited label search capabilities, we'll search for recent tasks
	// and filter client-side for those with our target labels
	jql := "updated >= -90d ORDER BY updated DESC"

	// URL encode the JQL
	encodedJQL := url.QueryEscape(jql)

	// Build the search URL - we'll use a larger maxResults and filter client-side
	searchMaxResults := maxResults * 3 // Get more results to ensure we find enough matching ones
	if searchMaxResults > 5000 {
		searchMaxResults = 5000 // JIRA limit
	}

	searchURL := fmt.Sprintf("%s/rest/api/3/search?jql=%s&maxResults=%d&fields=assignee,reporter,labels",
		jiraConfig.BaseURL(), encodedJQL, searchMaxResults)

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

	// Filter for tasks that actually have the label prefix and limit to requested count
	tasks := make([]usecase.JiraTaskInfo, 0, maxResults)
	count := 0

	for _, issue := range searchResponse.Issues {
		if count >= maxResults {
			break
		}

		// Check if any label starts with the prefix
		hasTargetLabel := false
		for _, label := range issue.Fields.Labels {
			if strings.HasPrefix(strings.ToLower(label), strings.ToLower(labelPrefix)) {
				hasTargetLabel = true
				break
			}
		}

		// Only include tasks that have labels starting with our prefix
		if !hasTargetLabel {
			continue
		}

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
		count++
	}

	return tasks, nil
}

// SearchTasksWithFilters searches for tasks with additional filtering options
func (a *JiraQueryAdapter) SearchTasksWithFilters(ctx context.Context, filters usecase.JiraSearchFilters) ([]usecase.JiraTaskInfo, error) {
	jiraConfig, err := a.configService.GetJiraConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get JIRA config: %w", err)
	}

	// Build JQL query with filters
	var jqlParts []string

	// Base condition - recent tasks
	jqlParts = append(jqlParts, "updated >= -90d")

	// Add project filter
	if filters.ProjectKey != "" {
		jqlParts = append(jqlParts, fmt.Sprintf("project = \"%s\"", filters.ProjectKey))
	}

	// Add sprint filter
	if filters.SprintName != "" {
		jqlParts = append(jqlParts, fmt.Sprintf("sprint = \"%s\"", filters.SprintName))
	}

	// TODO: Team filtering would require looking up team members first
	// For now, we'll filter after getting results

	// Combine all parts
	jql := strings.Join(jqlParts, " AND ")
	jql += " ORDER BY updated DESC"

	// URL encode the JQL
	encodedJQL := url.QueryEscape(jql)

	// Build the search URL
	searchMaxResults := filters.MaxResults * 3 // Get more results for filtering
	if searchMaxResults > 5000 {
		searchMaxResults = 5000 // JIRA limit
	}

	searchURL := fmt.Sprintf("%s/rest/api/3/search?jql=%s&maxResults=%d&fields=assignee,reporter,labels",
		jiraConfig.BaseURL(), encodedJQL, searchMaxResults)

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

	// Filter and convert results
	tasks := make([]usecase.JiraTaskInfo, 0, filters.MaxResults)
	count := 0

	for _, issue := range searchResponse.Issues {
		if count >= filters.MaxResults {
			break
		}

		// Check if any label starts with the prefix
		hasTargetLabel := false
		for _, label := range issue.Fields.Labels {
			if strings.HasPrefix(strings.ToLower(label), strings.ToLower(filters.LabelPrefix)) {
				hasTargetLabel = true
				break
			}
		}

		// Only include tasks that have labels starting with our prefix
		if !hasTargetLabel {
			continue
		}

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

		// TODO: Add team filtering here if needed
		// For now, we'll let the use case handle team-specific filtering

		tasks = append(tasks, task)
		count++
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
