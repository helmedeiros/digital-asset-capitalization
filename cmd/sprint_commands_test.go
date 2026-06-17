package main

import (
	"bytes"
	"errors"
	"flag"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	configapp "github.com/helmedeiros/digital-asset-capitalization/internal/config/application"
	configdomain "github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
	sprintusecase "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/application/usecase"
	sprintdomain "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	sprintports "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
)

// stubSprintService is a hand-rolled minimum SprintService so the
// Action tests can drive each branch without the JIRA infrastructure.
// Only the methods the sprint Actions actually call are exercised;
// others panic so a future regression that calls something unexpected
// is immediately obvious in tests.
type stubSprintService struct {
	listResult    *sprintusecase.ListSprintsResult
	listErr       error
	processResult string
	processErr    error
	optsResult    string
	optsErr       error
	pushCSV       string
	pushResult    *sprintusecase.PushResult
	pushErr       error
}

func (s *stubSprintService) ListSprints(string, string) (*sprintusecase.ListSprintsResult, error) {
	return s.listResult, s.listErr
}
func (s *stubSprintService) ProcessJiraIssues(string, string, string) (string, error) {
	return s.processResult, s.processErr
}
func (s *stubSprintService) ProcessJiraIssuesWithOptions(string, string, string, bool, ...sprintusecase.SprintAllocationOption) (string, error) {
	return s.optsResult, s.optsErr
}
func (s *stubSprintService) PushAllocationToJira(string, string, string, bool, bool, ...sprintusecase.SprintAllocationOption) (string, *sprintusecase.PushResult, error) {
	return s.pushCSV, s.pushResult, s.pushErr
}
func (s *stubSprintService) ProcessSprint(string, *sprintdomain.Sprint) error { panic("not used") }
func (s *stubSprintService) ProcessTeamIssues(*sprintdomain.Team) error       { panic("not used") }
func (s *stubSprintService) ProcessJiraIssuesWithStrategy(string, string, string, bool) (string, error) {
	panic("not used")
}

// teamResolverFor returns a real TeamResolverService backed by a tiny
// nicknames map so the team-identifier resolution branch can be
// exercised without mocking. "voyager" -> "FN" and that's it.
func teamResolverFor(t *testing.T) *configapp.TeamResolverService {
	t.Helper()
	cfg, err := configdomain.NewTeamConfigWithNicknames(
		map[string][]string{"FN": {"alice"}},
		map[string][]string{"FN": {"voyager"}},
	)
	require.NoError(t, err)
	return configapp.NewTeamResolverService(cfg)
}

// newContextWithFlags builds a *cli.Context whose String/Bool lookups
// return the supplied values. Used to drive Action methods directly
// without the full cli.App boot.
func newContextWithFlags(t *testing.T, strFlags map[string]string, boolFlags map[string]bool) *cli.Context {
	t.Helper()
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	for k, v := range strFlags {
		set.String(k, v, "")
	}
	for k, v := range boolFlags {
		set.Bool(k, v, "")
	}
	return cli.NewContext(nil, set, nil)
}

// captureStdout swaps os.Stdout for the duration of fn and returns
// whatever was written. The sprint Actions print results via fmt.Print
// directly to stdout, so the test needs to redirect.
func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	actionErr := fn()
	require.NoError(t, w.Close())
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String(), actionErr
}

