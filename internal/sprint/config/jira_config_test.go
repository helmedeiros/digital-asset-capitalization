package config

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// JiraConfigTestCase represents a test case for Jira configuration
type JiraConfigTestCase struct {
	Name    string
	EnvVars map[string]string
	WantErr bool
	ErrType error
}

// setupEnvVars sets up environment variables for a test and returns a cleanup function
func setupEnvVars(vars map[string]string) func() {
	// Clear all relevant environment variables
	os.Unsetenv("JIRA_BASE_URL")
	os.Unsetenv("JIRA_EMAIL")
	os.Unsetenv("JIRA_TOKEN")

	// Set test env vars
	for k, v := range vars {
		os.Setenv(k, v)
	}

	// Return cleanup function
	return func() {
		os.Unsetenv("JIRA_BASE_URL")
		os.Unsetenv("JIRA_EMAIL")
		os.Unsetenv("JIRA_TOKEN")
	}
}

func TestNewJiraConfig(t *testing.T) {
	// Save current env vars
	oldBaseURL := os.Getenv(envJiraBaseURL)
	oldEmail := os.Getenv(envJiraEmail)
	oldToken := os.Getenv(envJiraToken)

	// Restore env vars after test
	defer func() {
		os.Setenv(envJiraBaseURL, oldBaseURL)
		os.Setenv(envJiraEmail, oldEmail)
		os.Setenv(envJiraToken, oldToken)
	}()

	tests := []struct {
		name    string
		baseURL string
		email   string
		token   string
		wantErr bool
		errType error
	}{
		{
			name:    "valid configuration",
			baseURL: "https://example.atlassian.net",
			email:   "test@example.com",
			token:   "test-token",
			wantErr: false,
		},
		{
			name:    "missing base URL",
			baseURL: "",
			email:   "test@example.com",
			token:   "test-token",
			wantErr: true,
			errType: ErrMissingBaseURL,
		},
		{
			name:    "missing email",
			baseURL: "https://example.atlassian.net",
			email:   "",
			token:   "test-token",
			wantErr: true,
			errType: ErrMissingEmail,
		},
		{
			name:    "missing token",
			baseURL: "https://example.atlassian.net",
			email:   "test@example.com",
			token:   "",
			wantErr: true,
			errType: ErrMissingToken,
		},
		{
			name:    "invalid base URL",
			baseURL: "not-a-url",
			email:   "test@example.com",
			token:   "test-token",
			wantErr: true,
			errType: ErrInvalidBaseURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test environment variables
			os.Setenv(envJiraBaseURL, tt.baseURL)
			os.Setenv(envJiraEmail, tt.email)
			os.Setenv(envJiraToken, tt.token)

			config, err := NewJiraConfig()

			// Check error cases
			if tt.wantErr {
				assert.Error(t, err, "Expected error but got none")
				assert.True(t, errors.Is(err, tt.errType), "Expected error type %v but got %v", tt.errType, err)
				return
			}

			// Check successful cases
			require.NoError(t, err, "Unexpected error")
			assert.Equal(t, tt.baseURL, config.GetBaseURL(), "Base URL mismatch")
			assert.Equal(t, tt.email, config.GetEmail(), "Email mismatch")
		})
	}
}

func TestGetAuthHeader(t *testing.T) {
	config := &JiraConfig{
		email: "test@example.com",
		token: "test-token",
	}

	expected := "Basic " + "dGVzdEBleGFtcGxlLmNvbTp0ZXN0LXRva2Vu"
	assert.Equal(t, expected, config.GetAuthHeader(), "Auth header mismatch")
}
