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

// RenderCleanList renders data in a clean, readable list format
func RenderCleanList(title string, items []map[string]interface{}, categoryKey string, nameKey string, descKey string) string {
	if len(items) == 0 {
		return fmt.Sprintf("%s No items to display.\n", InfoText("⏺"))
	}

	var result strings.Builder

	// Add title with bullet
	result.WriteString(fmt.Sprintf("%s %s\n", InfoText("⏺"), BoldText(title)))

	// Group items by category if categoryKey is provided
	if categoryKey != "" {
		categories := make(map[string][]map[string]interface{})
		for _, item := range items {
			category := "Other"
			if cat, exists := item[categoryKey]; exists && cat != nil {
				category = fmt.Sprintf("%v", cat)
			}
			categories[category] = append(categories[category], item)
		}

		// Sort categories for consistent output
		var sortedCategories []string
		for cat := range categories {
			sortedCategories = append(sortedCategories, cat)
		}
		sort.Strings(sortedCategories)

		// Render each category
		for _, category := range sortedCategories {
			categoryItems := categories[category]
			result.WriteString(fmt.Sprintf("  - %s (", category))

			var itemNames []string
			for _, item := range categoryItems {
				name := ""
				if nameVal, exists := item[nameKey]; exists && nameVal != nil {
					name = fmt.Sprintf("%v", nameVal)
				}
				if desc, exists := item[descKey]; exists && desc != nil && desc != "" {
					name = fmt.Sprintf("%s: %v", name, desc)
				}
				if name != "" {
					itemNames = append(itemNames, name)
				}
			}

			result.WriteString(strings.Join(itemNames, ", "))
			result.WriteString(")\n")
		}

		// Add total count
		result.WriteString(fmt.Sprintf("\n  Total: %d items.\n", len(items)))
	} else {
		// Simple list without categories
		for _, item := range items {
			name := ""
			if nameVal, exists := item[nameKey]; exists && nameVal != nil {
				name = fmt.Sprintf("%v", nameVal)
			}
			if desc, exists := item[descKey]; exists && desc != nil && desc != "" {
				name = fmt.Sprintf("%s: %v", name, desc)
			}
			if name != "" {
				result.WriteString(fmt.Sprintf("  - %s\n", name))
			}
		}

		result.WriteString(fmt.Sprintf("\n  Total: %d items.\n", len(items)))
	}

	return result.String()
}

// RenderDetailView renders a detailed view with expandable sections
func RenderDetailView(title string, data map[string]interface{}, _ []string) string {
	var result strings.Builder

	// Command execution header
	if cmd, exists := data["_command"]; exists && cmd != nil {
		result.WriteString(fmt.Sprintf("%s %s\n", InfoText("⏺"), MutedText(fmt.Sprintf("Bash(%v)", cmd))))
	}

	// Asset/item header with name
	if name, exists := data["name"]; exists && name != nil {
		result.WriteString(fmt.Sprintf("  %s %s: %s\n", InfoText("⎿"), BoldText("Asset"), BoldText(fmt.Sprintf("%v", name))))
	} else {
		result.WriteString(fmt.Sprintf("  %s %s\n", InfoText("⎿"), BoldText(title)))
	}

	// Description with wrapping and expansion
	if desc, exists := data["description"]; exists && desc != nil {
		descStr := fmt.Sprintf("%v", desc)
		if len(descStr) > 200 {
			// Truncated description with expansion hint
			truncated := descStr[:200]
			lastSpace := strings.LastIndex(truncated, " ")
			if lastSpace > 0 {
				truncated = truncated[:lastSpace]
			}
			result.WriteString(fmt.Sprintf("    %s: %s...\n", "Description", truncated))
			result.WriteString(fmt.Sprintf("    … +%d lines (ctrl+r to expand)\n\n", strings.Count(descStr[200:], "\n")+1))
		} else {
			result.WriteString(fmt.Sprintf("    %s: %s\n\n", "Description", descStr))
		}
	}

	// Main summary with bullet
	summary := extractSummary(data)
	if summary != "" {
		result.WriteString(fmt.Sprintf("%s %s\n\n", InfoText("⏺"), BoldText(summary)))
	}

	// Key details in clean format
	keyFields := []string{"purpose", "revenue", "implementation", "goal", "documentation", "why", "benefits", "how", "metrics"}

	for _, field := range keyFields {
		if value, exists := data[field]; exists && value != nil && fmt.Sprintf("%v", value) != "" {
			fieldName := strings.ToUpper(field[:1]) + field[1:]
			result.WriteString(fmt.Sprintf("  - %s: %v\n", fieldName, value))
		}
	}

	return result.String()
}

// extractSummary creates a summary from asset data
func extractSummary(data map[string]interface{}) string {
	name, hasName := data["name"]
	desc, hasDesc := data["description"]

	if hasName && hasDesc {
		nameStr := fmt.Sprintf("%v", name)
		descStr := fmt.Sprintf("%v", desc)

		// Try to extract a concise summary from the description
		if len(descStr) > 100 {
			sentences := strings.Split(descStr, ". ")
			if len(sentences) > 0 && len(sentences[0]) < 150 {
				return fmt.Sprintf("%s %s", nameStr, strings.ToLower(sentences[0]))
			}
		}

		// Fallback to name with brief desc
		return fmt.Sprintf("%s provides additional capabilities", nameStr)
	}

	return ""
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
