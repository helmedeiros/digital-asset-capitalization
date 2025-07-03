package infrastructure

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// BoardsResponse represents the response from the boards API
type BoardsResponse struct {
	Values []Board `json:"values"`
}

// SprintsResponse represents the response from the sprints API
type SprintsResponse struct {
	Values []Sprint `json:"values"`
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

// GetSprints retrieves sprints from the Jira API
func (c *HTTPClient) GetSprints(jiraURL string) ([]Sprint, error) {
	body, err := c.Get(jiraURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get sprints: %w", err)
	}

	var response SprintsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sprints response: %w", err)
	}

	return response.Values, nil
}
