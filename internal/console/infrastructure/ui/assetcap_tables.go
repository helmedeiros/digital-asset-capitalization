package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Asset status constants to avoid goconst warnings
const (
	AssetStatusActive    = "active"
	AssetStatusInactive  = "inactive"
	AssetStatusCompleted = "completed"
	AssetStatusDone      = "done"
)

// AssetCapTableFactory creates specialized tables for AssetCap data types
type AssetCapTableFactory struct {
	palette ColorPalette
	style   TableStyle
}

// NewAssetCapTableFactory creates a new table factory
func NewAssetCapTableFactory() *AssetCapTableFactory {
	return &AssetCapTableFactory{
		palette: DefaultPalette(),
		style:   AssetCapTableStyle(),
	}
}

// AssetCapTableStyle returns a table style optimized for AssetCap
func AssetCapTableStyle() TableStyle {
	return TableStyle{
		HeaderColor:  ColorPrimary,
		RowColor:     ColorOutput,
		BorderColor:  ColorMuted,
		AlternateRow: true,
		ShowBorder:   true,
		Padding:      1,
	}
}

// CreateAssetListTable creates a table optimized for asset listings
func (f *AssetCapTableFactory) CreateAssetListTable() *Table {
	columns := []TableColumn{
		{Header: "Name", Key: "name", Align: "left", Width: 25, Formatter: AssetNameFormatter},
		{Header: "Status", Key: "status", Align: "center", Width: 12, Formatter: AssetStatusFormatter},
		{Header: "Team", Key: "owning_team", Align: "left", Width: 15, Formatter: TeamFormatter},
		{Header: "Tasks", Key: "task_count", Align: "right", Width: 8, Formatter: CountFormatter},
		{Header: "Updated", Key: "updated_at", Align: "left", Width: 12, Formatter: RelativeDateFormatter},
	}

	table := NewTable(columns)
	table.Style = f.style
	return table
}

// CreateAssetDetailTable creates a table for detailed asset view
func (f *AssetCapTableFactory) CreateAssetDetailTable() *Table {
	columns := []TableColumn{
		{Header: "Property", Key: "property", Align: "left", Width: 20},
		{Header: "Value", Key: "value", Align: "left", Width: 50, Formatter: AssetPropertyFormatter},
	}

	table := NewTable(columns)
	table.Style = f.style
	table.Style.AlternateRow = false // Better for key-value display
	return table
}

// CreateTaskListTable creates a table optimized for task listings
func (f *AssetCapTableFactory) CreateTaskListTable() *Table {
	columns := []TableColumn{
		{Header: "Key", Key: "key", Align: "left", Width: 12, Formatter: TaskKeyFormatter},
		{Header: "Summary", Key: "summary", Align: "left", Width: 35, Formatter: TaskSummaryFormatter},
		{Header: "Status", Key: "status", Align: "center", Width: 12, Formatter: TaskStatusFormatter},
		{Header: "Type", Key: "type", Align: "left", Width: 10, Formatter: WorkTypeFormatter},
		{Header: "Priority", Key: "priority", Align: "center", Width: 8, Formatter: PriorityFormatter},
		{Header: "Sprint", Key: "sprint", Align: "left", Width: 15, Formatter: SprintFormatter},
	}

	table := NewTable(columns)
	table.Style = f.style
	return table
}

// CreateInvestmentSummaryTable creates a table for investment summaries
func (f *AssetCapTableFactory) CreateInvestmentSummaryTable() *Table {
	columns := []TableColumn{
		{Header: "Asset", Key: "asset", Align: "left", Width: 25, Formatter: AssetNameFormatter},
		{Header: "Investment", Key: "total_investment", Align: "right", Width: 15, Formatter: InvestmentFormatter},
		{Header: "Hours", Key: "total_hours", Align: "right", Width: 10, Formatter: HoursFormatter},
		{Header: "Period", Key: "period", Align: "left", Width: 20, Formatter: PeriodFormatter},
		{Header: "Engineers", Key: "engineer_count", Align: "right", Width: 10, Formatter: CountFormatter},
	}

	table := NewTable(columns)
	table.Style = f.style
	return table
}

