package infrastructure

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/application/usecase"
	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

// Mock config service for testing
type MockConfigService struct {
	mock.Mock
}

func (m *MockConfigService) GetJiraConfig() (*domain.JiraConfig, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.JiraConfig), args.Error(1)
}

// MockHTTPClient for testing HTTP requests
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

// HTTPClientInterface defines the interface for HTTP client
type HTTPClientInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

// Helper function to create a mock HTTP response
func createMockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

// Helper function to create adapter with mock HTTP client
func createAdapterWithMockHTTP(configService ConfigServiceInterface, httpClient HTTPClientInterface) *JiraQueryAdapter {
	adapter := &JiraQueryAdapter{
		configService: configService,
	}
	// Set up the mock client wrapper
	if mockClient, ok := httpClient.(*MockHTTPClient); ok {
		adapter.httpClient = &http.Client{
			Transport: &mockTransport{mock: mockClient},
		}
	} else {
		adapter.httpClient = &http.Client{}
	}
	return adapter
}

// mockTransport implements http.RoundTripper to integrate with http.Client
type mockTransport struct {
	mock *MockHTTPClient
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.mock.Do(req)
}

func TestNewJiraQueryAdapter(t *testing.T) {
	t.Run("should create adapter successfully", func(t *testing.T) {
		mockConfigService := &MockConfigService{}

		adapter, err := NewJiraQueryAdapter(mockConfigService)
		assert.NoError(t, err)
		assert.NotNil(t, adapter)
	})

	t.Run("should return error for nil config service", func(t *testing.T) {
		adapter, err := NewJiraQueryAdapter(nil)
		assert.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "config service is required")
	})
}

