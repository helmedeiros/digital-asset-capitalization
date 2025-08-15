package ui

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Status constants to avoid goconst warnings
const (
	StatusActive    = "active"
	StatusInactive  = "inactive"
	StatusFailed    = "failed"
	StatusCompleted = "completed"
	StatusError     = "error"
	StatusSuccess   = "success"
	StatusDone      = "done"
)

// TableColumn represents a column in a table
type TableColumn struct {
	Header    string
	Key       string
	Width     int
	Align     string // "left", "right", "center"
	Formatter func(interface{}) string
}

// Table represents a formatted table
type Table struct {
	Columns []TableColumn
	Rows    []map[string]interface{}
	Style   TableStyle
}

// TableStyle defines table appearance
type TableStyle struct {
	HeaderColor  Color
	RowColor     Color
	BorderColor  Color
	AlternateRow bool
	ShowBorder   bool
	Padding      int
}

// DefaultTableStyle returns a default table style
func DefaultTableStyle() TableStyle {
	return TableStyle{
		HeaderColor:  ColorPrimary,
		RowColor:     ColorOutput,
		BorderColor:  ColorMuted,
		AlternateRow: true,
		ShowBorder:   true,
		Padding:      1,
	}
}

// NewTable creates a new table
func NewTable(columns []TableColumn) *Table {
	return &Table{
		Columns: columns,
		Rows:    make([]map[string]interface{}, 0),
		Style:   DefaultTableStyle(),
	}
}

// AddRow adds a row to the table
func (t *Table) AddRow(row map[string]interface{}) {
	t.Rows = append(t.Rows, row)
}

// Render renders the table as a string
func (t *Table) Render() string {
	if len(t.Columns) == 0 {
		return ""
	}

	// Calculate column widths
	t.calculateColumnWidths()

	var result strings.Builder

	// Render header
	if t.Style.ShowBorder {
		result.WriteString(t.renderTopBorder())
		result.WriteString("\n")
	}

	result.WriteString(t.renderHeader())
	result.WriteString("\n")

	if t.Style.ShowBorder {
		result.WriteString(t.renderMiddleBorder())
		result.WriteString("\n")
	}

	// Render rows
	for i, row := range t.Rows {
		isAlternate := t.Style.AlternateRow && i%2 == 1
		result.WriteString(t.renderRow(row, isAlternate))
		result.WriteString("\n")
	}

	// Render bottom border
	if t.Style.ShowBorder {
		result.WriteString(t.renderBottomBorder())
	}

	return result.String()
}

// calculateColumnWidths calculates optimal column widths
func (t *Table) calculateColumnWidths() {
	for i := range t.Columns {
		column := &t.Columns[i]

		// Start with header width
		minWidth := len(column.Header)

		// Check all row values
		for _, row := range t.Rows {
			if value, exists := row[column.Key]; exists {
				formatted := t.formatValue(value, column.Formatter)
				if len(formatted) > minWidth {
					minWidth = len(formatted)
				}
			}
		}

		// Apply minimum width if specified, otherwise use calculated
		if column.Width == 0 {
			column.Width = minWidth + t.Style.Padding*2
		} else if column.Width < minWidth {
			column.Width = minWidth + t.Style.Padding*2
		}
	}
}

// renderTopBorder renders the top border
func (t *Table) renderTopBorder() string {
	var result strings.Builder
	result.WriteString(Colorize("┌", t.Style.BorderColor))

	for i, column := range t.Columns {
		result.WriteString(Colorize(strings.Repeat("─", column.Width), t.Style.BorderColor))
		if i < len(t.Columns)-1 {
			result.WriteString(Colorize("┬", t.Style.BorderColor))
		}
	}

	result.WriteString(Colorize("┐", t.Style.BorderColor))
	return result.String()
}

// renderMiddleBorder renders the middle border
func (t *Table) renderMiddleBorder() string {
	var result strings.Builder
	result.WriteString(Colorize("├", t.Style.BorderColor))

	for i, column := range t.Columns {
		result.WriteString(Colorize(strings.Repeat("─", column.Width), t.Style.BorderColor))
		if i < len(t.Columns)-1 {
			result.WriteString(Colorize("┼", t.Style.BorderColor))
		}
	}

	result.WriteString(Colorize("┤", t.Style.BorderColor))
	return result.String()
}

