package ui

import (
	"fmt"
	"strings"
	"time"
)

// PromptSession manages the Claude Code-style prompt interface
type PromptSession struct {
	history   []PromptEntry
	maxWidth  int
	palette   ColorPalette
	showTime  bool
	userStyle string
}

// PromptEntry represents a single prompt/response entry
type PromptEntry struct {
	Timestamp time.Time
	Input     string
	Output    string
	Duration  time.Duration
	Success   bool
}

// NewPromptSession creates a new prompt session
func NewPromptSession(maxWidth int) *PromptSession {
	if maxWidth <= 0 {
		maxWidth = 80
	}

	return &PromptSession{
		history:   make([]PromptEntry, 0),
		maxWidth:  maxWidth,
		palette:   DefaultPalette(),
		showTime:  true,
		userStyle: "claude-code",
	}
}

// AddEntry adds a new prompt entry to the history
func (ps *PromptSession) AddEntry(input, output string, duration time.Duration, success bool) {
	entry := PromptEntry{
		Timestamp: time.Now(),
		Input:     input,
		Output:    output,
		Duration:  duration,
		Success:   success,
	}
	ps.history = append(ps.history, entry)
}

// RenderFullSession renders the complete session including all history
func (ps *PromptSession) RenderFullSession() string {
	var result strings.Builder

	// Render all historical entries
	for _, entry := range ps.history {
		result.WriteString(ps.renderHistoryEntry(entry))
		result.WriteString("\n")
	}

	return result.String()
}

// RenderCurrentPrompt renders the current input prompt (always at bottom)
func (ps *PromptSession) RenderCurrentPrompt() string {
	var result strings.Builder

	// Add some separation from previous content
	result.WriteString("\n")

	// Create the input prompt box
	prompt := ps.createInputPrompt()
	result.WriteString(prompt)

	return result.String()
}

// RenderHistoryEntry renders a single history entry in muted style
func (ps *PromptSession) renderHistoryEntry(entry PromptEntry) string {
	var result strings.Builder

	// Render the input in muted style (Claude Code style)
	inputLine := ps.renderMutedInput(entry.Input, entry.Timestamp)
	result.WriteString(inputLine)
	result.WriteString("\n")

	// Render the output
	if entry.Output != "" {
		outputLines := ps.renderOutput(entry.Output, entry.Success)
		result.WriteString(outputLines)
		result.WriteString("\n")
	}

	// Add duration info if available
	if entry.Duration > 0 && entry.Duration > 100*time.Millisecond {
		durationText := ps.renderDuration(entry.Duration)
		result.WriteString(durationText)
		result.WriteString("\n")
	}

	// Add visual separator
	result.WriteString(ps.renderSeparator())
	result.WriteString("\n")

	return result.String()
}

// renderMutedInput renders the input prompt in muted gray style (like Claude Code)
func (ps *PromptSession) renderMutedInput(input string, timestamp time.Time) string {
	var result strings.Builder

	// Create the prompt indicator in muted color
	promptSymbol := MutedText("> ")

	// Format timestamp if enabled
	timeStr := ""
	if ps.showTime {
		timeStr = MutedText(fmt.Sprintf("[%s] ", timestamp.Format("15:04:05")))
	}

	// Wrap input text in muted color
	mutedInput := MutedText(input)

	// Combine all parts
	result.WriteString(timeStr + promptSymbol + mutedInput)

	return result.String()
}

// renderOutput renders the output with appropriate styling
func (ps *PromptSession) renderOutput(output string, success bool) string {
	if !success {
		return ps.renderErrorOutput(output)
	}

	// Regular output - try to detect and format different types
	if ps.isStructuredData(output) {
		return ps.renderStructuredOutput(output)
	}

	return ps.renderPlainOutput(output)
}

// renderErrorOutput renders error output with error styling
func (ps *PromptSession) renderErrorOutput(output string) string {
	var result strings.Builder

	// Error indicator
	result.WriteString(ErrorText("ERROR: "))
	result.WriteString("\n")

	// Indent error message
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			result.WriteString("   ")
			result.WriteString(ErrorText(line))
			result.WriteString("\n")
		}
	}

	return result.String()
}

