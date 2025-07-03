package infrastructure

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCLIUserInteraction(t *testing.T) {
	ui := NewCLIUserInteraction()
	assert.NotNil(t, ui)
}

func TestCLIUserInteraction_DisplayMessages(t *testing.T) {
	t.Run("should display message", func(t *testing.T) {
		var output bytes.Buffer
		ui := NewCLIUserInteractionWithIO(nil, &output)

		ui.DisplayMessage("Test message")

		assert.Contains(t, output.String(), "Test message")
	})

	t.Run("should display success message", func(t *testing.T) {
		var output bytes.Buffer
		ui := NewCLIUserInteractionWithIO(nil, &output)

		ui.DisplaySuccess("Success message")

		assert.Contains(t, output.String(), "Success message")
		assert.Contains(t, output.String(), "✅") // Success indicator
	})

	t.Run("should display warning message", func(t *testing.T) {
		var output bytes.Buffer
		ui := NewCLIUserInteractionWithIO(nil, &output)

		ui.DisplayWarning("Warning message")

		assert.Contains(t, output.String(), "Warning message")
		assert.Contains(t, output.String(), "⚠️") // Warning indicator
	})

	t.Run("should display error message", func(t *testing.T) {
		var output bytes.Buffer
		ui := NewCLIUserInteractionWithIO(nil, &output)

		ui.DisplayError("Error message")

		assert.Contains(t, output.String(), "Error message")
		assert.Contains(t, output.String(), "❌") // Error indicator
	})
}

func TestCLIUserInteraction_PromptString(t *testing.T) {
	t.Run("should prompt for string input", func(t *testing.T) {
		// Create a mock input/output
		input := "test input"
		mockInput := strings.NewReader(input + "\n")

		var output bytes.Buffer
		ui := NewCLIUserInteractionWithIO(mockInput, &output)

		result, err := ui.PromptString("Enter value:")

		require.NoError(t, err)
		assert.Equal(t, "test input", result)
		assert.Contains(t, output.String(), "Enter value:")
	})

	t.Run("should handle empty input", func(t *testing.T) {
		mockInput := strings.NewReader("\n")
		var output bytes.Buffer
		ui := NewCLIUserInteractionWithIO(mockInput, &output)

		result, err := ui.PromptString("Enter value:")

		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestCLIUserInteraction_PromptStringWithDefault(t *testing.T) {
	t.Run("should return input when provided", func(t *testing.T) {
		mockInput := strings.NewReader("custom value\n")
		var output bytes.Buffer
		ui := NewCLIUserInteractionWithIO(mockInput, &output)

		result, err := ui.PromptStringWithDefault("Enter value:", "default")

		require.NoError(t, err)
		assert.Equal(t, "custom value", result)
		assert.Contains(t, output.String(), "[default]")
	})

	t.Run("should return default when input is empty", func(t *testing.T) {
		mockInput := strings.NewReader("\n")
		var output bytes.Buffer
		ui := NewCLIUserInteractionWithIO(mockInput, &output)

		result, err := ui.PromptStringWithDefault("Enter value:", "default")

		require.NoError(t, err)
		assert.Equal(t, "default", result)
	})
}

func TestCLIUserInteraction_PromptConfirm(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"should return true for y", "y\n", true},
		{"should return true for Y", "Y\n", true},
		{"should return true for yes", "yes\n", true},
		{"should return true for YES", "YES\n", true},
		{"should return false for n", "n\n", false},
		{"should return false for N", "N\n", false},
		{"should return false for no", "no\n", false},
		{"should return false for NO", "NO\n", false},
		{"should return false for empty", "\n", false},
		{"should return false for invalid", "maybe\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockInput := strings.NewReader(tt.input)
			var output bytes.Buffer
			ui := NewCLIUserInteractionWithIO(mockInput, &output)

			result, err := ui.PromptConfirm("Confirm?")

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
			assert.Contains(t, output.String(), "[y/N]")
		})
	}
}

func TestCLIUserInteraction_PromptSelect(t *testing.T) {
	options := []string{"option1", "option2", "option3"}

	t.Run("should return selected option by number", func(t *testing.T) {
		mockInput := strings.NewReader("2\n")
		var output bytes.Buffer
		ui := NewCLIUserInteractionWithIO(mockInput, &output)

		result, err := ui.PromptSelect("Choose:", options)

		require.NoError(t, err)
		assert.Equal(t, "option2", result)
		assert.Contains(t, output.String(), "1) option1")
		assert.Contains(t, output.String(), "2) option2")
		assert.Contains(t, output.String(), "3) option3")
	})

	t.Run("should return selected option by name", func(t *testing.T) {
		mockInput := strings.NewReader("option3\n")
		var output bytes.Buffer
		ui := NewCLIUserInteractionWithIO(mockInput, &output)

		result, err := ui.PromptSelect("Choose:", options)

		require.NoError(t, err)
		assert.Equal(t, "option3", result)
	})

	t.Run("should handle out of range number selection", func(t *testing.T) {
		mockInput := strings.NewReader("0\n")
		var output bytes.Buffer
		ui := NewCLIUserInteractionWithIO(mockInput, &output)

		// This will cause EOF error since we can't loop infinitely in test
		_, err := ui.PromptSelect("Choose:", options)

		assert.Error(t, err) // Should get EOF error when input runs out
	})
}

func TestCLIUserInteraction_PromptMultiSelect(t *testing.T) {
	options := []string{"option1", "option2", "option3"}

	t.Run("should return multiple selected options", func(t *testing.T) {
		mockInput := strings.NewReader("1,3\n")
		var output bytes.Buffer
		ui := NewCLIUserInteractionWithIO(mockInput, &output)

		result, err := ui.PromptMultiSelect("Choose multiple:", options)

		require.NoError(t, err)
		assert.Contains(t, result, "option1")
		assert.Contains(t, result, "option3")
		assert.Len(t, result, 2)
	})

	t.Run("should handle mixed number and name selection", func(t *testing.T) {
		mockInput := strings.NewReader("1,option3\n")
		var output bytes.Buffer
		ui := NewCLIUserInteractionWithIO(mockInput, &output)

		result, err := ui.PromptMultiSelect("Choose multiple:", options)

		require.NoError(t, err)
		assert.Contains(t, result, "option1")
		assert.Contains(t, result, "option3")
		assert.Len(t, result, 2)
	})

	t.Run("should return empty for no selection", func(t *testing.T) {
		mockInput := strings.NewReader("\n")
		var output bytes.Buffer
		ui := NewCLIUserInteractionWithIO(mockInput, &output)

		result, err := ui.PromptMultiSelect("Choose multiple:", options)

		require.NoError(t, err)
		assert.Empty(t, result)
	})
}
