package ui

import (
	"strings"
	"testing"
)

func TestColorize(t *testing.T) {
	text := "Hello World"
	result := Colorize(text, ColorSuccess)

	// Should contain the text and color codes
	if !strings.Contains(result, text) {
		t.Errorf("Expected result to contain '%s', got '%s'", text, result)
	}

	// Should start with color code and end with reset
	if !strings.HasPrefix(result, string(ColorSuccess)) {
		t.Errorf("Expected result to start with color code")
	}

	if !strings.HasSuffix(result, Reset) {
		t.Errorf("Expected result to end with reset code")
	}
}

func TestBoldText(t *testing.T) {
	text := "Bold Text"
	result := BoldText(text)

	expected := Bold + text + Reset
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestErrorText(t *testing.T) {
	text := "Error message"
	result := ErrorText(text)

	if !strings.Contains(result, text) {
		t.Errorf("Expected result to contain '%s'", text)
	}

	if !strings.Contains(result, string(ColorError)) {
		t.Errorf("Expected result to contain error color")
	}
}

func TestSuccessText(t *testing.T) {
	text := "Success message"
	result := SuccessText(text)

	if !strings.Contains(result, text) {
		t.Errorf("Expected result to contain '%s'", text)
	}

	if !strings.Contains(result, string(ColorSuccess)) {
		t.Errorf("Expected result to contain success color")
	}
}

func TestProgressBar(t *testing.T) {
	result := CreateProgressBar(7, 10, 20, ColorSuccess, ColorMuted)

	// Should contain progress indicators
	if !strings.Contains(result, "█") || !strings.Contains(result, "░") {
		t.Errorf("Expected progress bar to contain fill and empty characters")
	}

	// Should contain percentage
	if !strings.Contains(result, "70.0%") {
		t.Errorf("Expected progress bar to show correct percentage")
	}
}

func TestBox(t *testing.T) {
	text := "Boxed content"
	result := Box(text, ColorPrimary)

	// Should contain the text
	if !strings.Contains(result, text) {
		t.Errorf("Expected box to contain '%s'", text)
	}

	// Should apply color formatting (no box drawing characters anymore)
	if !strings.Contains(result, string(ColorPrimary)) {
		t.Error("Expected box to apply color formatting")
	}
}

func TestDefaultPalette(t *testing.T) {
	palette := DefaultPalette()

	// Check that palette has all required colors
	if palette.Primary == "" {
		t.Error("Expected palette to have primary color")
	}

	if palette.Success == "" {
		t.Error("Expected palette to have success color")
	}

	if palette.Error == "" {
		t.Error("Expected palette to have error color")
	}
}

func TestPaletteFormat(t *testing.T) {
	palette := DefaultPalette()
	text := "Test text"

	result := palette.Format(text, "success")

	if !strings.Contains(result, text) {
		t.Errorf("Expected formatted text to contain '%s'", text)
	}
}

func TestStyled(t *testing.T) {
	text := "Styled text"
	result := Styled(text, Bold, string(ColorPrimary))

	if !strings.Contains(result, text) {
		t.Errorf("Expected styled text to contain '%s'", text)
	}

	if !strings.HasPrefix(result, Bold+string(ColorPrimary)) {
		t.Error("Expected styled text to start with style codes")
	}

	if !strings.HasSuffix(result, Reset) {
		t.Error("Expected styled text to end with reset")
	}
}