// renderStructuredOutput renders structured data with formatting
func (ps *PromptSession) renderStructuredOutput(output string) string {
	// This is a simplified version - in practice you'd parse JSON/YAML/etc
	var result strings.Builder

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			result.WriteString("\n")
			continue
		}

		// Detect key-value pairs
		if strings.Contains(line, ":") && !strings.HasPrefix(trimmed, "http") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				// Format as key: value with colors
				result.WriteString(Colorize(key+":", ColorPrimary))
				result.WriteString(" ")
				result.WriteString(ps.formatValue(value))
				result.WriteString("\n")
				continue
			}
		}

		// Regular line
		result.WriteString(line)
		result.WriteString("\n")
	}

	return result.String()
}

// renderPlainOutput renders plain text output
func (ps *PromptSession) renderPlainOutput(output string) string {
	return output
}

// formatValue formats a value based on its content
func (ps *PromptSession) formatValue(value string) string {
	// Detect different types of values and apply appropriate formatting

	// URLs
	if strings.HasPrefix(value, "http") {
		return Link(value)
	}

	// File paths
	if strings.Contains(value, "/") && (strings.Contains(value, ".") || strings.HasPrefix(value, "/")) {
		return Code(value)
	}

	// Numbers with units (money, percentages, etc.)
	if strings.Contains(value, "$") || strings.Contains(value, "EUR") || strings.Contains(value, "USD") {
		return Code(value)
	}

	if strings.Contains(value, "%") {
		return Code(value)
	}

	// Boolean-like values
	if value == "true" || value == "success" || value == "active" {
		return SuccessText(value)
	}

	if value == "false" || value == "failed" || value == "inactive" {
		return ErrorText(value)
	}

	// Default
	return value
}

// renderDuration renders execution duration
func (ps *PromptSession) renderDuration(duration time.Duration) string {
	durationText := fmt.Sprintf("Completed in %v", duration.Round(time.Millisecond))
	return "   " + MutedText(durationText)
}

// renderSeparator renders a visual separator between entries
func (ps *PromptSession) renderSeparator() string {
	return MutedText(strings.Repeat("─", ps.maxWidth/2))
}

// createInputPrompt creates the current input prompt
func (ps *PromptSession) createInputPrompt() string {
	var result strings.Builder

	// Create a modern-looking input box
	result.WriteString(ps.renderInputBox())

	return result.String()
}

// renderInputBox renders the input box UI
func (ps *PromptSession) renderInputBox() string {
	// Simple prompt without box drawing
	return PromptText("> ")
}

// RenderInputWithText renders the input box with entered text
func (ps *PromptSession) RenderInputWithText(input string) string {
	// Simple prompt with input, no box drawing
	return PromptText("> ") + input
}

// isStructuredData tries to detect if output is structured data
func (ps *PromptSession) isStructuredData(output string) bool {
	// Simple heuristics to detect structured data
	lines := strings.Split(output, "\n")
	colonCount := 0

	for _, line := range lines {
		if strings.Contains(line, ":") && !strings.HasPrefix(strings.TrimSpace(line), "http") {
			colonCount++
		}
	}

	// If more than 2 lines have colons, likely structured
	return colonCount > 2
}

// ClearScreen clears the terminal screen (platform-specific)
func (ps *PromptSession) ClearScreen() string {
	// ANSI escape sequence to clear screen
	return "\033[2J\033[H"
}

// SetShowTime configures whether to show timestamps
func (ps *PromptSession) SetShowTime(show bool) {
	ps.showTime = show
}

// SetMaxWidth sets the maximum width for the prompt session
func (ps *PromptSession) SetMaxWidth(width int) {
	if width > 0 {
		ps.maxWidth = width
	}
}

// GetHistory returns the prompt history
func (ps *PromptSession) GetHistory() []PromptEntry {
	return ps.history
}

// ClearHistory clears the prompt history
func (ps *PromptSession) ClearHistory() {
	ps.history = make([]PromptEntry, 0)
}

// RenderWelcome renders a Claude Code-style welcome message
func (ps *PromptSession) RenderWelcome(appName string) string {
	var result strings.Builder

	// Simple welcome without box drawing
	title := fmt.Sprintf("%s AI Console", appName)
	subtitle := "Ask me anything about your digital assets, tasks, or investments."
	helpText := "Type 'help' for guidance or 'exit' to quit."

	result.WriteString(BoldText(PrimaryText(title)))
	result.WriteString("\n")
	result.WriteString(subtitle)
	result.WriteString("\n")
	result.WriteString(MutedText(helpText))
	result.WriteString("\n\n")

	return result.String()
}
