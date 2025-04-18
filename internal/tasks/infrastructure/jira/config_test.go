package jira

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
	}{
		{
			name: "valid config",
			envVars: map[string]string{
				"JIRA_BASE_URL": "https://test.atlassian.net",
				"JIRA_EMAIL":    "test@example.com",
				"JIRA_TOKEN":    "test-token",
			},
			wantErr: false,
		},
		{
			name: "invalid base URL",
			envVars: map[string]string{
				"JIRA_BASE_URL": "invalid-url",
				"JIRA_EMAIL":    "test@example.com",
				"JIRA_TOKEN":    "test-token",
			},
			wantErr: true,
		},
		{
			name: "empty email",
			envVars: map[string]string{
				"JIRA_BASE_URL": "https://test.atlassian.net",
				"JIRA_EMAIL":    "",
				"JIRA_TOKEN":    "test-token",
			},
			wantErr: true,
		},
		{
			name: "empty token",
			envVars: map[string]string{
				"JIRA_BASE_URL": "https://test.atlassian.net",
				"JIRA_EMAIL":    "test@example.com",
				"JIRA_TOKEN":    "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env vars
			origEnv := make(map[string]string)
			for k, v := range tt.envVars {
				origEnv[k] = os.Getenv(k)
				os.Setenv(k, v)
			}
			defer func() {
				// Restore original env vars
				for k, v := range origEnv {
					os.Setenv(k, v)
				}
			}()

			config, err := newConfig()
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, config)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, config)
			assert.Equal(t, tt.envVars["JIRA_BASE_URL"], config.GetBaseURL())
			assert.Equal(t, tt.envVars["JIRA_EMAIL"], config.GetEmail())
			assert.Equal(t, tt.envVars["JIRA_TOKEN"], config.GetToken())
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				BaseURL: "https://test.atlassian.net",
				Email:   "test@example.com",
				Token:   "test-token",
			},
			wantErr: false,
		},
		{
			name: "invalid base URL",
			config: &Config{
				BaseURL: "invalid-url",
				Email:   "test@example.com",
				Token:   "test-token",
			},
			wantErr: true,
		},
		{
			name: "empty email",
			config: &Config{
				BaseURL: "https://test.atlassian.net",
				Email:   "",
				Token:   "test-token",
			},
			wantErr: true,
		},
		{
			name: "empty token",
			config: &Config{
				BaseURL: "https://test.atlassian.net",
				Email:   "test@example.com",
				Token:   "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, IsConfigurationError(err))
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestConfig_ValidateBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid URL",
			config: &Config{
				BaseURL: "https://test.atlassian.net",
			},
			wantErr: false,
		},
		{
			name: "invalid URL",
			config: &Config{
				BaseURL: "invalid-url",
			},
			wantErr: true,
		},
		{
			name: "empty URL",
			config: &Config{
				BaseURL: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validateBaseURL()
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, IsConfigurationError(err))
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestConfig_ValidateCredentials(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid credentials",
			config: &Config{
				Email: "test@example.com",
				Token: "test-token",
			},
			wantErr: false,
		},
		{
			name: "empty email",
			config: &Config{
				Email: "",
				Token: "test-token",
			},
			wantErr: true,
		},
		{
			name: "empty token",
			config: &Config{
				Email: "test@example.com",
				Token: "",
			},
			wantErr: true,
		},
		{
			name: "both empty",
			config: &Config{
				Email: "",
				Token: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validateCredentials()
			if tt.wantErr {
				require.Error(t, err)
				assert.True(t, IsConfigurationError(err))
				return
			}
			require.NoError(t, err)
		})
	}
}
