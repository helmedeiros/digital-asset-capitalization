package formatting

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
)

func TestFormatSprintList_PlainText(t *testing.T) {
	formatter := NewOutputFormatter()
	sprints := []ports.Sprint{
		{
			ID:        "1",
			Name:      "Sprint 1",
			State:     "active",
			StartDate: "2025-04-01T00:00:00Z",
			EndDate:   "2025-04-15T00:00:00Z",
			Goal:      "Goal 1",
		},
	}
	boards := []ports.BoardInfo{
		{ID: 1, Name: "Board 1", Type: "scrum", HasSprints: true},
		{ID: 2, Name: "Board 2", Type: "kanban", HasSprints: false},
	}
	output := formatter.FormatSprintList("FN", "Q2 2025", sprints, boards)

	assert.Contains(t, output, "Sprint List: FN (Q2 2025)")
	assert.Contains(t, output, "Boards:")
	assert.Contains(t, output, "• Board 1 (scrum) - active")
	assert.Contains(t, output, "• Board 2 (kanban) - no sprints")
	assert.Contains(t, output, "Found 1 sprints:")
	assert.Contains(t, output, "1. Sprint 1")
	assert.Contains(t, output, "[active]")
	assert.Contains(t, output, "Apr 1-15")
	assert.Contains(t, output, "Goal 1")
}

func TestFormatSprintList_NoSprints(t *testing.T) {
	formatter := NewOutputFormatter()
	output := formatter.FormatSprintList("FN", "Q2 2025", nil, nil)
	assert.Contains(t, output, "No sprints found for the specified period.")
}

func TestFormatBoardSummary_PlainText(t *testing.T) {
	formatter := NewOutputFormatter()
	boards := []ports.BoardInfo{
		{ID: 1, Name: "Board 1", Type: "scrum", HasSprints: true},
		{ID: 2, Name: "Board 2", Type: "kanban", HasSprints: false},
	}
	output := formatter.FormatBoardSummary(boards)
	assert.Contains(t, output, "Board Summary:")
	assert.Contains(t, output, "• 1 Scrum boards (support sprints)")
	assert.Contains(t, output, "• 1 Kanban boards (no sprints)")
	assert.Contains(t, output, "• 1 boards with sprints found")
}

func TestFormatDate_PlainText(t *testing.T) {
	formatter := NewOutputFormatter()
	assert.Equal(t, "Jan 02, 2025", formatter.formatDate("2025-01-02T00:00:00Z"))
	assert.Equal(t, "N/A", formatter.formatDate(""))
	assert.Equal(t, "invalid-date", formatter.formatDate("invalid-date"))
}

func TestFormatInfo_Success_Warning(t *testing.T) {
	formatter := NewOutputFormatter()
	assert.Equal(t, "info", formatter.FormatInfo("info"))
	assert.Equal(t, "success", formatter.FormatSuccess("success"))
	assert.Equal(t, "warn", formatter.FormatWarning("warn"))
}

func TestFormatDateRange(t *testing.T) {
	formatter := NewOutputFormatter()

	// Test no dates
	assert.Equal(t, "No dates", formatter.formatDateRange("", ""))

	// Test only start date
	assert.Equal(t, "Starts Apr 1", formatter.formatDateRange("2025-04-01T00:00:00Z", ""))

	// Test only end date
	assert.Equal(t, "Ends Apr 15", formatter.formatDateRange("", "2025-04-15T00:00:00Z"))

	// Test same month
	assert.Equal(t, "Apr 1-15", formatter.formatDateRange("2025-04-01T00:00:00Z", "2025-04-15T00:00:00Z"))

	// Test different months
	assert.Equal(t, "Apr 1 - May 1", formatter.formatDateRange("2025-04-01T00:00:00Z", "2025-05-01T00:00:00Z"))

	// Test invalid dates
	assert.Equal(t, "invalid1 - invalid2", formatter.formatDateRange("invalid1", "invalid2"))
}

func TestTruncateString(t *testing.T) {
	formatter := NewOutputFormatter()

	// Test short string
	assert.Equal(t, "short", formatter.truncateString("short", 10))

	// Test truncation
	assert.Equal(t, "very lo...", formatter.truncateString("very long string", 10))

	// Test exact length
	assert.Equal(t, "exact", formatter.truncateString("exact", 5))
}

func TestFormatGoal(t *testing.T) {
	formatter := NewOutputFormatter()

	// Test empty goal
	assert.Equal(t, "", formatter.formatGoal(""))

	// Test simple goal
	assert.Equal(t, "Simple goal", formatter.formatGoal("Simple goal"))

	// Test goal with newlines
	assert.Equal(t, "Goal with spaces", formatter.formatGoal("Goal\nwith\nspaces"))

	// Test goal with leading dashes
	assert.Equal(t, "Clean goal", formatter.formatGoal("- 1. Clean goal"))

	// Test goal with multiple spaces
	assert.Equal(t, "Multiple spaces", formatter.formatGoal("Multiple   spaces"))
}

func TestWrapText(t *testing.T) {
	formatter := NewOutputFormatter()

	// Test short text
	assert.Equal(t, "short", formatter.wrapText("short", 80, "    "))

	// Test wrapping
	result := formatter.wrapText("This is a very long text that needs to be wrapped", 20, "    ")
	assert.Contains(t, result, "This is a very long")
	assert.Contains(t, result, "\n    text that needs")
	assert.Contains(t, result, "\n    to be wrapped")

	// Test single word longer than width
	result = formatter.wrapText("supercalifragilisticexpialidocious", 10, "  ")
	assert.Equal(t, "supercalifragilisticexpialidocious", result)
}

func TestIsSameMonth(t *testing.T) {
	formatter := NewOutputFormatter()

	// Test same month
	assert.True(t, formatter.isSameMonth("2025-04-01T00:00:00Z", "2025-04-15T00:00:00Z"))

	// Test different months
	assert.False(t, formatter.isSameMonth("2025-04-01T00:00:00Z", "2025-05-01T00:00:00Z"))

	// Test invalid dates
	assert.False(t, formatter.isSameMonth("invalid1", "invalid2"))
	assert.False(t, formatter.isSameMonth("2025-04-01T00:00:00Z", "invalid"))
}

func TestFormatBoardSummary_EmptyBoards(t *testing.T) {
	formatter := NewOutputFormatter()
	output := formatter.FormatBoardSummary([]ports.BoardInfo{})
	assert.Contains(t, output, "Board Summary:")
	assert.Contains(t, output, "• 0 Scrum boards")
	assert.Contains(t, output, "• 0 Kanban boards")
	assert.Contains(t, output, "• 0 boards with sprints")
}

func TestFormatSprintList_EmptyBoards(t *testing.T) {
	formatter := NewOutputFormatter()
	sprints := []ports.Sprint{
		{
			ID:        "1",
			Name:      "Sprint 1",
			State:     "active",
			StartDate: "2025-04-01T00:00:00Z",
			EndDate:   "2025-04-15T00:00:00Z",
			Goal:      "",
		},
	}
	output := formatter.FormatSprintList("FN", "Q2 2025", sprints, []ports.BoardInfo{})
	assert.Contains(t, output, "Sprint List: FN (Q2 2025)")
	assert.Contains(t, output, "Found 1 sprints:")
	assert.NotContains(t, output, "Boards:")
}
