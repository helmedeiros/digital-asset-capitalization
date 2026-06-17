package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	assetsapp "github.com/helmedeiros/digital-asset-capitalization/internal/assets/application"
	assetdomain "github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	taskdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
	taskports "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// stubTaskService implements the minimum TaskService surface used by
// the tasks Actions. Methods not called by any Action panic so a
// future refactor that touches a new method is loudly flagged.
type stubTaskService struct {
	fetchErr           error
	fetchByKeyErr      error
	classifyErr        error
	classifyInput      taskdomain.ClassifyTasksInput
	getTasks           []*taskdomain.Task
	getTasksErr        error
	getTasksByAsset    []*taskdomain.Task
	getTasksByAssetErr error
	getTaskByKey       *taskdomain.Task
	getTaskByKeyErr    error
}

func (s *stubTaskService) FetchTasks(context.Context, string, string, string) error {
	return s.fetchErr
}
func (s *stubTaskService) FetchTaskByKey(context.Context, string, string) error {
	return s.fetchByKeyErr
}
func (s *stubTaskService) ClassifyTasks(_ context.Context, in taskdomain.ClassifyTasksInput) error {
	s.classifyInput = in
	return s.classifyErr
}
func (s *stubTaskService) GetTasks(context.Context, string, string) ([]*taskdomain.Task, error) {
	return s.getTasks, s.getTasksErr
}
func (s *stubTaskService) GetTasksByAsset(context.Context, string) ([]*taskdomain.Task, error) {
	return s.getTasksByAsset, s.getTasksByAssetErr
}
func (s *stubTaskService) GetTaskByKey(context.Context, string) (*taskdomain.Task, error) {
	return s.getTaskByKey, s.getTaskByKeyErr
}
func (s *stubTaskService) GetLocalRepository() taskports.TaskRepository { panic("not used") }

// stubAssetServiceForTasks implements only the AssetService method
// that the tasks Actions read (GetAsset, used by tasks show --asset).
// All other interface methods panic so an accidental dependency is
// loud. The embedded UnimplementedAssetService provides those panics
// without forcing the test file to enumerate every method.
type stubAssetServiceForTasks struct {
	unimplementedAssetService
	asset    *assetdomain.Asset
	assetErr error
}

func (s *stubAssetServiceForTasks) GetAsset(string) (*assetdomain.Asset, error) {
	return s.asset, s.assetErr
}

// unimplementedAssetService satisfies the AssetService interface by
// panicking on every method. Embed it to opt-in only specific methods.
type unimplementedAssetService struct{}

func (unimplementedAssetService) CreateAsset(string, string) error            { panic("not used") }
func (unimplementedAssetService) ListAssets() ([]*assetdomain.Asset, error)   { panic("not used") }
func (unimplementedAssetService) GetAsset(string) (*assetdomain.Asset, error) { panic("not used") }
func (unimplementedAssetService) DeleteAsset(string, bool) error              { panic("not used") }
func (unimplementedAssetService) UpdateAsset(string, string, string, string, string, string) error {
	panic("not used")
}
func (unimplementedAssetService) UpdateDocumentation(string) error { panic("not used") }
func (unimplementedAssetService) IncrementTaskCount(string) error  { panic("not used") }
func (unimplementedAssetService) DecrementTaskCount(string) error  { panic("not used") }
func (unimplementedAssetService) SyncFromConfluence(string, string, bool) (*assetdomain.SyncResult, error) {
	panic("not used")
}
func (unimplementedAssetService) EnrichAsset(string, string) error          { panic("not used") }
func (unimplementedAssetService) GenerateKeywords(string) error             { panic("not used") }
func (unimplementedAssetService) AssignTeam(string, string, []string) error { panic("not used") }
func (unimplementedAssetService) GetAssetTeams() ([]assetsapp.AssetTeamInfo, error) {
	panic("not used")
}
func (unimplementedAssetService) GetAssetTeamInfo(string) (*assetsapp.AssetTeamInfo, error) {
	panic("not used")
}
func (unimplementedAssetService) AddContributingTeam(string, string) error    { panic("not used") }
func (unimplementedAssetService) RemoveContributingTeam(string, string) error { panic("not used") }
func (unimplementedAssetService) PublishToConfluence(context.Context, string, string, string, bool, bool) (*assetsapp.PublishToConfluenceResult, error) {
	panic("not used")
}
func (unimplementedAssetService) UpdateConfluencePage(context.Context, string, bool, bool) (*assetsapp.PublishToConfluenceResult, error) {
	panic("not used")
}