func TestJiraQueryAdapter_SearchTasksByLabelPrefix(t *testing.T) {
	t.Run("should handle config service error", func(t *testing.T) {
		mockConfigService := &MockConfigService{}
		mockHTTPClient := &MockHTTPClient{}
		adapter := createAdapterWithMockHTTP(mockConfigService, mockHTTPClient)

		mockConfigService.On("GetJiraConfig").Return(nil, fmt.Errorf("config error"))

		ctx := context.Background()
		_, err := adapter.SearchTasksByLabelPrefix(ctx, "cap-asset", 10)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config error")

		mockConfigService.AssertExpectations(t)
	})

	t.Run("should search tasks successfully", func(t *testing.T) {
		mockConfigService := &MockConfigService{}
		mockHTTPClient := &MockHTTPClient{}
		adapter := createAdapterWithMockHTTP(mockConfigService, mockHTTPClient)

		jiraConfig, _ := domain.NewJiraConfig("https://example.atlassian.net", "user@example.com", "token")
		mockConfigService.On("GetJiraConfig").Return(jiraConfig, nil)

		responseBody := `{
			"issues": [
				{
					"key": "TEST-1",
					"fields": {
						"labels": ["cap-asset-test"],
						"assignee": {"displayName": "John Doe"},
						"reporter": {"displayName": "Jane Smith"}
					}
				}
			],
			"total": 1
		}`

		mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
			createMockResponse(200, responseBody), nil)

		ctx := context.Background()
		tasks, err := adapter.SearchTasksByLabelPrefix(ctx, "cap-asset", 10)

		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "TEST-1", tasks[0].Key)
		assert.Equal(t, "John Doe", tasks[0].Assignee)
		assert.Equal(t, "Jane Smith", tasks[0].Reporter)

		mockConfigService.AssertExpectations(t)
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("should handle HTTP error", func(t *testing.T) {
		mockConfigService := &MockConfigService{}
		mockHTTPClient := &MockHTTPClient{}
		adapter := createAdapterWithMockHTTP(mockConfigService, mockHTTPClient)

		jiraConfig, _ := domain.NewJiraConfig("https://example.atlassian.net", "user@example.com", "token")
		mockConfigService.On("GetJiraConfig").Return(jiraConfig, nil)

		mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
			nil, fmt.Errorf("network error"))

		ctx := context.Background()
		_, err := adapter.SearchTasksByLabelPrefix(ctx, "cap-asset", 10)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "network error")

		mockConfigService.AssertExpectations(t)
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("should handle HTTP status error", func(t *testing.T) {
		mockConfigService := &MockConfigService{}
		mockHTTPClient := &MockHTTPClient{}
		adapter := createAdapterWithMockHTTP(mockConfigService, mockHTTPClient)

		jiraConfig, _ := domain.NewJiraConfig("https://example.atlassian.net", "user@example.com", "token")
		mockConfigService.On("GetJiraConfig").Return(jiraConfig, nil)

		errorBody := `{"errorMessages": ["Unauthorized"]}`
		mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
			createMockResponse(401, errorBody), nil)

		ctx := context.Background()
		_, err := adapter.SearchTasksByLabelPrefix(ctx, "cap-asset", 10)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "JIRA API error (status 401)")
		assert.Contains(t, err.Error(), "Unauthorized")

		mockConfigService.AssertExpectations(t)
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("should handle JSON decode error", func(t *testing.T) {
		mockConfigService := &MockConfigService{}
		mockHTTPClient := &MockHTTPClient{}
		adapter := createAdapterWithMockHTTP(mockConfigService, mockHTTPClient)

		jiraConfig, _ := domain.NewJiraConfig("https://example.atlassian.net", "user@example.com", "token")
		mockConfigService.On("GetJiraConfig").Return(jiraConfig, nil)

		// Mock response with invalid JSON
		invalidJSON := `{"issues": [invalid json}`
		mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
			createMockResponse(200, invalidJSON), nil)

		ctx := context.Background()
		_, err := adapter.SearchTasksByLabelPrefix(ctx, "cap-asset", 10)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode response")

		mockConfigService.AssertExpectations(t)
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("should handle large max results with JIRA limit", func(t *testing.T) {
		mockConfigService := &MockConfigService{}
		mockHTTPClient := &MockHTTPClient{}
		adapter := createAdapterWithMockHTTP(mockConfigService, mockHTTPClient)

		jiraConfig, _ := domain.NewJiraConfig("https://example.atlassian.net", "user@example.com", "token")
		mockConfigService.On("GetJiraConfig").Return(jiraConfig, nil)

		responseBody := `{"issues": [], "total": 0}`
		mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
			// Verify that maxResults is capped at 5000
			return strings.Contains(req.URL.String(), "maxResults=5000")
		})).Return(createMockResponse(200, responseBody), nil)

		ctx := context.Background()
		_, err := adapter.SearchTasksByLabelPrefix(ctx, "cap-asset", 2000) // Should be capped

		assert.NoError(t, err)

		mockConfigService.AssertExpectations(t)
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("should filter tasks by label prefix case insensitive", func(t *testing.T) {
		mockConfigService := &MockConfigService{}
		mockHTTPClient := &MockHTTPClient{}
		adapter := createAdapterWithMockHTTP(mockConfigService, mockHTTPClient)

		jiraConfig, _ := domain.NewJiraConfig("https://example.atlassian.net", "user@example.com", "token")
		mockConfigService.On("GetJiraConfig").Return(jiraConfig, nil)

		responseBody := `{
			"issues": [
				{
					"key": "TEST-1",
					"fields": {
						"labels": ["CAP-ASSET-TEST"],
						"assignee": null,
						"reporter": null
					}
				}
			],
			"total": 1
		}`

		mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
			createMockResponse(200, responseBody), nil)

		ctx := context.Background()
		tasks, err := adapter.SearchTasksByLabelPrefix(ctx, "cap-asset", 10)

		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "TEST-1", tasks[0].Key)

		mockConfigService.AssertExpectations(t)
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("should respect max results limit in filtering", func(t *testing.T) {
		mockConfigService := &MockConfigService{}
		mockHTTPClient := &MockHTTPClient{}
		adapter := createAdapterWithMockHTTP(mockConfigService, mockHTTPClient)

		jiraConfig, _ := domain.NewJiraConfig("https://example.atlassian.net", "user@example.com", "token")
		mockConfigService.On("GetJiraConfig").Return(jiraConfig, nil)

		// Create response with 3 matching tasks but request only 2
		responseBody := `{
			"issues": [
				{
					"key": "TEST-1",
					"fields": {
						"labels": ["cap-asset-test1"],
						"assignee": null,
						"reporter": null
					}
				},
				{
					"key": "TEST-2",
					"fields": {
						"labels": ["cap-asset-test2"],
						"assignee": null,
						"reporter": null
					}
				},
				{
					"key": "TEST-3",
					"fields": {
						"labels": ["cap-asset-test3"],
						"assignee": null,
						"reporter": null
					}
				}
			],
			"total": 3
		}`

		mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
			createMockResponse(200, responseBody), nil)

		ctx := context.Background()
		tasks, err := adapter.SearchTasksByLabelPrefix(ctx, "cap-asset", 2) // Limit to 2

		require.NoError(t, err)
		assert.Len(t, tasks, 2) // Should only return 2 tasks
		assert.Equal(t, "TEST-1", tasks[0].Key)
		assert.Equal(t, "TEST-2", tasks[1].Key)

		mockConfigService.AssertExpectations(t)
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("should handle missing assignee and reporter fields", func(t *testing.T) {
		mockConfigService := &MockConfigService{}
		mockHTTPClient := &MockHTTPClient{}
		adapter := createAdapterWithMockHTTP(mockConfigService, mockHTTPClient)

		jiraConfig, _ := domain.NewJiraConfig("https://example.atlassian.net", "user@example.com", "token")
		mockConfigService.On("GetJiraConfig").Return(jiraConfig, nil)

		responseBody := `{
			"issues": [
				{
					"key": "TEST-1",
					"fields": {
						"labels": ["cap-asset-test"],
						"assignee": null,
						"reporter": null
					}
				}
			],
			"total": 1
		}`

		mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
			createMockResponse(200, responseBody), nil)

		ctx := context.Background()
		tasks, err := adapter.SearchTasksByLabelPrefix(ctx, "cap-asset", 10)

		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "", tasks[0].Assignee)
		assert.Equal(t, "", tasks[0].Reporter)

		mockConfigService.AssertExpectations(t)
		mockHTTPClient.AssertExpectations(t)
	})
}

