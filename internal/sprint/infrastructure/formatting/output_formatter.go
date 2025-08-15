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

// FormatSprintList formats the sprint list output with clean, professional formatting
func (f *OutputFormatter) FormatSprintList(project, period string, sprints []ports.Sprint, boardInfo []ports.BoardInfo) string {
	var output strings.Builder

	// Clean header
	fmt.Fprintf(&output, "Sprint List: %s (%s)\n", project, period)
	output.WriteString(strings.Repeat("-", 50) + "\n\n")

	// Board information - simplified
	if len(boardInfo) > 0 {
		output.WriteString("Boards:\n")
		for _, board := range boardInfo {
			status := "active"
			if !board.HasSprints {
				status = "no sprints"
			}
			fmt.Fprintf(&output, "  • %s (%s) - %s\n", board.Name, board.Type, status)
		}
		output.WriteString("\n")
	}

	// Sprint information
	if len(sprints) == 0 {
		output.WriteString("No sprints found for the specified period.\n")
		return output.String()
	}

	fmt.Fprintf(&output, "Found %d sprints:\n\n", len(sprints))

	for i, sprint := range sprints {
		// Sprint header - clean and scannable
		fmt.Fprintf(&output, "%2d. %-20s [%s]  %s\n",
			i+1,
			f.truncateString(sprint.Name, 20),
			sprint.State,
			f.formatDateRange(sprint.StartDate, sprint.EndDate))

		// Goal - properly wrapped and indented
		if sprint.Goal != "" && strings.TrimSpace(sprint.Goal) != "" {
			goal := f.formatGoal(sprint.Goal)
			if goal != "" {
				fmt.Fprintf(&output, "    %s\n", goal)
			}
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

// formatDateRange formats start and end dates as a compact range
func (f *OutputFormatter) formatDateRange(startDateStr, endDateStr string) string {
	if startDateStr == "" && endDateStr == "" {
		return "No dates"
	}

	if startDateStr == "" {
		return "Ends " + f.formatShortDate(endDateStr)
	}

	if endDateStr == "" {
		return "Starts " + f.formatShortDate(startDateStr)
	}

	start := f.formatShortDate(startDateStr)
	end := f.formatShortDate(endDateStr)

	// If same month/year, show compact format
	if f.isSameMonth(startDateStr, endDateStr) {
		if startT, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			if endT, err := time.Parse(time.RFC3339, endDateStr); err == nil {
				return fmt.Sprintf("%s %d-%d", startT.Format("Jan"), startT.Day(), endT.Day())
			}
		}
	}

	return fmt.Sprintf("%s - %s", start, end)
}

// formatShortDate formats a date in compact format
func (f *OutputFormatter) formatShortDate(dateStr string) string {
	if dateStr == "" {
		return ""
	}

	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return t.Format("Jan 2")
	}

	return dateStr
}

// isSameMonth checks if two dates are in the same month/year
func (f *OutputFormatter) isSameMonth(date1Str, date2Str string) bool {
	t1, err1 := time.Parse(time.RFC3339, date1Str)
	t2, err2 := time.Parse(time.RFC3339, date2Str)

	if err1 != nil || err2 != nil {
		return false
	}

	return t1.Year() == t2.Year() && t1.Month() == t2.Month()
}

// truncateString truncates a string to the specified length
func (f *OutputFormatter) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// formatGoal cleans and formats sprint goals for better readability
func (f *OutputFormatter) formatGoal(goal string) string {
	if goal == "" {
		return ""
	}

	// Clean up the goal text
	goal = strings.TrimSpace(goal)

	// Replace multiple newlines with single space
	goal = strings.ReplaceAll(goal, "\n", " ")
	goal = strings.ReplaceAll(goal, "\r", " ")

	// Clean up multiple spaces
	for strings.Contains(goal, "  ") {
		goal = strings.ReplaceAll(goal, "  ", " ")
	}

	// Remove leading dashes and numbers that make it look messy
	goal = strings.TrimLeft(goal, "- 123456789.")
	goal = strings.TrimSpace(goal)

	// If too long, wrap it intelligently
	if len(goal) > 80 {
		return f.wrapText(goal, 80, "    ")
	}

	return goal
}

// wrapText wraps text to specified width with given prefix for continuation lines
func (f *OutputFormatter) wrapText(text string, width int, prefix string) string {
	if len(text) <= width {
		return text
	}

	var result strings.Builder
	words := strings.Fields(text)
	var currentLine strings.Builder
	lineLen := 0

	for _, word := range words {
		wordLen := len(word)

		// If adding this word would exceed the width, start a new line
		if lineLen+wordLen+1 > width && lineLen > 0 {
			result.WriteString(currentLine.String())
			result.WriteString("\n" + prefix)
			currentLine.Reset()
			lineLen = len(prefix)
		}

		if currentLine.Len() > 0 {
			currentLine.WriteString(" ")
			lineLen++
		}
		currentLine.WriteString(word)
		lineLen += wordLen
	}

	// Add the last line
	if currentLine.Len() > 0 {
		result.WriteString(currentLine.String())
	}

	return result.String()
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
