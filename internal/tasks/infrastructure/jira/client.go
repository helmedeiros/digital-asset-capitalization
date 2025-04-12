package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/jira/api"
)

// Client defines the interface for Jira API interactions
type Client interface {
	// FetchTasks retrieves tasks from Jira for a given project and sprint
	FetchTasks(ctx context.Context, project, sprint string) ([]*domain.Task, error)

	// UpdateLabels updates the labels of a Jira issue
	UpdateLabels(ctx context.Context, issueKey string, labels []string) error
}

// HTTPClient defines the interface for making HTTP requests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// ClientFactory is a function type for creating new Jira clients
type ClientFactory func(config *Config) (Client, error)

// NewClient is the default implementation of ClientFactory
var NewClient ClientFactory = newClient

// client implements the Client interface
type client struct {
	httpClient HTTPClient
	config     *Config
}

// NewClient creates a new Jira client instance
func newClient(config *Config) (Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config,
	}, nil
}

// mapJiraStatus converts a Jira status to our domain TaskStatus
func mapJiraStatus(status string) domain.TaskStatus {
	switch strings.ToUpper(status) {
	case "TO DO", "OPEN", "BACKLOG":
		return domain.TaskStatusTodo
	case "IN PROGRESS", "IN DEVELOPMENT":
		return domain.TaskStatusInProgress
	case "DONE", "CLOSED", "RESOLVED":
		return domain.TaskStatusDone
	case "BLOCKED", "IMPEDIMENT":
		return domain.TaskStatusBlocked
	default:
		return domain.TaskStatusTodo
	}
}

// mapJiraType converts a Jira issue type to our domain TaskType
func mapJiraType(issueType string) domain.TaskType {
	switch strings.ToUpper(issueType) {
	case "STORY":
		return domain.TaskTypeStory
	case "BUG":
		return domain.TaskTypeBug
	case "EPIC":
		return domain.TaskTypeEpic
	case "SUB-TASK":
		return domain.TaskTypeSubtask
	default:
		return domain.TaskTypeTask
	}
}

// parseTime attempts to parse a time string in various Jira formats
func parseTime(timeStr string) (time.Time, error) {
	formats := []string{
		"2006-01-02T15:04:05.000-0700",
		time.RFC3339,
	}

	var lastErr error
	for _, format := range formats {
		t, err := time.Parse(format, timeStr)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}

	return time.Time{}, fmt.Errorf("failed to parse time %q: %w", timeStr, lastErr)
}

// wasWorkedOnDuringSprint checks if an issue was worked on during the specific sprint period
func wasWorkedOnDuringSprint(issue api.Issue, sprintStart, sprintEnd time.Time) bool {
	if sprintStart.IsZero() || sprintEnd.IsZero() {
		return false
	}

	// Check changelog history for any activity during the sprint period (inclusive)
	for _, history := range issue.Fields.Changelog.Histories {
		historyTime, err := parseTime(history.Created)
		if err != nil {
			continue
		}

		// Check if the changelog entry is within the sprint period (inclusive)
		if !historyTime.Before(sprintStart) && !historyTime.After(sprintEnd) {
			// Check if any of the changes indicate work was done
			for _, item := range history.Items {
				// Look for meaningful changes that indicate work
				if item.Field == "status" || item.Field == "assignee" || item.Field == "description" ||
					item.Field == "comment" || item.Field == "resolution" {
					return true
				}
			}
		}
	}

	return false
}

