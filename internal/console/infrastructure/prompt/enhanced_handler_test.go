package prompt

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/application"
	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
)

// Test methods that don't require complex service mocking

func TestEnhancedHandler_EndSession_NoService(t *testing.T) {
	handler := NewEnhancedHandler(nil)
	handler.sessionContext = nil

	ctx := context.Background()
	err := handler.endSession(ctx)

	assert.NoError(t, err)
}

func TestEnhancedHandler_HandleEnhancedInput_EmptyInput(t *testing.T) {
	handler := NewEnhancedHandler(nil)
	ctx := context.Background()

	err := handler.handleEnhancedInput(ctx, "")
	assert.NoError(t, err)
}

func TestEnhancedHandler_HandleSpecialCommands_WithoutService(t *testing.T) {
	t.Cleanup(setupTestEnvironment(t))
	handler := NewEnhancedHandler(nil)

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"help command", "help", true},
		{"history command", "history", true},
		{"clear command", "clear", true},
		{"HELP uppercase", "HELP", true},
		{"regular command", "show assets", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.handleSpecialCommands(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnhancedHandler_AddToHistory_WithoutService(t *testing.T) {
	handler := NewEnhancedHandler(nil)

	// Test that addToHistory doesn't panic when there's no prompt session
	assert.NotPanics(t, func() {
		handler.addToHistory("test input", "test output", 0, true)
	})
}

func TestEnhancedHandler_SpinnerMethods_Standalone(t *testing.T) {
	handler := NewEnhancedHandler(nil)

	// Test spinner methods don't panic
	assert.NotPanics(t, func() {
		handler.startProcessingIndicator("Testing...")
	})

	assert.NotPanics(t, func() {
		handler.stopProcessingIndicator()
	})

	// Test stopping when no spinner is active
	assert.NotPanics(t, func() {
		handler.stopProcessingIndicator()
	})
}

// TestEnhancedHandler_Spinner_StartStopIsRaceFree asserts that
// startProcessingIndicator and stopProcessingIndicator do not race on
// h.spinner. Before the fix, the animate goroutine read h.spinner three
// times per tick while stopProcessingIndicator wrote h.spinner = nil from
// the caller's goroutine; the data race was reproducible under `go test
// -race`. Running start/stop multiple times here -- and especially within
// less than one tick interval -- exercises both the happy stop path and
// the case where the goroutine is still picking up the spinner pointer
// when stop fires.
func TestEnhancedHandler_Spinner_StartStopIsRaceFree(t *testing.T) {
	handler := NewEnhancedHandler(nil)

	for i := 0; i < 5; i++ {
		handler.startProcessingIndicator("Working...")
		handler.stopProcessingIndicator()
	}

	// Also drive a stop on an idle handler -- previously a no-op, must
	// stay a no-op.
	handler.stopProcessingIndicator()
}

func TestEnhancedHandler_DisplayMethods_Standalone(t *testing.T) {
	handler := NewEnhancedHandler(nil)

	// Test that all display methods don't panic
	assert.NotPanics(t, func() {
		handler.displayEnhancedWelcome()
	})

	assert.NotPanics(t, func() {
		handler.displayEnhancedGoodbye()
	})

	assert.NotPanics(t, func() {
		handler.displayEnhancedError("test error")
	})

	assert.NotPanics(t, func() {
		handler.displayEnhancedInfo("test info")
	})

	assert.NotPanics(t, func() {
		handler.displayEnhancedSuccess("test success")
	})

	assert.NotPanics(t, func() {
		handler.displayEnhancedHelp()
	})

	assert.NotPanics(t, func() {
		handler.clearScreen()
	})

	assert.NotPanics(t, func() {
		handler.displayCommandHistory()
	})
}

func TestEnhancedHandler_DisplayResult_Standalone(t *testing.T) {
	handler := NewEnhancedHandler(nil)

	testResults := []*application.ProcessResult{
		{
			Success: true,
			Output:  nil,
		},
		{
			Success: true,
			Output:  "simple string output",
		},
		{
			Success: true,
			Output: map[string]interface{}{
				"name":   "Test Asset",
				"status": "active",
			},
		},
		{
			Success: true,
			Output: []interface{}{
				map[string]interface{}{"name": "Asset1"},
				map[string]interface{}{"name": "Asset2"},
			},
		},
	}

	for i, result := range testResults {
		t.Run(fmt.Sprintf("result_%d", i), func(t *testing.T) {
			assert.NotPanics(t, func() {
				handler.displayEnhancedResult(result)
			})
		})
	}
}

func TestEnhancedHandler_DisplayContext_Standalone(t *testing.T) {
	handler := NewEnhancedHandler(nil)

	testContext := domain.NewContext("test-session")

	assert.NotPanics(t, func() {
		handler.displayEnhancedContext(testContext)
	})
}

func TestEnhancedHandler_DisplayClarification_Standalone(t *testing.T) {
	handler := NewEnhancedHandler(nil)

	assert.NotPanics(t, func() {
		handler.displayEnhancedClarification("Please clarify", []string{"option1", "option2"})
	})
}

func TestEnhancedHandler_CompletePromptWithInput_Standalone(t *testing.T) {
	handler := NewEnhancedHandler(nil)

	assert.NotPanics(t, func() {
		handler.completePromptWithInput("test input")
	})
}

func TestEnhancedHandler_DisplayMapOutput_Standalone(t *testing.T) {
	handler := NewEnhancedHandler(nil)

	testMaps := []map[string]interface{}{
		{
			"name":   "Test Asset",
			"status": "active",
		},
		{
			"assets": []interface{}{
				map[string]interface{}{"name": "Asset1"},
				map[string]interface{}{"name": "Asset2"},
			},
		},
		{}, // empty map
	}

	for i, testMap := range testMaps {
		t.Run(fmt.Sprintf("map_%d", i), func(t *testing.T) {
			assert.NotPanics(t, func() {
				handler.displayEnhancedMapOutput(testMap)
			})
		})
	}
}

func TestEnhancedHandler_DisplayListOutput_Standalone(t *testing.T) {
	handler := NewEnhancedHandler(nil)

	testLists := [][]interface{}{
		{"item1", "item2", "item3"},
		{
			map[string]interface{}{"command": "help", "description": "Show help"},
			map[string]interface{}{"command": "exit", "description": "Exit console"},
		},
		{}, // empty list
	}

	for i, testList := range testLists {
		t.Run(fmt.Sprintf("list_%d", i), func(t *testing.T) {
			assert.NotPanics(t, func() {
				handler.displayEnhancedListOutput(testList)
			})
		})
	}
}

func TestEnhancedHandler_AddToCommandHistory(t *testing.T) {
	t.Run("empty input is skipped", func(t *testing.T) {
		h := NewEnhancedHandler(nil)
		h.addToCommandHistory("")
		assert.Empty(t, h.commandHistory)
	})

	t.Run("consecutive duplicate is skipped", func(t *testing.T) {
		h := NewEnhancedHandler(nil)
		h.addToCommandHistory("show assets")
		h.addToCommandHistory("show assets")
		assert.Equal(t, []string{"show assets"}, h.commandHistory)
	})

	t.Run("non-consecutive duplicate is recorded", func(t *testing.T) {
		h := NewEnhancedHandler(nil)
		h.addToCommandHistory("show assets")
		h.addToCommandHistory("list tasks")
		h.addToCommandHistory("show assets")
		assert.Equal(t, []string{"show assets", "list tasks", "show assets"}, h.commandHistory)
	})

	t.Run("caps history at 100 entries with FIFO eviction", func(t *testing.T) {
		h := NewEnhancedHandler(nil)
		for i := 0; i < 105; i++ {
			h.addToCommandHistory(fmt.Sprintf("cmd-%03d", i))
		}
		assert.Len(t, h.commandHistory, 100)
		assert.Equal(t, "cmd-005", h.commandHistory[0], "earliest non-evicted entry")
		assert.Equal(t, "cmd-104", h.commandHistory[99], "most recent entry")
	})

	t.Run("resets historyIndex on insert", func(t *testing.T) {
		h := NewEnhancedHandler(nil)
		h.historyIndex = 7
		h.addToCommandHistory("show assets")
		assert.Equal(t, -1, h.historyIndex)
	})
}

func TestEnhancedHandler_FormatResultForHistory(t *testing.T) {
	h := NewEnhancedHandler(nil)

	tests := []struct {
		name   string
		result *application.ProcessResult
		want   string
	}{
		{
			name:   "nil output reports generic success",
			result: &application.ProcessResult{Success: true, Output: nil},
			want:   "Command executed successfully",
		},
		{
			name: "map with message returns that message",
			result: &application.ProcessResult{Output: map[string]interface{}{
				"message": "Asset created",
				"id":      "42",
			}},
			want: "Asset created",
		},
		{
			name: "map without message reports field count",
			result: &application.ProcessResult{Output: map[string]interface{}{
				"id":     "42",
				"status": "active",
			}},
			want: "Returned 2 fields",
		},
		{
			name:   "slice reports item count",
			result: &application.ProcessResult{Output: []interface{}{"a", "b", "c"}},
			want:   "Returned 3 items",
		},
		{
			name:   "string is returned as-is",
			result: &application.ProcessResult{Output: "hello"},
			want:   "hello",
		},
		{
			name:   "other types fall through to fmt %v",
			result: &application.ProcessResult{Output: 42},
			want:   "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, h.formatResultForHistory(tt.result))
		})
	}
}

func TestEnhancedHandler_FormatKeyName(t *testing.T) {
	h := NewEnhancedHandler(nil)

	tests := []struct {
		in   string
		want string
	}{
		{"name", "Name"},
		{"created_at", "Created At"},
		{"total_investment", "Total Investment"},
		{"", ""},
		{"already_Capitalized_INPUT", "Already Capitalized Input"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, h.formatKeyName(tt.in))
		})
	}
}

