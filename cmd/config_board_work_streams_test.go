package main

import (
	"errors"
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	configdomain "github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

// fakeTeamConfigWithBoards extends stubTeamConfigService with a
// recording SetBoardWorkStream and a custom error injection point.
// Same embedding pattern as fakeTeamConfigWithExcluded.
type fakeTeamConfigWithBoards struct {
	stubTeamConfigService

	gotProject    string
	gotBoardID    int
	gotWorkStream string
	setErr        error
}

func (s *fakeTeamConfigWithBoards) SetBoardWorkStream(project string, boardID int, ws string) error {
	s.gotProject = project
	s.gotBoardID = boardID
	s.gotWorkStream = ws
	return s.setErr
}

// board-work-streams set

func TestApp_configBoardWorkStreamsSetAction_RequiresProject(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &fakeTeamConfigWithBoards{}}
	ctx := newContextWithFlags(t,
		map[string]string{"work-stream": "Product"},
		nil,
	)
	err := a.configBoardWorkStreamsSetAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project is required")
}

func TestApp_configBoardWorkStreamsSetAction_RequiresBoardID(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &fakeTeamConfigWithBoards{}}
	// boardID defaults to 0 when unset.
	ctx := newContextWithFlags(t,
		map[string]string{"project": "COP", "work-stream": "Product"},
		nil,
	)
	err := a.configBoardWorkStreamsSetAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "board ID is required")
}

func TestApp_configBoardWorkStreamsSetAction_RequiresWorkStream(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &fakeTeamConfigWithBoards{}}
	ctx := newContextWithIntFlag(t, map[string]string{"project": "COP"}, "board", 5119)
	err := a.configBoardWorkStreamsSetAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "work-stream is required")
}

func TestApp_configBoardWorkStreamsSetAction_ServiceErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &fakeTeamConfigWithBoards{setErr: errors.New("disk")}}
	ctx := newContextWithIntAndStrings(t,
		map[string]string{"project": "COP", "work-stream": "Product"},
		"board", 5119,
	)
	err := a.configBoardWorkStreamsSetAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to set board work stream")
}

func TestApp_configBoardWorkStreamsSetAction_Success(t *testing.T) {
	// no t.Parallel: prints to stdout
	stub := &fakeTeamConfigWithBoards{}
	a := &App{teamConfigService: stub}
	ctx := newContextWithIntAndStrings(t,
		map[string]string{"project": "COP", "work-stream": "Product"},
		"board", 5119,
	)
	out, err := captureStdout(t, func() error { return a.configBoardWorkStreamsSetAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Set board 5119 -> 'Product' for project 'COP'")
	assert.Equal(t, "COP", stub.gotProject)
	assert.Equal(t, 5119, stub.gotBoardID)
	assert.Equal(t, "Product", stub.gotWorkStream)
}

// board-work-streams show

func TestApp_configBoardWorkStreamsShowAction_RequiresProject(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &stubTeamConfigService{}}
	err := a.configBoardWorkStreamsShowAction(newContextWithFlags(t, nil, nil))
	require.Error(t, err)
}

func TestApp_configBoardWorkStreamsShowAction_LoadErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &stubTeamConfigService{teamConfigErr: errors.New("disk")}}
	ctx := newContextWithFlags(t, map[string]string{"project": "COP"}, nil)
	err := a.configBoardWorkStreamsShowAction(ctx)
	require.Error(t, err)
}

