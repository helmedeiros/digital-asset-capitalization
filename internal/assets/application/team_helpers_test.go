package application

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	configservice "github.com/helmedeiros/digital-asset-capitalization/internal/config/application/service"
	configdomain "github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

// stubConfigRepo is a hand-rolled minimal ConfigurationRepository so we
// can build a real *configservice.ConfigService for the team-lookup
// helper tests. Only LoadTeamConfig is exercised; the rest panic to
// surface unexpected calls.
type stubConfigRepo struct {
	teamConfig    *configdomain.TeamConfig
	teamConfigErr error
}

func (r *stubConfigRepo) LoadJiraConfig() (*configdomain.JiraConfig, error) {
	panic("LoadJiraConfig should not be called by these tests")
}
func (r *stubConfigRepo) SaveJiraConfig(*configdomain.JiraConfig) error {
	panic("SaveJiraConfig should not be called by these tests")
}
func (r *stubConfigRepo) LoadTeamConfig() (*configdomain.TeamConfig, error) {
	return r.teamConfig, r.teamConfigErr
}
func (r *stubConfigRepo) SaveTeamConfig(*configdomain.TeamConfig) error {
	panic("SaveTeamConfig should not be called by these tests")
}
func (r *stubConfigRepo) ConfigExists() (bool, error) {
	return r.teamConfig != nil, nil
}
func (r *stubConfigRepo) InitializeConfigDirectory() error {
	panic("InitializeConfigDirectory should not be called by these tests")
}

// teamConfigWithConfluence builds the smallest TeamConfig that lets us
// drive getConfluenceParentPageForTeam through every branch: a known
// project with a parent page, a known project without one, and (via
// the team's absence) an unknown project.
func teamConfigWithConfluence(t *testing.T) *configdomain.TeamConfig {
	t.Helper()
	tc, err := configdomain.NewTeamConfigComplete(
		map[string][]string{
			"FN": {"alice"},
			"AD": {"bob"},
		},
		nil, nil, nil, nil,
		map[string]string{"FN": "12345"},
	)
	require.NoError(t, err)
	return tc
}

func teamConfigWithNicknames(t *testing.T) *configdomain.TeamConfig {
	t.Helper()
	tc, err := configdomain.NewTeamConfigWithNicknames(
		map[string][]string{"FN": {"alice"}},
		map[string][]string{"FN": {"fortuna", "Fortuna Team"}},
	)
	require.NoError(t, err)
	return tc
}

func TestAssetServiceImpl_getConfluenceParentPageForTeam(t *testing.T) {
	t.Run("empty team name short-circuits to empty string", func(t *testing.T) {
		s := &AssetServiceImpl{}
		assert.Equal(t, "", s.getConfluenceParentPageForTeam(""))
	})

	t.Run("nil configService short-circuits to empty string", func(t *testing.T) {
		s := &AssetServiceImpl{}
		assert.Equal(t, "", s.getConfluenceParentPageForTeam("FN"))
	})

	t.Run("LoadTeamConfig error returns empty string", func(t *testing.T) {
		repo := &stubConfigRepo{teamConfigErr: errors.New("repo down")}
		s := &AssetServiceImpl{configService: configservice.NewConfigService(repo)}
		assert.Equal(t, "", s.getConfluenceParentPageForTeam("FN"))
	})

	t.Run("known project returns the configured parent page", func(t *testing.T) {
		repo := &stubConfigRepo{teamConfig: teamConfigWithConfluence(t)}
		s := &AssetServiceImpl{configService: configservice.NewConfigService(repo)}
		assert.Equal(t, "12345", s.getConfluenceParentPageForTeam("FN"))
	})

	t.Run("known project without a configured page returns empty string", func(t *testing.T) {
		repo := &stubConfigRepo{teamConfig: teamConfigWithConfluence(t)}
		s := &AssetServiceImpl{configService: configservice.NewConfigService(repo)}
		assert.Equal(t, "", s.getConfluenceParentPageForTeam("AD"))
	})

	t.Run("unknown project returns empty string", func(t *testing.T) {
		repo := &stubConfigRepo{teamConfig: teamConfigWithConfluence(t)}
		s := &AssetServiceImpl{configService: configservice.NewConfigService(repo)}
		assert.Equal(t, "", s.getConfluenceParentPageForTeam("GHOST"))
	})
}

func TestAssetServiceImpl_resolveTeamIdentifier(t *testing.T) {
	t.Run("nil configService returns the identifier unchanged", func(t *testing.T) {
		s := &AssetServiceImpl{}
		assert.Equal(t, "fortuna", s.resolveTeamIdentifier("fortuna"))
	})

	t.Run("LoadTeamConfig error returns the identifier unchanged", func(t *testing.T) {
		repo := &stubConfigRepo{teamConfigErr: errors.New("repo down")}
		s := &AssetServiceImpl{configService: configservice.NewConfigService(repo)}
		assert.Equal(t, "fortuna", s.resolveTeamIdentifier("fortuna"))
	})

	t.Run("known project key resolves to itself", func(t *testing.T) {
		repo := &stubConfigRepo{teamConfig: teamConfigWithNicknames(t)}
		s := &AssetServiceImpl{configService: configservice.NewConfigService(repo)}
		assert.Equal(t, "FN", s.resolveTeamIdentifier("FN"))
	})

	t.Run("nickname resolves to the canonical project key", func(t *testing.T) {
		repo := &stubConfigRepo{teamConfig: teamConfigWithNicknames(t)}
		s := &AssetServiceImpl{configService: configservice.NewConfigService(repo)}
		assert.Equal(t, "FN", s.resolveTeamIdentifier("fortuna"))
	})

	t.Run("unknown identifier returns itself unchanged", func(t *testing.T) {
		repo := &stubConfigRepo{teamConfig: teamConfigWithNicknames(t)}
		s := &AssetServiceImpl{configService: configservice.NewConfigService(repo)}
		assert.Equal(t, "mystery-team", s.resolveTeamIdentifier("mystery-team"))
	})
}

func TestAssetServiceImpl_getTribeAndCompanyForTeam(t *testing.T) {
	// Pin the getTribeForTeam / getCompanyForTeam branches at the same
	// time -- they're structurally identical to getConfluenceParentPageForTeam
	// and the existing test surface already partly covers them, but the
	// nil-configService branch wasn't exercised.
	t.Run("nil configService returns empty for both", func(t *testing.T) {
		s := &AssetServiceImpl{}
		assert.Equal(t, "", s.getTribeForTeam("FN"))
		assert.Equal(t, "", s.getCompanyForTeam("FN"))
	})

	t.Run("LoadTeamConfig error returns empty for both", func(t *testing.T) {
		repo := &stubConfigRepo{teamConfigErr: errors.New("nope")}
		s := &AssetServiceImpl{configService: configservice.NewConfigService(repo)}
		assert.Equal(t, "", s.getTribeForTeam("FN"))
		assert.Equal(t, "", s.getCompanyForTeam("FN"))
	})
}