func TestEnhancedHandler_CreateGenericTable(t *testing.T) {
	h := NewEnhancedHandler(nil)

	t.Run("produces one column per key in sorted order with humanized headers", func(t *testing.T) {
		table := h.createGenericTable(map[string]interface{}{
			"updated_at": "2026-06-14",
			"name":       "Asset A",
			"status":     "active",
		})
		require.NotNil(t, table)
		require.Len(t, table.Columns, 3)

		// SortMapKeys yields lexical order: name, status, updated_at.
		assert.Equal(t, []string{"name", "status", "updated_at"},
			[]string{table.Columns[0].Key, table.Columns[1].Key, table.Columns[2].Key})
		assert.Equal(t, []string{"Name", "Status", "Updated At"},
			[]string{table.Columns[0].Header, table.Columns[1].Header, table.Columns[2].Header})

		for _, col := range table.Columns {
			assert.Equal(t, "left", col.Align)
		}
	})

	t.Run("assigns a formatter to status/state columns", func(t *testing.T) {
		table := h.createGenericTable(map[string]interface{}{
			"name":   "Asset A",
			"status": "active",
		})
		statusCol := table.Columns[1]
		require.Equal(t, "status", statusCol.Key)
		assert.NotNil(t, statusCol.Formatter, "status should get the status formatter")
	})

	t.Run("assigns a date formatter to timestamp columns", func(t *testing.T) {
		table := h.createGenericTable(map[string]interface{}{
			"created_at": "2026-06-14",
			"id":         "1",
		})
		// created_at sorts before id.
		require.Equal(t, "created_at", table.Columns[0].Key)
		assert.NotNil(t, table.Columns[0].Formatter)
	})

	t.Run("money branch only matches exact-keyed money fields (current behavior)", func(t *testing.T) {
		// The production switch matches on exact lowered key ("cost",
		// "investment", "price", "amount"), then guards on key containing
		// "total". Since none of those exact keys contain "total", the
		// MoneyFormatter branch is effectively unreachable today --
		// neither "cost" nor "total_cost" gets a formatter. This test
		// pins that observable behavior so a future refactor of the
		// switch shape can't quietly change it.
		table := h.createGenericTable(map[string]interface{}{
			"cost":       1.0,
			"total_cost": 2.0,
			"name":       "x",
		})
		require.Len(t, table.Columns, 3)
		byKey := map[string]func(interface{}) string{}
		for _, col := range table.Columns {
			byKey[col.Key] = col.Formatter
		}
		assert.Nil(t, byKey["cost"])
		assert.Nil(t, byKey["total_cost"])
		assert.Nil(t, byKey["name"])
	})
}

