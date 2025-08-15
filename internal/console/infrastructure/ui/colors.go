package ui

import (
	"fmt"
	"strings"
)

// Color type constants to avoid goconst warnings
const (
	ColorTypeError   = "error"
	ColorTypeSuccess = "success"
	ColorTypeWarning = "warning"
)

// ANSI color codes for terminal output
const (
	// Standard colors
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"
	Italic = "\033[3m"

	// Foreground colors
	Black   = "\033[30m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"

	// Bright foreground colors
	BrightBlack   = "\033[90m" // Gray
	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
	BrightWhite   = "\033[97m"

	// Background colors
	BgBlack   = "\033[40m"
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgWhite   = "\033[47m"
)

// Color represents a terminal color
type Color string

// Predefined color schemes
const (
	ColorError   Color = Red
	ColorSuccess Color = Green
	ColorWarning Color = Yellow
	ColorInfo    Color = Cyan
	ColorMuted   Color = BrightBlack
	ColorPrimary Color = Blue
	ColorInput   Color = White
	ColorOutput  Color = White
	ColorPrompt  Color = BrightBlue
)

// Colorize wraps text with the specified color
func Colorize(text string, color Color) string {
	return string(color) + text + Reset
}

// Bold makes text bold
func BoldText(text string) string {
	return Bold + text + Reset
}

// Dim makes text dimmed
func DimText(text string) string {
	return Dim + text + Reset
}

// ItalicText makes text italic
func ItalicText(text string) string {
	return Italic + text + Reset
}

// Styled applies multiple styles to text
func Styled(text string, styles ...string) string {
	prefix := strings.Join(styles, "")
	return prefix + text + Reset
}

// ErrorText returns red error text
func ErrorText(text string) string {
	return Colorize(text, ColorError)
}

// SuccessText returns green success text
func SuccessText(text string) string {
	return Colorize(text, ColorSuccess)
}

// WarningText returns yellow warning text
func WarningText(text string) string {
	return Colorize(text, ColorWarning)
}

// InfoText returns cyan info text
func InfoText(text string) string {
	return Colorize(text, ColorInfo)
}

// MutedText returns gray muted text
func MutedText(text string) string {
	return Colorize(text, ColorMuted)
}

// PrimaryText returns blue primary text
func PrimaryText(text string) string {
	return Colorize(text, ColorPrimary)
}

// PromptText returns styled prompt text
func PromptText(text string) string {
	return Colorize(BoldText(text), ColorPrompt)
}

// Header creates a styled header
func Header(text string) string {
	return BoldText(PrimaryText(text))
}

// Subheader creates a styled subheader
func Subheader(text string) string {
	return Colorize(text, ColorPrimary)
}

// Code formats text as code
func Code(text string) string {
	return Colorize(text, BrightYellow)
}

// Link formats text as a link
func Link(text string) string {
	return Colorize(text, BrightCyan)
}

// Highlight highlights important text
func Highlight(text string) string {
	return Styled(text, BgYellow, Black)
}

// CreateColorPalette defines semantic colors for different UI elements
type ColorPalette struct {
	Primary    Color
	Secondary  Color
	Success    Color
	Error      Color
	Warning    Color
	Info       Color
	Muted      Color
	Background Color
	Foreground Color
}

// DefaultPalette returns the default color palette
func DefaultPalette() ColorPalette {
	return ColorPalette{
		Primary:    ColorPrimary,
		Secondary:  ColorMuted,
		Success:    ColorSuccess,
		Error:      ColorError,
		Warning:    ColorWarning,
		Info:       ColorInfo,
		Muted:      ColorMuted,
		Background: Black,
		Foreground: White,
	}
}

// Format applies color formatting with optional styles
func (p ColorPalette) Format(text string, colorType string, styles ...string) string {
	var color Color
	switch colorType {
	case "primary":
		color = p.Primary
	case "secondary":
		color = p.Secondary
	case ColorTypeSuccess:
		color = p.Success
	case ColorTypeError:
		color = p.Error
	case ColorTypeWarning:
		color = p.Warning
	case "info":
		color = p.Info
	case "muted":
		color = p.Muted
	default:
		color = p.Foreground
	}

	result := string(color) + text + Reset

	// Apply additional styles
	for _, style := range styles {
		switch style {
		case "bold":
			result = Bold + result
		case "dim":
			result = Dim + result
		case "italic":
			result = Italic + result
		}
	}

	return result
}

// Gradient creates a gradient effect between two colors (simplified)
func Gradient(text string, startColor, _ Color) string {
	// For now, just use start color - full gradient would require more complex logic
	return Colorize(text, startColor)
}

// CreateProgressBar creates a colored progress bar string
func CreateProgressBar(current, total int, width int, fillColor, emptyColor Color) string {
	if total <= 0 {
		return ""
	}

	filled := int(float64(current) / float64(total) * float64(width))
	if filled > width {
		filled = width
	}

	// Color the filled and empty parts
	coloredBar := Colorize(strings.Repeat("█", filled), fillColor) +
		Colorize(strings.Repeat("░", width-filled), emptyColor)

	percentage := float64(current) / float64(total) * 100
	return fmt.Sprintf("%s %6.1f%%", coloredBar, percentage)
}

// Box creates a simple highlighted text block without box drawing
func Box(text string, borderColor Color) string {
	lines := strings.Split(text, "\n")
	var result strings.Builder

	for _, line := range lines {
		// Simple formatting with color, no box drawing
		result.WriteString(Colorize(line, borderColor))
		result.WriteString("\n")
	}

	return result.String()
}
