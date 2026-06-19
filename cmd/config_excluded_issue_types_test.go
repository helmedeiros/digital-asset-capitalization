package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configdomain "github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

// stubTeamConfigService is defined in config_team_tribe_company_test.go.
// Here we exercise the excluded-issue-types methods on it.

// excluded-issue-types set

func TestApp_configExcludedIssueTypesSetAction_RequiresProject(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &stubTeamConfigService{}}
	ctx := newContextWithFlags(t, map[string]string{"types": "Experiment"}, nil)
	err := a.configExcludedIssueTypesSetAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project is required")
}

func TestApp_configExcludedIssueTypesSetAction_RequiresTypes(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &stubTeamConfigService{}}
	ctx := newContextWithFlags(t, map[string]string{"project": "COP"}, nil)
	err := a.configExcludedIssueTypesSetAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "types is required")
}

func TestApp_configExcludedIssueTypesSetAction_ServiceErrorWraps(t *testing.T) {
	t.Parallel()
	stub := &fakeTeamConfigWithExcluded{setErr: errors.New("disk")}
	a := &App{teamConfigService: stub}
	ctx := newContextWithFlags(t, map[string]string{"project": "COP", "types": "Experiment"}, nil)
	err := a.configExcludedIssueTypesSetAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to set excluded issue types")
}

func TestApp_configExcludedIssueTypesSetAction_Success(t *testing.T) {
	// no t.Parallel: prints to stdout
	stub := &fakeTeamConfigWithExcluded{}
	a := &App{teamConfigService: stub}
	ctx := newContextWithFlags(t, map[string]string{
		"project": "COP",
		"types":   " Experiment , , Spike ",
	}, nil)
	out, err := captureStdout(t, func() error { return a.configExcludedIssueTypesSetAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Set excluded issue types for project 'COP': Experiment, Spike")
	assert.Equal(t, []string{"Experiment", "Spike"}, stub.gotTypes,
		"comma-separated list should be trimmed and empty entries dropped")
}

// excluded-issue-types clear

func TestApp_configExcludedIssueTypesClearAction_RequiresProject(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &fakeTeamConfigWithExcluded{}}
	ctx := newContextWithFlags(t, nil, nil)
	err := a.configExcludedIssueTypesClearAction(ctx)
	require.Error(t, err)
}

func TestApp_configExcludedIssueTypesClearAction_ServiceErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &fakeTeamConfigWithExcluded{setErr: errors.New("disk")}}
	ctx := newContextWithFlags(t, map[string]string{"project": "COP"}, nil)
	err := a.configExcludedIssueTypesClearAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to clear")
}

func TestApp_configExcludedIssueTypesClearAction_Success(t *testing.T) {
	// no t.Parallel: prints to stdout
	stub := &fakeTeamConfigWithExcluded{}
	a := &App{teamConfigService: stub}
	ctx := newContextWithFlags(t, map[string]string{"project": "COP"}, nil)
	out, err := captureStdout(t, func() error { return a.configExcludedIssueTypesClearAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Cleared excluded issue types for project 'COP'")
	assert.Nil(t, stub.gotTypes, "clear must pass nil through to the service")
}

// excluded-issue-types list

func TestApp_configExcludedIssueTypesListAction_LoadErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &stubTeamConfigService{teamConfigErr: errors.New("disk")}}
	err := a.configExcludedIssueTypesListAction(nil)
	require.Error(t, err)
}

func TestApp_configExcludedIssueTypesListAction_NoTeamsConfigured(t *testing.T) {
	// no t.Parallel: prints to stdout
	cfg, err := configdomain.NewTeamConfig(map[string][]string{})
	require.NoError(t, err)
	a := &App{teamConfigService: &stubTeamConfigService{teamConfig: cfg}}
	out, err := captureStdout(t, func() error { return a.configExcludedIssueTypesListAction(nil) })
	require.NoError(t, err)
	assert.Contains(t, out, "No teams configured")
}

func TestApp_configExcludedIssueTypesListAction_NoExclusionsConfigured(t *testing.T) {
	// no t.Parallel: prints to stdout
	cfg, err := configdomain.NewTeamConfig(map[string][]string{"FN": {"alice"}, "AD": {"bob"}})
	require.NoError(t, err)
	a := &App{teamConfigService: &stubTeamConfigService{teamConfig: cfg}}
	out, err := captureStdout(t, func() error { return a.configExcludedIssueTypesListAction(nil) })
	require.NoError(t, err)
	assert.Contains(t, out, "Excluded Issue Types:")
	assert.Contains(t, out, "No excluded issue types configured for any project")
}

func TestApp_configExcludedIssueTypesListAction_RendersProjectsWithExclusions(t *testing.T) {
	// no t.Parallel: prints to stdout
	cfg, err := configdomain.NewTeamConfig(map[string][]string{"COP": {"alice"}, "FN": {"bob"}})
	require.NoError(t, err)
	require.NoError(t, cfg.SetExcludedIssueTypes("COP", []string{"Experiment", "Spike"}))
	a := &App{teamConfigService: &stubTeamConfigService{teamConfig: cfg}}
	out, err := captureStdout(t, func() error { return a.configExcludedIssueTypesListAction(nil) })
	require.NoError(t, err)
	assert.Contains(t, out, "COP: Experiment, Spike")
	assert.NotContains(t, out, "No excluded issue types configured for any project")
}

// excluded-issue-types show

func TestApp_configExcludedIssueTypesShowAction_RequiresProject(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &fakeTeamConfigWithExcluded{}}
	err := a.configExcludedIssueTypesShowAction(newContextWithFlags(t, nil, nil))
	require.Error(t, err)
}

func TestApp_configExcludedIssueTypesShowAction_ServiceErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &fakeTeamConfigWithExcluded{getErr: errors.New("disk")}}
	ctx := newContextWithFlags(t, map[string]string{"project": "COP"}, nil)
	err := a.configExcludedIssueTypesShowAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get excluded issue types")
}

func TestApp_configExcludedIssueTypesShowAction_None(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{teamConfigService: &fakeTeamConfigWithExcluded{}}
	ctx := newContextWithFlags(t, map[string]string{"project": "COP"}, nil)
	out, err := captureStdout(t, func() error { return a.configExcludedIssueTypesShowAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Project 'COP' has no excluded issue types")
}

func TestApp_configExcludedIssueTypesShowAction_RendersList(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{teamConfigService: &fakeTeamConfigWithExcluded{
		types: []string{"Experiment", "Spike"},
	}}
	ctx := newContextWithFlags(t, map[string]string{"project": "COP"}, nil)
	out, err := captureStdout(t, func() error { return a.configExcludedIssueTypesShowAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Project 'COP' excludes: Experiment, Spike")
}

// fakeTeamConfigWithExcluded specialises stubTeamConfigService for the
// set/clear/show paths so tests can also assert what arguments the
// service received.
type fakeTeamConfigWithExcluded struct {
	stubTeamConfigService

	gotTypes []string
	setErr   error
	types    []string
	getErr   error
}

func (s *fakeTeamConfigWithExcluded) SetExcludedIssueTypesForProject(_ string, types []string) error {
	s.gotTypes = types
	return s.setErr
}
func (s *fakeTeamConfigWithExcluded) GetExcludedIssueTypesForProject(string) ([]string, error) {
	return s.types, s.getErr
}