func TestEnhancedHandler_CreateTableForData(t *testing.T) {
	h := NewEnhancedHandler(nil)

	// Each branch in createTableForData dispatches to a different factory
	// method. We can't distinguish factory tables by identity, but we can
	// distinguish the generic fallback by its column count + key set, which
	// is what we assert for the generic case. For the typed branches we
	// just assert that the dispatch produced a table at all -- the factory
	// methods themselves are covered by ui package tests.
	tests := []struct {
		name   string
		sample map[string]interface{}
	}{
		{"asset-shaped", map[string]interface{}{"name": "A", "status": "active"}},
		{"task-shaped", map[string]interface{}{"key": "TASK-1", "summary": "do thing"}},
		{"investment-shaped via asset+investment", map[string]interface{}{"asset": "A", "investment": 10.0}},
		{"investment-shaped via total_investment", map[string]interface{}{"total_investment": 10.0}},
		{"engineer-shaped", map[string]interface{}{"name": "Eng", "level": "L4", "hours": 40.0, "cost": 1000.0}},
		{"sprint-shaped via dates", map[string]interface{}{"name": "Sprint 1", "start_date": "x", "end_date": "y"}},
		{"sprint-shaped via state+goal", map[string]interface{}{"state": "active", "goal": "ship"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := h.createTableForData(tt.sample)
			require.NotNil(t, table)
			assert.NotEmpty(t, table.Columns, "branch should produce a table with columns")
		})
	}

	t.Run("unknown shape falls through to createGenericTable", func(t *testing.T) {
		sample := map[string]interface{}{"foo": "bar", "baz": 1}
		table := h.createTableForData(sample)
		require.NotNil(t, table)
		require.Len(t, table.Columns, len(sample), "generic table has one column per key")
		// Verify keys round-trip through createGenericTable
		keys := map[string]bool{}
		for _, col := range table.Columns {
			keys[col.Key] = true
		}
		assert.True(t, keys["foo"] && keys["baz"])
	})
}

