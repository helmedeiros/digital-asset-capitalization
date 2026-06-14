package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewPromptSession(t *testing.T) {
	tests := []struct {
		name     string
		maxWidth int
		expected int
	}{
		{"default width", 0, 80},
		{"negative width", -10, 80},
		{"custom width", 120, 120},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := NewPromptSession(tt.maxWidth)
			assert.NotNil(t, ps)
			assert.Equal(t, tt.expected, ps.maxWidth)
			assert.Empty(t, ps.history)
			assert.True(t, ps.showTime)
			assert.Equal(t, "claude-code", ps.userStyle)
		})
	}
}

func TestAddEntry(t *testing.T) {
	ps := NewPromptSession(80)

	ps.AddEntry("test input", "test output", 100*time.Millisecond, true)

	assert.Len(t, ps.history, 1)
	assert.Equal(t, "test input", ps.history[0].Input)
	assert.Equal(t, "test output", ps.history[0].Output)
	assert.Equal(t, 100*time.Millisecond, ps.history[0].Duration)
	assert.True(t, ps.history[0].Success)
	assert.WithinDuration(t, time.Now(), ps.history[0].Timestamp, 1*time.Second)
}

func TestRenderWelcome(t *testing.T) {
	ps := NewPromptSession(80)

	welcome := ps.RenderWelcome("TestApp")

	assert.Contains(t, welcome, "TestApp AI Console")
	assert.Contains(t, welcome, "Ask me anything")
	assert.Contains(t, welcome, "Type 'help'")
}

func TestRenderInputWithText(t *testing.T) {
	ps := NewPromptSession(80)

	result := ps.RenderInputWithText("show assets")

	assert.Contains(t, result, ">")
	assert.Contains(t, result, "show assets")
}

func TestGetHistory(t *testing.T) {
	ps := NewPromptSession(80)

	ps.AddEntry("cmd1", "output1", 50*time.Millisecond, true)
	ps.AddEntry("cmd2", "output2", 100*time.Millisecond, false)

	history := ps.GetHistory()
	assert.Len(t, history, 2)
	assert.Equal(t, "cmd1", history[0].Input)
	assert.Equal(t, "cmd2", history[1].Input)
}

func TestClearHistory(t *testing.T) {
	ps := NewPromptSession(80)

	ps.AddEntry("cmd1", "output1", 50*time.Millisecond, true)
	ps.AddEntry("cmd2", "output2", 100*time.Millisecond, false)

	ps.ClearHistory()
	assert.Empty(t, ps.history)
}

func TestSetShowTime(t *testing.T) {
	ps := NewPromptSession(80)

	ps.SetShowTime(false)
	assert.False(t, ps.showTime)

	ps.SetShowTime(true)
	assert.True(t, ps.showTime)
}

func TestSetMaxWidth(t *testing.T) {
	ps := NewPromptSession(80)

	ps.SetMaxWidth(100)
	assert.Equal(t, 100, ps.maxWidth)

	// Test invalid width (should not change)
	ps.SetMaxWidth(0)
	assert.Equal(t, 100, ps.maxWidth)

	ps.SetMaxWidth(-10)
	assert.Equal(t, 100, ps.maxWidth)
}

func TestRenderMutedInput(t *testing.T) {
	ps := NewPromptSession(80)
	timestamp := time.Now()

	// Test with timestamp
	ps.showTime = true
	result := ps.renderMutedInput("test command", timestamp)
	assert.Contains(t, result, ">")
	assert.Contains(t, result, "test command")
	assert.Contains(t, result, timestamp.Format("15:04:05"))

	// Test without timestamp
	ps.showTime = false
	result = ps.renderMutedInput("test command", timestamp)
	assert.Contains(t, result, ">")
	assert.Contains(t, result, "test command")
	assert.NotContains(t, result, timestamp.Format("15:04:05"))
}

