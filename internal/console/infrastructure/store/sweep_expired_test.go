package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
)

// TestSweepExpired_DropsOnlyExpiredSessions drives the sweep helper
// that backs cleanupRoutine. cleanupRoutine itself is a 5-minute
// ticker loop and is impractical to drive end-to-end; the extracted
// sweepExpired carries the same delete logic.
func TestSweepExpired_DropsOnlyExpiredSessions(t *testing.T) {
	store := NewMemoryStore(Config{MaxSessions: 10, SessionTTL: time.Minute})

	fresh := domain.NewContext("fresh")
	stale := domain.NewContext("stale")
	require.NoError(t, store.Save(context.Background(), fresh))
	require.NoError(t, store.Save(context.Background(), stale))

	// Backdate the stale entry's LastActivity past the TTL. Save deep-
	// copies, so reach into the stored copy directly.
	store.mu.Lock()
	store.contexts["stale"].LastActivity = time.Now().Add(-2 * time.Minute)
	store.mu.Unlock()

	store.sweepExpired(time.Now())

	store.mu.RLock()
	_, freshExists := store.contexts["fresh"]
	_, staleExists := store.contexts["stale"]
	store.mu.RUnlock()
	assert.True(t, freshExists, "fresh session must survive")
	assert.False(t, staleExists, "stale session must be swept")
}

func TestSweepExpired_NoEntriesIsNoop(t *testing.T) {
	store := NewMemoryStore(Config{MaxSessions: 10, SessionTTL: time.Minute})
	// Must not panic on an empty map.
	store.sweepExpired(time.Now())
	assert.Equal(t, 0, store.GetStats().TotalSessions)
}
