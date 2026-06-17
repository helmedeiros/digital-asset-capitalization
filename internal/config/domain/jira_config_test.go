package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJiraConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		baseURL string
		email   string
		token   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid configuration",
			baseURL: "https://company.atlassian.net",
			email:   "user@company.com",
			token:   "valid-token-123",
			wantErr: false,
		},
		{
			name:    "empty base URL",
			baseURL: "",
			email:   "user@company.com",
			token:   "valid-token-123",
			wantErr: true,
			errMsg:  "base URL is required",
		},
		{
			name:    "invalid base URL format",
			baseURL: "not-a-url",
			email:   "user@company.com",
			token:   "valid-token-123",
			wantErr: true,
			errMsg:  "invalid base URL format",
		},
		{
			name:    "empty email",
			baseURL: "https://company.atlassian.net",
			email:   "",
			token:   "valid-token-123",
			wantErr: true,
			errMsg:  "email is required",
		},
		{
			name:    "invalid email format",
			baseURL: "https://company.atlassian.net",
			email:   "not-an-email",
			token:   "valid-token-123",
			wantErr: true,
			errMsg:  "invalid email format",
		},
		{
			name:    "empty token",
			baseURL: "https://company.atlassian.net",
			email:   "user@company.com",
			token:   "",
			wantErr: true,
			errMsg:  "token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := NewJiraConfig(tt.baseURL, tt.email, tt.token)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, config)
			} else {
				require.NoError(t, err)
				require.NotNil(t, config)
				assert.Equal(t, tt.baseURL, config.BaseURL())
				assert.Equal(t, tt.email, config.Email())
				assert.Equal(t, tt.token, config.Token())
			}
		})
	}
}

func TestJiraConfig_IsValid(t *testing.T) {
	t.Parallel()
	validConfig, err := NewJiraConfig("https://company.atlassian.net", "user@company.com", "token-123")
	require.NoError(t, err)

	assert.True(t, validConfig.IsValid())
}

func TestJiraConfig_AuthHeader(t *testing.T) {
	t.Parallel()
	config, err := NewJiraConfig("https://company.atlassian.net", "user@company.com", "token-123")
	require.NoError(t, err)

	authHeader := config.AuthHeader()
	assert.NotEmpty(t, authHeader)
	assert.Contains(t, authHeader, "Basic ")
}

func TestJiraConfig_String(t *testing.T) {
	t.Parallel()
	config, err := NewJiraConfig("https://company.atlassian.net", "user@company.com", "token-123")
	require.NoError(t, err)

	str := config.String()
	assert.Contains(t, str, "https://company.atlassian.net")
	assert.Contains(t, str, "user@company.com")
	assert.NotContains(t, str, "token-123") // Token should be masked
}
