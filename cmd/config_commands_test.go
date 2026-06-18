package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/application/usecase"
	configdomain "github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

// stubConfigService implements the ConfigService interface used by
// configInitAction. GetJiraConfig isn't called by the extracted
// Actions in this PR, so it panics to flag accidental coupling.
type stubConfigService struct {
	initResult *usecase.InitializeConfigResult
	initErr    error
}

func (s *stubConfigService) InitializeConfig(bool) (*usecase.InitializeConfigResult, error) {
	return s.initResult, s.initErr
}

func (s *stubConfigService) GetJiraConfig() (*configdomain.JiraConfig, error) {
	panic("not used")
}

func TestApp_configInitAction_NoServiceReturnsError(t *testing.T) {
	t.Parallel()
	a := &App{}
	ctx := newContextWithFlags(t, nil, nil)
	err := a.configInitAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "configuration service not available")
}

func TestApp_configInitAction_ServiceErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{configService: &stubConfigService{initErr: errors.New("boom")}}
	ctx := newContextWithFlags(t, nil, map[string]bool{"non-interactive": true})
	err := a.configInitAction(ctx)
	require.Error(t, err)
	assert.Equal(t, "boom", err.Error())
}

func TestApp_configInitAction_PrintsResultMessage(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{configService: &stubConfigService{
		initResult: &usecase.InitializeConfigResult{Message: "configured"},
	}}
	ctx := newContextWithFlags(t, nil, map[string]bool{"non-interactive": true})
	out, err := captureStdout(t, func() error { return a.configInitAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "configured")
}

func TestApp_configShowAction_PrintsMaskedToken(t *testing.T) {
	// Uses t.Setenv so the env state is restored after the test.
	t.Setenv("JIRA_BASE_URL", "https://example.invalid")
	t.Setenv("JIRA_EMAIL", "user@example.invalid")
	t.Setenv("JIRA_TOKEN", "tok-1234567890")
	a := &App{}
	out, err := captureStdout(t, func() error { return a.configShowAction(nil) })
	require.NoError(t, err)
	assert.Contains(t, out, "JIRA_BASE_URL: https://example.invalid")
	assert.Contains(t, out, "user@example.invalid")
	// The token must not appear in full.
	assert.NotContains(t, out, "tok-1234567890")
}

func TestApp_configShowAction_UnsetTokenPrintsPlaceholder(t *testing.T) {
	t.Setenv("JIRA_BASE_URL", "")
	t.Setenv("JIRA_EMAIL", "")
	t.Setenv("JIRA_TOKEN", "")
	a := &App{}
	out, err := captureStdout(t, func() error { return a.configShowAction(nil) })
	require.NoError(t, err)
	assert.Contains(t, out, "<not set>")
}

func TestApp_configValidateAction_ReturnsErrorWhenEnvMissing(t *testing.T) {
	t.Setenv("JIRA_BASE_URL", "")
	t.Setenv("JIRA_EMAIL", "")
	t.Setenv("JIRA_TOKEN", "")
	a := &App{}
	_, err := captureStdout(t, func() error { return a.configValidateAction(nil) })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "configuration validation failed")
}

func TestApp_configValidateAction_SucceedsWhenEverythingPresent(t *testing.T) {
	// Set env vars.
	t.Setenv("JIRA_BASE_URL", "https://example.invalid")
	t.Setenv("JIRA_EMAIL", "user@example.invalid")
	t.Setenv("JIRA_TOKEN", "tok")

	// Stand up a temp dir for the teams file. The validate action
	// reads from configDir/teamsFile (package constants); chdir into
	// TempDir and create .assetcap/teams.json so the relative-path
	// lookup succeeds.
	t.Chdir(t.TempDir())
	require.NoError(t, os.MkdirAll(configDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, teamsFile), []byte("{}"), 0o644))

	a := &App{}
	out, err := captureStdout(t, func() error { return a.configValidateAction(nil) })
	require.NoError(t, err)
	assert.Contains(t, out, "Configuration is valid")
}
