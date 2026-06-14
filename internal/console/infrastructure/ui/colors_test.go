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

// assertWrapsWithReset checks that a colour/style helper preserved the
// raw text in its output and terminated the styled span with Reset.
// All the helpers below share that contract; deduplicating the
// assertion keeps each subtest a single readable line.
func assertWrapsWithReset(t *testing.T, got, raw string) {
	t.Helper()
	if !strings.Contains(got, raw) {
		t.Errorf("expected output to contain raw text %q, got %q", raw, got)
	}
	if !strings.HasSuffix(got, Reset) {
		t.Errorf("expected output to end with Reset, got %q", got)
	}
}

func TestDimText(t *testing.T) {
	got := DimText("hello")
	assertWrapsWithReset(t, got, "hello")
	if !strings.HasPrefix(got, Dim) {
		t.Errorf("expected output to start with Dim, got %q", got)
	}
}

func TestItalicText(t *testing.T) {
	got := ItalicText("hi")
	assertWrapsWithReset(t, got, "hi")
	if !strings.HasPrefix(got, Italic) {
		t.Errorf("expected output to start with Italic, got %q", got)
	}
}

func TestSemanticTextWrappers(t *testing.T) {
	cases := []struct {
		name string
		fn   func(string) string
	}{
		{"MutedText", MutedText},
		{"PrimaryText", PrimaryText},
		{"Code", Code},
		{"Link", Link},
		{"Subheader", Subheader},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertWrapsWithReset(t, c.fn("payload"), "payload")
		})
	}
}

func TestPromptText_AppliesBoldAndColor(t *testing.T) {
	got := PromptText("> ")
	// PromptText is Colorize(BoldText(text)) so the raw text and a
	// Bold marker must both survive into the output.
	if !strings.Contains(got, "> ") {
		t.Errorf("expected raw text in output, got %q", got)
	}
	if !strings.Contains(got, Bold) {
		t.Errorf("expected output to contain Bold, got %q", got)
	}
}

func TestHeader_AppliesBoldOverPrimary(t *testing.T) {
	got := Header("Title")
	if !strings.Contains(got, "Title") {
		t.Errorf("expected raw text in output, got %q", got)
	}
	if !strings.Contains(got, Bold) {
		t.Errorf("expected Header to apply Bold, got %q", got)
	}
}

func TestHighlight_WrapsWithBackgroundAndForeground(t *testing.T) {
	got := Highlight("warn")
	// Highlight is Styled(text, BgYellow, Black) so both style strings
	// must appear before the text and a Reset must close the run.
	if !strings.Contains(got, "warn") {
		t.Errorf("expected raw text in output, got %q", got)
	}
	if !strings.HasSuffix(got, Reset) {
		t.Errorf("expected output to end with Reset, got %q", got)
	}
	if !strings.Contains(got, BgYellow) || !strings.Contains(got, Black) {
		t.Errorf("expected output to contain both BgYellow and Black style runs, got %q", got)
	}
}

func TestGradient_StubFallsBackToStartColor(t *testing.T) {
	// The documented behaviour is "for now, just use start color".
	// Pin that so a future real gradient implementation surfaces in tests.
	got := Gradient("g", ColorSuccess, ColorError)
	assertWrapsWithReset(t, got, "g")
	if !strings.HasPrefix(got, string(ColorSuccess)) {
		t.Errorf("expected Gradient to begin with the start colour, got %q", got)
	}
}

func TestPaletteFormat_AllColorTypesAndStyles(t *testing.T) {
	palette := DefaultPalette()
	text := "payload"

	t.Run("color-type table including the fallthrough", func(t *testing.T) {
		// One subtest per color type so the failure message names the
		// branch that broke.
		cases := []string{"primary", "secondary", ColorTypeSuccess, ColorTypeError, ColorTypeWarning, "info", "muted", "unknown-falls-back-to-foreground"}
		for _, colorType := range cases {
			t.Run(colorType, func(t *testing.T) {
				got := palette.Format(text, colorType)
				if !strings.Contains(got, text) {
					t.Errorf("expected output to contain %q, got %q", text, got)
				}
				if !strings.HasSuffix(got, Reset) {
					t.Errorf("expected output to end with Reset, got %q", got)
				}
			})
		}
	})

	t.Run("each named style prepends its escape code", func(t *testing.T) {
		cases := map[string]string{"bold": Bold, "dim": Dim, "italic": Italic}
		for styleName, escape := range cases {
			t.Run(styleName, func(t *testing.T) {
				got := palette.Format(text, "primary", styleName)
				if !strings.HasPrefix(got, escape) {
					t.Errorf("expected output to begin with %q escape, got %q", styleName, got)
				}
			})
		}
	})

	t.Run("unknown style names are silently ignored", func(t *testing.T) {
		got := palette.Format(text, "primary", "neon-glow")
		// Output still starts with the primary colour (not Bold/Dim/Italic)
		// and ends with Reset.
		assertWrapsWithReset(t, got, text)
	})

	t.Run("multiple styles stack in argument order", func(t *testing.T) {
		got := palette.Format(text, "primary", "bold", "italic")
		// Italic is appended last so it ends up at the very front.
		if !strings.HasPrefix(got, Italic) {
			t.Errorf("expected output to begin with Italic (the last style applied), got %q", got)
		}
		if !strings.Contains(got, Bold) {
			t.Errorf("expected output to also contain Bold, got %q", got)
		}
	})
}
