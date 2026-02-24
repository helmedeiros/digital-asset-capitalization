package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

func TestSprintLockStorage_FindLock(t *testing.T) {
	t.Run("should return nil when no lock exists", func(t *testing.T) {
		dir := t.TempDir()
		storage := NewSprintLockStorage(dir, "sprint_locks.json")

		lock, err := storage.FindLock(context.Background(), "COP", "Sprint 1")
		require.NoError(t, err)
		assert.Nil(t, lock)
	})

	t.Run("should return nil when file does not exist", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "nonexistent")
		storage := NewSprintLockStorage(dir, "sprint_locks.json")

		lock, err := storage.FindLock(context.Background(), "COP", "Sprint 1")
		require.NoError(t, err)
		assert.Nil(t, lock)
	})

	t.Run("should return lock when it exists", func(t *testing.T) {
		dir := t.TempDir()
		storage := NewSprintLockStorage(dir, "sprint_locks.json")

		// Save a lock first
		original := domain.NewSprintLock("COP", "Sprint 1", 5)
		err := storage.SaveLock(context.Background(), original)
		require.NoError(t, err)

		// Find it
		lock, err := storage.FindLock(context.Background(), "COP", "Sprint 1")
		require.NoError(t, err)
		require.NotNil(t, lock)
		assert.Equal(t, "COP", lock.Project)
		assert.Equal(t, "Sprint 1", lock.Sprint)
		assert.Equal(t, 5, lock.TaskCount)
	})

	t.Run("should return nil for different sprint", func(t *testing.T) {
		dir := t.TempDir()
		storage := NewSprintLockStorage(dir, "sprint_locks.json")

		original := domain.NewSprintLock("COP", "Sprint 1", 5)
		err := storage.SaveLock(context.Background(), original)
		require.NoError(t, err)

		lock, err := storage.FindLock(context.Background(), "COP", "Sprint 2")
		require.NoError(t, err)
		assert.Nil(t, lock)
	})

	t.Run("should return error for invalid JSON", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(filepath.Join(dir, "sprint_locks.json"), []byte("invalid"), 0644)
		require.NoError(t, err)

		storage := NewSprintLockStorage(dir, "sprint_locks.json")
		lock, err := storage.FindLock(context.Background(), "COP", "Sprint 1")
		assert.Error(t, err)
		assert.Nil(t, lock)
	})
}

func TestSprintLockStorage_SaveLock(t *testing.T) {
	t.Run("should save and retrieve lock", func(t *testing.T) {
		dir := t.TempDir()
		storage := NewSprintLockStorage(dir, "sprint_locks.json")

		lock := domain.NewSprintLock("COP", "Sprint 1", 10)
		err := storage.SaveLock(context.Background(), lock)
		require.NoError(t, err)

		// Verify file exists
		_, err = os.Stat(filepath.Join(dir, "sprint_locks.json"))
		assert.NoError(t, err)

		// Retrieve
		found, err := storage.FindLock(context.Background(), "COP", "Sprint 1")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, 10, found.TaskCount)
	})

	t.Run("should overwrite existing lock", func(t *testing.T) {
		dir := t.TempDir()
		storage := NewSprintLockStorage(dir, "sprint_locks.json")

		lock1 := domain.NewSprintLock("COP", "Sprint 1", 5)
		err := storage.SaveLock(context.Background(), lock1)
		require.NoError(t, err)

		lock2 := domain.NewSprintLock("COP", "Sprint 1", 8)
		err = storage.SaveLock(context.Background(), lock2)
		require.NoError(t, err)

		found, err := storage.FindLock(context.Background(), "COP", "Sprint 1")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, 8, found.TaskCount)
	})

	t.Run("should store multiple locks independently", func(t *testing.T) {
		dir := t.TempDir()
		storage := NewSprintLockStorage(dir, "sprint_locks.json")

		lock1 := domain.NewSprintLock("COP", "Sprint 1", 5)
		lock2 := domain.NewSprintLock("COP", "Sprint 2", 3)
		require.NoError(t, storage.SaveLock(context.Background(), lock1))
		require.NoError(t, storage.SaveLock(context.Background(), lock2))

		found1, err := storage.FindLock(context.Background(), "COP", "Sprint 1")
		require.NoError(t, err)
		require.NotNil(t, found1)
		assert.Equal(t, 5, found1.TaskCount)

		found2, err := storage.FindLock(context.Background(), "COP", "Sprint 2")
		require.NoError(t, err)
		require.NotNil(t, found2)
		assert.Equal(t, 3, found2.TaskCount)
	})

	t.Run("should return error for nil lock", func(t *testing.T) {
		dir := t.TempDir()
		storage := NewSprintLockStorage(dir, "sprint_locks.json")

		err := storage.SaveLock(context.Background(), nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sprint lock cannot be nil")
	})
}
