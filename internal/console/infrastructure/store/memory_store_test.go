package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain/ports"
)

func TestNewMemoryStore(t *testing.T) {
	config := Config{
		MaxSessions: 10,
		SessionTTL:  5 * time.Minute,
	}

	store := NewMemoryStore(config)

	assert.NotNil(t, store)
	assert.Equal(t, 10, store.maxSessions)
	assert.Equal(t, 5*time.Minute, store.sessionTTL)
}

func TestMemoryStore_SaveAndLoad(t *testing.T) {
	store := NewMemoryStore(DefaultConfig())
	ctx := context.Background()

	// Create a test context
	sessionContext := domain.NewContext("test-session-id")

	// Test Save
	err := store.Save(ctx, sessionContext)
	assert.NoError(t, err)

	// Test Load
	loaded, err := store.Load(ctx, "test-session-id")
	assert.NoError(t, err)
	assert.Equal(t, "test-session-id", loaded.SessionID)
}

func TestMemoryStore_LoadNonExistent(t *testing.T) {
	store := NewMemoryStore(DefaultConfig())
	ctx := context.Background()

	_, err := store.Load(ctx, "non-existent")
	assert.Equal(t, ports.ErrContextNotFound, err)
}

func TestMemoryStore_Update(t *testing.T) {
	store := NewMemoryStore(DefaultConfig())
	ctx := context.Background()

	// Create and save a context
	sessionContext := domain.NewContext("test-session")
	err := store.Save(ctx, sessionContext)
	require.NoError(t, err)

	// Update the context
	err = store.Update(ctx, "test-session", func(ctx *domain.Context) error {
		ctx.CurrentProject = "test-project"
		return nil
	})
	assert.NoError(t, err)

	// Verify the update
	loaded, err := store.Load(ctx, "test-session")
	assert.NoError(t, err)
	assert.Equal(t, "test-project", loaded.CurrentProject)
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore(DefaultConfig())
	ctx := context.Background()

	// Create and save a context
	sessionContext := domain.NewContext("test-session")
	err := store.Save(ctx, sessionContext)
	require.NoError(t, err)

	// Delete the context
	err = store.Delete(ctx, "test-session")
	assert.NoError(t, err)

	// Verify it's deleted
	_, err = store.Load(ctx, "test-session")
	assert.Equal(t, ports.ErrContextNotFound, err)
}

func TestMemoryStore_List(t *testing.T) {
	store := NewMemoryStore(DefaultConfig())
	ctx := context.Background()

	// Initially empty
	sessions, err := store.List(ctx)
	assert.NoError(t, err)
	assert.Empty(t, sessions)

	// Add some sessions
	ctx1 := domain.NewContext("session-1")
	ctx2 := domain.NewContext("session-2")

	err = store.Save(ctx, ctx1)
	require.NoError(t, err)
	err = store.Save(ctx, ctx2)
	require.NoError(t, err)

	// List should return both
	sessions, err = store.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, sessions, 2)
	assert.Contains(t, sessions, "session-1")
	assert.Contains(t, sessions, "session-2")
}

func TestMemoryStore_GetStats(t *testing.T) {
	store := NewMemoryStore(DefaultConfig())
	ctx := context.Background()

	// Initial stats
	stats := store.GetStats()
	assert.Equal(t, 0, stats.TotalSessions)

	// Add a session
	sessionContext := domain.NewContext("test-session")
	err := store.Save(ctx, sessionContext)
	require.NoError(t, err)

	// Check stats
	stats = store.GetStats()
	assert.Equal(t, 1, stats.TotalSessions)
	assert.Equal(t, 100, stats.MaxSessions)
}

func TestMemoryStore_StoreFull(t *testing.T) {
	// Create store with max 1 session
	config := Config{
		MaxSessions: 1,
		SessionTTL:  5 * time.Minute,
	}
	store := NewMemoryStore(config)
	ctx := context.Background()

	// Add first session
	ctx1 := domain.NewContext("session-1")
	err := store.Save(ctx, ctx1)
	assert.NoError(t, err)

	// Try to add second session - should fail
	ctx2 := domain.NewContext("session-2")
	err = store.Save(ctx, ctx2)
	assert.Equal(t, ports.ErrStoreFull, err)

	// But updating existing session should work
	err = store.Save(ctx, ctx1)
	assert.NoError(t, err)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, 100, config.MaxSessions)
	assert.Equal(t, 30*time.Minute, config.SessionTTL)
}

func TestMemoryStore_Update_NonExistent(t *testing.T) {
	store := NewMemoryStore(DefaultConfig())
	ctx := context.Background()

	// Try to update non-existent session
	err := store.Update(ctx, "non-existent", func(_ *domain.Context) error {
		return nil
	})
	assert.Equal(t, ports.ErrContextNotFound, err)
}

func TestMemoryStore_Update_Error(t *testing.T) {
	store := NewMemoryStore(DefaultConfig())
	ctx := context.Background()

	// Create and save a context
	sessionContext := domain.NewContext("test-session")
	err := store.Save(ctx, sessionContext)
	require.NoError(t, err)

	// Update with error
	expectedErr := assert.AnError
	err = store.Update(ctx, "test-session", func(_ *domain.Context) error {
		return expectedErr
	})
	assert.Equal(t, expectedErr, err)
}

func TestMemoryStore_Delete_NonExistent(t *testing.T) {
	store := NewMemoryStore(DefaultConfig())
	ctx := context.Background()

	// Delete non-existent session should succeed (no-op)
	err := store.Delete(ctx, "non-existent")
	assert.NoError(t, err)
}

func TestMemoryStore_CleanupExpired(t *testing.T) {
	store := NewMemoryStore(DefaultConfig())
	ctx := context.Background()

	// Add a session
	sessionContext := domain.NewContext("test-session")
	err := store.Save(ctx, sessionContext)
	require.NoError(t, err)

	// Cleanup with timeout (none should be expired with recent session)
	err = store.CleanupExpired(ctx, 30)
	assert.NoError(t, err)

	// Session should still exist
	_, err = store.Load(ctx, "test-session")
	assert.NoError(t, err)
}