// renderBottomBorder renders the bottom border
func (t *Table) renderBottomBorder() string {
	var result strings.Builder
	result.WriteString(Colorize("└", t.Style.BorderColor))

	for i, column := range t.Columns {
		result.WriteString(Colorize(strings.Repeat("─", column.Width), t.Style.BorderColor))
		if i < len(t.Columns)-1 {
			result.WriteString(Colorize("┴", t.Style.BorderColor))
		}
	}

	result.WriteString(Colorize("┘", t.Style.BorderColor))
	return result.String()
}

// renderHeader renders the table header
func (t *Table) renderHeader() string {
	var result strings.Builder

	if t.Style.ShowBorder {
		result.WriteString(Colorize("│", t.Style.BorderColor))
	}

	for i, column := range t.Columns {
		headerText := t.alignText(BoldText(Colorize(column.Header, t.Style.HeaderColor)), column.Width, column.Align)
		result.WriteString(headerText)

		if i < len(t.Columns)-1 && t.Style.ShowBorder {
			result.WriteString(Colorize("│", t.Style.BorderColor))
		}
	}

	if t.Style.ShowBorder {
		result.WriteString(Colorize("│", t.Style.BorderColor))
	}

	return result.String()
}

// renderRow renders a table row
func (t *Table) renderRow(row map[string]interface{}, isAlternate bool) string {
	var result strings.Builder

	rowColor := t.Style.RowColor
	if isAlternate {
		rowColor = ColorMuted
	}

	if t.Style.ShowBorder {
		result.WriteString(Colorize("│", t.Style.BorderColor))
	}

	for i, column := range t.Columns {
		value := ""
		if val, exists := row[column.Key]; exists {
			value = t.formatValue(val, column.Formatter)
		}

		cellText := t.alignText(Colorize(value, rowColor), column.Width, column.Align)
		result.WriteString(cellText)

		if i < len(t.Columns)-1 && t.Style.ShowBorder {
			result.WriteString(Colorize("│", t.Style.BorderColor))
		}
	}

	if t.Style.ShowBorder {
		result.WriteString(Colorize("│", t.Style.BorderColor))
	}

	return result.String()
}

// alignText aligns text within a given width
func (t *Table) alignText(text string, width int, align string) string {
	// Remove ANSI codes for length calculation
	cleanText := stripANSI(text)

	if len(cleanText) >= width {
		return text
	}

	padding := width - len(cleanText)

	switch align {
	case "right":
		return strings.Repeat(" ", padding) + text
	case "center":
		leftPad := padding / 2
		rightPad := padding - leftPad
		return strings.Repeat(" ", leftPad) + text + strings.Repeat(" ", rightPad)
	default: // "left"
		return text + strings.Repeat(" ", padding)
	}
}

// formatValue formats a value using the provided formatter or default
func (t *Table) formatValue(value interface{}, formatter func(interface{}) string) string {
	if formatter != nil {
		return formatter(value)
	}

	return DefaultFormatter(value)
}

// stripANSI removes ANSI color codes for length calculation
func stripANSI(text string) string {
	// Simple ANSI code removal - in practice you might want a more robust solution
	result := ""
	inEscape := false

	for _, char := range text {
		if char == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if char == 'm' {
				inEscape = false
			}
			continue
		}
		result += string(char)
	}

	return result
}

// DefaultFormatter provides default formatting for common types
func DefaultFormatter(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%.2f", v)
	case bool:
		if v {
			return SuccessText("true")
		}
		return ErrorText("false")
	case time.Time:
		return v.Format("2006-01-02 15:04")
	case []string:
		return strings.Join(v, ", ")
	default:
		// Handle slices of interfaces
		if reflect.TypeOf(v).Kind() == reflect.Slice {
			s := reflect.ValueOf(v)
			var items []string
			for i := 0; i < s.Len(); i++ {
				items = append(items, fmt.Sprintf("%v", s.Index(i).Interface()))
			}
			return strings.Join(items, ", ")
		}
		return fmt.Sprintf("%v", v)
	}
}

// MoneyFormatter formats monetary values
func MoneyFormatter(value interface{}) string {
	switch v := value.(type) {
	case string:
		return Code(v)
	case float64:
		return Code(fmt.Sprintf("$%.2f", v))
	case float32:
		return Code(fmt.Sprintf("$%.2f", v))
	default:
		if str := fmt.Sprintf("%v", v); strings.Contains(str, "$") || strings.Contains(str, "EUR") || strings.Contains(str, "USD") {
			return Code(str)
		}
		return DefaultFormatter(value)
	}
}

// DateFormatter formats date values
func DateFormatter(value interface{}) string {
	switch v := value.(type) {
	case time.Time:
		return MutedText(v.Format("2006-01-02"))
	case string:
		if t, err := time.Parse("2006-01-02 15:04:05", v); err == nil {
			return MutedText(t.Format("2006-01-02"))
		}
		return v
	default:
		return DefaultFormatter(value)
	}
}

