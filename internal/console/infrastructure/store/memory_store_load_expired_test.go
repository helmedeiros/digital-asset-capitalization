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

// TestMemoryStore_Load_Expired drives the previously-uncovered
// expiration branch in MemoryStore.Load. We save with a generous TTL,
// then shrink the TTL via the same-package accessor so the next Load
// finds the entry but treats it as expired.
func TestMemoryStore_Load_Expired(t *testing.T) {
	store := NewMemoryStore(Config{MaxSessions: 10, SessionTTL: time.Hour})

	sessionContext := domain.NewContext("expiring-session")
	require.NoError(t, store.Save(context.Background(), sessionContext))

	// Shrink TTL so the stored context is treated as expired.
	store.sessionTTL = time.Nanosecond

	_, err := store.Load(context.Background(), "expiring-session")
	assert.ErrorIs(t, err, ports.ErrContextExpired)

	// Second load reports not-found because the expired session was
	// deleted by the previous Load.
	_, err = store.Load(context.Background(), "expiring-session")
	assert.ErrorIs(t, err, ports.ErrContextNotFound)
}
