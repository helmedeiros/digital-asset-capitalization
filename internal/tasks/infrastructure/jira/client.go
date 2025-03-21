package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/jira/model"
)

// Client defines the interface for Jira API interactions
type Client interface {
	// FetchTasks retrieves tasks from Jira for a given project and sprint
	FetchTasks(ctx context.Context, project, sprint string) ([]*domain.Task, error)
}

// HTTPClient defines the interface for making HTTP requests
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// client implements the Client interface
type client struct {
	httpClient HTTPClient
	baseURL    string
	email      string
	token      string
}

// NewClient creates a new Jira client instance
func NewClient(baseURL, email, token string) Client {
	return &client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
		email:   email,
		token:   token,
	}
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

// FetchTasks retrieves tasks from Jira for a given project and sprint
func (c *client) FetchTasks(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
	if project == "" {
		return nil, fmt.Errorf("project is required")
	}

	// Build JQL query
	jql := fmt.Sprintf("project = %q", project)
	if sprint != "" {
		jql += fmt.Sprintf(" AND sprint = %q", sprint)
	}

	// Build request URL
	url := fmt.Sprintf("%s/rest/api/2/search", c.baseURL)

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication headers
	req.SetBasicAuth(c.email, c.token)
	req.Header.Set("Accept", "application/json")

	// Add query parameters
	q := req.URL.Query()
	q.Add("jql", jql)
	q.Add("fields", "summary,status,project,sprint,created,updated,description")
	req.URL.RawQuery = q.Encode()

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response
	var searchResp model.SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to domain tasks
	tasks := make([]*domain.Task, 0, len(searchResp.Issues))
	for _, issue := range searchResp.Issues {
		created, err := time.Parse(time.RFC3339, issue.Fields.Created)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created time: %w", err)
		}

		updated, err := time.Parse(time.RFC3339, issue.Fields.Updated)
		if err != nil {
			return nil, fmt.Errorf("failed to parse updated time: %w", err)
		}

		task, err := domain.NewTask(
			issue.Key,
			issue.Fields.Summary,
			issue.Fields.Project.Key,
			issue.Fields.Sprint.Name,
			"JIRA",
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create task: %w", err)
		}

		task.UpdateDescription(issue.Fields.Description)
		if err := task.UpdateStatus(mapJiraStatus(issue.Fields.Status.Name)); err != nil {
			return nil, fmt.Errorf("failed to update task status: %w", err)
		}

		// Override timestamps from Jira
		task.CreatedAt = created
		task.UpdatedAt = updated

		tasks = append(tasks, task)
	}

	return tasks, nil
}
