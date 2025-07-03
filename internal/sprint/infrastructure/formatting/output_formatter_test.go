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

	assert.Contains(t, output, "🏃 Sprint List for Project FN (Q2 2025)")
	assert.Contains(t, output, "📋 Boards Found:")
	assert.Contains(t, output, "✅ Board 1 (ID: 1) - scrum")
	assert.Contains(t, output, "⚠️  Board 2 (ID: 2) - kanban (no sprints)")
	assert.Contains(t, output, "📅 Sprints Found (1):")
	assert.Contains(t, output, "1. Sprint 1 (ID: 1)")
	assert.Contains(t, output, "📅 Apr 01, 2025 → Apr 15, 2025")
	assert.Contains(t, output, "📊 State: active")
	assert.Contains(t, output, "🎯 Goal: Goal 1")
}

func TestFormatSprintList_NoSprints(t *testing.T) {
	formatter := NewOutputFormatter()
	output := formatter.FormatSprintList("FN", "Q2 2025", nil, nil)
	assert.Contains(t, output, "❌ No sprints found for the specified period.")
}

func TestFormatBoardSummary_PlainText(t *testing.T) {
	formatter := NewOutputFormatter()
	boards := []ports.BoardInfo{
		{ID: 1, Name: "Board 1", Type: "scrum", HasSprints: true},
		{ID: 2, Name: "Board 2", Type: "kanban", HasSprints: false},
	}
	output := formatter.FormatBoardSummary(boards)
	assert.Contains(t, output, "📋 Board Summary:")
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