func TestIsStructuredData(t *testing.T) {
	ps := NewPromptSession(80)

	tests := []struct {
		name     string
		output   string
		expected bool
	}{
		{
			name: "structured data with colons",
			output: `name: Test Asset
status: active
type: service
description: A test asset`,
			expected: true,
		},
		{
			name: "URLs should not count",
			output: `url: https://example.com:8080
https://another.com:443
http://test.com:80`,
			expected: false,
		},
		{
			name:     "plain text",
			output:   "This is just plain text without structure",
			expected: false,
		},
		{
			name: "mixed content",
			output: `Some text here
key1: value1
key2: value2`,
			expected: false, // Only 2 lines with colons
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ps.isStructuredData(tt.output)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatValue(t *testing.T) {
	ps := NewPromptSession(80)

	tests := []struct {
		name  string
		value string
		check func(string) bool
	}{
		{
			name:  "URL",
			value: "https://example.com",
			check: func(s string) bool { return strings.Contains(s, "https://example.com") },
		},
		{
			name:  "file path",
			value: "/path/to/file.txt",
			check: func(s string) bool { return strings.Contains(s, "/path/to/file.txt") },
		},
		{
			name:  "money USD",
			value: "$100.00",
			check: func(s string) bool { return strings.Contains(s, "$100.00") },
		},
		{
			name:  "percentage",
			value: "85.5%",
			check: func(s string) bool { return strings.Contains(s, "85.5%") },
		},
		{
			name:  "boolean true",
			value: "true",
			check: func(s string) bool { return s != "" },
		},
		{
			name:  "boolean false",
			value: "false",
			check: func(s string) bool { return s != "" },
		},
		{
			name:  "regular text",
			value: "regular value",
			check: func(s string) bool { return s == "regular value" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ps.formatValue(tt.value)
			assert.True(t, tt.check(result))
		})
	}
}

func TestRenderHistoryEntry(t *testing.T) {
	ps := NewPromptSession(80)

	entry := PromptEntry{
		Timestamp: time.Now(),
		Input:     "test command",
		Output:    "test output",
		Duration:  150 * time.Millisecond,
		Success:   true,
	}

	result := ps.renderHistoryEntry(entry)

	assert.Contains(t, result, "test command")
	assert.Contains(t, result, "test output")
	assert.Contains(t, result, "150ms")
}

func TestRenderErrorOutput(t *testing.T) {
	ps := NewPromptSession(80)

	errorOutput := "Error: Something went wrong\nDetails: File not found"
	result := ps.renderErrorOutput(errorOutput)

	assert.Contains(t, result, "ERROR:")
	assert.Contains(t, result, "Something went wrong")
	assert.Contains(t, result, "File not found")
}

func TestRenderDuration(t *testing.T) {
	ps := NewPromptSession(80)

	duration := 250 * time.Millisecond
	result := ps.renderDuration(duration)

	assert.Contains(t, result, "Completed in")
	assert.Contains(t, result, "250ms")
}

func TestClearScreen(t *testing.T) {
	ps := NewPromptSession(80)

	result := ps.ClearScreen()
	assert.Equal(t, "\033[2J\033[H", result)
}

func TestRenderFullSession(t *testing.T) {
	ps := NewPromptSession(80)

	t.Run("empty history yields empty string", func(t *testing.T) {
		assert.Equal(t, "", ps.RenderFullSession())
	})

	t.Run("populated history renders each entry's input", func(t *testing.T) {
		ps.AddEntry("first input", "first output", 100*time.Millisecond, true)
		ps.AddEntry("second input", "second output", 200*time.Millisecond, false)
		got := ps.RenderFullSession()
		assert.Contains(t, got, "first input")
		assert.Contains(t, got, "second input")
	})
}

func TestRenderCurrentPrompt(t *testing.T) {
	ps := NewPromptSession(80)
	got := ps.RenderCurrentPrompt()
	assert.NotEmpty(t, got)
	assert.True(t, strings.HasPrefix(got, "\n"), "RenderCurrentPrompt should lead with a newline separator")
}