func TestEnhancedHandler_DisplayEnhancedCommandHelp(t *testing.T) {
	t.Cleanup(setupTestEnvironment(t))
	h := NewEnhancedHandler(nil)

	tests := []struct {
		name     string
		commands []interface{}
	}{
		{"empty list", []interface{}{}},
		{
			"command without description or examples",
			[]interface{}{map[string]interface{}{"command": "exit"}},
		},
		{
			"command with description only",
			[]interface{}{map[string]interface{}{"command": "help", "description": "Show help"}},
		},
		{
			"command with description and examples",
			[]interface{}{map[string]interface{}{
				"command":     "list",
				"description": "List assets",
				"examples":    []interface{}{"list assets", "list assets --filter active"},
			}},
		},
		{
			"command name is a long string (exercises padding clamp)",
			[]interface{}{map[string]interface{}{
				"command":     strings.Repeat("x", 40),
				"description": "Long-named command",
			}},
		},
		{
			"malformed entries are skipped without panic",
			[]interface{}{
				"not a map",
				map[string]interface{}{"description": "missing command key"},
				map[string]interface{}{"command": 42, "description": "command is not a string"},
				map[string]interface{}{"command": "ok", "description": "fine"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				h.displayEnhancedCommandHelp(tt.commands)
			})
		})
	}
}

func TestEnhancedHandler_DisplayEnhancedMapOutput_AllBranches(t *testing.T) {
	t.Cleanup(setupTestEnvironment(t))
	h := NewEnhancedHandler(nil)

	tests := []struct {
		name   string
		output map[string]interface{}
	}{
		{
			"tabular data (array of objects) triggers displayAsTable",
			map[string]interface{}{
				"assets": []interface{}{
					map[string]interface{}{"name": "A", "status": "active"},
					map[string]interface{}{"name": "B", "status": "archived"},
				},
			},
		},
		{
			"single asset detail triggers displayAssetDetail",
			map[string]interface{}{
				"name":        "Asset A",
				"id":          "42",
				"description": "primary asset",
				"status":      "active",
			},
		},
		{
			"map with message key prints the message and continues",
			map[string]interface{}{
				"message": "Asset created",
				"other":   "field",
			},
		},
		{
			"map with commands key dispatches to displayEnhancedCommandHelp",
			map[string]interface{}{
				"commands": []interface{}{
					map[string]interface{}{"command": "help", "description": "Show help"},
				},
			},
		},
		{
			"map with non-string message value falls through without crash",
			map[string]interface{}{
				"message": 42, // not a string -- branch type assertion fails
				"other":   "field",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() { h.displayEnhancedMapOutput(tt.output) })
		})
	}
}

func TestEnhancedHandler_DisplayEnhancedListOutput_AllBranches(t *testing.T) {
	t.Cleanup(setupTestEnvironment(t))
	h := NewEnhancedHandler(nil)

	tests := []struct {
		name   string
		output []interface{}
	}{
		{
			"list of objects with table-shape fields triggers displayListAsTable",
			[]interface{}{
				map[string]interface{}{"name": "A", "id": "1", "status": "active"},
				map[string]interface{}{"name": "B", "id": "2", "status": "archived"},
			},
		},
		{
			"list of objects without table-shape fields renders as numbered list",
			[]interface{}{
				map[string]interface{}{"foo": "bar"},
				map[string]interface{}{"baz": "qux"},
			},
		},
		{
			"list of scalars renders as numbered list",
			[]interface{}{"one", "two", "three"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() { h.displayEnhancedListOutput(tt.output) })
		})
	}
}