func TestApp_configBoardWorkStreamsShowAction_NoMappings(t *testing.T) {
	// no t.Parallel: prints to stdout
	cfg, err := configdomain.NewTeamConfig(map[string][]string{"COP": {"alice"}})
	require.NoError(t, err)
	a := &App{teamConfigService: &stubTeamConfigService{teamConfig: cfg}}
	ctx := newContextWithFlags(t, map[string]string{"project": "COP"}, nil)
	out, err := captureStdout(t, func() error { return a.configBoardWorkStreamsShowAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Project 'COP' has no board-to-workstream mappings")
}

func TestApp_configBoardWorkStreamsShowAction_RendersMappings(t *testing.T) {
	// no t.Parallel: prints to stdout
	cfg, err := configdomain.NewTeamConfig(map[string][]string{"COP": {"alice"}})
	require.NoError(t, err)
	require.NoError(t, cfg.SetBoardWorkStreams("COP", map[int]string{5119: "Product"}))

	a := &App{teamConfigService: &stubTeamConfigService{teamConfig: cfg}}
	ctx := newContextWithFlags(t, map[string]string{"project": "COP"}, nil)
	out, err := captureStdout(t, func() error { return a.configBoardWorkStreamsShowAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Board Work Streams for 'COP':")
	assert.Contains(t, out, "Board 5119 -> Product")
}

// board-work-streams list

func TestApp_configBoardWorkStreamsListAction_LoadErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &stubTeamConfigService{teamConfigErr: errors.New("disk")}}
	err := a.configBoardWorkStreamsListAction(nil)
	require.Error(t, err)
}

func TestApp_configBoardWorkStreamsListAction_NoTeams(t *testing.T) {
	// no t.Parallel: prints to stdout
	cfg, err := configdomain.NewTeamConfig(map[string][]string{})
	require.NoError(t, err)
	a := &App{teamConfigService: &stubTeamConfigService{teamConfig: cfg}}
	out, err := captureStdout(t, func() error { return a.configBoardWorkStreamsListAction(nil) })
	require.NoError(t, err)
	assert.Contains(t, out, "No teams configured")
}

func TestApp_configBoardWorkStreamsListAction_NoMappingsAcrossProjects(t *testing.T) {
	// no t.Parallel: prints to stdout
	cfg, err := configdomain.NewTeamConfig(map[string][]string{"COP": {"alice"}, "FN": {"bob"}})
	require.NoError(t, err)
	a := &App{teamConfigService: &stubTeamConfigService{teamConfig: cfg}}
	out, err := captureStdout(t, func() error { return a.configBoardWorkStreamsListAction(nil) })
	require.NoError(t, err)
	assert.Contains(t, out, "Board Work Streams:")
	assert.Contains(t, out, "No board-to-workstream mappings configured")
}

func TestApp_configBoardWorkStreamsListAction_RendersProjectsWithMappings(t *testing.T) {
	// no t.Parallel: prints to stdout
	cfg, err := configdomain.NewTeamConfig(map[string][]string{"COP": {"alice"}, "FN": {"bob"}})
	require.NoError(t, err)
	require.NoError(t, cfg.SetBoardWorkStreams("COP", map[int]string{5119: "Product"}))

	a := &App{teamConfigService: &stubTeamConfigService{teamConfig: cfg}}
	out, err := captureStdout(t, func() error { return a.configBoardWorkStreamsListAction(nil) })
	require.NoError(t, err)
	assert.Contains(t, out, "COP:")
	assert.Contains(t, out, "Board 5119 -> Product")
}

// newContextWithIntFlag is a small helper for the int-only flag tests.
// urfave/cli requires Int flags to be registered separately from String
// flags on the FlagSet.
func newContextWithIntFlag(t *testing.T, strFlags map[string]string, intName string, intVal int) *cli.Context {
	t.Helper()
	return newContextWithIntAndStrings(t, strFlags, intName, intVal)
}

// newContextWithIntAndStrings is the workhorse for the int-flag tests.
// Same pattern as newContextWithFlags but with one additional int flag.
func newContextWithIntAndStrings(t *testing.T, strFlags map[string]string, intName string, intVal int) *cli.Context {
	t.Helper()
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	for k, v := range strFlags {
		set.String(k, v, "")
	}
	set.Int(intName, intVal, "")
	return cli.NewContext(nil, set, nil)
}
