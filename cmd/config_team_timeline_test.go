package main

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configdomain "github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

// fakeTeamConfigWithSave embeds stubTeamConfigService and adds
// recording for SaveTeamConfig + an injectable save error so the
// add/remove Actions can be tested end-to-end. Same embedding
// pattern as fakeTeamConfigWithExcluded.
type fakeTeamConfigWithSave struct {
	stubTeamConfigService

	savedConfig *configdomain.TeamConfig
	saveErr     error
}

func (s *fakeTeamConfigWithSave) SaveTeamConfig(cfg *configdomain.TeamConfig) error {
	s.savedConfig = cfg
	return s.saveErr
}

// team-timeline show

func TestApp_configTeamTimelineShowAction_RequiresProject(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &stubTeamConfigService{}}
	err := a.configTeamTimelineShowAction(newContextWithFlags(t, nil, nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project is required")
}

func TestApp_configTeamTimelineShowAction_LoadErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &stubTeamConfigService{teamConfigErr: errors.New("disk")}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	err := a.configTeamTimelineShowAction(ctx)
	require.Error(t, err)
}

func TestApp_configTeamTimelineShowAction_NoTimelineFallsBackToFlatTeam(t *testing.T) {
	// no t.Parallel: prints to stdout
	cfg, err := configdomain.NewTeamConfig(map[string][]string{"FN": {"alice", "bob"}})
	require.NoError(t, err)
	a := &App{teamConfigService: &stubTeamConfigService{teamConfig: cfg}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	out, err := captureStdout(t, func() error { return a.configTeamTimelineShowAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "no team timeline configured")
	assert.Contains(t, out, "Using flat team list")
	assert.Contains(t, out, "alice, bob")
}

func TestApp_configTeamTimelineShowAction_NoTimelineNoTeamSilent(t *testing.T) {
	// no t.Parallel: prints to stdout
	cfg, err := configdomain.NewTeamConfig(map[string][]string{})
	require.NoError(t, err)
	a := &App{teamConfigService: &stubTeamConfigService{teamConfig: cfg}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	out, err := captureStdout(t, func() error { return a.configTeamTimelineShowAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "no team timeline configured")
	// No "Current team:" line since GetTeam returns (_, false).
	assert.NotContains(t, out, "Current team:")
}

func TestApp_configTeamTimelineShowAction_RendersTimelineWithActiveAndDeparted(t *testing.T) {
	// no t.Parallel: prints to stdout
	cfg, err := configdomain.NewTeamConfig(map[string][]string{"FN": {"alice", "bob"}})
	require.NoError(t, err)
	joined := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, cfg.AddMemberWithDates("FN", "alice", joined))
	require.NoError(t, cfg.AddMemberWithDates("FN", "bob", joined))
	left := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, cfg.SetMemberLeft("FN", "bob", left))

	a := &App{teamConfigService: &stubTeamConfigService{teamConfig: cfg}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	out, err := captureStdout(t, func() error { return a.configTeamTimelineShowAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Team Timeline for 'FN':")
	assert.Contains(t, out, "alice")
	assert.Contains(t, out, "[active]")
	assert.Contains(t, out, "bob")
	assert.Contains(t, out, "[departed]")
	assert.Contains(t, out, "2026-06-01")
	assert.Contains(t, out, "Active members")
}

// team-timeline add

func TestApp_configTeamTimelineAddAction_RequiresAllFlags(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		flags map[string]string
	}{
		{"no project", map[string]string{"member": "alice", "joined": "2026-01-01"}},
		{"no member", map[string]string{"project": "FN", "joined": "2026-01-01"}},
		{"no joined", map[string]string{"project": "FN", "member": "alice"}},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			a := &App{teamConfigService: &fakeTeamConfigWithSave{}}
			err := a.configTeamTimelineAddAction(newContextWithFlags(t, c.flags, nil))
			require.Error(t, err)
			assert.Contains(t, err.Error(), "are required")
		})
	}
}

func TestApp_configTeamTimelineAddAction_InvalidDate(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &fakeTeamConfigWithSave{}}
	ctx := newContextWithFlags(t, map[string]string{
		"project": "FN", "member": "alice", "joined": "yesterday",
	}, nil)
	err := a.configTeamTimelineAddAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid date format")
}

func TestApp_configTeamTimelineAddAction_LoadErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &fakeTeamConfigWithSave{
		stubTeamConfigService: stubTeamConfigService{teamConfigErr: errors.New("disk")},
	}}
	ctx := newContextWithFlags(t, map[string]string{
		"project": "FN", "member": "alice", "joined": "2026-01-01",
	}, nil)
	err := a.configTeamTimelineAddAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load team config")
}