func TestEnhancedHandler_DisplayEnhancedResult_DurationBranch(t *testing.T) {
	t.Cleanup(setupTestEnvironment(t))
	h := NewEnhancedHandler(nil)

	tests := []struct {
		name   string
		result *application.ProcessResult
	}{
		{
			"result with significant duration prints the elapsed line",
			&application.ProcessResult{
				Output:   "ok",
				Duration: 250 * time.Millisecond,
			},
		},
		{
			"string output path",
			&application.ProcessResult{Output: "plain string"},
		},
		{
			"default case for unsupported output type",
			&application.ProcessResult{Output: 42},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotPanics(t, func() { h.displayEnhancedResult(tt.result) })
		})
	}
}

func TestEnhancedHandler_DisplayInteractivePrompt(t *testing.T) {
	t.Cleanup(setupTestEnvironment(t))
	h := NewEnhancedHandler(nil)
	assert.NotPanics(t, func() { h.displayInteractivePrompt() })
}

func TestEnhancedHandler_HandleEnhancedContextCommand_EarlyReturns(t *testing.T) {
	t.Cleanup(setupTestEnvironment(t))
	h := NewEnhancedHandler(nil)
	ctx := context.Background()

	t.Run("nil command returns nil", func(t *testing.T) {
		err := h.handleEnhancedContextCommand(ctx, &application.ProcessResult{Command: nil})
		assert.NoError(t, err)
	})

	t.Run("single-word interpreted returns nil before service call", func(t *testing.T) {
		result := &application.ProcessResult{
			Command: &domain.Command{Interpreted: "context"},
		}
		err := h.handleEnhancedContextCommand(ctx, result)
		assert.NoError(t, err)
	})

	t.Run("unknown action falls through switch without service call", func(t *testing.T) {
		result := &application.ProcessResult{
			Command: &domain.Command{Interpreted: "context unknownaction"},
		}
		err := h.handleEnhancedContextCommand(ctx, result)
		assert.NoError(t, err)
	})
}

func TestHandler_HandleContextCommand_EarlyReturns(t *testing.T) {
	t.Cleanup(setupTestEnvironment(t))
	h := &Handler{promptStyle: DefaultStyle()}
	ctx := context.Background()

	t.Run("nil command returns nil", func(t *testing.T) {
		err := h.handleContextCommand(ctx, &application.ProcessResult{Command: nil})
		assert.NoError(t, err)
	})

	t.Run("single-word interpreted returns nil before service call", func(t *testing.T) {
		result := &application.ProcessResult{
			Command: &domain.Command{Interpreted: "context"},
		}
		err := h.handleContextCommand(ctx, result)
		assert.NoError(t, err)
	})

	t.Run("unknown action falls through switch without service call", func(t *testing.T) {
		result := &application.ProcessResult{
			Command: &domain.Command{Interpreted: "context whatever"},
		}
		err := h.handleContextCommand(ctx, result)
		assert.NoError(t, err)
	})
}

func TestHandler_HandleInput_EmptyInput(t *testing.T) {
	h := &Handler{promptStyle: DefaultStyle()}

	err := h.handleInput(context.Background(), "")
	assert.NoError(t, err)
}

func TestEnhancedHandler_DisplayCommandHistory_PopulatedAndOverflow(t *testing.T) {
	t.Cleanup(setupTestEnvironment(t))

	t.Run("populated history renders without panic", func(t *testing.T) {
		h := NewEnhancedHandler(nil)
		for i := 0; i < 5; i++ {
			h.commandHistory = append(h.commandHistory, fmt.Sprintf("cmd-%d", i))
		}
		assert.NotPanics(t, func() { h.displayCommandHistory() })
	})

	t.Run("history with more than 20 entries triggers the trimming branch", func(t *testing.T) {
		h := NewEnhancedHandler(nil)
		for i := 0; i < 25; i++ {
			h.commandHistory = append(h.commandHistory, fmt.Sprintf("cmd-%02d", i))
		}
		assert.NotPanics(t, func() { h.displayCommandHistory() })
	})
}
