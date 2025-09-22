package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	sprintPorts "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain/ports"
)

// UserInput implements UserInput interface for command-line interaction
type UserInput struct {
	reader *bufio.Reader
}

// NewUserInput creates a new UserInput instance
func NewUserInput() *UserInput {
	return &UserInput{
		reader: bufio.NewReader(os.Stdin),
	}
}

// Confirm asks the user for a yes/no confirmation via command line
func (ui *UserInput) Confirm(format string, args ...interface{}) (bool, error) {
	// Print the formatted message
	fmt.Printf(format+" (y/n): ", args...)

	// Read user input
	input, err := ui.reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read user input: %w", err)
	}

	// Clean and normalize the input
	input = strings.TrimSpace(strings.ToLower(input))

	// Check for valid responses
	switch input {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid input: %s. Please answer with 'y' or 'n'", input)
	}
}

// SelectSprint presents sprint options to user and returns selected sprint
func (ui *UserInput) SelectSprint(candidates []ports.SprintCandidate) (*sprintPorts.Sprint, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no sprint candidates provided")
	}

	// Display header
	fmt.Printf("❌ Sprint not found exactly.\n")
	fmt.Printf("🔍 Found similar sprints:\n\n")

	// Display candidates
	for i, candidate := range candidates {
		dateRange := ""
		if candidate.Sprint.StartDate != "" && candidate.Sprint.EndDate != "" {
			startDate := ui.formatDate(candidate.Sprint.StartDate)
			endDate := ui.formatDate(candidate.Sprint.EndDate)
			if startDate != "" && endDate != "" {
				dateRange = fmt.Sprintf(" - %s → %s", startDate, endDate)
			}
		}

		status := ""
		if candidate.Sprint.State != "" {
			status = fmt.Sprintf(" [%s]", candidate.Sprint.State)
		}

		fmt.Printf("%d) %s (ID: %s)%s%s\n", i+1, candidate.Sprint.Name, candidate.Sprint.ID, dateRange, status)
		if candidate.Reason != "" {
			fmt.Printf("   📝 Match reason: %s\n", candidate.Reason)
		}
		fmt.Println()
	}

	// Prompt for selection
	fmt.Printf("Select sprint (1-%d) or press Enter to cancel: ", len(candidates))

	// Read user input
	input, err := ui.reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read user input: %w", err)
	}

	// Clean input
	input = strings.TrimSpace(input)

	// Handle empty input (cancel)
	if input == "" {
		fmt.Println("🚫 Sprint selection cancelled")
		return nil, nil
	}

	// Parse selection
	selection, err := strconv.Atoi(input)
	if err != nil {
		return nil, fmt.Errorf("invalid selection: %s. Please enter a number between 1 and %d", input, len(candidates))
	}

	// Validate range
	if selection < 1 || selection > len(candidates) {
		return nil, fmt.Errorf("invalid selection: %d. Please enter a number between 1 and %d", selection, len(candidates))
	}

	// Return selected sprint
	selectedCandidate := candidates[selection-1]
	fmt.Printf("✅ Using %s (ID: %s)\n", selectedCandidate.Sprint.Name, selectedCandidate.Sprint.ID)

	return &selectedCandidate.Sprint, nil
}

// formatDate attempts to format a date string for display
func (ui *UserInput) formatDate(dateStr string) string {
	if dateStr == "" {
		return ""
	}

	// Try common date formats
	formats := []string{
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t.Format("Jan 02, 2006")
		}
	}

	// If parsing fails, return as-is
	return dateStr
}
