package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
			// Verify request
			assert.Equal(t, http.MethodGet, r.Method, "Method should be GET")
			assert.Equal(t, "/rest/api/3/search", r.URL.Path, "Path should match")
			assert.Equal(t, "project = TEST AND sprint in (\"Sprint 1\") ORDER BY key ASC", r.URL.Query().Get("jql"), "JQL should match")
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
		assert.Contains(t, err.Error(), "unexpected status code: 500", "Error message should indicate server error")
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
					"https://test.atlassian.net/rest/api/2/field": {
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
					"invalid-url/rest/api/2/field": fmt.Errorf("invalid URL"),
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
					"https://test.atlassian.net/rest/api/3/search?jql=project+%3D+TEST+AND+sprint+in+%28%27Sprint+1%27%29&fields=*all": {
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
					"invalid-url/rest/api/3/search?jql=project+%3D+TEST+AND+sprint+in+%28%27Sprint+1%27%29&fields=*all": fmt.Errorf("invalid URL"),
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