func TestApp_sprintListAction_DelegatesAndPrints(t *testing.T) {
	// no t.Parallel: this test swaps os.Stdout, which is global.
	stub := &stubSprintService{
		listResult: &sprintusecase.ListSprintsResult{
			Project: "FN",
			Period:  "Q1 2026",
			Sprints: []sprintports.Sprint{{Name: "Penguins"}},
		},
	}
	a := &App{sprintService: stub}
	ctx := newContextWithFlags(t,
		map[string]string{"project": "FN", "period": "Q1 2026"},
		nil,
	)
	out, err := captureStdout(t, func() error { return a.sprintListAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Penguins", "formatter should render the stub sprint")
}

func TestApp_sprintListAction_ResolverErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{
		sprintService: &stubSprintService{},
		teamResolver:  teamResolverFor(t),
	}
	ctx := newContextWithFlags(t,
		map[string]string{"project": "no-such-nickname", "period": "Q1"},
		nil,
	)
	err := a.sprintListAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown project or team nickname")
}

func TestApp_sprintListAction_ResolverHitFlowsResolvedProject(t *testing.T) {
	// serial: swaps os.Stdout
	stub := &stubSprintService{
		listResult: &sprintusecase.ListSprintsResult{Sprints: nil},
	}
	a := &App{sprintService: stub, teamResolver: teamResolverFor(t)}
	ctx := newContextWithFlags(t,
		map[string]string{"project": "voyager", "period": "Q1"},
		nil,
	)
	_, err := captureStdout(t, func() error { return a.sprintListAction(ctx) })
	require.NoError(t, err)
}

func TestApp_sprintListAction_ServiceErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{sprintService: &stubSprintService{listErr: errors.New("jira down")}}
	ctx := newContextWithFlags(t,
		map[string]string{"project": "FN", "period": "Q1"},
		nil,
	)
	err := a.sprintListAction(ctx)
	require.Error(t, err)
	assert.Equal(t, "jira down", err.Error())
}

func TestApp_sprintAllocateAction_PlainCSV(t *testing.T) {
	// serial: swaps os.Stdout
	stub := &stubSprintService{processResult: "header\nrow1\n"}
	a := &App{sprintService: stub}
	ctx := newContextWithFlags(t,
		map[string]string{"project": "FN", "sprint": "Penguins"},
		// sprint-bounded defaults to false here so the plain branch runs.
		map[string]bool{"sprint-bounded": false},
	)
	out, err := captureStdout(t, func() error { return a.sprintAllocateAction(ctx) })
	require.NoError(t, err)
	assert.Equal(t, "header\nrow1\n", out)
}

func TestApp_sprintAllocateAction_SprintBoundedPath(t *testing.T) {
	// serial: swaps os.Stdout
	stub := &stubSprintService{optsResult: "opts-csv"}
	a := &App{sprintService: stub}
	ctx := newContextWithFlags(t,
		map[string]string{"project": "FN", "sprint": "Penguins"},
		map[string]bool{"sprint-bounded": true},
	)
	out, err := captureStdout(t, func() error { return a.sprintAllocateAction(ctx) })
	require.NoError(t, err)
	assert.Equal(t, "opts-csv", out)
}

func TestApp_sprintAllocateAction_WorkStreamsParsed(t *testing.T) {
	// serial: swaps os.Stdout
	stub := &stubSprintService{optsResult: "csv"}
	a := &App{sprintService: stub}
	ctx := newContextWithFlags(t,
		map[string]string{
			"project":      "FN",
			"sprint":       "Penguins",
			"work-streams": " product, , operational ", // exercises the trim+drop branch
		},
		nil,
	)
	_, err := captureStdout(t, func() error { return a.sprintAllocateAction(ctx) })
	require.NoError(t, err)
}

func TestApp_sprintAllocateAction_ResolverErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{sprintService: &stubSprintService{}, teamResolver: teamResolverFor(t)}
	ctx := newContextWithFlags(t,
		map[string]string{"project": "bogus", "sprint": "Penguins"},
		nil,
	)
	err := a.sprintAllocateAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown project or team nickname")
}

func TestApp_sprintAllocateAction_ServiceErrorBubbles(t *testing.T) {
	t.Parallel()
	stub := &stubSprintService{processErr: errors.New("compute failed")}
	a := &App{sprintService: stub}
	ctx := newContextWithFlags(t,
		map[string]string{"project": "FN", "sprint": "P"},
		map[string]bool{"sprint-bounded": false},
	)
	err := a.sprintAllocateAction(ctx)
	require.Error(t, err)
	assert.Equal(t, "compute failed", err.Error())
}

func TestParseCommaSeparated(t *testing.T) {
	t.Parallel()
	t.Run("empty input returns nil", func(t *testing.T) {
		assert.Nil(t, parseCommaSeparated(""))
	})
	t.Run("trims and drops empty elements", func(t *testing.T) {
		assert.Equal(t, []string{"a", "b"}, parseCommaSeparated(" a , , b "))
	})
	t.Run("preserves order", func(t *testing.T) {
		assert.Equal(t, []string{"x", "y", "z"}, parseCommaSeparated("x,y,z"))
	})
}

// suppress unused imports if a future trim accidentally drops one.
var _ = strings.TrimSpace
