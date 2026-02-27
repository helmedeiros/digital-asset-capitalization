package infrastructure

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
)

// Board represents a Jira board
type Board struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// Sprint represents a Jira sprint
type Sprint struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	State     string `json:"state"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Goal      string `json:"goal"`
}

// UnmarshalJSON custom unmarshaling to handle both string and number ID types
func (s *Sprint) UnmarshalJSON(data []byte) error {
	// Create a temporary struct to handle the raw JSON
	type TempSprint struct {
		ID        interface{} `json:"id"`
		Name      string      `json:"name"`
		State     string      `json:"state"`
		StartDate string      `json:"startDate"`
		EndDate   string      `json:"endDate"`
		Goal      string      `json:"goal"`
	}

	var temp TempSprint
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Convert ID to string regardless of whether it's a number or string
	var idStr string
	switch v := temp.ID.(type) {
	case string:
		idStr = v
	case float64:
		idStr = fmt.Sprintf("%.0f", v)
	case int:
		idStr = fmt.Sprintf("%d", v)
	default:
		idStr = fmt.Sprintf("%v", v)
	}

	// Set the fields
	s.ID = idStr
	s.Name = temp.Name
	s.State = temp.State
	s.StartDate = temp.StartDate
	s.EndDate = temp.EndDate
	s.Goal = temp.Goal

	return nil
}

// BoardsResponse represents the response from the boards API
type BoardsResponse struct {
	Values []Board `json:"values"`
}

// HTTPClient handles HTTP requests to the Jira API
type HTTPClient struct {
	client  *http.Client
	baseURL string
	auth    string
}

// NewHTTPClient creates a new HTTP client for Jira API
func NewHTTPClient(baseURL, auth string) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: time.Second * 10,
		},
		baseURL: baseURL,
		auth:    auth,
	}
}

// Get performs a GET request to the Jira API
func (c *HTTPClient) Get(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.auth)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error response from Jira: %s - %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return body, nil
}

// Put performs a PUT request to the Jira API
func (c *HTTPClient) Put(url string, body []byte) error {
	req, err := http.NewRequest("PUT", url, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("failed to create PUT request: %w", err)
	}

	req.Header.Set("Authorization", c.auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send PUT request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error response from Jira PUT: %s - %s", resp.Status, string(respBody))
	}

	return nil
}

// JiraResponse represents the response from a Jira API search query
type JiraResponse struct {
	Issues []domain.JiraIssue `json:"issues"`
}

// GetJiraIssues retrieves issues from the Jira API
func (c *HTTPClient) GetJiraIssues(jiraURL string) ([]domain.JiraIssue, error) {
	body, err := c.Get(jiraURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get Jira issues: %w", err)
	}

	var response JiraResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Jira response: %w", err)
	}

	return response.Issues, nil
}

// GetBoards retrieves boards from the Jira API
func (c *HTTPClient) GetBoards(jiraURL string) ([]Board, error) {
	body, err := c.Get(jiraURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get boards: %w", err)
	}

	var response BoardsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal boards response: %w", err)
	}

	return response.Values, nil
}

// SprintsResponse represents the response from the sprints API
type SprintsResponse struct {
	Values     []json.RawMessage `json:"values"`
	IsLast     bool              `json:"isLast"`
	StartAt    int               `json:"startAt"`
	MaxResults int               `json:"maxResults"`
}

// GetSprints retrieves sprints from the Jira API with pagination support
func (c *HTTPClient) GetSprints(jiraURL string) ([]Sprint, error) {
	var allSprints []Sprint
	startAt := 0
	maxResults := 50 // Default page size

	for {
		// Add pagination parameters
		separator := "&"
		if !strings.Contains(jiraURL, "?") {
			separator = "?"
		}
		paginatedURL := fmt.Sprintf("%s%sstartAt=%d&maxResults=%d", jiraURL, separator, startAt, maxResults)

		body, err := c.Get(paginatedURL)
		if err != nil {
			return nil, fmt.Errorf("failed to get sprints: %w", err)
		}

		var response SprintsResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal sprints response: %w", err)
		}

		// Process sprints from this page
		for _, raw := range response.Values {
			var sprint Sprint
			if err := json.Unmarshal(raw, &sprint); err != nil {
				return nil, fmt.Errorf("failed to unmarshal sprint: %w; raw=%s", err, string(raw))
			}
			allSprints = append(allSprints, sprint)
		}

		// Check if this is the last page
		if response.IsLast || len(response.Values) == 0 {
			break
		}

		// Move to next page
		startAt += len(response.Values)
	}

	return allSprints, nil
}
