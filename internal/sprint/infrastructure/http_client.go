package infrastructure

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
)

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
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
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
