package migration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

// stubMigrationRepo satisfies ports.TaskRepository while letting tests
// inject errors on the two methods SprintMigration actually calls
// (FindAll and Save). All other methods panic — they're not on the
// migration path, and a panic surfaces an accidental dependency
// growth in code review rather than silently no-op'ing.
type stubMigrationRepo struct {
	tasks      []*domain.Task
	findAllErr error
	saveErrFor string // Task.Key whose Save call should fail
	saveErr    error
	savedTasks []*domain.Task
	saveCalls  int
}

func (s *stubMigrationRepo) FindAll(context.Context) ([]*domain.Task, error) {
	return s.tasks, s.findAllErr
}

func (s *stubMigrationRepo) Save(_ context.Context, t *domain.Task) error {
	s.saveCalls++
	if t.Key == s.saveErrFor {
		return s.saveErr
	}
	s.savedTasks = append(s.savedTasks, t)
	return nil
}

func (s *stubMigrationRepo) SaveAll(context.Context, []*domain.Task) error {
	panic("SaveAll not used by migration")
}
func (s *stubMigrationRepo) FindByKey(context.Context, string) (*domain.Task, error) {
	panic("FindByKey not used by migration")
}
func (s *stubMigrationRepo) FindByProjectAndSprint(context.Context, string, string) ([]*domain.Task, error) {
	panic("FindByProjectAndSprint not used by migration")
}
func (s *stubMigrationRepo) FindByProject(context.Context, string) ([]*domain.Task, error) {
	panic("FindByProject not used by migration")
}
func (s *stubMigrationRepo) FindBySprint(context.Context, string) ([]*domain.Task, error) {
	panic("FindBySprint not used by migration")
}
func (s *stubMigrationRepo) FindByPlatform(context.Context, string) ([]*domain.Task, error) {
	panic("FindByPlatform not used by migration")
}
func (s *stubMigrationRepo) Delete(context.Context, string) error {
	panic("Delete not used by migration")
}
func (s *stubMigrationRepo) DeleteByProjectAndSprint(context.Context, string, string) error {
	panic("DeleteByProjectAndSprint not used by migration")
}
func (s *stubMigrationRepo) UpdateLabels(context.Context, string, []string, []string) error {
	panic("UpdateLabels not used by migration")
}

// MigrateToArrayFormat

func TestSprintMigration_MigrateToArrayFormat_FindAllErrorWraps(t *testing.T) {
	t.Parallel()
	repo := &stubMigrationRepo{findAllErr: errors.New("disk")}
	mig := NewSprintMigration(repo, t.TempDir())
	_, err := mig.MigrateToArrayFormat(context.Background(), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch tasks")
}

func TestSprintMigration_MigrateToArrayFormat_BackupErrorWraps(t *testing.T) {
	t.Parallel()
	repo := &stubMigrationRepo{tasks: []*domain.Task{{Key: "T-1", Sprint: "A"}}}
	// backupDir under a non-directory file → MkdirAll fails.
	tempDir := t.TempDir()
	notADir := filepath.Join(tempDir, "file")
	require.NoError(t, os.WriteFile(notADir, []byte("x"), 0644))
	backupDir := filepath.Join(notADir, "subdir")

	mig := NewSprintMigration(repo, backupDir)
	_, err := mig.MigrateToArrayFormat(context.Background(), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create backup")
}

func TestSprintMigration_MigrateToArrayFormat_PerTaskMigrationErrorRecorded(t *testing.T) {
	t.Parallel()
	// A task whose Sprint can't be parsed: a single "," yields no
	// non-empty entries. dryRun=true skips createBackup (which mutates
	// tasks via Task.MarshalJSON) so the per-task error path is the
	// only thing exercised here.
	repo := &stubMigrationRepo{tasks: []*domain.Task{
		{Key: "T-bad", Sprint: ","},
	}}
	mig := NewSprintMigration(repo, t.TempDir())
	result, err := mig.MigrateToArrayFormat(context.Background(), true)
	require.NoError(t, err)
	require.NotEmpty(t, result.Errors, "the bad task should record a migration error")
	assert.Contains(t, result.Errors[0], "T-bad")
	assert.Contains(t, result.Errors[0], "failed to parse sprint string")
}

func TestSprintMigration_MigrateToArrayFormat_SaveErrorRecorded(t *testing.T) {
	t.Parallel()
	// To survive both the createBackup MarshalJSON mutation AND the
	// migrateTask early-return guard, the task must have Sprint and
	// Sprints out-of-sync (Sprint="A, B" but Sprints=["A"]). MarshalJSON
	// only auto-syncs when one side is empty; with both populated the
	// mismatch is preserved into migrateTask, which then runs the
	// SetSprints branch and triggers the Save call we want to fail.
	repo := &stubMigrationRepo{
		tasks: []*domain.Task{
			{Key: "T-1", Sprint: "A, B", Sprints: []string{"A"}},
		},
		saveErrFor: "T-1",
		saveErr:    errors.New("disk full"),
	}
	mig := NewSprintMigration(repo, t.TempDir())
	result, err := mig.MigrateToArrayFormat(context.Background(), false)
	require.NoError(t, err)
	require.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0], "Failed to save task T-1")
}

