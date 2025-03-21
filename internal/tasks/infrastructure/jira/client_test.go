package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockHTTPClient struct {
	response *http.Response
	err      error
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.response, m.err
}

func createMockResponse(t *testing.T, statusCode int, body interface{}) *http.Response {
	jsonBytes, err := json.Marshal(body)
	require.NoError(t, err, "Failed to marshal mock response")

	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(bytes.NewReader(jsonBytes)),
	}
}

func TestNewClient(t *testing.T) {
	config := &Config{
		baseURL: "https://test.atlassian.net",
		email:   "test@example.com",
		token:   "test-token",
	}

	client, err := NewClient(config)
	require.NoError(t, err, "Should not return error")
	assert.NotNil(t, client, "Client should not be nil")
}

func TestClient_FetchTasks(t *testing.T) {
	ctx := context.Background()

	t.Run("empty project", func(t *testing.T) {
		config := &Config{
			baseURL: "https://test.atlassian.net",
			email:   "test@example.com",
			token:   "test-token",
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
			assert.Equal(t, "project = \"TEST\" AND sprint = \"Sprint 1\" ORDER BY resolved ASC, created DESC", r.URL.Query().Get("jql"), "JQL should match")
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
						},
					},
				},
			}
			json.NewEncoder(w).Encode(responseData)
		}))
		defer server.Close()

		config := &Config{
			baseURL: server.URL,
			email:   "test@example.com",
			token:   "test-token",
		}
		client, err := NewClient(config)
		require.NoError(t, err, "Should not return error")
		tasks, err := client.FetchTasks(ctx, "TEST", "Sprint 1")
		require.NoError(t, err, "Should not return error")
		require.Len(t, tasks, 1, "Should return one task")

		task := tasks[0]
		assert.Equal(t, "TEST-1", task.Key, "Task key should match")
		assert.Equal(t, "Test Issue", task.Summary, "Task summary should match")
		assert.Equal(t, domain.TaskStatusInProgress, task.Status, "Task status should match")
		assert.Equal(t, "TEST", task.Project, "Task project should match")
		assert.Equal(t, "Sprint 1", task.Sprint, "Task sprint should match")
		assert.Equal(t, "JIRA", task.Platform, "Task platform should be JIRA")
		assert.Equal(t, "Test Description", task.Description, "Task description should match")
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Internal Server Error"}`))
		}))
		defer server.Close()

		config := &Config{
			baseURL: server.URL,
			email:   "test@example.com",
			token:   "test-token",
		}
		client, err := NewClient(config)
		require.NoError(t, err, "Should not return error")
		tasks, err := client.FetchTasks(ctx, "TEST", "Sprint 1")
		require.Error(t, err, "Should return error")
		assert.Nil(t, tasks, "Tasks should be nil")
		assert.Contains(t, err.Error(), "unexpected status code: 500", "Error message should indicate server error")
	})

	t.Run("invalid response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`invalid json`))
		}))
		defer server.Close()

		config := &Config{
			baseURL: server.URL,
			email:   "test@example.com",
			token:   "test-token",
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