// StatusFormatter formats status values with colors
func StatusFormatter(value interface{}) string {
	status := strings.ToLower(fmt.Sprintf("%v", value))

	switch status {
	case StatusActive, StatusCompleted, StatusSuccess, StatusDone, "enabled":
		return SuccessText(fmt.Sprintf("%v", value))
	case StatusInactive, StatusFailed, StatusError, "disabled":
		return ErrorText(fmt.Sprintf("%v", value))
	case "pending", "in progress", "warning":
		return WarningText(fmt.Sprintf("%v", value))
	default:
		return InfoText(fmt.Sprintf("%v", value))
	}
}

// PercentageFormatter formats percentage values
func PercentageFormatter(value interface{}) string {
	switch v := value.(type) {
	case float64:
		return fmt.Sprintf("%.1f%%", v)
	case float32:
		return fmt.Sprintf("%.1f%%", v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return fmt.Sprintf("%.1f%%", f)
		}
		return v
	default:
		return DefaultFormatter(value)
	}
}

// KeyValuePair represents a key-value pair for simple displays
type KeyValuePair struct {
	Key   string
	Value interface{}
}

// RenderKeyValueList renders a list of key-value pairs
func RenderKeyValueList(pairs []KeyValuePair, style TableStyle) string {
	if len(pairs) == 0 {
		return ""
	}

	// Find the longest key for alignment
	maxKeyLength := 0
	for _, pair := range pairs {
		if len(pair.Key) > maxKeyLength {
			maxKeyLength = len(pair.Key)
		}
	}

	var result strings.Builder

	for _, pair := range pairs {
		key := Colorize(pair.Key+":", style.HeaderColor)
		padding := strings.Repeat(" ", maxKeyLength-len(pair.Key)+2)
		value := Colorize(DefaultFormatter(pair.Value), style.RowColor)

		result.WriteString(fmt.Sprintf("%s%s%s\n", key, padding, value))
	}

	return result.String()
}

// CreateAssetTable creates a table optimized for asset display
func CreateAssetTable() *Table {
	columns := []TableColumn{
		{Header: "Name", Key: "name", Align: "left"},
		{Header: "Status", Key: "status", Align: "left", Formatter: StatusFormatter},
		{Header: "Created", Key: "created_at", Align: "left", Formatter: DateFormatter},
		{Header: "Team", Key: "owning_team", Align: "left"},
		{Header: "Tasks", Key: "task_count", Align: "right"},
	}

	table := NewTable(columns)
	table.Style.HeaderColor = ColorPrimary
	return table
}

// CreateTaskTable creates a table optimized for task display
func CreateTaskTable() *Table {
	columns := []TableColumn{
		{Header: "Key", Key: "key", Align: "left", Width: 12},
		{Header: "Summary", Key: "summary", Align: "left", Width: 40},
		{Header: "Status", Key: "status", Align: "left", Formatter: StatusFormatter},
		{Header: "Type", Key: "type", Align: "left"},
		{Header: "Sprint", Key: "sprint", Align: "left"},
	}

	table := NewTable(columns)
	table.Style.HeaderColor = ColorPrimary
	return table
}

// CreateInvestmentTable creates a table optimized for investment display
func CreateInvestmentTable() *Table {
	columns := []TableColumn{
		{Header: "Asset", Key: "asset", Align: "left"},
		{Header: "Investment", Key: "total_investment", Align: "right", Formatter: MoneyFormatter},
		{Header: "Hours", Key: "total_hours", Align: "right"},
		{Header: "Period", Key: "period", Align: "left"},
		{Header: "Engineers", Key: "engineer_count", Align: "right"},
	}

	table := NewTable(columns)
	table.Style.HeaderColor = ColorPrimary
	return table
}

// SimpleList renders a simple list with bullets
func SimpleList(items []string, bullet string) string {
	if bullet == "" {
		bullet = "•"
	}

	var result strings.Builder
	for _, item := range items {
		result.WriteString(fmt.Sprintf("%s %s\n", InfoText(bullet), item))
	}

	return result.String()
}

// NumberedList renders a numbered list
func NumberedList(items []string) string {
	var result strings.Builder
	for i, item := range items {
		result.WriteString(fmt.Sprintf("%s %s\n", InfoText(fmt.Sprintf("%d.", i+1)), item))
	}

	return result.String()
}

// SortMapKeys sorts map keys for consistent display
func SortMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
