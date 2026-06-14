package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewContext(t *testing.T) {
	sessionID := "test-session-123"
	ctx := NewContext(sessionID)

	require.NotNil(t, ctx)
	assert.Equal(t, sessionID, ctx.SessionID)
	assert.WithinDuration(t, time.Now(), ctx.StartTime, time.Second)
	assert.WithinDuration(t, time.Now(), ctx.LastActivity, time.Second)
	assert.Empty(t, ctx.Commands)
	assert.NotNil(t, ctx.CommandResults)
	assert.Empty(t, ctx.RecentAssets)
	assert.Empty(t, ctx.RecentTasks)
	assert.NotNil(t, ctx.Variables)
	assert.Equal(t, "table", ctx.PreferredFormat)
	assert.Equal(t, "normal", ctx.Verbosity)
}

func TestContext_CommandManagement(t *testing.T) {
	ctx := NewContext("test-session")

	// Add commands
	cmd1 := Command{
		ID:          "cmd-1",
		Raw:         "show assets",
		Interpreted: "assets list",
		Confidence:  0.9,
		Timestamp:   time.Now(),
	}
	cmd2 := Command{
		ID:          "cmd-2",
		Raw:         "create asset Test",
		Interpreted: "assets create --name Test",
		Confidence:  0.85,
		Timestamp:   time.Now(),
	}

	ctx.AddCommand(cmd1)
	ctx.AddCommand(cmd2)

	// Test command history
	assert.Len(t, ctx.Commands, 2)
	assert.Equal(t, cmd1.ID, ctx.Commands[0].ID)
	assert.Equal(t, cmd2.ID, ctx.Commands[1].ID)

	// Test GetLastCommand
	lastCmd, exists := ctx.GetLastCommand()
	require.True(t, exists)
	assert.Equal(t, cmd2.ID, lastCmd.ID)

	// Test GetCommandHistory
	history := ctx.GetCommandHistory(1)
	assert.Len(t, history, 1)
	assert.Equal(t, cmd2.ID, history[0].ID)

	history = ctx.GetCommandHistory(5) // More than available
	assert.Len(t, history, 2)

	history = ctx.GetCommandHistory(0) // Zero
	assert.Empty(t, history)

	// Test with no commands
	emptyCtx := NewContext("empty")
	lastCmd, exists = emptyCtx.GetLastCommand()
	assert.False(t, exists)
	assert.Nil(t, lastCmd)
}

func TestContext_CommandResults(t *testing.T) {
	ctx := NewContext("test-session")

	result1 := CommandResult{
		CommandID: "cmd-1",
		Success:   true,
		Output:    "Assets listed",
		Duration:  100 * time.Millisecond,
	}
	result2 := CommandResult{
		CommandID: "cmd-2",
		Success:   false,
		Error:     assert.AnError,
		Duration:  50 * time.Millisecond,
	}

	ctx.AddCommandResult(result1)
	ctx.AddCommandResult(result2)

	assert.Len(t, ctx.CommandResults, 2)
	assert.Equal(t, result1, ctx.CommandResults["cmd-1"])
	assert.Equal(t, result2, ctx.CommandResults["cmd-2"])
}

func TestContext_EntityUpdates(t *testing.T) {
	ctx := NewContext("test-session")

	// Test asset context
	asset1 := struct{ Name string }{Name: "Asset1"}
	ctx.UpdateAssetContext("Asset1", asset1)
	assert.Equal(t, asset1, ctx.LastAsset)
	assert.Equal(t, []string{"Asset1"}, ctx.RecentAssets)

	asset2 := struct{ Name string }{Name: "Asset2"}
	ctx.UpdateAssetContext("Asset2", asset2)
	assert.Equal(t, asset2, ctx.LastAsset)
	assert.Equal(t, []string{"Asset2", "Asset1"}, ctx.RecentAssets)

	// Add same asset again (should move to front)
	ctx.UpdateAssetContext("Asset1", asset1)
	assert.Equal(t, []string{"Asset1", "Asset2"}, ctx.RecentAssets)

	// Test task context
	task1 := struct{ Key string }{Key: "TASK-1"}
	ctx.UpdateTaskContext("TASK-1", task1)
	assert.Equal(t, task1, ctx.LastTask)
	assert.Equal(t, []string{"TASK-1"}, ctx.RecentTasks)

	// Test team context
	team := struct{ Name string }{Name: "Team1"}
	ctx.UpdateTeamContext(team)
	assert.Equal(t, team, ctx.LastTeam)
}

func TestContext_RecentEntityLimits(t *testing.T) {
	ctx := NewContext("test-session")

	// Add more than 5 assets
	for i := 1; i <= 7; i++ {
		asset := struct{ Name string }{Name: "Asset"}
		ctx.UpdateAssetContext(string(rune('A'+i-1)), asset)
	}

	// Should only keep last 5
	assert.Len(t, ctx.RecentAssets, 5)
	assert.Equal(t, []string{"G", "F", "E", "D", "C"}, ctx.RecentAssets)

	// Same for tasks
	for i := 1; i <= 7; i++ {
		task := struct{ Key string }{Key: "TASK"}
		ctx.UpdateTaskContext(string(rune('0'+i)), task)
	}

	assert.Len(t, ctx.RecentTasks, 5)
	assert.Equal(t, []string{"7", "6", "5", "4", "3"}, ctx.RecentTasks)
}