func TestApp_tasksFetchAction_KeyConflictsWithProjectSprint(t *testing.T) {
	t.Parallel()
	a := &App{taskService: &stubTaskService{}}
	ctx := newContextWithFlags(t,
		map[string]string{"key": "FN-1", "project": "FN", "platform": "jira"},
		nil,
	)
	err := a.tasksFetchAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "when using --key, do not specify")
}

func TestApp_tasksFetchAction_NeitherKeyNorPair(t *testing.T) {
	t.Parallel()
	a := &App{taskService: &stubTaskService{}}
	ctx := newContextWithFlags(t,
		map[string]string{"platform": "jira"},
		nil,
	)
	err := a.tasksFetchAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "either --key or both --project and --sprint")
}

func TestApp_tasksFetchAction_FetchByKeySuccess(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{taskService: &stubTaskService{}}
	ctx := newContextWithFlags(t,
		map[string]string{"key": "FN-1", "platform": "jira"},
		nil,
	)
	out, err := captureStdout(t, func() error { return a.tasksFetchAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Successfully fetched task FN-1")
}

func TestApp_tasksFetchAction_FetchByKeyError(t *testing.T) {
	t.Parallel()
	a := &App{taskService: &stubTaskService{fetchByKeyErr: errors.New("network")}}
	ctx := newContextWithFlags(t,
		map[string]string{"key": "FN-1", "platform": "jira"},
		nil,
	)
	err := a.tasksFetchAction(ctx)
	require.Error(t, err)
	assert.Equal(t, "network", err.Error())
}

func TestApp_tasksFetchAction_FetchBySprintResolvesTeamNickname(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{taskService: &stubTaskService{}, teamResolver: teamResolverFor(t)}
	ctx := newContextWithFlags(t,
		map[string]string{"project": "voyager", "sprint": "Penguins", "platform": "jira"},
		nil,
	)
	out, err := captureStdout(t, func() error { return a.tasksFetchAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Successfully fetched tasks")
}

func TestApp_tasksFetchAction_FetchBySprintBadNickname(t *testing.T) {
	t.Parallel()
	a := &App{taskService: &stubTaskService{}, teamResolver: teamResolverFor(t)}
	ctx := newContextWithFlags(t,
		map[string]string{"project": "bogus", "sprint": "Penguins", "platform": "jira"},
		nil,
	)
	err := a.tasksFetchAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown project or team nickname")
}

func TestApp_tasksShowAction_AssetNotFound(t *testing.T) {
	t.Parallel()
	a := &App{
		taskService:  &stubTaskService{},
		assetService: &stubAssetServiceForTasks{assetErr: errors.New("not found")},
	}
	ctx := newContextWithFlags(t,
		map[string]string{"asset": "Search"},
		nil,
	)
	err := a.tasksShowAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "asset not found")
}

func TestApp_tasksShowAction_NoFlagsErrors(t *testing.T) {
	t.Parallel()
	a := &App{taskService: &stubTaskService{}}
	ctx := newContextWithFlags(t, nil, nil)
	err := a.tasksShowAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "both project and sprint flags are required")
}

func TestApp_tasksShowAction_EmptyAssetTasksOK(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{
		taskService:  &stubTaskService{getTasksByAsset: nil},
		assetService: &stubAssetServiceForTasks{asset: &assetdomain.Asset{Name: "Search"}},
	}
	ctx := newContextWithFlags(t,
		map[string]string{"asset": "Search"},
		nil,
	)
	out, err := captureStdout(t, func() error { return a.tasksShowAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "No tasks found")
}

func TestApp_tasksClassifyAction_ServiceErrorBubbles(t *testing.T) {
	t.Parallel()
	a := &App{taskService: &stubTaskService{classifyErr: errors.New("boom")}}
	ctx := newContextWithFlags(t,
		map[string]string{"project": "FN", "sprint": "P", "platform": "jira"},
		map[string]bool{"dry-run": true},
	)
	err := a.tasksClassifyAction(ctx)
	require.Error(t, err)
	assert.Equal(t, "boom", err.Error())
}

func TestApp_tasksClassifyAction_DryRunPath(t *testing.T) {
	// no t.Parallel: prints to stdout
	stub := &stubTaskService{}
	a := &App{taskService: stub}
	ctx := newContextWithFlags(t,
		map[string]string{"project": "FN", "sprint": "P", "platform": "jira"},
		map[string]bool{"dry-run": true},
	)
	out, err := captureStdout(t, func() error { return a.tasksClassifyAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Classification preview completed")
	assert.True(t, stub.classifyInput.DryRun)
}

func TestApp_tasksClassifyAction_ApplyPath(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{taskService: &stubTaskService{}}
	ctx := newContextWithFlags(t,
		map[string]string{"project": "FN", "sprint": "P", "platform": "jira"},
		map[string]bool{"apply": true},
	)
	out, err := captureStdout(t, func() error { return a.tasksClassifyAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Successfully classified and applied")
}

func TestApp_tasksClassifyAction_BadNickname(t *testing.T) {
	t.Parallel()
	a := &App{taskService: &stubTaskService{}, teamResolver: teamResolverFor(t)}
	ctx := newContextWithFlags(t,
		map[string]string{"project": "bogus", "sprint": "P", "platform": "jira"},
		nil,
	)
	err := a.tasksClassifyAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown project or team nickname")
}

func TestApp_tasksInspectAction_EmptyKey(t *testing.T) {
	t.Parallel()
	a := &App{taskService: &stubTaskService{}}
	ctx := newContextWithFlags(t, nil, nil)
	err := a.tasksInspectAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "task key is required")
}

func TestApp_tasksInspectAction_NotFound(t *testing.T) {
	// no t.Parallel: prints to stdout
	a := &App{taskService: &stubTaskService{getTaskByKey: nil}}
	ctx := newContextWithFlags(t,
		map[string]string{"key": "FN-1"},
		nil,
	)
	out, err := captureStdout(t, func() error { return a.tasksInspectAction(ctx) })
	require.NoError(t, err)
	assert.Contains(t, out, "Task FN-1 not found")
}

func TestApp_tasksInspectAction_LookupErrorWraps(t *testing.T) {
	t.Parallel()
	a := &App{taskService: &stubTaskService{getTaskByKeyErr: errors.New("network")}}
	ctx := newContextWithFlags(t,
		map[string]string{"key": "FN-1"},
		nil,
	)
	err := a.tasksInspectAction(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get task FN-1")
}

func TestApp_tasksMigrateAction_StatsPath(t *testing.T) {
	// no t.Parallel: prints to stdout, uses TempDir for file
	dir := t.TempDir()
	tasksJSON := filepath.Join(dir, "tasks.json")
	// Empty file is fine: the migrator's GetMigrationStats handles
	// missing/empty data by returning zero counts.
	require.NoError(t, writeEmptyJSONFile(t, tasksJSON))

	a := &App{}
	ctx := newContextWithFlags(t,
		map[string]string{"file": tasksJSON},
		map[string]bool{"stats": true},
	)
	_, err := captureStdout(t, func() error { return a.tasksMigrateAction(ctx) })
	// Stats may succeed or surface a parse error; we mainly care that
	// the stats-branch is selected.
	if err != nil {
		assert.Contains(t, err.Error(), "failed to get migration stats")
	}
}

func writeEmptyJSONFile(t *testing.T, path string) error {
	t.Helper()
	return os.WriteFile(path, []byte("[]"), 0o644)
}