// ValidateCompatibility

func TestSprintMigration_ValidateCompatibility_FindAllErrorWraps(t *testing.T) {
	t.Parallel()
	repo := &stubMigrationRepo{findAllErr: errors.New("disk")}
	mig := NewSprintMigration(repo, t.TempDir())
	err := mig.ValidateCompatibility(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch tasks")
}

// GetMigrationStats

func TestSprintMigration_GetMigrationStats_FindAllErrorWraps(t *testing.T) {
	t.Parallel()
	repo := &stubMigrationRepo{findAllErr: errors.New("disk")}
	mig := NewSprintMigration(repo, t.TempDir())
	_, err := mig.GetMigrationStats(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch tasks")
}

func TestSprintMigration_GetMigrationStats_TaskWithSprintButNoSprintsArrayCountsAsNeedsMigration(t *testing.T) {
	t.Parallel()
	// One task has Sprint but no Sprints array (needs-migration).
	// One has Sprint empty (skipped).
	// One has Sprint matching the joined Sprints (already migrated).
	// One has Sprint NOT matching the joined Sprints (needs migration).
	repo := &stubMigrationRepo{tasks: []*domain.Task{
		{Key: "T-1", Sprint: "Alpha"},                             // needs migration (no Sprints)
		{Key: "T-2", Sprint: ""},                                  // skipped
		{Key: "T-3", Sprint: "X, Y", Sprints: []string{"X", "Y"}}, // already migrated
		{Key: "T-4", Sprint: "Mismatch", Sprints: []string{"X"}},  // needs migration (mismatch)
	}}
	mig := NewSprintMigration(repo, t.TempDir())
	result, err := mig.GetMigrationStats(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 4, result.TasksProcessed)
	assert.Equal(t, 1, result.Statistics["already_migrated"])
	assert.Equal(t, 2, result.Statistics["needs_migration"])
}

// RollbackMigration

func TestSprintMigration_RollbackMigration_NoBackupFileFound(t *testing.T) {
	t.Parallel()
	repo := &stubMigrationRepo{}
	// Existing but empty backup dir → no matching files.
	mig := NewSprintMigration(repo, t.TempDir())
	err := mig.RollbackMigration(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no backup file found")
}

func TestSprintMigration_RollbackMigration_SkipsDirsAndNonBackupFiles(t *testing.T) {
	t.Parallel()
	repo := &stubMigrationRepo{}
	backupDir := t.TempDir()
	// A subdirectory in the backup dir — RollbackMigration's IsDir branch
	// must skip it without misinterpreting as a backup.
	require.NoError(t, os.Mkdir(filepath.Join(backupDir, "subdir"), 0755))
	// A file with a non-backup prefix.
	require.NoError(t, os.WriteFile(filepath.Join(backupDir, "other.json"), []byte("{}"), 0644))
	// A file with the right prefix but a malformed timestamp.
	require.NoError(t, os.WriteFile(filepath.Join(backupDir, "tasks_backup_NOTATIMESTAMP.json"), []byte("{}"), 0644))

	mig := NewSprintMigration(repo, backupDir)
	err := mig.RollbackMigration(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no backup file found")
}

func TestSprintMigration_RollbackMigration_HappyPath(t *testing.T) {
	t.Parallel()
	repo := &stubMigrationRepo{}
	mig := NewSprintMigration(repo, t.TempDir())

	// Seed a backup by calling createBackup directly.
	tasks := []*domain.Task{{Key: "T-1", Sprint: "Alpha", Sprints: []string{"Alpha"}}}
	backupPath, err := mig.createBackup(tasks)
	require.NoError(t, err)
	require.FileExists(t, backupPath)

	// Now rollback — it should read the backup back and call Save.
	require.NoError(t, mig.RollbackMigration(context.Background()))
	assert.Equal(t, 1, repo.saveCalls)
}

func TestSprintMigration_RollbackMigration_BadJSONInBackup(t *testing.T) {
	t.Parallel()
	repo := &stubMigrationRepo{}
	backupDir := t.TempDir()
	// Write a backup file with the right name pattern but invalid JSON.
	backupName := "tasks_backup_20260619_120000.json"
	require.NoError(t, os.WriteFile(filepath.Join(backupDir, backupName), []byte("not json"), 0644))

	mig := NewSprintMigration(repo, backupDir)
	err := mig.RollbackMigration(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal backup")
}

func TestSprintMigration_RollbackMigration_SaveErrorWraps(t *testing.T) {
	t.Parallel()
	repo := &stubMigrationRepo{
		saveErrFor: "T-1",
		saveErr:    errors.New("disk"),
	}
	mig := NewSprintMigration(repo, t.TempDir())

	tasks := []*domain.Task{{Key: "T-1", Sprint: "Alpha"}}
	_, err := mig.createBackup(tasks)
	require.NoError(t, err)

	err = mig.RollbackMigration(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to restore task T-1")
}
