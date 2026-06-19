package infrastructure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

func mustNewJiraConfig(t *testing.T) *domain.JiraConfig {
	t.Helper()
	cfg, err := domain.NewJiraConfig("https://example.atlassian.net", "test@example.com", "token")
	require.NoError(t, err)
	return cfg
}

func mustNewTeamConfig(t *testing.T) *domain.TeamConfig {
	t.Helper()
	cfg, err := domain.NewTeamConfig(map[string][]string{"FN": {"alice"}})
	require.NoError(t, err)
	return cfg
}

// Read-error wraps for LoadJiraConfig / LoadTeamConfig: trigger a
// real os.ReadFile permission error by chmod-ing the file to 0000.
// (Linux/macOS only — root would bypass; CI runs as non-root.)

func TestFileRepository_LoadJiraConfig_ReadFileErrorWraps(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "jira.json")
	require.NoError(t, os.WriteFile(path, []byte(`{}`), 0644))
	require.NoError(t, os.Chmod(path, 0000))
	t.Cleanup(func() { _ = os.Chmod(path, 0644) })

	repo := NewFileRepository(dir)
	_, err := repo.LoadJiraConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read jira config")
}

func TestFileRepository_LoadTeamConfig_ReadFileErrorWraps(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "teams.json")
	require.NoError(t, os.WriteFile(path, []byte(`{}`), 0644))
	require.NoError(t, os.Chmod(path, 0000))
	t.Cleanup(func() { _ = os.Chmod(path, 0644) })

	repo := NewFileRepository(dir)
	_, err := repo.LoadTeamConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read team config")
}

// LoadTeamConfig: invalid team_timeline date strings (joined / left).

func TestFileRepository_LoadTeamConfig_InvalidJoinedDateWraps(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	teamsJSON := `{
		"FN": {
			"team": ["alice"],
			"team_timeline": [
				{"member": "alice", "joined": "not-a-date"}
			]
		}
	}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "teams.json"), []byte(teamsJSON), 0644))

	repo := NewFileRepository(dir)
	_, err := repo.LoadTeamConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse joined date for alice in FN")
}

func TestFileRepository_LoadTeamConfig_InvalidLeftDateWraps(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	teamsJSON := `{
		"FN": {
			"team": ["alice"],
			"team_timeline": [
				{"member": "alice", "joined": "2026-01-01", "left": "yesterday"}
			]
		}
	}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "teams.json"), []byte(teamsJSON), 0644))

	repo := NewFileRepository(dir)
	_, err := repo.LoadTeamConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse left date for alice in FN")
}

func TestFileRepository_LoadTeamConfig_SkipsInvalidBoardIDs(t *testing.T) {
	t.Parallel()
	// LoadTeamConfig parses board_work_streams as map[string]string and
	// silently skips entries whose key cannot be Atoi'd. Pin the skip
	// branch by feeding one invalid + one valid entry.
	dir := t.TempDir()
	teamsJSON := `{
		"FN": {
			"team": ["alice"],
			"board_work_streams": {
				"5119": "Product",
				"notanint": "Operational"
			}
		}
	}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "teams.json"), []byte(teamsJSON), 0644))

	repo := NewFileRepository(dir)
	cfg, err := repo.LoadTeamConfig()
	require.NoError(t, err)
	mapping := cfg.GetBoardWorkStreams("FN")
	require.Len(t, mapping, 1)
	assert.Equal(t, "Product", mapping[5119])
}

// writeFile error paths via SaveJiraConfig (path validation triggers
// MkdirAll/WriteFile failures).

func TestFileRepository_SaveJiraConfig_MkdirAllErrorWraps(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Make a regular file at the would-be config dir path so MkdirAll
	// can't create a directory at that location.
	conflictPath := filepath.Join(dir, "file-not-dir")
	require.NoError(t, os.WriteFile(conflictPath, []byte("x"), 0644))
	// Now point the repo to a subdirectory under that file — MkdirAll
	// will fail trying to create a directory inside a regular file.
	repo := NewFileRepository(filepath.Join(conflictPath, "subdir"))

	configdomainCfg := mustNewJiraConfig(t)
	err := repo.SaveJiraConfig(configdomainCfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create directory")
}

func TestFileRepository_SaveTeamConfig_MkdirAllErrorWraps(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	conflictPath := filepath.Join(dir, "file-not-dir")
	require.NoError(t, os.WriteFile(conflictPath, []byte("x"), 0644))
	repo := NewFileRepository(filepath.Join(conflictPath, "subdir"))

	cfg := mustNewTeamConfig(t)
	err := repo.SaveTeamConfig(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create directory")
}

// JiraTeamSyncAdapter — uncovered branches.

func TestJiraTeamSyncAdapter_GetProjectMembers_EmptyProjectKey(t *testing.T) {
	t.Parallel()
	cs := &MockConfigService{}
	a, err := NewJiraTeamSyncAdapter(cs)
	require.NoError(t, err)
	_, err = a.GetProjectMembers("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project key is required")
}

func TestJiraTeamSyncAdapter_GetProjectRoles_EmptyProjectKey(t *testing.T) {
	t.Parallel()
	cs := &MockConfigService{}
	a, err := NewJiraTeamSyncAdapter(cs)
	require.NoError(t, err)
	_, err = a.GetProjectRoles("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project key is required")
}

func TestJiraTeamSyncAdapter_GetProjectRoles_ConfigErrorWraps(t *testing.T) {
	t.Parallel()
	cs := &MockConfigService{}
	cs.On("GetJiraConfig").Return(nil, assertAnyErr())
	a, err := NewJiraTeamSyncAdapter(cs)
	require.NoError(t, err)
	_, err = a.GetProjectRoles("FN")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get JIRA config")
	cs.AssertExpectations(t)
}

// assertAnyErr returns a sentinel error used by the config-error
// test above. testify's mock.AnythingOfType doesn't apply to
// concrete error values; this keeps the assertion intent obvious.
func assertAnyErr() error {
	return errFakeConfig
}

var errFakeConfig = newErr("fake config error")

func newErr(msg string) error { return errString(msg) }

type errString string

func (e errString) Error() string { return string(e) }
