package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/shared/httputil"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/jira/api"
)

type mockHTTPClient struct {
	responses map[string]*http.Response
	errors    map[string]error
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	url := req.URL.String()
	if err, ok := m.errors[url]; ok {
		return nil, err
	}
	if resp, ok := m.responses[url]; ok {
		return resp, nil
	}
	return nil, fmt.Errorf("no mock response for URL: %s", url)
}

func TestNewClient(t *testing.T) {
	config := &Config{
		BaseURL: "http://localhost:8080",
		Email:   "test@example.com",
		Token:   "test-token",
	}

	client, err := NewClient(config)
	require.NoError(t, err, "Should not return error")
	assert.NotNil(t, client, "Client should not be nil")
}

func TestClient_FetchTasks(t *testing.T) {
	ctx := context.Background()

	t.Run("empty project", func(t *testing.T) {
		config := &Config{
			BaseURL: "http://localhost:8080",
			Email:   "test@example.com",
			Token:   "test-token",
		}
		client, err := NewClient(config)
		require.NoError(t, err, "Should not return error")
		tasks, err := client.FetchTasks(ctx, "", "Sprint 1")
		require.Error(t, err, "Should return error")
		assert.Nil(t, tasks, "Tasks should be nil")
		assert.Contains(t, err.Error(), "project is required", "Error message should indicate project is required")
	})

	t.Run("successful fetch", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/rest/api/3/field" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[]`))
				return
			}
			// Verify request
			assert.Equal(t, http.MethodGet, r.Method, "Method should be GET")
			assert.Equal(t, "/rest/api/3/search/jql", r.URL.Path, "Path should match")
			// With dual strategy, we expect either the specific query or the broader query
			jql := r.URL.Query().Get("jql")
			expectedSpecific := "project = TEST AND sprint in (\"Sprint 1\") ORDER BY key ASC"
			expectedBroad := "project = TEST AND (sprint in (\"Sprint 1\") OR (updated >= -30d AND sprint is not EMPTY)) ORDER BY key ASC"
			assert.True(t, jql == expectedSpecific || jql == expectedBroad, "JQL should match either specific or broad query, got: %s", jql)
			assert.Equal(t, "*all", r.URL.Query().Get("fields"), "Fields should match")
			assert.Equal(t, "changelog", r.URL.Query().Get("expand"), "Expand should match")

			// Verify auth header
			username, password, ok := r.BasicAuth()
			assert.True(t, ok, "Should have basic auth")
			assert.Equal(t, "test@example.com", username, "Username should match")
			assert.Equal(t, "test-token", password, "Password should match")

			// Return response
			now := time.Now().Format(time.RFC3339)
			responseData := map[string]interface{}{
				"issues": []map[string]interface{}{
					{
						"key": "TEST-1",
						"fields": map[string]interface{}{
							"summary": "Test Issue",
							"status": map[string]interface{}{
								"name": "In Progress",
							},
							"project": map[string]interface{}{
								"key": "TEST",
							},
							"customfield_10100": []map[string]interface{}{
								{
									"id":        1,
									"name":      "Sprint 1",
									"state":     "active",
									"startDate": now,
									"endDate":   now,
									"boardId":   1,
									"goal":      "Test sprint goal",
								},
							},
							"created": now,
							"updated": now,
							"description": map[string]interface{}{
								"type":    "doc",
								"version": 1,
								"content": []map[string]interface{}{
									{
										"type": "paragraph",
										"content": []map[string]interface{}{
											{
												"type": "text",
												"text": "Test Description",
											},
										},
									},
								},
							},
							"issuetype": map[string]interface{}{
								"name": "Story",
							},
						},
					},
					{
						"key": "TEST-2",
						"fields": map[string]interface{}{
							"summary": "Test Issue 2",
							"status": map[string]interface{}{
								"name": "To Do",
							},
							"project": map[string]interface{}{
								"key": "TEST",
							},
							"customfield_10100": []map[string]interface{}{
								{
									"id":        1,
									"name":      "Sprint 1",
									"state":     "active",
									"startDate": now,
									"endDate":   now,
									"boardId":   1,
									"goal":      "Test sprint goal",
								},
							},
							"created": now,
							"updated": now,
							"description": map[string]interface{}{
								"type":    "doc",
								"version": 1,
								"content": []map[string]interface{}{
									{
										"type": "paragraph",
										"content": []map[string]interface{}{
											{
												"type": "text",
												"text": "Test Description",
											},
										},
									},
								},
							},
							"issuetype": map[string]interface{}{
								"name": "Bug",
							},
						},
					},
					{
						"key": "TEST-3",
						"fields": map[string]interface{}{
							"summary": "Test Issue 3",
							"status": map[string]interface{}{
								"name": "Done",
							},
							"project": map[string]interface{}{
								"key": "TEST",
							},
							"customfield_10100": []map[string]interface{}{
								{
									"id":        1,
									"name":      "Sprint 1",
									"state":     "active",
									"startDate": now,
									"endDate":   now,
									"boardId":   1,
									"goal":      "Test sprint goal",
								},
							},
							"created": now,
							"updated": now,
							"description": map[string]interface{}{
								"type":    "doc",
								"version": 1,
								"content": []map[string]interface{}{
									{
										"type": "paragraph",
										"content": []map[string]interface{}{
											{
												"type": "text",
												"text": "Test Description",
											},
										},
									},
								},
							},
							"issuetype": map[string]interface{}{
								"name": "Epic",
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(responseData)
		}))
		defer server.Close()

		config := &Config{
			BaseURL: server.URL,
			Email:   "test@example.com",
			Token:   "test-token",
		}
		client, err := NewClient(config)
		require.NoError(t, err, "Should not return error")
		tasks, err := client.FetchTasks(ctx, "TEST", "Sprint 1")
		require.NoError(t, err, "Should not return error")
		require.Len(t, tasks, 3, "Should return three tasks")

		task1 := tasks[0]
		assert.Equal(t, "TEST-1", task1.Key, "Task key should match")
		assert.Equal(t, "Test Issue", task1.Summary, "Task summary should match")
		assert.Equal(t, domain.TaskStatusInProgress, task1.Status, "Task status should match")
		assert.Equal(t, "TEST", task1.Project, "Task project should match")
		assert.Equal(t, "Sprint 1", task1.Sprint, "Task sprint should match")
		assert.Equal(t, "JIRA", task1.Platform, "Task platform should be JIRA")
		assert.Equal(t, "Test Description", task1.Description, "Task description should match")
		assert.Equal(t, domain.TaskTypeStory, task1.Type, "Task type should match")

		task2 := tasks[1]
		assert.Equal(t, "TEST-2", task2.Key, "Task key should match")
		assert.Equal(t, "Test Issue 2", task2.Summary, "Task summary should match")
		assert.Equal(t, domain.TaskStatusTodo, task2.Status, "Task status should match")
		assert.Equal(t, "TEST", task2.Project, "Task project should match")
		assert.Equal(t, "Sprint 1", task2.Sprint, "Task sprint should match")
		assert.Equal(t, "JIRA", task2.Platform, "Task platform should be JIRA")
		assert.Equal(t, "Test Description", task2.Description, "Task description should match")
		assert.Equal(t, domain.TaskTypeBug, task2.Type, "Task type should match")

		task3 := tasks[2]
		assert.Equal(t, "TEST-3", task3.Key, "Task key should match")
		assert.Equal(t, "Test Issue 3", task3.Summary, "Task summary should match")
		assert.Equal(t, domain.TaskStatusDone, task3.Status, "Task status should match")
		assert.Equal(t, "TEST", task3.Project, "Task project should match")
		assert.Equal(t, "Sprint 1", task3.Sprint, "Task sprint should match")
		assert.Equal(t, "JIRA", task3.Platform, "Task platform should be JIRA")
		assert.Equal(t, "Test Description", task3.Description, "Task description should match")
		assert.Equal(t, domain.TaskTypeEpic, task3.Type, "Task type should match")
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Internal Server Error"}`))
		}))
		defer server.Close()

		// 500 is retryable; cap MaxAttempts at 1 so the test runs fast.
		// Retry behaviour itself is covered in internal/shared/httputil.
		config := &Config{
			BaseURL:     server.URL,
			Email:       "test@example.com",
			Token:       "test-token",
			MaxAttempts: 1,
		}
		client, err := NewClient(config)
		require.NoError(t, err, "Should not return error")
		tasks, err := client.FetchTasks(ctx, "TEST", "Sprint 1")
		require.Error(t, err, "Should return error")
		assert.Nil(t, tasks, "Tasks should be nil")
		assert.Contains(t, err.Error(), "JIRA search failed after 1 attempts")
	})

	t.Run("invalid response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Write([]byte(`invalid json`))
		}))
		defer server.Close()

		config := &Config{
			BaseURL: server.URL,
			Email:   "test@example.com",
			Token:   "test-token",
		}
		client, err := NewClient(config)
		require.NoError(t, err, "Should not return error")
		tasks, err := client.FetchTasks(ctx, "TEST", "Sprint 1")
		require.Error(t, err, "Should return error")
		assert.Nil(t, tasks, "Tasks should be nil")
		assert.Contains(t, err.Error(), "failed to decode response", "Error message should indicate decode failure")
	})
}