// CreateInvestmentDetailTable creates a table for detailed investment breakdown
func (f *AssetCapTableFactory) CreateInvestmentDetailTable() *Table {
	columns := []TableColumn{
		{Header: "Engineer", Key: "name", Align: "left", Width: 20, Formatter: EngineerNameFormatter},
		{Header: "Level", Key: "level", Align: "left", Width: 12, Formatter: EngineerLevelFormatter},
		{Header: "Hours", Key: "hours", Align: "right", Width: 10, Formatter: HoursFormatter},
		{Header: "Rate", Key: "rate", Align: "right", Width: 12, Formatter: HourlyRateFormatter},
		{Header: "Cost", Key: "cost", Align: "right", Width: 15, Formatter: InvestmentFormatter},
	}

	table := NewTable(columns)
	table.Style = f.style
	return table
}

// CreateSprintTable creates a table for sprint information
func (f *AssetCapTableFactory) CreateSprintTable() *Table {
	columns := []TableColumn{
		{Header: "Sprint", Key: "name", Align: "left", Width: 20, Formatter: SprintNameFormatter},
		{Header: "Status", Key: "state", Align: "center", Width: 12, Formatter: SprintStatusFormatter},
		{Header: "Start Date", Key: "start_date", Align: "left", Width: 12, Formatter: DateFormatter},
		{Header: "End Date", Key: "end_date", Align: "left", Width: 12, Formatter: DateFormatter},
		{Header: "Goal", Key: "goal", Align: "left", Width: 30, Formatter: GoalFormatter},
	}

	table := NewTable(columns)
	table.Style = f.style
	return table
}

// Specialized Formatters for AssetCap Data

// AssetNameFormatter formats asset names with appropriate styling
func AssetNameFormatter(value interface{}) string {
	name := DefaultFormatter(value)
	if name == "" {
		return MutedText("(unnamed)")
	}
	return BoldText(name)
}

// AssetStatusFormatter formats asset status with color coding
func AssetStatusFormatter(value interface{}) string {
	status := strings.ToLower(fmt.Sprintf("%v", value))

	switch status {
	case AssetStatusActive, "live", "production":
		return SuccessText("Active")
	case "development", "dev", "in-progress":
		return InfoText("Development")
	case "deprecated", "retired", AssetStatusInactive:
		return ErrorText("Deprecated")
	case "planned", "future":
		return WarningText("Planned")
	default:
		return MutedText(fmt.Sprintf("%v", value))
	}
}

// TeamFormatter formats team names with styling
func TeamFormatter(value interface{}) string {
	if value == nil || fmt.Sprintf("%v", value) == "" {
		return MutedText("(no team)")
	}
	return Colorize(fmt.Sprintf("%v", value), ColorInfo)
}

// CountFormatter formats count values
func CountFormatter(value interface{}) string {
	count := fmt.Sprintf("%v", value)
	if count == "0" {
		return MutedText("0")
	}
	return BoldText(count)
}

// RelativeDateFormatter formats dates relative to now
func RelativeDateFormatter(value interface{}) string {
	switch v := value.(type) {
	case time.Time:
		return formatRelativeTime(v)
	case string:
		if t, err := time.Parse("2006-01-02 15:04:05", v); err == nil {
			return formatRelativeTime(t)
		}
		if t, err := time.Parse("2006-01-02", v); err == nil {
			return formatRelativeTime(t)
		}
		return v
	default:
		return DefaultFormatter(value)
	}
}

// TaskKeyFormatter formats task keys (e.g., JIRA keys)
func TaskKeyFormatter(value interface{}) string {
	key := fmt.Sprintf("%v", value)
	if key == "" {
		return MutedText("(no key)")
	}
	return Link(key) // Style as clickable link
}

// TaskSummaryFormatter formats task summaries with truncation
func TaskSummaryFormatter(value interface{}) string {
	summary := fmt.Sprintf("%v", value)
	if len(summary) > 30 {
		return summary[:27] + "..."
	}
	return summary
}