func TestContext_ProjectSprintSpace(t *testing.T) {
	ctx := NewContext("test-session")

	ctx.SetCurrentProject("PROJECT1")
	assert.Equal(t, "PROJECT1", ctx.CurrentProject)

	ctx.SetCurrentSprint("Sprint 23")
	assert.Equal(t, "Sprint 23", ctx.CurrentSprint)

	ctx.SetCurrentSpace("MZN")
	assert.Equal(t, "MZN", ctx.CurrentSpace)
}

func TestContext_Variables(t *testing.T) {
	ctx := NewContext("test-session")

	// Set variables
	ctx.SetVariable("key1", "value1")
	ctx.SetVariable("key2", 42)
	ctx.SetVariable("key3", true)

	// Get variables
	val, exists := ctx.GetVariable("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", val)

	val, exists = ctx.GetVariable("key2")
	assert.True(t, exists)
	assert.Equal(t, 42, val)

	val, exists = ctx.GetVariable("key3")
	assert.True(t, exists)
	assert.Equal(t, true, val)

	// Non-existent variable
	val, exists = ctx.GetVariable("missing")
	assert.False(t, exists)
	assert.Nil(t, val)
}

func TestContext_SessionManagement(t *testing.T) {
	ctx := NewContext("test-session")

	// Backdate StartTime/LastActivity so the assertions don't depend on
	// time.Sleep firing in time — they only need the past to be strictly
	// before time.Now().
	past := time.Now().Add(-time.Hour)
	ctx.StartTime = past
	ctx.LastActivity = past

	duration := ctx.GetSessionDuration()
	assert.Greater(t, duration, time.Duration(0))

	assert.False(t, ctx.IsExpired(2*time.Hour))
	assert.True(t, ctx.IsExpired(time.Nanosecond))

	oldActivity := ctx.LastActivity
	ctx.SetVariable("test", "value")
	assert.Greater(t, ctx.LastActivity, oldActivity)
}

func TestContext_Clear(t *testing.T) {
	ctx := NewContext("test-session")

	// Populate context
	ctx.AddCommand(Command{ID: "cmd-1"})
	ctx.AddCommandResult(CommandResult{CommandID: "cmd-1"})
	ctx.SetCurrentProject("PROJECT")
	ctx.SetCurrentSprint("Sprint 1")
	ctx.SetCurrentSpace("SPACE")
	ctx.UpdateAssetContext("Asset", struct{}{})
	ctx.UpdateTaskContext("TASK-1", struct{}{})
	ctx.UpdateTeamContext(struct{}{})
	ctx.SetVariable("key", "value")

	// Clear context
	ctx.Clear()

	// Verify everything is cleared except session ID
	assert.Equal(t, "test-session", ctx.SessionID)
	assert.Empty(t, ctx.Commands)
	assert.Empty(t, ctx.CommandResults)
	assert.Empty(t, ctx.CurrentProject)
	assert.Empty(t, ctx.CurrentSprint)
	assert.Empty(t, ctx.CurrentSpace)
	assert.Nil(t, ctx.LastAsset)
	assert.Nil(t, ctx.LastTask)
	assert.Nil(t, ctx.LastTeam)
	assert.Empty(t, ctx.RecentAssets)
	assert.Empty(t, ctx.RecentTasks)
	assert.Empty(t, ctx.Variables)
}

func TestContext_GetSummary(t *testing.T) {
	ctx := NewContext("test-session-123")

	// Add some data
	ctx.AddCommand(Command{ID: "cmd-1"})
	ctx.AddCommand(Command{ID: "cmd-2"})
	ctx.SetCurrentProject("FN")
	ctx.SetCurrentSprint("Sprint 23")
	ctx.SetCurrentSpace("MZN")
	ctx.UpdateAssetContext("Payment", struct{}{})
	ctx.UpdateAssetContext("Auth", struct{}{})
	ctx.UpdateTaskContext("FN-123", struct{}{})

	summary := ctx.GetSummary()

	assert.Contains(t, summary, "Session: test-session-123")
	assert.Contains(t, summary, "Commands executed: 2")
	assert.Contains(t, summary, "Current project: FN")
	assert.Contains(t, summary, "Current sprint: Sprint 23")
	assert.Contains(t, summary, "Current space: MZN")
	assert.Contains(t, summary, "Recent assets: [Auth Payment]")
	assert.Contains(t, summary, "Recent tasks: [FN-123]")
}

func TestContext_ThreadSafety(t *testing.T) {
	ctx := NewContext("test-session")
	done := make(chan bool, 3)

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			ctx.SetVariable("key1", i)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			ctx.UpdateAssetContext("Asset", struct{}{})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			ctx.AddCommand(Command{ID: "cmd"})
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// Just verify no panic occurred
	assert.NotNil(t, ctx)
}
