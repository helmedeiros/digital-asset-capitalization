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
	"sync"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/shared/httputil"
	sharedjira "github.com/helmedeiros/digital-asset-capitalization/internal/shared/jira"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/jira/api"
)

// Client defines the interface for Jira API interactions
type Client interface {
	// FetchTasks retrieves tasks from Jira for a given project and sprint
	FetchTasks(ctx context.Context, project, sprint string) ([]*domain.Task, error)

	// FetchTaskByKey retrieves a single task from Jira by its key
	FetchTaskByKey(ctx context.Context, key string) (*domain.Task, error)

	// UpdateLabels updates the labels of a Jira issue using add/remove operations
	UpdateLabels(ctx context.Context, issueKey string, addLabels, removeLabels []string) error
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
	httpClient  HTTPClient
	config      *Config
	retryPolicy httputil.RetryPolicy

	// customFieldIDs is lazily populated via resolveFieldIDsOnce on first
	// use. After the Once.Do fires, customFieldIDs is read-only for the
	// rest of the client's lifetime, so the field is safe to read from
	// any goroutine that calls resolveFieldIDs() to drive the Once.
	resolveFieldIDsOnce sync.Once
	customFieldIDs      *sharedjira.CustomFieldIDs
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
		// Honoured per-call via httputil.DoWithRetry so transient blips,
		// brief 5xx hiccups, and 429 rate-limit replies don't fail a whole
		// sprint sync. Zero-valued fields fall back to httputil defaults.
		retryPolicy: httputil.RetryPolicy{
			MaxAttempts: config.MaxAttempts,
			BackoffBase: config.BackoffBase,
		},
	}, nil
}