func Test_mapJiraStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected domain.TaskStatus
	}{
		{
			name:     "to do status",
			status:   "To Do",
			expected: domain.TaskStatusTodo,
		},
		{
			name:     "open status",
			status:   "Open",
			expected: domain.TaskStatusTodo,
		},
		{
			name:     "backlog status",
			status:   "Backlog",
			expected: domain.TaskStatusTodo,
		},
		{
			name:     "in progress status",
			status:   "In Progress",
			expected: domain.TaskStatusInProgress,
		},
		{
			name:     "in development status",
			status:   "In Development",
			expected: domain.TaskStatusInProgress,
		},
		{
			name:     "done status",
			status:   "Done",
			expected: domain.TaskStatusDone,
		},
		{
			name:     "closed status",
			status:   "Closed",
			expected: domain.TaskStatusDone,
		},
		{
			name:     "resolved status",
			status:   "Resolved",
			expected: domain.TaskStatusDone,
		},
		{
			name:     "blocked status",
			status:   "Blocked",
			expected: domain.TaskStatusBlocked,
		},
		{
			name:     "impediment status",
			status:   "Impediment",
			expected: domain.TaskStatusBlocked,
		},
		{
			name:     "unknown status",
			status:   "Unknown",
			expected: domain.TaskStatusTodo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapJiraStatus(tt.status)
			assert.Equal(t, tt.expected, result, "Status mapping should match")
		})
	}
}

