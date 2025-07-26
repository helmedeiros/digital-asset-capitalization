package infrastructure

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

const searchPath = "/rest/api/3/search"

// MockConfigService for testing
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

func TestNewJiraTeamSyncAdapter(t *testing.T) {
	tests := []struct {
		name          string
		configService interface{}
		wantErr       bool
		errContains   string
	}{
		{
			name:          "successful creation",
			configService: &MockConfigService{},
			wantErr:       false,
		},
		{
			name:          "nil config service",
			configService: nil,
			wantErr:       true,
			errContains:   "config service is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var configService ConfigServiceInterface
			if tt.configService != nil {
				configService = tt.configService.(*MockConfigService)
			}

			adapter, err := NewJiraTeamSyncAdapter(configService)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, adapter)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, adapter)
				assert.NotNil(t, adapter.httpClient)
			}
		})
	}
}

func TestJiraTeamSyncAdapter_GetProjectMembers(t *testing.T) {
	tests := []struct {
		name           string
		projectKey     string
		setupMocks     func(*MockConfigService, *httptest.Server)
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
		expectedCount  int
	}{
		{
			name:       "successful assignable users fetch",
			projectKey: "TEST",
			setupMocks: func(mockConfig *MockConfigService, server *httptest.Server) {
				jiraConfig, _ := domain.NewJiraConfig(server.URL, "test@example.com", "token123")
				mockConfig.On("GetJiraConfig").Return(jiraConfig, nil)
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == searchPath && r.URL.Query().Get("jql") != "" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{
						"issues": [
							{
								"fields": {
									"assignee": {
										"accountId": "123",
										"displayName": "John Doe",
										"emailAddress": "john@example.com",
										"active": true
									}
								}
							},
							{
								"fields": {
									"assignee": {
										"accountId": "456",
										"displayName": "Jane Smith",
										"emailAddress": "jane@example.com",
										"active": true
									}
								}
							}
						],
						"total": 2
					}`))
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			},
			wantErr:       false,
			expectedCount: 2,
		},
		{
			name:        "empty project key",
			projectKey:  "",
			setupMocks:  func(*MockConfigService, *httptest.Server) {},
			wantErr:     true,
			errContains: "project key is required",
		},
		{
			name:       "config service error",
			projectKey: "TEST",
			setupMocks: func(mockConfig *MockConfigService, _ *httptest.Server) {
				mockConfig.On("GetJiraConfig").Return(nil, assert.AnError)
			},
			wantErr:     true,
			errContains: "failed to get JIRA config",
		},
		{
			name:       "JIRA API error",
			projectKey: "TEST",
			setupMocks: func(mockConfig *MockConfigService, server *httptest.Server) {
				jiraConfig, _ := domain.NewJiraConfig(server.URL, "test@example.com", "token123")
				mockConfig.On("GetJiraConfig").Return(jiraConfig, nil)
			},
			serverResponse: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"errorMessages":["Authentication failed"]}`))
			},
			wantErr:     true,
			errContains: "JIRA API error",
		},
		{
			name:       "inactive users filtered out",
			projectKey: "TEST",
			setupMocks: func(mockConfig *MockConfigService, server *httptest.Server) {
				jiraConfig, _ := domain.NewJiraConfig(server.URL, "test@example.com", "token123")
				mockConfig.On("GetJiraConfig").Return(jiraConfig, nil)
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == searchPath {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{
						"issues": [
							{
								"fields": {
									"assignee": {
										"accountId": "123",
										"displayName": "John Doe",
										"emailAddress": "john@example.com",
										"active": true
									}
								}
							},
							{
								"fields": {
									"assignee": {
										"accountId": "456",
										"displayName": "Inactive User",
										"emailAddress": "inactive@example.com",
										"active": false
									}
								}
							}
						],
						"total": 2
					}`))
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			},
			wantErr:       false,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			mockConfig := &MockConfigService{}
			tt.setupMocks(mockConfig, server)

			adapter, _ := NewJiraTeamSyncAdapter(mockConfig)
			adapter.httpClient = server.Client()

			result, err := adapter.GetProjectMembers(tt.projectKey)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.projectKey, result.ProjectKey)
				assert.Len(t, result.Members, tt.expectedCount)

				// Verify that all returned members are active
				for _, member := range result.Members {
					assert.NotEmpty(t, member.AccountID)
					assert.NotEmpty(t, member.DisplayName)
				}
			}

			mockConfig.AssertExpectations(t)
		})
	}
}

func TestJiraTeamSyncAdapter_GetAssignableUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == searchPath && r.URL.Query().Get("jql") != "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"issues": [
					{
						"fields": {
							"assignee": {
								"accountId": "123",
								"displayName": "John Doe",
								"emailAddress": "john@example.com",
								"name": "john.doe",
								"active": true
							}
						}
					}
				],
				"total": 1
			}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	mockConfig := &MockConfigService{}
	jiraConfig, _ := domain.NewJiraConfig(server.URL, "test@example.com", "token123")
	mockConfig.On("GetJiraConfig").Return(jiraConfig, nil)

	adapter, _ := NewJiraTeamSyncAdapter(mockConfig)
	adapter.httpClient = server.Client()

	members, err := adapter.GetAssignableUsers("TEST")

	assert.NoError(t, err)
	assert.Len(t, members, 1)
	assert.Equal(t, "123", members[0].AccountID)
	assert.Equal(t, "John Doe", members[0].DisplayName)
	assert.Equal(t, "john@example.com", members[0].Email)
	assert.Equal(t, "john.doe", members[0].Name)

	mockConfig.AssertExpectations(t)
}

func TestJiraTeamSyncAdapter_GetProjectRoles(t *testing.T) {
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/3/project/TEST/role":
			// Return role URLs
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"Developers": "` + serverURL + `/rest/api/3/project/TEST/role/10100",
				"Administrators": "` + serverURL + `/rest/api/3/project/TEST/role/10101"
			}`))
		case "/rest/api/3/project/TEST/role/10100":
			// Return developers role details
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"role": {
					"id": 10100,
					"name": "Developers",
					"description": "Developer role"
				},
				"actors": [
					{
						"accountId": "123",
						"displayName": "John Developer",
						"emailAddress": "john@example.com",
						"active": true
					}
				]
			}`))
		case "/rest/api/3/project/TEST/role/10101":
			// Return administrators role details
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"role": {
					"id": 10101,
					"name": "Administrators",
					"description": "Administrator role"  
				},
				"actors": [
					{
						"accountId": "456",
						"displayName": "Jane Admin",
						"emailAddress": "jane@example.com",
						"active": true
					}
				]
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	serverURL = server.URL
	defer server.Close()

	mockConfig := &MockConfigService{}
	jiraConfig, _ := domain.NewJiraConfig(server.URL, "test@example.com", "token123")
	mockConfig.On("GetJiraConfig").Return(jiraConfig, nil)

	adapter, _ := NewJiraTeamSyncAdapter(mockConfig)
	adapter.httpClient = server.Client()

	roles, err := adapter.GetProjectRoles("TEST")

	assert.NoError(t, err)
	assert.Len(t, roles, 2)

	assert.Contains(t, roles, "Developers")
	assert.Contains(t, roles, "Administrators")

	assert.Len(t, roles["Developers"], 1)
	assert.Equal(t, "John Developer", roles["Developers"][0].DisplayName)

	assert.Len(t, roles["Administrators"], 1)
	assert.Equal(t, "Jane Admin", roles["Administrators"][0].DisplayName)

	mockConfig.AssertExpectations(t)
}

func TestConvertJiraUsersToTeamMembers(t *testing.T) {
	adapter := &JiraTeamSyncAdapter{}

	jiraUsers := []JiraUser{
		{
			AccountID:    "123",
			DisplayName:  "John Doe",
			EmailAddress: "john@example.com",
			Name:         "john.doe",
			Active:       true,
		},
		{
			AccountID:    "456",
			DisplayName:  "Jane Smith",
			EmailAddress: "jane@example.com",
			Name:         "",
			Active:       true,
		},
		{
			AccountID:    "789",
			DisplayName:  "Inactive User",
			EmailAddress: "inactive@example.com",
			Name:         "inactive.user",
			Active:       false,
		},
	}

	members := adapter.convertJiraUsersToTeamMembers(jiraUsers)

	// Should only include active users
	assert.Len(t, members, 2)

	// First member
	assert.Equal(t, "123", members[0].AccountID)
	assert.Equal(t, "John Doe", members[0].DisplayName)
	assert.Equal(t, "john@example.com", members[0].Email)
	assert.Equal(t, "john.doe", members[0].Name)

	// Second member - name should fallback to display name when empty
	assert.Equal(t, "456", members[1].AccountID)
	assert.Equal(t, "Jane Smith", members[1].DisplayName)
	assert.Equal(t, "jane@example.com", members[1].Email)
	assert.Equal(t, "Jane Smith", members[1].Name) // Fallback to DisplayName
}