// TaskStatusFormatter formats task status with color coding
func TaskStatusFormatter(value interface{}) string {
	status := strings.ToLower(fmt.Sprintf("%v", value))

	switch status {
	case AssetStatusDone, AssetStatusCompleted, "closed", "resolved":
		return SuccessText("Done")
	case "in progress", "in-progress", AssetStatusActive, "working":
		return InfoText("In Progress")
	case "todo", "to do", "open", "new":
		return WarningText("To Do")
	case "blocked", "impediment":
		return ErrorText("Blocked")
	case "review", "code review", "testing":
		return WarningText("Review")
	default:
		return MutedText(fmt.Sprintf("%v", value))
	}
}

// WorkTypeFormatter formats work types with color coding
func WorkTypeFormatter(value interface{}) string {
	workType := strings.ToLower(fmt.Sprintf("%v", value))

	switch workType {
	case "feature", "enhancement", "story":
		return InfoText("⚡")
	case "bug", "defect", "issue":
		return ErrorText("🐛")
	case "task", "chore", "maintenance":
		return WarningText("CONFIG")
	case "epic", "initiative":
		return PrimaryText("EPIC")
	default:
		return MutedText("TASK")
	}
}

// PriorityFormatter formats priority levels
func PriorityFormatter(value interface{}) string {
	priority := strings.ToLower(fmt.Sprintf("%v", value))

	switch priority {
	case "critical", "highest", "urgent":
		return ErrorText("CRITICAL")
	case "high", "important":
		return WarningText("HIGH")
	case "medium", "normal":
		return InfoText("MEDIUM")
	case "low", "minor":
		return MutedText("LOW")
	default:
		return MutedText("UNKNOWN")
	}
}

// SprintFormatter formats sprint names
func SprintFormatter(value interface{}) string {
	sprint := fmt.Sprintf("%v", value)
	if sprint == "" {
		return MutedText("(no sprint)")
	}
	return Colorize(sprint, ColorInfo)
}

// InvestmentFormatter formats monetary values
func InvestmentFormatter(value interface{}) string {
	str := fmt.Sprintf("%v", value)

	// If it already contains currency symbol, just colorize
	if strings.Contains(str, "$") || strings.Contains(str, "EUR") || strings.Contains(str, "USD") {
		return Code(BoldText(str))
	}

	// Try to parse as number and format
	if f, err := strconv.ParseFloat(str, 64); err == nil {
		return Code(BoldText(fmt.Sprintf("$%.2f", f)))
	}

	return Code(str)
}

// HoursFormatter formats hour values
func HoursFormatter(value interface{}) string {
	str := fmt.Sprintf("%v", value)

	if f, err := strconv.ParseFloat(str, 64); err == nil {
		if f == 0 {
			return MutedText("0h")
		}
		return fmt.Sprintf("%.1fh", f)
	}

	if !strings.HasSuffix(str, "h") {
		str += "h"
	}
	return str
}

// PeriodFormatter formats time periods
func PeriodFormatter(value interface{}) string {
	period := fmt.Sprintf("%v", value)
	if strings.Contains(period, " to ") {
		parts := strings.Split(period, " to ")
		if len(parts) == 2 {
			start := formatShortDate(parts[0])
			end := formatShortDate(parts[1])
			return fmt.Sprintf("%s → %s", MutedText(start), MutedText(end))
		}
	}
	return MutedText(period)
}

// EngineerNameFormatter formats engineer names
func EngineerNameFormatter(value interface{}) string {
	name := fmt.Sprintf("%v", value)
	if name == "" {
		return MutedText("(unknown)")
	}
	return BoldText(name)
}

// EngineerLevelFormatter formats engineer levels with appropriate styling
func EngineerLevelFormatter(value interface{}) string {
	level := strings.ToLower(fmt.Sprintf("%v", value))

	switch level {
	case "junior", "jr":
		return InfoText("Jr")
	case "mid", "middle", "intermediate":
		return SuccessText("Mid")
	case "senior", "sr":
		return WarningText("Sr")
	case "staff":
		return PrimaryText("Staff")
	case "principal", "lead":
		return Colorize("Principal", ColorPrimary)
	default:
		return fmt.Sprintf("%v", value)
	}
}

