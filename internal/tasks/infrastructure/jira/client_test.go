package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/jira/model"
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
	baseURL := "https://test.atlassian.net"
	email := "test@example.com"
	token := "test-token"

	client := NewClient(baseURL, email, token)
	assert.NotNil(t, client, "Client should not be nil")
}

func TestClient_FetchTasks(t *testing.T) {
	mockClient := &mockHTTPClient{}
	client := &client{
		httpClient: mockClient,
		baseURL:    "https://test.atlassian.net",
		email:      "test@example.com",
		token:      "test-token",
	}
	ctx := context.Background()

	t.Run("fetch tasks with empty project", func(t *testing.T) {
		tasks, err := client.FetchTasks(ctx, "", "Sprint 1")
		require.Error(t, err, "Should return error for empty project")
		assert.Nil(t, tasks, "Tasks should be nil")
		assert.Contains(t, err.Error(), "project is required", "Error message should indicate project is required")
	})

	t.Run("fetch tasks successfully", func(t *testing.T) {
		mockResponse := model.SearchResponse{
			Issues: []model.Issue{
				{
					Key: "TEST-1",
					Fields: model.Fields{
						Summary:     "Test Issue",
						Status:      model.Status{Name: "In Progress"},
						Project:     model.Project{Key: "TEST"},
						Sprint:      model.Sprint{Name: "Sprint 1"},
						Created:     "2024-03-20T10:00:00.000Z",
						Updated:     "2024-03-20T11:00:00.000Z",
						Description: "Test Description",
					},
				},
			},
		}

		mockClient.response = createMockResponse(t, http.StatusOK, mockResponse)
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

		expectedCreated, err := time.Parse(time.RFC3339, "2024-03-20T10:00:00.000Z")
		require.NoError(t, err, "Should parse created time")
		assert.Equal(t, expectedCreated, task.CreatedAt, "Task created time should match")

		expectedUpdated, err := time.Parse(time.RFC3339, "2024-03-20T11:00:00.000Z")
		require.NoError(t, err, "Should parse updated time")
		assert.Equal(t, expectedUpdated, task.UpdatedAt, "Task updated time should match")
	})

	t.Run("handle HTTP error", func(t *testing.T) {
		mockClient.response = &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(bytes.NewReader([]byte(`{"error": "Bad Request"}`))),
		}

		tasks, err := client.FetchTasks(ctx, "TEST", "Sprint 1")
		require.Error(t, err, "Should return error")
		assert.Nil(t, tasks, "Tasks should be nil")
		assert.Contains(t, err.Error(), "unexpected status code: 400", "Error message should indicate bad request")
	})

	t.Run("handle network error", func(t *testing.T) {
		mockClient.response = nil
		mockClient.err = io.EOF

		tasks, err := client.FetchTasks(ctx, "TEST", "Sprint 1")
		require.Error(t, err, "Should return error")
		assert.Nil(t, tasks, "Tasks should be nil")
		assert.Contains(t, err.Error(), "failed to execute request", "Error message should indicate request failure")
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
