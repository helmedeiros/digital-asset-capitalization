package infrastructure

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEnvironmentProvider(t *testing.T) {
	provider := NewEnvironmentProvider()
	assert.NotNil(t, provider)
}

func TestEnvironmentProvider_GetJiraConfig(t *testing.T) {
	provider := NewEnvironmentProvider()

	t.Run("should return empty values when environment variables are not set", func(t *testing.T) {
		// Clean environment
		cleanup := setupCleanEnvironment(t)
		defer cleanup()

		assert.Empty(t, provider.GetJiraBaseURL())
		assert.Empty(t, provider.GetJiraEmail())
		assert.Empty(t, provider.GetJiraToken())
	})

	t.Run("should return values when environment variables are set", func(t *testing.T) {
		cleanup := setupCleanEnvironment(t)
		defer cleanup()

		// Set environment variables
		os.Setenv("JIRA_BASE_URL", "https://test.atlassian.net")
		os.Setenv("JIRA_EMAIL", "test@example.com")
		os.Setenv("JIRA_TOKEN", "test-token")

		assert.Equal(t, "https://test.atlassian.net", provider.GetJiraBaseURL())
		assert.Equal(t, "test@example.com", provider.GetJiraEmail())
		assert.Equal(t, "test-token", provider.GetJiraToken())
	})
}

func TestEnvironmentProvider_SetJiraConfig(t *testing.T) {
	provider := NewEnvironmentProvider()

	t.Run("should set environment variables successfully", func(t *testing.T) {
		cleanup := setupCleanEnvironment(t)
		defer cleanup()

		err := provider.SetJiraBaseURL("https://company.atlassian.net")
		require.NoError(t, err)
		assert.Equal(t, "https://company.atlassian.net", os.Getenv("JIRA_BASE_URL"))

		err = provider.SetJiraEmail("user@company.com")
		require.NoError(t, err)
		assert.Equal(t, "user@company.com", os.Getenv("JIRA_EMAIL"))

		err = provider.SetJiraToken("user-token")
		require.NoError(t, err)
		assert.Equal(t, "user-token", os.Getenv("JIRA_TOKEN"))
	})
}

func TestEnvironmentProvider_IsConfigured(t *testing.T) {
	provider := NewEnvironmentProvider()

	t.Run("should return false when no environment variables are set", func(t *testing.T) {
		cleanup := setupCleanEnvironment(t)
		defer cleanup()

		assert.False(t, provider.IsConfigured())
	})

	t.Run("should return false when only some environment variables are set", func(t *testing.T) {
		cleanup := setupCleanEnvironment(t)
		defer cleanup()

		os.Setenv("JIRA_BASE_URL", "https://test.atlassian.net")
		assert.False(t, provider.IsConfigured())

		os.Setenv("JIRA_EMAIL", "test@example.com")
		assert.False(t, provider.IsConfigured())
	})

	t.Run("should return true when all environment variables are set", func(t *testing.T) {
		cleanup := setupCleanEnvironment(t)
		defer cleanup()

		os.Setenv("JIRA_BASE_URL", "https://test.atlassian.net")
		os.Setenv("JIRA_EMAIL", "test@example.com")
		os.Setenv("JIRA_TOKEN", "test-token")

		assert.True(t, provider.IsConfigured())
	})
}

func TestEnvironmentProvider_GetMissingVars(t *testing.T) {
	provider := NewEnvironmentProvider()

	t.Run("should return all variables when none are set", func(t *testing.T) {
		cleanup := setupCleanEnvironment(t)
		defer cleanup()

		missing := provider.GetMissingVars()
		assert.Contains(t, missing, "JIRA_BASE_URL")
		assert.Contains(t, missing, "JIRA_EMAIL")
		assert.Contains(t, missing, "JIRA_TOKEN")
		assert.Len(t, missing, 3)
	})

	t.Run("should return only missing variables", func(t *testing.T) {
		cleanup := setupCleanEnvironment(t)
		defer cleanup()

		os.Setenv("JIRA_BASE_URL", "https://test.atlassian.net")
		os.Setenv("JIRA_EMAIL", "test@example.com")

		missing := provider.GetMissingVars()
		assert.Contains(t, missing, "JIRA_TOKEN")
		assert.Len(t, missing, 1)
	})

	t.Run("should return empty slice when all variables are set", func(t *testing.T) {
		cleanup := setupCleanEnvironment(t)
		defer cleanup()

		os.Setenv("JIRA_BASE_URL", "https://test.atlassian.net")
		os.Setenv("JIRA_EMAIL", "test@example.com")
		os.Setenv("JIRA_TOKEN", "test-token")

		missing := provider.GetMissingVars()
		assert.Empty(t, missing)
	})
}

// setupCleanEnvironment cleans up environment variables and returns a cleanup function
func setupCleanEnvironment(t *testing.T) func() {
	t.Helper()

	// Save original values
	origBaseURL := os.Getenv("JIRA_BASE_URL")
	origEmail := os.Getenv("JIRA_EMAIL")
	origToken := os.Getenv("JIRA_TOKEN")

	// Clear environment variables
	os.Unsetenv("JIRA_BASE_URL")
	os.Unsetenv("JIRA_EMAIL")
	os.Unsetenv("JIRA_TOKEN")

	return func() {
		// Restore original values
		if origBaseURL != "" {
			os.Setenv("JIRA_BASE_URL", origBaseURL)
		} else {
			os.Unsetenv("JIRA_BASE_URL")
		}
		if origEmail != "" {
			os.Setenv("JIRA_EMAIL", origEmail)
		} else {
			os.Unsetenv("JIRA_EMAIL")
		}
		if origToken != "" {
			os.Setenv("JIRA_TOKEN", origToken)
		} else {
			os.Unsetenv("JIRA_TOKEN")
		}
	}
}
