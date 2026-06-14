package jira

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configservice "github.com/helmedeiros/digital-asset-capitalization/internal/config/application/service"
	configdomain "github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

// stubConfigRepoForJira is the minimum ConfigurationRepository surface
// the test needs to drive NewRepository's path through
// *service.ConfigService.GetJiraConfig. Other port methods panic so
// any future leak surfaces immediately.
type stubConfigRepoForJira struct {
	jiraConfig    *configdomain.JiraConfig
	jiraConfigErr error
}

func (r *stubConfigRepoForJira) LoadJiraConfig() (*configdomain.JiraConfig, error) {
	return r.jiraConfig, r.jiraConfigErr
}
func (r *stubConfigRepoForJira) SaveJiraConfig(*configdomain.JiraConfig) error {
	panic("SaveJiraConfig should not be called by these tests")
}
func (r *stubConfigRepoForJira) LoadTeamConfig() (*configdomain.TeamConfig, error) {
	panic("LoadTeamConfig should not be called by these tests")
}
func (r *stubConfigRepoForJira) SaveTeamConfig(*configdomain.TeamConfig) error {
	panic("SaveTeamConfig should not be called by these tests")
}
func (r *stubConfigRepoForJira) ConfigExists() (bool, error) { return r.jiraConfig != nil, nil }
func (r *stubConfigRepoForJira) InitializeConfigDirectory() error {
	panic("InitializeConfigDirectory should not be called by these tests")
}

// TestNewRepository_FromConfigService drives the *real*
// NewRepository(configService) constructor (the existing TestNewRepository
// in this package despite its name actually exercises
// NewRepositoryLegacy). The fakeable NewClient global from
// jira_task_repository_test.go lets us swap the client factory so the
// test never makes a real HTTP call.
func TestNewRepository_FromConfigService(t *testing.T) {
	originalNewClient := NewClient
	t.Cleanup(func() { NewClient = originalNewClient })

	t.Run("config service error wraps", func(t *testing.T) {
		repo := &stubConfigRepoForJira{jiraConfigErr: errors.New("disk gone")}
		svc := configservice.NewConfigService(repo)

		got, err := NewRepository(svc)
		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "failed to get Jira configuration")
	})

	t.Run("client factory error wraps", func(t *testing.T) {
		jiraConfig, err := configdomain.NewJiraConfig("https://jira.example.com", "user@example.com", "tok")
		require.NoError(t, err)
		repo := &stubConfigRepoForJira{jiraConfig: jiraConfig}
		svc := configservice.NewConfigService(repo)

		NewClient = func(_ *Config) (Client, error) {
			return nil, errors.New("client boom")
		}

		got, err := NewRepository(svc)
		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "failed to create Jira client")
	})

	t.Run("happy path returns a wired TaskRepository", func(t *testing.T) {
		jiraConfig, err := configdomain.NewJiraConfig("https://jira.example.com", "user@example.com", "tok")
		require.NoError(t, err)
		repo := &stubConfigRepoForJira{jiraConfig: jiraConfig}
		svc := configservice.NewConfigService(repo)

		NewClient = func(_ *Config) (Client, error) {
			return &mockClient{}, nil
		}

		got, err := NewRepository(svc)
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.NotNil(t, got.client)
	})
}