func TestApp_configTeamTimelineAddAction_SaveErrorWraps(t *testing.T) {
	t.Parallel()
	cfg, err := configdomain.NewTeamConfig(map[string][]string{"FN": {"alice"}})
	require.NoError(t, err)
	a := &App{teamConfigService: &fakeTeamConfigWithSave{
		stubTeamConfigService: stubTeamConfigService{teamConfig: cfg},
		saveErr:               errors.New("disk"),
	}}
	ctx := newContextWithFlags(t, map[string]string{
		"project": "FN", "member": "alice", "joined": "2026-01-01",
	}, nil)
	err = a.configTeamTimelineAddAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save team config")
}

func TestApp_configTeamTimelineAddAction_Success(t *testing.T) {
	// no t.Parallel: prints to stdout
	cfg, err := configdomain.NewTeamConfig(map[string][]string{"FN": {"alice"}})
	require.NoError(t, err)
	stub := &fakeTeamConfigWithSave{
		stubTeamConfigService: stubTeamConfigService{teamConfig: cfg},
	}
	a := &App{teamConfigService: stub}
	ctx := newContextWithFlags(t, map[string]string{
		"project": "FN", "member": "alice", "joined": "2026-01-01",
	}, nil)
	out, err := captureStdout(t, func() error { return a.configTeamTimelineAddAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Added 'alice' to project 'FN' timeline")
	assert.Same(t, cfg, stub.savedConfig, "SaveTeamConfig should be called with the loaded TeamConfig")
}

// team-timeline remove

func TestApp_configTeamTimelineRemoveAction_RequiresAllFlags(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &fakeTeamConfigWithSave{}}
	err := a.configTeamTimelineRemoveAction(newContextWithFlags(t, nil, nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "are required")
}

func TestApp_configTeamTimelineRemoveAction_InvalidDate(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &fakeTeamConfigWithSave{}}
	ctx := newContextWithFlags(t, map[string]string{
		"project": "FN", "member": "alice", "left": "yesterday",
	}, nil)
	err := a.configTeamTimelineRemoveAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid date format")
}

func TestApp_configTeamTimelineRemoveAction_LoadErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &fakeTeamConfigWithSave{
		stubTeamConfigService: stubTeamConfigService{teamConfigErr: errors.New("disk")},
	}}
	ctx := newContextWithFlags(t, map[string]string{
		"project": "FN", "member": "alice", "left": "2026-01-01",
	}, nil)
	err := a.configTeamTimelineRemoveAction(ctx)
	require.Error(t, err)
}

func TestApp_configTeamTimelineRemoveAction_MemberNotInTimeline(t *testing.T) {
	t.Parallel()
	cfg, err := configdomain.NewTeamConfig(map[string][]string{"FN": {"alice"}})
	require.NoError(t, err)
	// No AddMemberWithDates → SetMemberLeft returns an error.
	a := &App{teamConfigService: &fakeTeamConfigWithSave{
		stubTeamConfigService: stubTeamConfigService{teamConfig: cfg},
	}}
	ctx := newContextWithFlags(t, map[string]string{
		"project": "FN", "member": "alice", "left": "2026-01-01",
	}, nil)
	err = a.configTeamTimelineRemoveAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to set departure")
}

func TestApp_configTeamTimelineRemoveAction_Success(t *testing.T) {
	// no t.Parallel: prints to stdout
	cfg, err := configdomain.NewTeamConfig(map[string][]string{"FN": {"alice"}})
	require.NoError(t, err)
	require.NoError(t, cfg.AddMemberWithDates("FN", "alice",
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)))
	stub := &fakeTeamConfigWithSave{
		stubTeamConfigService: stubTeamConfigService{teamConfig: cfg},
	}
	a := &App{teamConfigService: stub}
	ctx := newContextWithFlags(t, map[string]string{
		"project": "FN", "member": "alice", "left": "2026-06-01",
	}, nil)
	out, err := captureStdout(t, func() error { return a.configTeamTimelineRemoveAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Set 'alice' departure from project 'FN'")
	assert.Same(t, cfg, stub.savedConfig)
}