func TestJiraQueryAdapter_SearchTasksWithFilters(t *testing.T) {
	t.Run("should handle config service error", func(t *testing.T) {
		mockConfigService := &MockConfigService{}
		mockHTTPClient := &MockHTTPClient{}
		adapter := createAdapterWithMockHTTP(mockConfigService, mockHTTPClient)

		mockConfigService.On("GetJiraConfig").Return(nil, fmt.Errorf("config error"))

		filters := usecase.JiraSearchFilters{
			LabelPrefix: "cap-asset",
			ProjectKey:  "TEST",
			MaxResults:  10,
		}

		ctx := context.Background()
		_, err := adapter.SearchTasksWithFilters(ctx, filters)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "config error")

		mockConfigService.AssertExpectations(t)
	})

	t.Run("should successfully search with project filter", func(t *testing.T) {
		mockConfigService := &MockConfigService{}
		mockHTTPClient := &MockHTTPClient{}
		adapter := createAdapterWithMockHTTP(mockConfigService, mockHTTPClient)

		jiraConfig, _ := domain.NewJiraConfig("https://example.atlassian.net", "user@example.com", "token")
		mockConfigService.On("GetJiraConfig").Return(jiraConfig, nil)

		responseBody := `{
			"issues": [
				{
					"key": "TEST-1",
					"fields": {
						"labels": ["cap-asset-test"],
						"assignee": {"emailAddress": "user@example.com"},
						"reporter": {"displayName": "Reporter"}
					}
				}
			],
			"total": 1
		}`

		mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
			return strings.Contains(req.URL.String(), "/rest/api/3/search/jql") &&
				strings.Contains(req.URL.String(), "project+%3D+%22TEST%22")
		})).Return(createMockResponse(200, responseBody), nil)

		filters := usecase.JiraSearchFilters{
			LabelPrefix: "cap-asset",
			ProjectKey:  "TEST",
			MaxResults:  10,
		}

		ctx := context.Background()
		tasks, err := adapter.SearchTasksWithFilters(ctx, filters)

		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "TEST-1", tasks[0].Key)
		assert.Equal(t, "user@example.com", tasks[0].Assignee)
		assert.Equal(t, "Reporter", tasks[0].Reporter)

		mockConfigService.AssertExpectations(t)
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("should successfully search with sprint filter", func(t *testing.T) {
		mockConfigService := &MockConfigService{}
		mockHTTPClient := &MockHTTPClient{}
		adapter := createAdapterWithMockHTTP(mockConfigService, mockHTTPClient)

		jiraConfig, _ := domain.NewJiraConfig("https://example.atlassian.net", "user@example.com", "token")
		mockConfigService.On("GetJiraConfig").Return(jiraConfig, nil)

		responseBody := `{
			"issues": [
				{
					"key": "SPRINT-1",
					"fields": {
						"labels": ["cap-asset-sprint-test"],
						"assignee": null,
						"reporter": {"accountId": "account123"}
					}
				}
			],
			"total": 1
		}`

		mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
			return strings.Contains(req.URL.String(), "/rest/api/3/search/jql") &&
				strings.Contains(req.URL.String(), "sprint+%3D+%22Sprint+1%22")
		})).Return(createMockResponse(200, responseBody), nil)

		filters := usecase.JiraSearchFilters{
			LabelPrefix: "cap-asset",
			SprintName:  "Sprint 1",
			MaxResults:  10,
		}

		ctx := context.Background()
		tasks, err := adapter.SearchTasksWithFilters(ctx, filters)

		require.NoError(t, err)
		assert.Len(t, tasks, 1)
		assert.Equal(t, "SPRINT-1", tasks[0].Key)
		assert.Equal(t, "", tasks[0].Assignee)
		assert.Equal(t, "account123", tasks[0].Reporter)

		mockConfigService.AssertExpectations(t)
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("should handle HTTP request error", func(t *testing.T) {
		mockConfigService := &MockConfigService{}
		mockHTTPClient := &MockHTTPClient{}
		adapter := createAdapterWithMockHTTP(mockConfigService, mockHTTPClient)

		jiraConfig, _ := domain.NewJiraConfig("https://example.atlassian.net", "user@example.com", "token")
		mockConfigService.On("GetJiraConfig").Return(jiraConfig, nil)

		mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(nil, fmt.Errorf("network error"))

		filters := usecase.JiraSearchFilters{
			LabelPrefix: "cap-asset",
			MaxResults:  10,
		}

		ctx := context.Background()
		_, err := adapter.SearchTasksWithFilters(ctx, filters)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to execute request")
		assert.Contains(t, err.Error(), "network error")

		mockConfigService.AssertExpectations(t)
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("should handle HTTP status error", func(t *testing.T) {
		mockConfigService := &MockConfigService{}
		mockHTTPClient := &MockHTTPClient{}
		adapter := createAdapterWithMockHTTP(mockConfigService, mockHTTPClient)

		jiraConfig, _ := domain.NewJiraConfig("https://example.atlassian.net", "user@example.com", "token")
		mockConfigService.On("GetJiraConfig").Return(jiraConfig, nil)

		errorBody := `{"errorMessages":["Forbidden"]}`
		mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
			createMockResponse(403, errorBody), nil)

		filters := usecase.JiraSearchFilters{
			LabelPrefix: "cap-asset",
			MaxResults:  10,
		}

		ctx := context.Background()
		_, err := adapter.SearchTasksWithFilters(ctx, filters)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "JIRA API error (status 403)")
		assert.Contains(t, err.Error(), "Forbidden")

		mockConfigService.AssertExpectations(t)
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("should handle JSON decode error", func(t *testing.T) {
		mockConfigService := &MockConfigService{}
		mockHTTPClient := &MockHTTPClient{}
		adapter := createAdapterWithMockHTTP(mockConfigService, mockHTTPClient)

		jiraConfig, _ := domain.NewJiraConfig("https://example.atlassian.net", "user@example.com", "token")
		mockConfigService.On("GetJiraConfig").Return(jiraConfig, nil)

		invalidJSON := `{"issues": [malformed json}`
		mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
			createMockResponse(200, invalidJSON), nil)

		filters := usecase.JiraSearchFilters{
			LabelPrefix: "cap-asset",
			MaxResults:  10,
		}

		ctx := context.Background()
		_, err := adapter.SearchTasksWithFilters(ctx, filters)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode response")

		mockConfigService.AssertExpectations(t)
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("should cap large max results to JIRA limit", func(t *testing.T) {
		mockConfigService := &MockConfigService{}
		mockHTTPClient := &MockHTTPClient{}
		adapter := createAdapterWithMockHTTP(mockConfigService, mockHTTPClient)

		jiraConfig, _ := domain.NewJiraConfig("https://example.atlassian.net", "user@example.com", "token")
		mockConfigService.On("GetJiraConfig").Return(jiraConfig, nil)

		responseBody := `{"issues": [], "total": 0}`
		mockHTTPClient.On("Do", mock.MatchedBy(func(req *http.Request) bool {
			// Verify that maxResults is capped at 5000
			return strings.Contains(req.URL.String(), "maxResults=5000")
		})).Return(createMockResponse(200, responseBody), nil)

		filters := usecase.JiraSearchFilters{
			LabelPrefix: "cap-asset",
			MaxResults:  2000, // Should be capped at 5000 internally
		}

		ctx := context.Background()
		_, err := adapter.SearchTasksWithFilters(ctx, filters)

		assert.NoError(t, err)

		mockConfigService.AssertExpectations(t)
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("should filter by label prefix correctly", func(t *testing.T) {
		mockConfigService := &MockConfigService{}
		mockHTTPClient := &MockHTTPClient{}
		adapter := createAdapterWithMockHTTP(mockConfigService, mockHTTPClient)

		jiraConfig, _ := domain.NewJiraConfig("https://example.atlassian.net", "user@example.com", "token")
		mockConfigService.On("GetJiraConfig").Return(jiraConfig, nil)

		// Response with mixed labels - only cap-asset tasks should be returned
		responseBody := `{
			"issues": [
				{
					"key": "TEST-1",
					"fields": {
						"labels": ["cap-asset-test"],
						"assignee": null,
						"reporter": null
					}
				},
				{
					"key": "TEST-2",
					"fields": {
						"labels": ["other-label"],
						"assignee": null,
						"reporter": null
					}
				}
			],
			"total": 2
		}`

		mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
			createMockResponse(200, responseBody), nil)

		filters := usecase.JiraSearchFilters{
			LabelPrefix: "cap-asset",
			MaxResults:  10,
		}

		ctx := context.Background()
		tasks, err := adapter.SearchTasksWithFilters(ctx, filters)

		require.NoError(t, err)
		assert.Len(t, tasks, 1) // Only TEST-1 should be returned
		assert.Equal(t, "TEST-1", tasks[0].Key)

		mockConfigService.AssertExpectations(t)
		mockHTTPClient.AssertExpectations(t)
	})

	t.Run("should respect max results limit", func(t *testing.T) {
		mockConfigService := &MockConfigService{}
		mockHTTPClient := &MockHTTPClient{}
		adapter := createAdapterWithMockHTTP(mockConfigService, mockHTTPClient)

		jiraConfig, _ := domain.NewJiraConfig("https://example.atlassian.net", "user@example.com", "token")
		mockConfigService.On("GetJiraConfig").Return(jiraConfig, nil)

		// Response with 3 matching tasks but we'll limit to 2
		responseBody := `{
			"issues": [
				{
					"key": "TEST-1",
					"fields": {
						"labels": ["cap-asset-test1"],
						"assignee": null,
						"reporter": null
					}
				},
				{
					"key": "TEST-2",
					"fields": {
						"labels": ["cap-asset-test2"],
						"assignee": null,
						"reporter": null
					}
				},
				{
					"key": "TEST-3",
					"fields": {
						"labels": ["cap-asset-test3"],
						"assignee": null,
						"reporter": null
					}
				}
			],
			"total": 3
		}`

		mockHTTPClient.On("Do", mock.AnythingOfType("*http.Request")).Return(
			createMockResponse(200, responseBody), nil)

		filters := usecase.JiraSearchFilters{
			LabelPrefix: "cap-asset",
			MaxResults:  2, // Limit to 2
		}

		ctx := context.Background()
		tasks, err := adapter.SearchTasksWithFilters(ctx, filters)

		require.NoError(t, err)
		assert.Len(t, tasks, 2) // Should only return 2 tasks
		assert.Equal(t, "TEST-1", tasks[0].Key)
		assert.Equal(t, "TEST-2", tasks[1].Key)

		mockConfigService.AssertExpectations(t)
		mockHTTPClient.AssertExpectations(t)
	})
}

func TestJiraQueryAdapter_ExtractUserIdentifier(t *testing.T) {
	mockConfigService := &MockConfigService{}
	adapter, _ := NewJiraQueryAdapter(mockConfigService)

	t.Run("should prefer email address", func(t *testing.T) {
		user := &JiraUser{
			EmailAddress: "user@example.com",
			DisplayName:  "John Doe",
			AccountID:    "account123",
		}

		result := adapter.extractUserIdentifier(user)
		assert.Equal(t, "user@example.com", result)
	})

	t.Run("should fallback to display name", func(t *testing.T) {
		user := &JiraUser{
			EmailAddress: "",
			DisplayName:  "John Doe",
			AccountID:    "account123",
		}

		result := adapter.extractUserIdentifier(user)
		assert.Equal(t, "John Doe", result)
	})

	t.Run("should fallback to account ID", func(t *testing.T) {
		user := &JiraUser{
			EmailAddress: "",
			DisplayName:  "",
			AccountID:    "account123",
		}

		result := adapter.extractUserIdentifier(user)
		assert.Equal(t, "account123", result)
	})

	t.Run("should handle all empty fields", func(t *testing.T) {
		user := &JiraUser{
			EmailAddress: "",
			DisplayName:  "",
			AccountID:    "",
		}

		result := adapter.extractUserIdentifier(user)
		assert.Equal(t, "", result)
	})
}

func TestJiraQueryAdapter_Types(t *testing.T) {
	t.Run("should create JiraSearchResponse correctly", func(t *testing.T) {
		response := JiraSearchResponse{
			Issues: []JiraIssue{
				{
					Key: "TEST-1",
					Fields: JiraIssueFields{
						Labels: []string{"cap-asset-test"},
						Assignee: &JiraUser{
							EmailAddress: "user@example.com",
						},
					},
				},
			},
			Total: 1,
		}

		assert.Len(t, response.Issues, 1)
		assert.Equal(t, "TEST-1", response.Issues[0].Key)
		assert.Equal(t, 1, response.Total)
	})

	t.Run("should create JiraIssue correctly", func(t *testing.T) {
		issue := JiraIssue{
			Key: "TEST-1",
			Fields: JiraIssueFields{
				Labels: []string{"cap-asset-test"},
				Assignee: &JiraUser{
					EmailAddress: "user@example.com",
				},
				Reporter: &JiraUser{
					DisplayName: "Jane Doe",
				},
			},
		}

		assert.Equal(t, "TEST-1", issue.Key)
		assert.Len(t, issue.Fields.Labels, 1)
		assert.NotNil(t, issue.Fields.Assignee)
		assert.NotNil(t, issue.Fields.Reporter)
	})

	t.Run("should create JiraUser correctly", func(t *testing.T) {
		user := JiraUser{
			DisplayName:  "John Doe",
			EmailAddress: "john.doe@example.com",
			AccountID:    "account123",
		}

		assert.Equal(t, "John Doe", user.DisplayName)
		assert.Equal(t, "john.doe@example.com", user.EmailAddress)
		assert.Equal(t, "account123", user.AccountID)
	})
}
