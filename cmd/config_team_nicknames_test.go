package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configapp "github.com/helmedeiros/digital-asset-capitalization/internal/config/application"
	configdomain "github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

// teamResolverWith builds a real TeamResolverService backed by the
// supplied teams and nicknames maps. The two-layer construction
// (NewTeamConfigWithNicknames then NewTeamResolverService) mirrors
// what initializeApp does at startup.
func teamResolverWith(t *testing.T, teams map[string][]string, nicknames map[string][]string) *configapp.TeamResolverService {
	t.Helper()
	cfg, err := configdomain.NewTeamConfigWithNicknames(teams, nicknames)
	require.NoError(t, err)
	return configapp.NewTeamResolverService(cfg)
}

// configTeamNicknamesAddAction

func TestApp_configTeamNicknamesAddAction_RequiresProject(t *testing.T) {
	t.Parallel()
	a := &App{}
	ctx := newContextWithFlags(t, map[string]string{"nickname": "pricing"}, nil)
	err := a.configTeamNicknamesAddAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project is required")
}

func TestApp_configTeamNicknamesAddAction_RequiresNickname(t *testing.T) {
	t.Parallel()
	a := &App{}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	err := a.configTeamNicknamesAddAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nickname is required")
}

func TestApp_configTeamNicknamesAddAction_PrintsParsedNicknames(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{}
	ctx := newContextWithFlags(t, map[string]string{
		"project":  "FN",
		"nickname": " pricing , , fintech ",
	}, nil)
	out, err := captureStdout(t, func() error { return a.configTeamNicknamesAddAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Would add nickname(s) pricing, fintech for project FN")
	assert.Contains(t, out, "requires implementation")
}

// configTeamNicknamesListAction

func TestApp_configTeamNicknamesListAction_NoResolverErrors(t *testing.T) {
	t.Parallel()
	a := &App{}
	err := a.configTeamNicknamesListAction(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "team resolver not available")
}

func TestApp_configTeamNicknamesListAction_EmptyMappings(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{teamResolver: teamResolverWith(t,
		map[string][]string{"FN": {"alice"}},
		nil, // no nicknames configured
	)}
	out, err := captureStdout(t, func() error { return a.configTeamNicknamesListAction(nil) })
	require.NoError(t, err)
	assert.Contains(t, out, "No team nicknames configured")
}

func TestApp_configTeamNicknamesListAction_RendersGroupedByProject(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{teamResolver: teamResolverWith(t,
		map[string][]string{"FN": {"alice"}, "AD": {"bob"}},
		map[string][]string{"FN": {"pricing", "fintech"}, "AD": {"ads"}},
	)}
	out, err := captureStdout(t, func() error { return a.configTeamNicknamesListAction(nil) })
	require.NoError(t, err)
	assert.Contains(t, out, "Team Nicknames:")
	assert.Contains(t, out, "FN:")
	assert.Contains(t, out, "AD:")
	// Order of nicknames within a project comes from the map iteration,
	// which is unstable. Assert membership rather than full strings.
	assert.Regexp(t, "pricing|fintech", out)
	assert.Contains(t, out, "ads")
}

// configTeamNicknamesShowAction

func TestApp_configTeamNicknamesShowAction_RequiresProject(t *testing.T) {
	t.Parallel()
	a := &App{}
	ctx := newContextWithFlags(t, nil, nil)
	err := a.configTeamNicknamesShowAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project is required")
}

func TestApp_configTeamNicknamesShowAction_NoResolverErrors(t *testing.T) {
	t.Parallel()
	a := &App{}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	err := a.configTeamNicknamesShowAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "team resolver not available")
}

func TestApp_configTeamNicknamesShowAction_UnknownProject(t *testing.T) {
	t.Parallel()
	a := &App{teamResolver: teamResolverWith(t,
		map[string][]string{"FN": {"alice"}}, nil,
	)}
	ctx := newContextWithFlags(t, map[string]string{"project": "bogus"}, nil)
	err := a.configTeamNicknamesShowAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown project")
}

func TestApp_configTeamNicknamesShowAction_PrintsDisplayName(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{teamResolver: teamResolverWith(t,
		map[string][]string{"FN": {"alice"}},
		map[string][]string{"FN": {"pricing"}},
	)}
	// Passing a nickname; the action resolves it to the canonical project
	// and prints the display name.
	ctx := newContextWithFlags(t, map[string]string{"project": "pricing"}, nil)
	out, err := captureStdout(t, func() error { return a.configTeamNicknamesShowAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Project:")
	assert.Contains(t, out, "FN")
}
