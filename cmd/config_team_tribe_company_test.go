package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configdomain "github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

// stubTeamConfigService implements the TeamConfigService interface
// with canned values per method. Methods not used by the Actions
// under test fall through to canned defaults rather than panicking
// since the interface is intentionally small.
type stubTeamConfigService struct {
	teamConfig    *configdomain.TeamConfig
	teamConfigErr error

	setTribeErr error
	tribe       string
	tribeErr    error

	setCompanyErr error
	company       string
	companyErr    error
}

func (s *stubTeamConfigService) GetTeamConfig() (*configdomain.TeamConfig, error) {
	return s.teamConfig, s.teamConfigErr
}
func (s *stubTeamConfigService) SaveTeamConfig(*configdomain.TeamConfig) error { return nil }
func (s *stubTeamConfigService) SetTribeForProject(string, string) error       { return s.setTribeErr }
func (s *stubTeamConfigService) GetTribeForProject(string) (string, error) {
	return s.tribe, s.tribeErr
}
func (s *stubTeamConfigService) SetCompanyForProject(string, string) error {
	return s.setCompanyErr
}
func (s *stubTeamConfigService) GetCompanyForProject(string) (string, error) {
	return s.company, s.companyErr
}
func (s *stubTeamConfigService) SetBoardWorkStream(string, int, string) error {
	return nil
}

// SetExcludedIssueTypesForProject and GetExcludedIssueTypesForProject
// satisfy the TeamConfigService interface (their dedicated tests use
// the fakeTeamConfigWithExcluded embedder in
// config_excluded_issue_types_test.go). The base stub returns zero
// values so it stays usable by every other Action test without
// pulling in fields they don't care about.
func (s *stubTeamConfigService) SetExcludedIssueTypesForProject(string, []string) error {
	return nil
}
func (s *stubTeamConfigService) GetExcludedIssueTypesForProject(string) ([]string, error) {
	return nil, nil
}

// teamConfigWith creates a real *TeamConfig with the supplied per-
// project tribe and company assignments. Empty maps are fine; the
// resulting config has projects but no annotations.
func teamConfigWith(t *testing.T, tribes map[string]string, companies map[string]string) *configdomain.TeamConfig {
	t.Helper()
	teams := map[string][]string{}
	for proj := range tribes {
		teams[proj] = []string{"alice"}
	}
	for proj := range companies {
		if _, ok := teams[proj]; !ok {
			teams[proj] = []string{"alice"}
		}
	}
	cfg, err := configdomain.NewTeamConfig(teams)
	require.NoError(t, err)
	for proj, tribe := range tribes {
		require.NoError(t, cfg.SetTribe(proj, tribe))
	}
	for proj, company := range companies {
		require.NoError(t, cfg.SetCompany(proj, company))
	}
	return cfg
}

// team-tribe set

func TestApp_configTeamTribeSetAction_RequiresProject(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &stubTeamConfigService{}}
	ctx := newContextWithFlags(t, map[string]string{"tribe": "Eng"}, nil)
	err := a.configTeamTribeSetAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project is required")
}

func TestApp_configTeamTribeSetAction_RequiresTribe(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &stubTeamConfigService{}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	err := a.configTeamTribeSetAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tribe is required")
}

func TestApp_configTeamTribeSetAction_ServiceErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &stubTeamConfigService{setTribeErr: errors.New("disk")}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN", "tribe": "Eng"}, nil)
	err := a.configTeamTribeSetAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to set tribe")
}

