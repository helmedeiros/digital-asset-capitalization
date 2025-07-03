package formatting

import (
	"fmt"
	"strings"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
)

// OutputFormatter handles formatting of sprint-related output
type OutputFormatter struct{}

// NewOutputFormatter creates a new output formatter
func NewOutputFormatter() *OutputFormatter {
	return &OutputFormatter{}
}

// FormatSprintList formats the sprint list output with plain text
func (f *OutputFormatter) FormatSprintList(project, period string, sprints []ports.Sprint, boardInfo []ports.BoardInfo) string {
	var output strings.Builder

	// Header
	fmt.Fprintf(&output, "🏃 Sprint List for Project %s (%s)\n", project, period)
	output.WriteString(strings.Repeat("=", 60) + "\n\n")

	// Board information
	if len(boardInfo) > 0 {
		fmt.Fprintf(&output, "📋 Boards Found:\n")
		for _, board := range boardInfo {
			if board.HasSprints {
				fmt.Fprintf(&output, "  ✅ %s (ID: %d) - %s\n", board.Name, board.ID, board.Type)
			} else {
				fmt.Fprintf(&output, "  ⚠️  %s (ID: %d) - %s (no sprints)\n", board.Name, board.ID, board.Type)
			}
		}
		output.WriteString("\n")
	}

	// Sprint information
	if len(sprints) == 0 {
		fmt.Fprintf(&output, "❌ No sprints found for the specified period.\n")
		return output.String()
	}

	fmt.Fprintf(&output, "📅 Sprints Found (%d):\n\n", len(sprints))

	for i, sprint := range sprints {
		// Sprint name and ID
		fmt.Fprintf(&output, "%d. %s (ID: %s)\n", i+1, sprint.Name, sprint.ID)

		// Dates
		startDate := f.formatDate(sprint.StartDate)
		endDate := f.formatDate(sprint.EndDate)
		fmt.Fprintf(&output, "   📅 %s → %s\n", startDate, endDate)

		// State
		fmt.Fprintf(&output, "   📊 State: %s\n", sprint.State)

		// Goal if available
		if sprint.Goal != "" {
			fmt.Fprintf(&output, "   🎯 Goal: %s\n", sprint.Goal)
		}

		output.WriteString("\n")
	}

	return output.String()
}

// FormatBoardSummary formats a summary of boards found
func (f *OutputFormatter) FormatBoardSummary(boards []ports.BoardInfo) string {
	var output strings.Builder

	fmt.Fprintf(&output, "📋 Board Summary:\n")

	scrumBoards := 0
	kanbanBoards := 0
	boardsWithSprints := 0

	for _, board := range boards {
		if board.Type == "scrum" {
			scrumBoards++
		} else if board.Type == "kanban" {
			kanbanBoards++
		}
		if board.HasSprints {
			boardsWithSprints++
		}
	}

	fmt.Fprintf(&output, "  • %d Scrum boards (support sprints)\n", scrumBoards)
	fmt.Fprintf(&output, "  • %d Kanban boards (no sprints)\n", kanbanBoards)
	fmt.Fprintf(&output, "  • %d boards with sprints found\n", boardsWithSprints)

	return output.String()
}

// formatDate formats a date string for better readability
func (f *OutputFormatter) formatDate(dateStr string) string {
	if dateStr == "" {
		return "N/A"
	}

	// Try to parse the date and format it nicely
	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return t.Format("Jan 02, 2006")
	}

	// If parsing fails, return the original string
	return dateStr
}

// FormatInfo formats informational messages
func (f *OutputFormatter) FormatInfo(message string) string {
	return message
}

// FormatSuccess formats success messages
func (f *OutputFormatter) FormatSuccess(message string) string {
	return message
}

// FormatWarning formats warning messages
func (f *OutputFormatter) FormatWarning(message string) string {
	return message
}