func TestWasWorkedOnDuringSprint(t *testing.T) {
	// Create a base time for testing
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	sprintStart := baseTime
	sprintEnd := baseTime.Add(14 * 24 * time.Hour) // 2 weeks sprint

	tests := []struct {
		name     string
		issue    api.Issue
		expected bool
	}{
		{
			name: "work done during sprint",
			issue: api.Issue{
				Fields: api.Fields{
					Changelog: api.Changelog{
						Histories: []api.ChangelogHistory{
							{
								Created: baseTime.Add(5 * 24 * time.Hour).Format(time.RFC3339),
								Items: []api.ChangelogItem{
									{Field: "status", FromString: "To Do", ToString: "In Progress"},
								},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "work done before sprint",
			issue: api.Issue{
				Fields: api.Fields{
					Changelog: api.Changelog{
						Histories: []api.ChangelogHistory{
							{
								Created: baseTime.Add(-1 * 24 * time.Hour).Format(time.RFC3339),
								Items: []api.ChangelogItem{
									{Field: "status", FromString: "To Do", ToString: "In Progress"},
								},
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "work done after sprint",
			issue: api.Issue{
				Fields: api.Fields{
					Changelog: api.Changelog{
						Histories: []api.ChangelogHistory{
							{
								Created: baseTime.Add(15 * 24 * time.Hour).Format(time.RFC3339),
								Items: []api.ChangelogItem{
									{Field: "status", FromString: "To Do", ToString: "In Progress"},
								},
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "multiple changes during sprint",
			issue: api.Issue{
				Fields: api.Fields{
					Changelog: api.Changelog{
						Histories: []api.ChangelogHistory{
							{
								Created: baseTime.Add(5 * 24 * time.Hour).Format(time.RFC3339),
								Items: []api.ChangelogItem{
									{Field: "status", FromString: "To Do", ToString: "In Progress"},
								},
							},
							{
								Created: baseTime.Add(7 * 24 * time.Hour).Format(time.RFC3339),
								Items: []api.ChangelogItem{
									{Field: "description", FromString: "Old", ToString: "New"},
								},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "work done at sprint boundaries",
			issue: api.Issue{
				Fields: api.Fields{
					Changelog: api.Changelog{
						Histories: []api.ChangelogHistory{
							{
								Created: sprintStart.Format(time.RFC3339),
								Items: []api.ChangelogItem{
									{Field: "status", FromString: "To Do", ToString: "In Progress"},
								},
							},
							{
								Created: sprintEnd.Format(time.RFC3339),
								Items: []api.ChangelogItem{
									{Field: "status", FromString: "In Progress", ToString: "Done"},
								},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "no relevant changes during sprint",
			issue: api.Issue{
				Fields: api.Fields{
					Changelog: api.Changelog{
						Histories: []api.ChangelogHistory{
							{
								Created: baseTime.Add(5 * 24 * time.Hour).Format(time.RFC3339),
								Items: []api.ChangelogItem{
									{Field: "labels", FromString: "", ToString: "bug"},
								},
							},
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wasWorkedOnDuringSprint(tt.issue, sprintStart, sprintEnd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFetchTasksWithMultipleSprints(t *testing.T) {
	// Create a mock client for testing
	mockClient := &client{
		httpClient: &mockHTTPClient{},
		config: &Config{
			BaseURL: "http://localhost:8080",
			Email:   "test@example.com",
			Token:   "test-token",
		},
	}

	// Create test data
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	sprintStart := baseTime
	sprintEnd := baseTime.Add(14 * 24 * time.Hour)

	tests := []struct {
		name     string
		issue    api.Issue
		sprint   string
		expected bool
	}{
		{
			name: "single sprint issue",
			issue: api.Issue{
				Key: "TEST-1",
				Fields: api.Fields{
					Summary: "Test Issue 1",
					Status:  api.Status{Name: "In Progress"},
					Project: api.Project{Key: "TEST"},
					Sprint: []api.Sprint{
						{
							Name:      "Sprint 1",
							StartDate: sprintStart.Format(time.RFC3339),
							EndDate:   sprintEnd.Format(time.RFC3339),
						},
					},
				},
			},
			sprint:   "Sprint 1",
			expected: true,
		},
		{
			name: "multiple sprints with work in requested sprint",
			issue: api.Issue{
				Key: "TEST-2",
				Fields: api.Fields{
					Summary: "Test Issue 2",
					Status:  api.Status{Name: "In Progress"},
					Project: api.Project{Key: "TEST"},
					Sprint: []api.Sprint{
						{
							Name:      "Sprint 1",
							StartDate: sprintStart.Format(time.RFC3339),
							EndDate:   sprintEnd.Format(time.RFC3339),
						},
						{
							Name:      "Sprint 2",
							StartDate: sprintEnd.Add(24 * time.Hour).Format(time.RFC3339),
							EndDate:   sprintEnd.Add(15 * 24 * time.Hour).Format(time.RFC3339),
						},
					},
					Changelog: api.Changelog{
						Histories: []api.ChangelogHistory{
							{
								Created: baseTime.Add(5 * 24 * time.Hour).Format(time.RFC3339),
								Items: []api.ChangelogItem{
									{Field: "status", FromString: "To Do", ToString: "In Progress"},
								},
							},
						},
					},
				},
			},
			sprint:   "Sprint 1",
			expected: true,
		},
		{
			name: "multiple sprints without work in requested sprint",
			issue: api.Issue{
				Key: "TEST-3",
				Fields: api.Fields{
					Summary: "Test Issue 3",
					Status:  api.Status{Name: "In Progress"},
					Project: api.Project{Key: "TEST"},
					Sprint: []api.Sprint{
						{
							Name:      "Sprint 1",
							StartDate: sprintStart.Format(time.RFC3339),
							EndDate:   sprintEnd.Format(time.RFC3339),
						},
						{
							Name:      "Sprint 2",
							StartDate: sprintEnd.Add(24 * time.Hour).Format(time.RFC3339),
							EndDate:   sprintEnd.Add(15 * 24 * time.Hour).Format(time.RFC3339),
						},
					},
					Changelog: api.Changelog{
						Histories: []api.ChangelogHistory{
							{
								Created: sprintEnd.Add(2 * 24 * time.Hour).Format(time.RFC3339),
								Items: []api.ChangelogItem{
									{Field: "status", FromString: "To Do", ToString: "In Progress"},
								},
							},
						},
					},
				},
			},
			sprint:   "Sprint 1",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a search result with the test issue
			searchResp := api.SearchResult{
				Issues: []api.Issue{tt.issue},
			}

			// Convert to domain tasks
			tasks, err := mockClient.convertToDomainTasks(searchResp, tt.sprint)
			require.NoError(t, err)

			// Check if the issue was included in the results
			if tt.expected {
				assert.Equal(t, 1, len(tasks), "Expected one task in results")
				assert.Equal(t, tt.issue.Key, tasks[0].Key, "Expected task key to match")
			} else {
				assert.Equal(t, 0, len(tasks), "Expected no tasks in results")
			}
		})
	}
}

func Test_mapJiraType(t *testing.T) {
	tests := []struct {
		name      string
		issueType string
		want      domain.TaskType
	}{
		{
			name:      "should map story",
			issueType: "Story",
			want:      domain.TaskTypeStory,
		},
		{
			name:      "should map bug",
			issueType: "Bug",
			want:      domain.TaskTypeBug,
		},
		{
			name:      "should map epic",
			issueType: "Epic",
			want:      domain.TaskTypeEpic,
		},
		{
			name:      "should map sub-task",
			issueType: "Sub-task",
			want:      domain.TaskTypeSubtask,
		},
		{
			name:      "should map unknown type to task",
			issueType: "Unknown",
			want:      domain.TaskTypeTask,
		},
		{
			name:      "should map empty type to task",
			issueType: "",
			want:      domain.TaskTypeTask,
		},
		{
			name:      "should map case insensitive",
			issueType: "STORY",
			want:      domain.TaskTypeStory,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapJiraType(tt.issueType)
			if got != tt.want {
				t.Errorf("mapJiraType() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mockTransport struct {
	responses map[string]*http.Response
	errors    map[string]error
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	url := req.URL.String()
	if err, ok := m.errors[url]; ok {
		return nil, err
	}
	if resp, ok := m.responses[url]; ok {
		return resp, nil
	}
	return nil, fmt.Errorf("no mock response for URL: %s", url)
}

func TestGetSprintFieldID(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		auth    string
		mock    *mockTransport
		wantErr bool
	}{
		{
			name:    "successful fetch",
			baseURL: "https://test.atlassian.net",
			auth:    "Basic dGVzdEBleGFtcGxlLmNvbTp0ZXN0LXRva2Vu",
			mock: &mockTransport{
				responses: map[string]*http.Response{
					"https://test.atlassian.net/rest/api/3/field": {
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`[
							{
								"id": "customfield_10100",
								"name": "Sprint",
								"schema": {
									"type": "array",
									"custom": "com.pyxis.greenhopper.jira:gh-sprint"
								}
							}
						]`)),
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid base URL",
			baseURL: "invalid-url",
			auth:    "Basic dGVzdEBleGFtcGxlLmNvbTp0ZXN0LXRva2Vu",
			mock: &mockTransport{
				errors: map[string]error{
					"invalid-url/rest/api/3/field": fmt.Errorf("invalid URL"),
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &HTTPClientImpl{
				client:  &http.Client{Transport: tt.mock},
				baseURL: tt.baseURL,
				auth:    tt.auth,
			}

			fieldID, err := client.getSprintFieldID()
			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, fieldID)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, fieldID)
			assert.Equal(t, "customfield_10100", fieldID)
		})
	}
}

func TestGetTasks(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		auth    string
		mock    *mockTransport
		wantErr bool
	}{
		{
			name:    "successful fetch",
			baseURL: "https://test.atlassian.net",
			auth:    "Basic dGVzdEBleGFtcGxlLmNvbTp0ZXN0LXRva2Vu",
			mock: &mockTransport{
				responses: map[string]*http.Response{
					"https://test.atlassian.net/rest/api/3/search/jql?jql=project+%3D+TEST+AND+sprint+in+%28%27Sprint+1%27%29&fields=*all": {
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"issues": [
								{
									"key": "TEST-1",
									"fields": {
										"summary": "Test Issue",
										"status": {"name": "To Do"},
										"issuetype": {"name": "Story"},
										"sprint": [
											{
												"id": 1,
												"name": "Sprint 1",
												"state": "active",
												"startDate": "2024-01-01T00:00:00.000Z",
												"endDate": "2024-01-14T00:00:00.000Z"
											}
										]
									}
								}
							]
						}`)),
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid base URL",
			baseURL: "invalid-url",
			auth:    "Basic dGVzdEBleGFtcGxlLmNvbTp0ZXN0LXRva2Vu",
			mock: &mockTransport{
				errors: map[string]error{
					"invalid-url/rest/api/3/search/jql?jql=project+%3D+TEST+AND+sprint+in+%28%27Sprint+1%27%29&fields=*all": fmt.Errorf("invalid URL"),
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &HTTPClientImpl{
				client:  &http.Client{Transport: tt.mock},
				baseURL: tt.baseURL,
				auth:    tt.auth,
			}

			tasks, err := client.GetTasks("TEST", "Sprint 1")
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, tasks)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, tasks)
			assert.Len(t, tasks, 1)
			assert.Equal(t, "TEST-1", tasks[0].Key)
			assert.Equal(t, "Test Issue", tasks[0].Summary)
			assert.Equal(t, "To Do", tasks[0].Status)
			assert.Equal(t, []string{"Sprint 1 (active)"}, tasks[0].Sprint)
		})
	}
}

func TestConvertToDomainTasks_WorkType(t *testing.T) {
	client := &client{
		httpClient: nil,
		config:     nil,
	}

	// Create a test issue with labels
	issue := api.Issue{
		Key: "TEST-1",
		Fields: api.Fields{
			Summary: "Test task",
			Project: api.Project{Key: "TEST"},
			Sprint:  []api.Sprint{{Name: "Sprint 1", StartDate: "2025-01-01T00:00:00.000Z", EndDate: "2025-01-14T00:00:00.000Z"}},
			Labels:  []string{"cap-development"},
			Created: "2025-01-01T00:00:00.000Z",
			Updated: "2025-01-01T00:00:00.000Z",
		},
	}

	searchResp := api.SearchResult{
		Issues: []api.Issue{issue},
	}

	tasks, err := client.convertToDomainTasks(searchResp, "Sprint 1")
	assert.NoError(t, err)
	assert.Len(t, tasks, 1)

	task := tasks[0]
	assert.Equal(t, domain.WorkTypeDevelopment, task.WorkType)
}

func TestClient_FetchTaskByKey(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		mockSetup     func(*MockHTTPClient)
		expectedError bool
		errorContains string
	}{
		{
			name: "successful fetch",
			key:  "TEST-123",
			mockSetup: func(mockClient *MockHTTPClient) {
				mockClient.DoFunc = func(req *http.Request) (*http.Response, error) {
					// Verify request URL and headers
					assert.Contains(t, req.URL.String(), "/rest/api/3/issue/TEST-123")
					assert.Equal(t, "application/json", req.Header.Get("Accept"))
					assert.Contains(t, req.Header.Get("Authorization"), "Basic")

					// Return mock response
					responseBody := `{
						"key": "TEST-123",
						"fields": {
							"summary": "Test Task",
							"description": "Test description",
							"status": {"name": "In Progress"},
							"issuetype": {"name": "Task"},
							"project": {"key": "TEST"},
							"assignee": {"displayName": "Test User"},
							"labels": [],
							"created": "2025-01-01T00:00:00.000+0000",
							"updated": "2025-01-01T00:00:00.000+0000"
						}
					}`
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader(responseBody)),
					}, nil
				}
			},
			expectedError: false,
		},
		{
			name: "task not found",
			key:  "TEST-404",
			mockSetup: func(mockClient *MockHTTPClient) {
				mockClient.DoFunc = func(_ *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: 404,
						Body:       io.NopCloser(strings.NewReader(`{"errorMessages":["Issue does not exist"]}`)),
					}, nil
				}
			},
			expectedError: true,
			errorContains: "not found",
		},
		{
			name: "empty key",
			key:  "",
			mockSetup: func(_ *MockHTTPClient) {
				// No setup needed as validation happens before HTTP call
			},
			expectedError: true,
			errorContains: "issue key is required",
		},
		{
			name: "server error",
			key:  "TEST-500",
			mockSetup: func(mockClient *MockHTTPClient) {
				mockClient.DoFunc = func(_ *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: 500,
						Body:       io.NopCloser(strings.NewReader(`{"errorMessages":["Internal Server Error"]}`)),
					}, nil
				}
			},
			expectedError: true,
			// 500 is retryable; the test uses the table client below
			// with MaxAttempts: 1 to skip the retry loop, so we report
			// the new single-attempt error shape.
			errorContains: "JIRA issue fetch failed after 1 attempts",
		},
		{
			name: "invalid JSON response",
			key:  "TEST-INVALID",
			mockSetup: func(mockClient *MockHTTPClient) {
				mockClient.DoFunc = func(_ *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader(`invalid json`)),
					}, nil
				}
			},
			expectedError: true,
			errorContains: "failed to parse response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP client
			mockHTTPClient := &MockHTTPClient{}
			tt.mockSetup(mockHTTPClient)

			// Create config
			config := &Config{
				BaseURL: "https://test.atlassian.net",
				Email:   "test@example.com",
				Token:   "test-token",
			}

			// Create client with MaxAttempts: 1 so retryable-status
			// cases don't drag the suite through real backoff sleeps.
			client := &client{
				httpClient:  mockHTTPClient,
				config:      config,
				retryPolicy: httputil.RetryPolicy{MaxAttempts: 1},
			}

			// Execute
			task, err := client.FetchTaskByKey(context.Background(), tt.key)

			// Verify
			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, task)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, task)
				assert.Equal(t, tt.key, task.Key)
			}
		})
	}
}

func TestClient_UpdateLabels(t *testing.T) {
	t.Run("successful update with add and remove operations", func(t *testing.T) {
		var receivedBody map[string]interface{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bodyBytes, _ := io.ReadAll(r.Body)
			json.Unmarshal(bodyBytes, &receivedBody)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		config := &Config{
			BaseURL: server.URL,
			Email:   "test@example.com",
			Token:   "test-token",
		}

		client, err := NewClient(config)
		require.NoError(t, err)

		err = client.UpdateLabels(context.Background(), "TEST-1",
			[]string{"cap-development", "cap-asset-new"},
			[]string{"cap-maintenance", "cap-asset-old"},
		)

		assert.NoError(t, err)

		// Verify the request body uses update operations
		update, ok := receivedBody["update"].(map[string]interface{})
		require.True(t, ok, "body should have 'update' key")
		labels, ok := update["labels"].([]interface{})
		require.True(t, ok, "update should have 'labels' key")

		// Should have 4 operations: 2 removes + 2 adds
		assert.Len(t, labels, 4)

		// First two should be removes
		op0 := labels[0].(map[string]interface{})
		assert.Equal(t, "cap-maintenance", op0["remove"])
		op1 := labels[1].(map[string]interface{})
		assert.Equal(t, "cap-asset-old", op1["remove"])

		// Last two should be adds
		op2 := labels[2].(map[string]interface{})
		assert.Equal(t, "cap-development", op2["add"])
		op3 := labels[3].(map[string]interface{})
		assert.Equal(t, "cap-asset-new", op3["add"])

		// Verify no 'fields' key exists
		_, hasFields := receivedBody["fields"]
		assert.False(t, hasFields, "body should not have 'fields' key")
	})

	t.Run("no operations when both add and remove are empty", func(t *testing.T) {
		config := &Config{
			BaseURL: "http://unused.example.com",
			Email:   "test@example.com",
			Token:   "test-token",
		}

		client, err := NewClient(config)
		require.NoError(t, err)

		err = client.UpdateLabels(context.Background(), "TEST-1", nil, nil)
		assert.NoError(t, err)
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Internal server error"}`))
		}))
		defer server.Close()

		// 500 is retryable; cap MaxAttempts at 1 so the test exercises
		// a single-attempt failure path quickly. The retry path itself
		// is covered exhaustively in internal/shared/httputil.
		config := &Config{
			BaseURL:     server.URL,
			Email:       "test@example.com",
			Token:       "test-token",
			MaxAttempts: 1,
		}

		client, err := NewClient(config)
		require.NoError(t, err)

		err = client.UpdateLabels(context.Background(), "TEST-1",
			[]string{"cap-development"}, nil,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "JIRA label update failed after 1 attempts")
	})
}

func TestNewRepositoryLegacy_ErrorHandling(t *testing.T) {
	// Clear environment variables to force error
	origBaseURL := os.Getenv("JIRA_BASE_URL")
	origEmail := os.Getenv("JIRA_EMAIL")
	origToken := os.Getenv("JIRA_TOKEN")

	os.Unsetenv("JIRA_BASE_URL")
	os.Unsetenv("JIRA_EMAIL")
	os.Unsetenv("JIRA_TOKEN")

	defer func() {
		if origBaseURL != "" {
			os.Setenv("JIRA_BASE_URL", origBaseURL)
		}
		if origEmail != "" {
			os.Setenv("JIRA_EMAIL", origEmail)
		}
		if origToken != "" {
			os.Setenv("JIRA_TOKEN", origToken)
		}
	}()

	_, err := NewRepositoryLegacy()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create Jira configuration")
}

func TestNewRepositoryLegacy_Success(t *testing.T) {
	// Set up environment variables for valid config
	os.Setenv("JIRA_BASE_URL", "https://test.atlassian.net")
	os.Setenv("JIRA_EMAIL", "test@example.com")
	os.Setenv("JIRA_TOKEN", "test-token")
	defer func() {
		os.Unsetenv("JIRA_BASE_URL")
		os.Unsetenv("JIRA_EMAIL")
		os.Unsetenv("JIRA_TOKEN")
	}()

	repo, err := NewRepositoryLegacy()
	assert.NoError(t, err)
	assert.NotNil(t, repo)
}

func TestIsRelevantToSprint(t *testing.T) {
	client := &client{
		httpClient: nil,
		config:     nil,
	}

	// Base time for testing
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sprintStart := baseTime.Format(time.RFC3339)
	sprintEnd := baseTime.Add(14 * 24 * time.Hour).Format(time.RFC3339)

	tests := []struct {
		name     string
		issue    api.Issue
		sprint   string
		expected bool
	}{
		{
			name: "no sprint filter",
			issue: api.Issue{
				Fields: api.Fields{
					Sprint: []api.Sprint{{Name: "Sprint 1"}},
				},
			},
			sprint:   "",
			expected: true,
		},
		{
			name: "no sprints assigned",
			issue: api.Issue{
				Fields: api.Fields{
					Sprint: []api.Sprint{},
				},
			},
			sprint:   "Sprint 1",
			expected: false,
		},
		{
			name: "single sprint match",
			issue: api.Issue{
				Fields: api.Fields{
					Sprint: []api.Sprint{{Name: "Sprint 1", StartDate: sprintStart, EndDate: sprintEnd}},
				},
			},
			sprint:   "Sprint 1",
			expected: true,
		},
		{
			name: "single sprint no match",
			issue: api.Issue{
				Fields: api.Fields{
					Sprint: []api.Sprint{{Name: "Sprint 2", StartDate: sprintStart, EndDate: sprintEnd}},
				},
			},
			sprint:   "Sprint 1",
			expected: false,
		},
		{
			name: "multi-sprint with target sprint",
			issue: api.Issue{
				Fields: api.Fields{
					Sprint: []api.Sprint{
						{Name: "Sprint 1", StartDate: sprintStart, EndDate: sprintEnd},
						{Name: "Sprint 2", StartDate: baseTime.Add(15 * 24 * time.Hour).Format(time.RFC3339), EndDate: baseTime.Add(29 * 24 * time.Hour).Format(time.RFC3339)},
					},
					Changelog: api.Changelog{
						Histories: []api.ChangelogHistory{
							{
								Created: baseTime.Add(5 * 24 * time.Hour).Format(time.RFC3339),
								Items: []api.ChangelogItem{
									{Field: "status", FromString: "To Do", ToString: "In Progress"},
								},
							},
						},
					},
				},
			},
			sprint:   "Sprint 1",
			expected: true,
		},
		{
			name: "multi-sprint without target sprint",
			issue: api.Issue{
				Fields: api.Fields{
					Sprint: []api.Sprint{
						{Name: "Sprint 2", StartDate: sprintStart, EndDate: sprintEnd},
						{Name: "Sprint 3", StartDate: baseTime.Add(15 * 24 * time.Hour).Format(time.RFC3339), EndDate: baseTime.Add(29 * 24 * time.Hour).Format(time.RFC3339)},
					},
				},
			},
			sprint:   "Sprint 1",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.isRelevantToSprint(tt.issue, tt.sprint)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsMultiSprintTaskRelevant(t *testing.T) {
	client := &client{
		httpClient: nil,
		config:     nil,
	}

	// Base time for testing
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	sprintStart := baseTime.Format(time.RFC3339)
	sprintEnd := baseTime.Add(14 * 24 * time.Hour).Format(time.RFC3339)

	targetSprint := api.Sprint{
		Name:      "Sprint 1",
		StartDate: sprintStart,
		EndDate:   sprintEnd,
	}

	tests := []struct {
		name     string
		issue    api.Issue
		expected bool
	}{
		{
			name: "work done during sprint",
			issue: api.Issue{
				Fields: api.Fields{
					Sprint: []api.Sprint{
						targetSprint,
						{Name: "Sprint 2", StartDate: baseTime.Add(15 * 24 * time.Hour).Format(time.RFC3339), EndDate: baseTime.Add(29 * 24 * time.Hour).Format(time.RFC3339)},
					},
					Changelog: api.Changelog{
						Histories: []api.ChangelogHistory{
							{
								Created: baseTime.Add(5 * 24 * time.Hour).Format(time.RFC3339),
								Items: []api.ChangelogItem{
									{Field: "status", FromString: "To Do", ToString: "In Progress"},
								},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "no work during sprint but most recent",
			issue: api.Issue{
				Fields: api.Fields{
					Sprint: []api.Sprint{
						{Name: "Sprint 0", StartDate: baseTime.Add(-15 * 24 * time.Hour).Format(time.RFC3339), EndDate: baseTime.Add(-1 * 24 * time.Hour).Format(time.RFC3339)},
						targetSprint,
					},
					Changelog: api.Changelog{
						Histories: []api.ChangelogHistory{
							{
								Created: baseTime.Add(-5 * 24 * time.Hour).Format(time.RFC3339),
								Items: []api.ChangelogItem{
									{Field: "status", FromString: "To Do", ToString: "In Progress"},
								},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "no work during sprint and not most recent",
			issue: api.Issue{
				Fields: api.Fields{
					Sprint: []api.Sprint{
						targetSprint,
						{Name: "Sprint 2", StartDate: baseTime.Add(15 * 24 * time.Hour).Format(time.RFC3339), EndDate: baseTime.Add(29 * 24 * time.Hour).Format(time.RFC3339)},
					},
					Changelog: api.Changelog{
						Histories: []api.ChangelogHistory{
							{
								Created: baseTime.Add(20 * 24 * time.Hour).Format(time.RFC3339),
								Items: []api.ChangelogItem{
									{Field: "status", FromString: "To Do", ToString: "In Progress"},
								},
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "no sprint dates - include to be safe",
			issue: api.Issue{
				Fields: api.Fields{
					Sprint: []api.Sprint{
						{Name: "Sprint 1"}, // No dates
						{Name: "Sprint 2"}, // Second sprint to make it multi-sprint
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.isMultiSprintTaskRelevant(tt.issue, targetSprint, "Sprint 1")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindMostRecentSprint(t *testing.T) {
	client := &client{
		httpClient: nil,
		config:     nil,
	}

	// Base time for testing
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		sprints  []api.Sprint
		expected *api.Sprint
	}{
		{
			name:     "empty sprints",
			sprints:  []api.Sprint{},
			expected: nil,
		},
		{
			name: "single sprint",
			sprints: []api.Sprint{
				{Name: "Sprint 1", StartDate: baseTime.Format(time.RFC3339), EndDate: baseTime.Add(14 * 24 * time.Hour).Format(time.RFC3339)},
			},
			expected: &api.Sprint{Name: "Sprint 1", StartDate: baseTime.Format(time.RFC3339), EndDate: baseTime.Add(14 * 24 * time.Hour).Format(time.RFC3339)},
		},
		{
			name: "multiple sprints - most recent by end date",
			sprints: []api.Sprint{
				{Name: "Sprint 1", StartDate: baseTime.Format(time.RFC3339), EndDate: baseTime.Add(14 * 24 * time.Hour).Format(time.RFC3339)},
				{Name: "Sprint 2", StartDate: baseTime.Add(15 * 24 * time.Hour).Format(time.RFC3339), EndDate: baseTime.Add(29 * 24 * time.Hour).Format(time.RFC3339)},
			},
			expected: &api.Sprint{Name: "Sprint 2", StartDate: baseTime.Add(15 * 24 * time.Hour).Format(time.RFC3339), EndDate: baseTime.Add(29 * 24 * time.Hour).Format(time.RFC3339)},
		},
		{
			name: "multiple sprints - fallback to start date",
			sprints: []api.Sprint{
				{Name: "Sprint 1", StartDate: baseTime.Format(time.RFC3339)},
				{Name: "Sprint 2", StartDate: baseTime.Add(15 * 24 * time.Hour).Format(time.RFC3339)},
			},
			expected: &api.Sprint{Name: "Sprint 2", StartDate: baseTime.Add(15 * 24 * time.Hour).Format(time.RFC3339)},
		},
		{
			name: "sprints with invalid dates",
			sprints: []api.Sprint{
				{Name: "Sprint 1", StartDate: "invalid-date"},
				{Name: "Sprint 2", StartDate: baseTime.Add(15 * 24 * time.Hour).Format(time.RFC3339)},
			},
			expected: &api.Sprint{Name: "Sprint 2", StartDate: baseTime.Add(15 * 24 * time.Hour).Format(time.RFC3339)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.findMostRecentSprint(tt.sprints)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expected.Name, result.Name)
				assert.Equal(t, tt.expected.StartDate, result.StartDate)
				assert.Equal(t, tt.expected.EndDate, result.EndDate)
			}
		})
	}
}

func TestFetchTasksWithFallback(t *testing.T) {
	tests := []struct {
		name               string
		specificQueryTasks []api.Issue
		broadQueryTasks    []api.Issue
		sprint             string
		expectedQueryCount int
		expectedTaskCount  int
		expectedTaskKey    string
	}{
		{
			name: "specific query returns tasks",
			specificQueryTasks: []api.Issue{
				{
					Key: "TEST-1",
					Fields: api.Fields{
						Summary:   "Test Task 1",
						Project:   api.Project{Key: "TEST"},
						Sprint:    []api.Sprint{{Name: "Sprint 1", StartDate: "2025-01-01T00:00:00.000Z", EndDate: "2025-01-14T00:00:00.000Z"}},
						Created:   "2025-01-01T00:00:00.000Z",
						Updated:   "2025-01-01T00:00:00.000Z",
						Status:    api.Status{Name: "In Progress"},
						IssueType: api.IssueType{Name: "Story"},
					},
				},
			},
			broadQueryTasks:    []api.Issue{},
			sprint:             "Sprint 1",
			expectedQueryCount: 2, // Dual strategy makes both queries
			expectedTaskCount:  1,
			expectedTaskKey:    "TEST-1",
		},
		{
			name:               "specific query returns no tasks, fallback to broad",
			specificQueryTasks: []api.Issue{},
			broadQueryTasks: []api.Issue{
				{
					Key: "TEST-2",
					Fields: api.Fields{
						Summary:   "Test Task 2",
						Project:   api.Project{Key: "TEST"},
						Sprint:    []api.Sprint{{Name: "Sprint 1", StartDate: "2025-01-01T00:00:00.000Z", EndDate: "2025-01-14T00:00:00.000Z"}},
						Created:   "2025-01-01T00:00:00.000Z",
						Updated:   "2025-01-01T00:00:00.000Z",
						Status:    api.Status{Name: "Done"},
						IssueType: api.IssueType{Name: "Story"},
					},
				},
			},
			sprint:             "Sprint 1",
			expectedQueryCount: 2, // Both queries called
			expectedTaskCount:  1,
			expectedTaskKey:    "TEST-2",
		},
		{
			name: "no sprint filter - only specific query",
			specificQueryTasks: []api.Issue{
				{
					Key: "TEST-3",
					Fields: api.Fields{
						Summary:   "Test Task 3",
						Project:   api.Project{Key: "TEST"},
						Sprint:    []api.Sprint{{Name: "Sprint 1", StartDate: "2025-01-01T00:00:00.000Z", EndDate: "2025-01-14T00:00:00.000Z"}},
						Created:   "2025-01-01T00:00:00.000Z",
						Updated:   "2025-01-01T00:00:00.000Z",
						Status:    api.Status{Name: "To Do"},
						IssueType: api.IssueType{Name: "Story"},
					},
				},
			},
			broadQueryTasks:    []api.Issue{},
			sprint:             "",
			expectedQueryCount: 1, // Only specific query called
			expectedTaskCount:  1,
			expectedTaskKey:    "TEST-3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var queryCount int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/rest/api/3/field" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`[]`))
					return
				}
				atomic.AddInt64(&queryCount, 1)
				w.Header().Set("Content-Type", "application/json")

				var responseData map[string]interface{}

				// Determine which query this is based on the URL
				if strings.Contains(r.URL.RawQuery, "sprint+is+not+EMPTY") {
					// This is the broad query
					responseData = map[string]interface{}{
						"issues": convertIssuesToResponse(tt.broadQueryTasks),
					}
				} else {
					// This is the specific query
					responseData = map[string]interface{}{
						"issues": convertIssuesToResponse(tt.specificQueryTasks),
					}
				}

				json.NewEncoder(w).Encode(responseData)
			}))
			defer server.Close()

			config := &Config{
				BaseURL: server.URL,
				Email:   "test@example.com",
				Token:   "test-token",
			}
			client, err := NewClient(config)
			require.NoError(t, err)

			tasks, err := client.FetchTasks(context.Background(), "TEST", tt.sprint)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedQueryCount, int(atomic.LoadInt64(&queryCount)), "Expected number of queries")
			assert.Equal(t, tt.expectedTaskCount, len(tasks), "Expected number of tasks")

			if tt.expectedTaskCount > 0 {
				assert.Equal(t, tt.expectedTaskKey, tasks[0].Key, "Expected task key")
			}
		})
	}
}

// Helper function to convert test issues to response format
func convertIssuesToResponse(issues []api.Issue) []map[string]interface{} {
	response := make([]map[string]interface{}, 0, len(issues))
	for _, issue := range issues {
		issueData := map[string]interface{}{
			"key": issue.Key,
			"fields": map[string]interface{}{
				"summary":   issue.Fields.Summary,
				"project":   map[string]interface{}{"key": issue.Fields.Project.Key},
				"created":   issue.Fields.Created,
				"updated":   issue.Fields.Updated,
				"status":    map[string]interface{}{"name": issue.Fields.Status.Name},
				"issuetype": map[string]interface{}{"name": issue.Fields.IssueType.Name},
				"description": map[string]interface{}{
					"type":    "doc",
					"version": 1,
					"content": []map[string]interface{}{
						{
							"type": "paragraph",
							"content": []map[string]interface{}{
								{
									"type": "text",
									"text": "Test Description",
								},
							},
						},
					},
				},
			},
		}

		// Add sprint information if available
		if len(issue.Fields.Sprint) > 0 {
			var sprints []map[string]interface{}
			for _, sprint := range issue.Fields.Sprint {
				sprintData := map[string]interface{}{
					"name":  sprint.Name,
					"state": "active",
				}
				if sprint.StartDate != "" {
					sprintData["startDate"] = sprint.StartDate
				}
				if sprint.EndDate != "" {
					sprintData["endDate"] = sprint.EndDate
				}
				sprints = append(sprints, sprintData)
			}
			issueData["fields"].(map[string]interface{})["customfield_10100"] = sprints
		}

		response = append(response, issueData)
	}
	return response
}

// TestClient_ResolveFieldIDs_IsRaceFree pins the fix for the race that
// fetchTasksWithDualStrategy used to trip: two goroutines calling
// resolveFieldIDs concurrently would each read fieldIDsResolved, both
// see false, both set it, and both write customFieldIDs without
// synchronisation. The replacement uses sync.Once so initialisation
// runs exactly once and subsequent reads are race-free by the Once.Do
// happens-before guarantee. Designed to run under -race.
func TestClient_ResolveFieldIDs_IsRaceFree(t *testing.T) {
	// Field-resolver endpoint returns an empty schema so customFieldIDs
	// stays nil after init -- we only care about the synchronisation,
	// not the parsed result.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/field") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	c := &client{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		config: &Config{
			BaseURL: server.URL,
			Email:   "x@example.com",
			Token:   "t",
		},
	}

	const callers = 32
	var wg sync.WaitGroup
	wg.Add(callers)
	for i := 0; i < callers; i++ {
		go func() {
			defer wg.Done()
			// Each goroutine reads the (initialised) value back. With the
			// previous check-then-set, all 32 of these reads would race
			// the writes inside resolveFieldIDs.
			_ = c.resolveFieldIDs()
		}()
	}
	wg.Wait()

	// One more call from the test goroutine to assert the second-call
	// happy path keeps returning the same (possibly-nil) cached value
	// without re-doing the work.
	_ = c.resolveFieldIDs()
}