func TestApp_configTeamTribeSetAction_Success(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{teamConfigService: &stubTeamConfigService{}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN", "tribe": "Eng"}, nil)
	out, err := captureStdout(t, func() error { return a.configTeamTribeSetAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Set tribe 'Eng' for project 'FN'")
}

// team-tribe list

func TestApp_configTeamTribeListAction_LoadErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &stubTeamConfigService{teamConfigErr: errors.New("disk")}}
	err := a.configTeamTribeListAction(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load team config")
}

func TestApp_configTeamTribeListAction_EmptyProjects(t *testing.T) {
	// no t.Parallel: prints to stdout
	cfg, err := configdomain.NewTeamConfig(map[string][]string{})
	require.NoError(t, err)
	a := &App{teamConfigService: &stubTeamConfigService{teamConfig: cfg}}
	out, err := captureStdout(t, func() error { return a.configTeamTribeListAction(nil) })
	require.NoError(t, err)
	assert.Contains(t, out, "No teams configured")
}

func TestApp_configTeamTribeListAction_RendersTribeGroupsAndOrphans(t *testing.T) {
	// no t.Parallel: prints to stdout
	cfg := teamConfigWith(t,
		map[string]string{"FN": "Engineering", "PAY": "Engineering", "AD": "Marketing"},
		nil,
	)
	// Add a fourth project without a tribe by extending team map.
	require.NoError(t, cfg.SetTeam("INF", []string{"infra-alice"}))

	a := &App{teamConfigService: &stubTeamConfigService{teamConfig: cfg}}
	out, err := captureStdout(t, func() error { return a.configTeamTribeListAction(nil) })
	require.NoError(t, err)
	assert.Contains(t, out, "Team Tribes:")
	assert.Contains(t, out, "Engineering")
	assert.Contains(t, out, "Marketing")
	assert.Contains(t, out, "(No tribe assigned)")
	assert.Contains(t, out, "INF")
}

// team-tribe show

func TestApp_configTeamTribeShowAction_RequiresProject(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &stubTeamConfigService{}}
	err := a.configTeamTribeShowAction(newContextWithFlags(t, nil, nil))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project is required")
}

func TestApp_configTeamTribeShowAction_NoTribePrintsBlank(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{teamConfigService: &stubTeamConfigService{tribe: ""}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	out, err := captureStdout(t, func() error { return a.configTeamTribeShowAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Project 'FN' has no tribe assigned")
}

func TestApp_configTeamTribeShowAction_WithTribe(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{teamConfigService: &stubTeamConfigService{tribe: "Engineering"}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	out, err := captureStdout(t, func() error { return a.configTeamTribeShowAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Project 'FN' belongs to tribe: Engineering")
}

func TestApp_configTeamTribeShowAction_LookupErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &stubTeamConfigService{tribeErr: errors.New("disk")}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	err := a.configTeamTribeShowAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get tribe")
}

// team-company set

func TestApp_configTeamCompanySetAction_RequiresProject(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &stubTeamConfigService{}}
	ctx := newContextWithFlags(t, map[string]string{"company": "Acme"}, nil)
	err := a.configTeamCompanySetAction(ctx)
	require.Error(t, err)
}

func TestApp_configTeamCompanySetAction_RequiresCompany(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &stubTeamConfigService{}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	err := a.configTeamCompanySetAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "company is required")
}

func TestApp_configTeamCompanySetAction_ServiceErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &stubTeamConfigService{setCompanyErr: errors.New("disk")}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN", "company": "Acme"}, nil)
	err := a.configTeamCompanySetAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to set company")
}

func TestApp_configTeamCompanySetAction_Success(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{teamConfigService: &stubTeamConfigService{}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN", "company": "Acme"}, nil)
	out, err := captureStdout(t, func() error { return a.configTeamCompanySetAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Set company 'Acme' for project 'FN'")
}

// team-company list

func TestApp_configTeamCompanyListAction_LoadErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &stubTeamConfigService{teamConfigErr: errors.New("disk")}}
	err := a.configTeamCompanyListAction(nil)
	require.Error(t, err)
}

func TestApp_configTeamCompanyListAction_EmptyProjects(t *testing.T) {
	// no t.Parallel: prints to stdout
	cfg, err := configdomain.NewTeamConfig(map[string][]string{})
	require.NoError(t, err)
	a := &App{teamConfigService: &stubTeamConfigService{teamConfig: cfg}}
	out, err := captureStdout(t, func() error { return a.configTeamCompanyListAction(nil) })
	require.NoError(t, err)
	assert.Contains(t, out, "No teams configured")
}

func TestApp_configTeamCompanyListAction_RendersCompanyGroupsAndOrphans(t *testing.T) {
	// no t.Parallel: prints to stdout
	cfg := teamConfigWith(t, nil,
		map[string]string{"FN": "Acme", "PAY": "Acme", "AD": "Beta Co"},
	)
	require.NoError(t, cfg.SetTeam("INF", []string{"infra-alice"}))

	a := &App{teamConfigService: &stubTeamConfigService{teamConfig: cfg}}
	out, err := captureStdout(t, func() error { return a.configTeamCompanyListAction(nil) })
	require.NoError(t, err)
	assert.Contains(t, out, "Team Companies:")
	assert.Contains(t, out, "Acme")
	assert.Contains(t, out, "Beta Co")
	assert.Contains(t, out, "(No company assigned)")
	assert.Contains(t, out, "INF")
}

// team-company show

func TestApp_configTeamCompanyShowAction_RequiresProject(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &stubTeamConfigService{}}
	err := a.configTeamCompanyShowAction(newContextWithFlags(t, nil, nil))
	require.Error(t, err)
}

func TestApp_configTeamCompanyShowAction_NoCompanyPrintsBlank(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{teamConfigService: &stubTeamConfigService{company: ""}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	out, err := captureStdout(t, func() error { return a.configTeamCompanyShowAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Project 'FN' has no company assigned")
}

func TestApp_configTeamCompanyShowAction_WithCompany(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{teamConfigService: &stubTeamConfigService{company: "Acme"}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	out, err := captureStdout(t, func() error { return a.configTeamCompanyShowAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Project 'FN' belongs to company: Acme")
}

func TestApp_configTeamCompanyShowAction_LookupErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{teamConfigService: &stubTeamConfigService{companyErr: errors.New("disk")}}
	ctx := newContextWithFlags(t, map[string]string{"project": "FN"}, nil)
	err := a.configTeamCompanyShowAction(ctx)
	require.Error(t, err)
}

// ensureTeamConfigService lazy init

func TestApp_ensureTeamConfigService_LazyInitAndIdempotent(t *testing.T) {
	t.Parallel()
	a := &App{}
	assert.Nil(t, a.teamConfigService)
	a.ensureTeamConfigService()
	svc := a.teamConfigService
	require.NotNil(t, svc, "first call must wire the service")
	a.ensureTeamConfigService()
	assert.Same(t, svc, a.teamConfigService, "subsequent calls must be no-ops")
}