// resolveFieldIDs lazily resolves custom field IDs on first use. Safe
// to call from multiple goroutines concurrently: the initialisation
// happens exactly once via sync.Once, and the post-init read of
// customFieldIDs synchronises through Once.Do's happens-before guarantee.
func (c *client) resolveFieldIDs() *sharedjira.CustomFieldIDs {
	c.resolveFieldIDsOnce.Do(func() {
		if c.config == nil {
			return
		}
		resolver := sharedjira.NewFieldResolver(c.config.GetBaseURL(), c.config.GetAuthHeader())
		fieldIDs, err := resolver.ResolveCustomFieldIDs()
		if err == nil {
			c.customFieldIDs = fieldIDs
		}
	})
	return c.customFieldIDs
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
	case "WON'T DO":
		return domain.TaskStatusWontDo
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
		// Check if this issue is relevant to the requested sprint
		if !c.isRelevantToSprint(issue, sprint) {
			continue
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

		// Handle sprint names
		var sprintNames []string
		if len(issue.Fields.Sprint) > 0 {
			for _, s := range issue.Fields.Sprint {
				if s.Name != "" {
					sprintNames = append(sprintNames, s.Name)
				}
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

		// Create task with multi-sprint support
		var task *domain.Task
		var err error
		if len(sprintNames) > 0 {
			task, err = domain.NewTaskWithSprints(issue.Key, issue.Fields.Summary, projectKey, sprintNames, "JIRA")
		} else {
			// Use NewTaskWithoutSprint for issues with no sprints
			task, err = domain.NewTaskWithoutSprint(issue.Key, issue.Fields.Summary, projectKey, "JIRA")
		}
		if err != nil {
			return nil, fmt.Errorf("failed to create task: %w", err)
		}

		// Set additional fields
		task.Description = issue.Fields.Description.ExtractAllText()
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

		// Populate TPD fields from custom field IDs
		if fieldIDs := c.resolveFieldIDs(); fieldIDs != nil {
			task.TPDBusinessUnits = issue.Fields.GetTPDBusinessUnits(fieldIDs.TPDBusinessUnit)
			task.EngineeringHours = issue.Fields.GetEngineeringHours(fieldIDs.EngineeringHours)
			task.WorkStream = issue.Fields.GetWorkStream(fieldIDs.WorkStream)
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// isRelevantToSprint determines if an issue is relevant to the requested sprint
// This handles both single-sprint and multi-sprint scenarios
func (c *client) isRelevantToSprint(issue api.Issue, sprint string) bool {
	if sprint == "" {
		return true // No sprint filter, include all issues
	}

	// Check if the issue has any sprints assigned
	if len(issue.Fields.Sprint) == 0 {
		return false // No sprints assigned, not relevant
	}

	// Look for the specific sprint in the issue's sprint list
	var targetSprint api.Sprint
	var sprintFound bool
	for _, s := range issue.Fields.Sprint {
		if s.Name == sprint {
			targetSprint = s
			sprintFound = true
			break
		}
	}

	if !sprintFound {
		return false // Sprint not found in issue's sprint list
	}

	// For single-sprint issues, always include them
	if len(issue.Fields.Sprint) == 1 {
		return true
	}

	// For multi-sprint issues, apply more sophisticated logic
	return c.isMultiSprintTaskRelevant(issue, targetSprint, sprint)
}

// isMultiSprintTaskRelevant determines if a multi-sprint task is relevant to the requested sprint
func (c *client) isMultiSprintTaskRelevant(issue api.Issue, targetSprint api.Sprint, sprintName string) bool {
	// Get sprint dates for the target sprint
	var sprintStart, sprintEnd time.Time
	if targetSprint.StartDate != "" {
		var err error
		sprintStart, err = parseTime(targetSprint.StartDate)
		if err != nil {
			return true // If we can't parse dates, include the issue to be safe
		}
	}
	if targetSprint.EndDate != "" {
		var err error
		sprintEnd, err = parseTime(targetSprint.EndDate)
		if err != nil {
			return true // If we can't parse dates, include the issue to be safe
		}
	}

	// If we don't have valid sprint dates, include the issue
	if sprintStart.IsZero() || sprintEnd.IsZero() {
		return true
	}

	// Check if there was meaningful work done during the sprint period
	if wasWorkedOnDuringSprint(issue, sprintStart, sprintEnd) {
		return true
	}

	// For multi-sprint tasks, also check if this is the most recent sprint
	// This handles cases where tasks are moved to new sprints without changelog entries
	mostRecentSprint := c.findMostRecentSprint(issue.Fields.Sprint)
	if mostRecentSprint != nil && mostRecentSprint.Name == sprintName {
		return true
	}

	// Fallback: if we can't determine the most recent sprint due to missing dates,
	// be permissive and include the task to avoid missing relevant work
	if mostRecentSprint == nil {
		return true
	}

	return false
}

// findMostRecentSprint finds the most recent sprint among the issue's sprints
func (c *client) findMostRecentSprint(sprints []api.Sprint) *api.Sprint {
	if len(sprints) == 0 {
		return nil
	}

	var mostRecent *api.Sprint
	var mostRecentTime time.Time

	for i := range sprints {
		sprint := &sprints[i]
		var sprintTime time.Time

		// Use end date if available, otherwise start date
		if sprint.EndDate != "" {
			if t, err := parseTime(sprint.EndDate); err == nil {
				sprintTime = t
			}
		} else if sprint.StartDate != "" {
			if t, err := parseTime(sprint.StartDate); err == nil {
				sprintTime = t
			}
		}

		if !sprintTime.IsZero() && (mostRecent == nil || sprintTime.After(mostRecentTime)) {
			mostRecent = sprint
			mostRecentTime = sprintTime
		}
	}

	return mostRecent
}

// FetchTasks retrieves tasks from Jira for a given project and sprint
func (c *client) FetchTasks(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
	if project == "" {
		return nil, fmt.Errorf("project is required")
	}

	// For sprint-specific queries, use a dual strategy to ensure we don't miss multi-sprint tasks
	if sprint != "" {
		return c.fetchTasksWithDualStrategy(ctx, project, sprint)
	}

	// For non-sprint queries, use the standard approach
	return c.fetchTasksWithQuery(ctx, project, sprint, false)
}

// fetchTasksWithDualStrategy combines results from both specific and broader queries
func (c *client) fetchTasksWithDualStrategy(ctx context.Context, project, sprint string) ([]*domain.Task, error) {
	// Run both queries in parallel to maximize coverage
	var specificTasks, broadTasks []*domain.Task
	var specificErr, broadErr error

	// Channel to collect results
	type result struct {
		tasks []*domain.Task
		err   error
		query string
	}

	results := make(chan result, 2)

	// Start specific query
	go func() {
		tasks, err := c.fetchTasksWithQuery(ctx, project, sprint, false)
		results <- result{tasks: tasks, err: err, query: "specific"}
	}()

	// Start broader query
	go func() {
		tasks, err := c.fetchTasksWithQuery(ctx, project, sprint, true)
		results <- result{tasks: tasks, err: err, query: "broad"}
	}()

	// Collect results
	for i := 0; i < 2; i++ {
		res := <-results
		if res.err != nil {
			if res.query == "specific" {
				specificErr = res.err
			} else {
				broadErr = res.err
			}
		} else {
			if res.query == "specific" {
				specificTasks = res.tasks
			} else {
				broadTasks = res.tasks
			}
		}
	}

	// If both queries failed, return the specific query error
	if specificErr != nil && broadErr != nil {
		return nil, specificErr
	}

	// Merge and deduplicate results
	return c.mergeAndDeduplicateTasks(specificTasks, broadTasks), nil
}

// mergeAndDeduplicateTasks combines task lists and removes duplicates based on task key
func (c *client) mergeAndDeduplicateTasks(specificTasks, broadTasks []*domain.Task) []*domain.Task {
	if len(specificTasks) == 0 {
		return broadTasks
	}
	if len(broadTasks) == 0 {
		return specificTasks
	}

	// Create a map to track unique tasks by key
	taskMap := make(map[string]*domain.Task)

	// Add specific tasks first (they have priority)
	for _, task := range specificTasks {
		taskMap[task.Key] = task
	}

	// Add broad tasks only if not already present
	for _, task := range broadTasks {
		if _, exists := taskMap[task.Key]; !exists {
			taskMap[task.Key] = task
		}
	}

	// Convert back to slice
	mergedTasks := make([]*domain.Task, 0, len(taskMap))
	for _, task := range taskMap {
		mergedTasks = append(mergedTasks, task)
	}

	// Sort by key for consistent output
	for i := 0; i < len(mergedTasks)-1; i++ {
		for j := i + 1; j < len(mergedTasks); j++ {
			if mergedTasks[i].Key > mergedTasks[j].Key {
				mergedTasks[i], mergedTasks[j] = mergedTasks[j], mergedTasks[i]
			}
		}
	}

	return mergedTasks
}

// search executes a JQL query and returns the raw search results
func (c *client) search(ctx context.Context, jql string) (*api.SearchResult, error) {
	// Build request URL with fields and expand parameters
	requestURL := fmt.Sprintf("%s/rest/api/3/search/jql?jql=%s&fields=*all&expand=changelog",
		c.config.GetBaseURL(),
		url.QueryEscape(jql))

	resp, err := httputil.DoWithRetry(c.httpClient, c.retryPolicy, "JIRA search", func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", c.config.GetAuthHeader())
		req.Header.Set("Accept", "application/json")
		return req, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status and body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var searchResp api.SearchResult
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &searchResp, nil
}

// fetchTasksWithQuery executes a JIRA query and returns tasks
func (c *client) fetchTasksWithQuery(ctx context.Context, project, sprint string, isBroaderQuery bool) ([]*domain.Task, error) {
	jql := fmt.Sprintf("project = %s", project)
	if sprint != "" {
		if isBroaderQuery {
			// Enhanced broader query: look for tasks that were updated recently and might be multi-sprint
			// This catches tasks that might have been moved between sprints or worked on across sprints
			jql += fmt.Sprintf(" AND (sprint in (\"%s\") OR (updated >= -30d AND sprint is not EMPTY))", sprint)
		} else {
			// Standard query - specific sprint only
			jql += fmt.Sprintf(" AND sprint in (\"%s\")", sprint)
		}
	}
	jql += " ORDER BY key ASC"

	searchResp, err := c.search(ctx, jql)
	if err != nil {
		return nil, fmt.Errorf("failed to search tasks: %w", err)
	}

	tasks, err := c.convertToDomainTasks(*searchResp, sprint)
	if err != nil {
		return nil, fmt.Errorf("failed to convert tasks: %w", err)
	}

	return tasks, nil
}

// FetchTaskByKey retrieves a single task from Jira by its key
func (c *client) FetchTaskByKey(ctx context.Context, key string) (*domain.Task, error) {
	if key == "" {
		return nil, fmt.Errorf("issue key is required")
	}

	// Build the URL for fetching a single issue
	issueURL := fmt.Sprintf("%s/rest/api/3/issue/%s?expand=changelog", c.config.BaseURL, key)

	resp, err := httputil.DoWithRetry(c.httpClient, c.retryPolicy, "JIRA issue fetch", func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, "GET", issueURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", c.config.GetAuthHeader())
		req.Header.Set("Accept", "application/json")
		return req, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("issue %s not found", key)
		}
		return nil, fmt.Errorf("Jira API returned status %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the response
	var issue api.Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to domain task
	task, err := c.convertSingleIssueToDomainTask(issue)
	if err != nil {
		return nil, fmt.Errorf("failed to convert issue to domain task: %w", err)
	}

	return task, nil
}

// convertSingleIssueToDomainTask converts a single Jira issue to a domain task
func (c *client) convertSingleIssueToDomainTask(issue api.Issue) (*domain.Task, error) {
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

	// Handle sprint names
	var sprintNames []string
	if len(issue.Fields.Sprint) > 0 {
		for _, s := range issue.Fields.Sprint {
			if s.Name != "" {
				sprintNames = append(sprintNames, s.Name)
			}
		}
	}

	// Use the project key from the issue key if not available in fields
	projectKey := ""
	if parts := strings.Split(issue.Key, "-"); len(parts) >= 2 {
		projectKey = parts[0]
	}

	// Handle epic link - check if parent exists and is an epic
	epicKey := ""
	if issue.Fields.Parent != nil && issue.Fields.Parent.Fields.IssueType.Name == "Epic" {
		epicKey = issue.Fields.Parent.Key
	}

	// Handle work type from labels
	workType := domain.WorkType("")
	for _, label := range issue.Fields.Labels {
		switch label {
		case "cap-maintenance":
			workType = domain.WorkTypeMaintenance
		case "cap-discovery":
			workType = domain.WorkTypeDiscovery
		case "cap-development":
			workType = domain.WorkTypeDevelopment
		}
		if workType != "" {
			break
		}
	}

	// Create task with multi-sprint support
	var task *domain.Task
	var err error
	if len(sprintNames) > 0 {
		task, err = domain.NewTaskWithSprints(issue.Key, issue.Fields.Summary, projectKey, sprintNames, "JIRA")
	} else {
		// Use NewTaskWithoutSprint for issues with no sprints
		task, err = domain.NewTaskWithoutSprint(issue.Key, issue.Fields.Summary, projectKey, "JIRA")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// Set additional fields
	task.Description = issue.Fields.Description.ExtractAllText()
	task.Status = mapJiraStatus(issue.Fields.Status.Name)
	task.Type = mapJiraType(issue.Fields.IssueType.Name)
	task.Priority = domain.TaskPriorityMedium // Default priority
	task.Labels = issue.Fields.Labels
	task.Epic = epicKey
	task.CreatedAt = created
	task.UpdatedAt = updated
	task.WorkType = workType

	// Populate TPD fields from custom field IDs. Going through
	// resolveFieldIDs (instead of reading c.customFieldIDs directly)
	// ensures we initialise the cache lazily even when FetchTaskByKey
	// is the first call against this client, and stays race-free with
	// any concurrent fetch-tasks path that drives the same sync.Once.
	if fieldIDs := c.resolveFieldIDs(); fieldIDs != nil {
		task.TPDBusinessUnits = issue.Fields.GetTPDBusinessUnits(fieldIDs.TPDBusinessUnit)
		task.EngineeringHours = issue.Fields.GetEngineeringHours(fieldIDs.EngineeringHours)
		task.WorkStream = issue.Fields.GetWorkStream(fieldIDs.WorkStream)
	}

	return task, nil
}

// labelOperation represents a single JIRA label add/remove operation
type labelOperation struct {
	Add    string `json:"add,omitempty"`
	Remove string `json:"remove,omitempty"`
}

// UpdateLabels updates the labels of a Jira issue using add/remove operations
// to avoid overwriting non-cap labels
func (c *client) UpdateLabels(ctx context.Context, issueKey string, addLabels, removeLabels []string) error {
	// Build label operations
	ops := make([]labelOperation, 0, len(addLabels)+len(removeLabels))
	for _, label := range removeLabels {
		ops = append(ops, labelOperation{Remove: label})
	}
	for _, label := range addLabels {
		ops = append(ops, labelOperation{Add: label})
	}

	if len(ops) == 0 {
		return nil
	}

	// Construct the request body using JIRA update operations
	body := struct {
		Update struct {
			Labels []labelOperation `json:"labels"`
		} `json:"update"`
	}{}
	body.Update.Labels = ops

	// Convert to JSON
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := httputil.DoWithRetry(c.httpClient, c.retryPolicy, "JIRA label update", func() (*http.Request, error) {
		// Rebuild bytes.Buffer per attempt so the body is fresh on retries.
		req, err := http.NewRequestWithContext(ctx, "PUT", fmt.Sprintf("%s/rest/api/3/issue/%s", c.config.GetBaseURL(), issueKey), bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", c.config.GetAuthHeader())
		return req, nil
	})
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update labels: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// HTTPClientImpl implements the HTTPClient interface for testing
type HTTPClientImpl struct {
	client  *http.Client
	baseURL string
	auth    string
}

// Field represents a JIRA field definition
type Field struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Schema struct {
		Type   string `json:"type"`
		Custom string `json:"custom"`
	} `json:"schema"`
}

// getSprintFieldID retrieves the custom field ID for sprint
func (c *HTTPClientImpl) getSprintFieldID() (string, error) {
	url := fmt.Sprintf("%s/rest/api/3/field", c.baseURL)
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

// GetTasks retrieves tasks from JIRA using the legacy API
func (c *HTTPClientImpl) GetTasks(project string, sprint string) ([]api.JiraIssue, error) {
	jql := fmt.Sprintf("project = %s", project)
	if sprint != "" {
		jql = fmt.Sprintf("%s AND sprint in ('%s')", jql, sprint)
	}

	url := fmt.Sprintf("%s/rest/api/3/search/jql?jql=%s&fields=*all", c.baseURL, url.QueryEscape(jql))
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