// convertToDomainTasks converts Jira issues to domain tasks
func (c *client) convertToDomainTasks(searchResp api.SearchResult, sprint string) ([]*domain.Task, error) {
	tasks := make([]*domain.Task, 0, len(searchResp.Issues))
	for _, issue := range searchResp.Issues {
		// Get sprint dates if available
		var sprintStart, sprintEnd time.Time
		if len(issue.Fields.Sprint) > 0 {
			for _, s := range issue.Fields.Sprint {
				if s.Name == sprint {
					var err error
					if s.StartDate != "" {
						sprintStart, err = parseTime(s.StartDate)
						if err != nil {
							continue
						}
					}
					if s.EndDate != "" {
						sprintEnd, err = parseTime(s.EndDate)
						if err != nil {
							continue
						}
					}
					break
				}
			}
		}

		// Skip if we couldn't get valid sprint dates
		if sprintStart.IsZero() || sprintEnd.IsZero() {
			continue
		}

		// For issues with multiple sprints, check if there was any work done during this sprint
		if len(issue.Fields.Sprint) > 1 {
			if !wasWorkedOnDuringSprint(issue, sprintStart, sprintEnd) {
				continue
			}
		}

		// Handle empty timestamps
		created := time.Now()
		updated := time.Now()

		if issue.Fields.Created != "" {
			var err error
			created, err = parseTime(issue.Fields.Created)
			if err != nil {
				return nil, fmt.Errorf("failed to parse created time: %w", err)
			}
		}

		if issue.Fields.Updated != "" {
			var err error
			updated, err = parseTime(issue.Fields.Updated)
			if err != nil {
				return nil, fmt.Errorf("failed to parse updated time: %w", err)
			}
		}

		// Handle empty sprint
		sprintName := ""
		if len(issue.Fields.Sprint) > 0 {
			var sprintNames []string
			for _, s := range issue.Fields.Sprint {
				if s.Name != "" {
					sprintNames = append(sprintNames, s.Name)
				}
			}
			if len(sprintNames) > 0 {
				sprintName = strings.Join(sprintNames, ", ")
			}
		}

		// Use the project key from the issue key if not available in fields
		projectKey := issue.Fields.Project.Key
		if projectKey == "" {
			parts := strings.Split(issue.Key, "-")
			if len(parts) > 0 {
				projectKey = parts[0]
			}
		}

		// Get the parent issue key for stories
		epicKey := ""
		if issue.Fields.Parent != nil {
			epicKey = issue.Fields.Parent.Key
		}

		task, err := domain.NewTask(issue.Key, issue.Fields.Summary, projectKey, sprintName, "JIRA")
		if err != nil {
			return nil, fmt.Errorf("failed to create task: %w", err)
		}

		// Set additional fields
		var description string
		if len(issue.Fields.Description.Content) > 0 {
			for _, content := range issue.Fields.Description.Content {
				if content.Type == "paragraph" {
					for _, text := range content.Content {
						if text.Type == "text" {
							description += text.Text
						}
					}
				}
			}
		}
		task.Description = strings.TrimSpace(description)
		task.Status = mapJiraStatus(issue.Fields.Status.Name)
		task.Type = mapJiraType(issue.Fields.IssueType.Name)
		task.Priority = domain.TaskPriorityMedium // Default priority since it's not available in the API
		task.Labels = issue.Fields.Labels
		task.Epic = epicKey
		task.CreatedAt = created
		task.UpdatedAt = updated

		// Set work type from labels
		for _, label := range issue.Fields.Labels {
			switch label {
			case "cap-maintenance":
				task.WorkType = domain.WorkTypeMaintenance
			case "cap-discovery":
				task.WorkType = domain.WorkTypeDiscovery
			case "cap-development":
				task.WorkType = domain.WorkTypeDevelopment
			}
			if task.WorkType != "" {
				break
			}
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// FetchTasks retrieves tasks from Jira for a given project and sprint
func (c *client) FetchTasks(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
	if project == "" {
		return nil, fmt.Errorf("project is required")
	}

	// Build JQL query - include issues in the sprint
	jql := fmt.Sprintf("project = %s", project)
	if sprint != "" {
		jql += fmt.Sprintf(" AND sprint in (\"%s\")", sprint)
	}
	jql += " ORDER BY key ASC"

	// Build request URL with fields and expand parameters
	url := fmt.Sprintf("%s/rest/api/3/search?jql=%s&fields=*all&expand=changelog",
		c.config.GetBaseURL(),
		url.QueryEscape(jql))

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication headers
	req.Header.Set("Authorization", c.config.GetAuthHeader())
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status and body
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var searchResp api.SearchResult
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.convertToDomainTasks(searchResp, sprint)
}

type HTTPClientImpl struct {
	client  *http.Client
	baseURL string
	auth    string
}

type Field struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Schema struct {
		Type   string `json:"type"`
		Custom string `json:"custom"`
	} `json:"schema"`
}

func (c *HTTPClientImpl) getSprintFieldID() (string, error) {
	url := fmt.Sprintf("%s/rest/api/2/field", c.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", c.auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	var fields []Field
	if err := json.NewDecoder(resp.Body).Decode(&fields); err != nil {
		return "", fmt.Errorf("error decoding response: %v", err)
	}

	for _, field := range fields {
		if field.Schema.Custom == "com.pyxis.greenhopper.jira:gh-sprint" {
			return field.ID, nil
		}
	}

	return "", fmt.Errorf("sprint field not found")
}

func (c *HTTPClientImpl) GetTasks(project string, sprint string) ([]api.JiraIssue, error) {
	jql := fmt.Sprintf("project = %s", project)
	if sprint != "" {
		jql = fmt.Sprintf("%s AND sprint in ('%s')", jql, sprint)
	}

	url := fmt.Sprintf("%s/rest/api/3/search?jql=%s&fields=*all", c.baseURL, url.QueryEscape(jql))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", c.auth)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error response from Jira: %s - %s", resp.Status, string(body))
	}

	var result api.SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}

	var tasks = make([]api.JiraIssue, 0, len(result.Issues))
	for _, issue := range result.Issues {
		task := api.JiraIssue{
			Key:     issue.Key,
			Summary: issue.Fields.Summary,
			Status:  issue.Fields.Status.Name,
		}
		if issue.Fields.Assignee.DisplayName != "" {
			task.Assignee = issue.Fields.Assignee.DisplayName
		}
		if len(issue.Fields.Sprint) > 0 {
			var sprintNames []string
			for _, sprint := range issue.Fields.Sprint {
				sprintNames = append(sprintNames, fmt.Sprintf("%s (%s)", sprint.Name, sprint.State))
			}
			task.Sprint = sprintNames
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// UpdateLabels updates the labels of a Jira issue
func (c *client) UpdateLabels(ctx context.Context, issueKey string, labels []string) error {
	// Construct the request body
	body := struct {
		Fields struct {
			Labels []string `json:"labels"`
		} `json:"fields"`
	}{}
	body.Fields.Labels = labels

	// Convert to JSON
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Construct the request
	req, err := http.NewRequestWithContext(ctx, "PUT", fmt.Sprintf("%s/rest/api/3/issue/%s", c.config.GetBaseURL(), issueKey), bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.config.GetAuthHeader())

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update labels: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}
