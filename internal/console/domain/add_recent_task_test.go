package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAddRecentTask_MoveToFrontAndCapAtFive drives addRecentTask
// through its three branches: insert when new, move-to-front when
// already present, and drop-tail when the list exceeds five.
// It's exercised via the public UpdateTaskContext entry point.
func TestAddRecentTask_MoveToFrontAndCapAtFive(t *testing.T) {
	ctx := NewContext("session-1")

	for _, key := range []string{"T-1", "T-2", "T-3", "T-4", "T-5"} {
		ctx.UpdateTaskContext(key, nil)
	}
	// Newest entries push older ones toward the tail.
	require.Equal(t, []string{"T-5", "T-4", "T-3", "T-2", "T-1"}, ctx.RecentTasks)

	// Re-mentioning T-3 moves it to the front, doesn't grow the list.
	ctx.UpdateTaskContext("T-3", nil)
	assert.Equal(t, []string{"T-3", "T-5", "T-4", "T-2", "T-1"}, ctx.RecentTasks)
	assert.Len(t, ctx.RecentTasks, 5)

	// A sixth fresh entry evicts the tail.
	ctx.UpdateTaskContext("T-6", nil)
	assert.Equal(t, []string{"T-6", "T-3", "T-5", "T-4", "T-2"}, ctx.RecentTasks)
	assert.Len(t, ctx.RecentTasks, 5)
}