// HourlyRateFormatter formats hourly rates
func HourlyRateFormatter(value interface{}) string {
	str := fmt.Sprintf("%v", value)

	if f, err := strconv.ParseFloat(str, 64); err == nil {
		return Code(fmt.Sprintf("$%.0f/h", f))
	}

	if !strings.Contains(str, "/h") && !strings.Contains(str, "hour") {
		return Code(str + "/h")
	}
	return Code(str)
}

// SprintNameFormatter formats sprint names with highlighting
func SprintNameFormatter(value interface{}) string {
	name := fmt.Sprintf("%v", value)
	if name == "" {
		return MutedText("(unnamed)")
	}

	// Highlight current sprint
	if strings.Contains(strings.ToLower(name), "current") {
		return SuccessText("● ") + BoldText(name)
	}

	return BoldText(name)
}

// SprintStatusFormatter formats sprint status
func SprintStatusFormatter(value interface{}) string {
	status := strings.ToLower(fmt.Sprintf("%v", value))

	switch status {
	case AssetStatusActive, "current", "open":
		return SuccessText("Active")
	case "closed", AssetStatusCompleted, AssetStatusDone:
		return MutedText("Completed")
	case "future", "planned":
		return InfoText("Planned")
	default:
		return fmt.Sprintf("%v", value)
	}
}

// GoalFormatter formats sprint goals with truncation
func GoalFormatter(value interface{}) string {
	goal := fmt.Sprintf("%v", value)
	if goal == "" {
		return MutedText("(no goal)")
	}

	if len(goal) > 25 {
		return goal[:22] + "..."
	}
	return goal
}

// AssetPropertyFormatter formats asset properties in detail view
func AssetPropertyFormatter(value interface{}) string {
	str := fmt.Sprintf("%v", value)

	// Format different types of values
	if strings.HasPrefix(str, "http") {
		return Link(str)
	}

	if strings.Contains(str, "@") && strings.Contains(str, ".") {
		return Link(str) // Email
	}

	if len(str) > 60 {
		return str[:57] + "..."
	}

	return str
}

// Helper functions

// formatRelativeTime formats a time relative to now
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return MutedText("just now")
	}
	if diff < time.Hour {
		minutes := int(diff.Minutes())
		return MutedText(fmt.Sprintf("%dm ago", minutes))
	}
	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return MutedText(fmt.Sprintf("%dh ago", hours))
	}
	if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		return MutedText(fmt.Sprintf("%dd ago", days))
	}

	// More than a week, show the date
	return MutedText(t.Format("Jan 2"))
}

// formatShortDate formats a date string in short format
func formatShortDate(dateStr string) string {
	if t, err := time.Parse("2006-01-02", dateStr); err == nil {
		return t.Format("Jan 2")
	}
	if t, err := time.Parse("2006-01-02 15:04:05", dateStr); err == nil {
		return t.Format("Jan 2")
	}
	return dateStr
}

// ConvertMapToAssetDetailRows converts a map to rows for asset detail display
func ConvertMapToAssetDetailRows(data map[string]interface{}) []map[string]interface{} {
	rows := make([]map[string]interface{}, 0, len(data))

	// Define the order for important properties
	propertyOrder := []string{
		"name", "id", "status", "description", "why", "benefits", "how", "metrics",
		"owning_team", "contributing_teams", "keywords", "launch_date", "doc_link",
		"created_at", "updated_at", "last_doc_update", "version", "task_count",
	}

	// Add properties in order
	added := make(map[string]bool)
	for _, key := range propertyOrder {
		if value, exists := data[key]; exists {
			rows = append(rows, map[string]interface{}{
				"property": formatPropertyName(key),
				"value":    value,
			})
			added[key] = true
		}
	}

	// Add any remaining properties
	for key, value := range data {
		if !added[key] {
			rows = append(rows, map[string]interface{}{
				"property": formatPropertyName(key),
				"value":    value,
			})
		}
	}

	return rows
}

// formatPropertyName formats property names for display
func formatPropertyName(key string) string {
	// Convert snake_case to Title Case
	parts := strings.Split(key, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	result := strings.Join(parts, " ")

	// Special cases
	switch strings.ToLower(key) {
	case "id":
		return "ID"
	case "doc_link":
		return "Documentation"
	case "task_count":
		return "Associated Tasks"
	case "last_doc_update":
		return "Last Doc Update"
	}

	return result
}
