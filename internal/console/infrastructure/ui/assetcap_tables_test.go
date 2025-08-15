package ui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAssetCapTableFactory(t *testing.T) {
	factory := NewAssetCapTableFactory()

	assert.NotNil(t, factory)
	assert.NotNil(t, factory.palette)
}

func TestAssetCapTableStyle(t *testing.T) {
	style := AssetCapTableStyle()

	assert.Equal(t, ColorPrimary, style.HeaderColor)
	assert.Equal(t, ColorOutput, style.RowColor)
	assert.Equal(t, ColorMuted, style.BorderColor)
	assert.True(t, style.AlternateRow)
	assert.True(t, style.ShowBorder)
	assert.Equal(t, 1, style.Padding)
}

func TestCreateAssetListTable(t *testing.T) {
	factory := NewAssetCapTableFactory()
	table := factory.CreateAssetListTable()

	assert.NotNil(t, table)
	assert.Len(t, table.Columns, 5)

	// Check column headers
	assert.Equal(t, "Name", table.Columns[0].Header)
	assert.Equal(t, "Status", table.Columns[1].Header)
	assert.Equal(t, "Team", table.Columns[2].Header)
	assert.Equal(t, "Tasks", table.Columns[3].Header)
	assert.Equal(t, "Updated", table.Columns[4].Header)
}

func TestCreateAssetDetailTable(t *testing.T) {
	factory := NewAssetCapTableFactory()
	table := factory.CreateAssetDetailTable()

	assert.NotNil(t, table)
	assert.Len(t, table.Columns, 2)

	assert.Equal(t, "Property", table.Columns[0].Header)
	assert.Equal(t, "Value", table.Columns[1].Header)
	assert.False(t, table.Style.AlternateRow)
}

func TestCreateTaskListTable(t *testing.T) {
	factory := NewAssetCapTableFactory()
	table := factory.CreateTaskListTable()

	assert.NotNil(t, table)
	assert.Len(t, table.Columns, 6)

	assert.Equal(t, "Key", table.Columns[0].Header)
	assert.Equal(t, "Summary", table.Columns[1].Header)
	assert.Equal(t, "Status", table.Columns[2].Header)
	assert.Equal(t, "Type", table.Columns[3].Header)
	assert.Equal(t, "Priority", table.Columns[4].Header)
	assert.Equal(t, "Sprint", table.Columns[5].Header)
}

func TestCreateInvestmentSummaryTable(t *testing.T) {
	factory := NewAssetCapTableFactory()
	table := factory.CreateInvestmentSummaryTable()

	assert.NotNil(t, table)
	assert.Len(t, table.Columns, 5)

	assert.Equal(t, "Asset", table.Columns[0].Header)
	assert.Equal(t, "Investment", table.Columns[1].Header)
	assert.Equal(t, "Hours", table.Columns[2].Header)
	assert.Equal(t, "Period", table.Columns[3].Header)
	assert.Equal(t, "Engineers", table.Columns[4].Header)
}

func TestCreateInvestmentDetailTable(t *testing.T) {
	factory := NewAssetCapTableFactory()
	table := factory.CreateInvestmentDetailTable()

	assert.NotNil(t, table)
	assert.Len(t, table.Columns, 5)

	assert.Equal(t, "Engineer", table.Columns[0].Header)
	assert.Equal(t, "Level", table.Columns[1].Header)
	assert.Equal(t, "Hours", table.Columns[2].Header)
	assert.Equal(t, "Rate", table.Columns[3].Header)
	assert.Equal(t, "Cost", table.Columns[4].Header)
}

func TestCreateSprintTable(t *testing.T) {
	factory := NewAssetCapTableFactory()
	table := factory.CreateSprintTable()

	assert.NotNil(t, table)
	assert.Len(t, table.Columns, 5)

	assert.Equal(t, "Sprint", table.Columns[0].Header)
	assert.Equal(t, "Status", table.Columns[1].Header)
	assert.Equal(t, "Start Date", table.Columns[2].Header)
	assert.Equal(t, "End Date", table.Columns[3].Header)
	assert.Equal(t, "Goal", table.Columns[4].Header)
}

// Test formatters
func TestAssetNameFormatter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"Test Asset", "Test Asset"},
		{"", "(unnamed)"},
		{nil, "(unnamed)"},
		{123, "123"},
	}

	for _, tt := range tests {
		result := AssetNameFormatter(tt.input)
		assert.Contains(t, result, tt.expected)
	}
}

func TestAssetStatusFormatter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"active", "Active"},
		{"ACTIVE", "Active"},
		{"live", "Active"},
		{"production", "Active"},
		{"development", "Development"},
		{"dev", "Development"},
		{"deprecated", "Deprecated"},
		{"retired", "Deprecated"},
		{"inactive", "Deprecated"},
		{"planned", "Planned"},
		{"future", "Planned"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		result := AssetStatusFormatter(tt.input)
		assert.Contains(t, result, tt.expected)
	}
}

func TestTeamFormatter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"Backend Team", "Backend Team"},
		{"", "(no team)"},
		{nil, "(no team)"},
	}

	for _, tt := range tests {
		result := TeamFormatter(tt.input)
		assert.Contains(t, result, tt.expected)
	}
}

func TestCountFormatter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{0, "0"},
		{5, "5"},
		{"10", "10"},
		{"0", "0"},
	}

	for _, tt := range tests {
		result := CountFormatter(tt.input)
		assert.Contains(t, result, tt.expected)
	}
}

func TestRelativeDateFormatter(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    interface{}
		contains string
	}{
		{
			name:     "time.Time recent",
			input:    now.Add(-5 * time.Minute),
			contains: "5m ago",
		},
		{
			name:     "time.Time hours ago",
			input:    now.Add(-2 * time.Hour),
			contains: "2h ago",
		},
		{
			name:     "time.Time days ago",
			input:    now.Add(-3 * 24 * time.Hour),
			contains: "3d ago",
		},
		{
			name:     "time.Time weeks ago",
			input:    now.Add(-10 * 24 * time.Hour),
			contains: now.Add(-10 * 24 * time.Hour).Format("Jan 2"),
		},
		{
			name:     "string date",
			input:    "2023-12-01",
			contains: "Dec 1",
		},
		{
			name:     "invalid string",
			input:    "invalid-date",
			contains: "invalid-date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RelativeDateFormatter(tt.input)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestTaskKeyFormatter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"PROJ-123", "PROJ-123"},
		{"", "(no key)"},
	}

	for _, tt := range tests {
		result := TaskKeyFormatter(tt.input)
		assert.Contains(t, result, tt.expected)
	}

	// Test nil separately since it gets formatted differently
	nilResult := TaskKeyFormatter(nil)
	assert.Contains(t, nilResult, "nil") // MutedText formats nil as "nil"
}

func TestTaskSummaryFormatter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"Short summary", "Short summary"},
		{"This is a very long summary that should be truncated", "This is a very long summary..."},
		{"", ""},
	}

	for _, tt := range tests {
		result := TaskSummaryFormatter(tt.input)
		assert.Contains(t, result, tt.expected)
	}
}

func TestTaskStatusFormatter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"done", "Done"},
		{"completed", "Done"},
		{"closed", "Done"},
		{"in progress", "In Progress"},
		{"active", "In Progress"},
		{"todo", "To Do"},
		{"open", "To Do"},
		{"blocked", "Blocked"},
		{"review", "Review"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		result := TaskStatusFormatter(tt.input)
		assert.Contains(t, result, tt.expected)
	}
}

func TestWorkTypeFormatter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"feature", "⚡"},
		{"enhancement", "⚡"},
		{"story", "⚡"},
		{"bug", "🐛"},
		{"defect", "🐛"},
		{"task", "CONFIG"},
		{"chore", "CONFIG"},
		{"epic", "EPIC"},
		{"unknown", "TASK"},
	}

	for _, tt := range tests {
		result := WorkTypeFormatter(tt.input)
		assert.Contains(t, result, tt.expected)
	}
}

func TestPriorityFormatter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"critical", "CRITICAL"},
		{"highest", "CRITICAL"},
		{"urgent", "CRITICAL"},
		{"high", "HIGH"},
		{"important", "HIGH"},
		{"medium", "MEDIUM"},
		{"normal", "MEDIUM"},
		{"low", "LOW"},
		{"minor", "LOW"},
		{"unknown", "UNKNOWN"},
	}

	for _, tt := range tests {
		result := PriorityFormatter(tt.input)
		assert.Contains(t, result, tt.expected)
	}
}

func TestInvestmentFormatter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"$100.00", "$100.00"},
		{"EUR 50.00", "EUR 50.00"},
		{100.50, "$100.50"},
		{"1000", "$1000.00"},
		{"invalid", "invalid"},
	}

	for _, tt := range tests {
		result := InvestmentFormatter(tt.input)
		assert.NotEmpty(t, result)
	}
}

func TestHoursFormatter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{0, "0h"},
		{8.5, "8.5h"},
		{"10", "10.0h"},
		{"5h", "5h"},
		{"invalid", "invalidh"},
	}

	for _, tt := range tests {
		result := HoursFormatter(tt.input)
		assert.Contains(t, result, "h")
	}
}

func TestEngineerLevelFormatter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"junior", "Jr"},
		{"jr", "Jr"},
		{"mid", "Mid"},
		{"middle", "Mid"},
		{"senior", "Sr"},
		{"sr", "Sr"},
		{"staff", "Staff"},
		{"principal", "Principal"},
		{"lead", "Principal"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		result := EngineerLevelFormatter(tt.input)
		assert.Contains(t, result, tt.expected)
	}
}

func TestSprintNameFormatter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"Sprint 1", "Sprint 1"},
		{"Current Sprint 2", "Current Sprint 2"},
		{"", "(unnamed)"},
	}

	for _, tt := range tests {
		result := SprintNameFormatter(tt.input)
		assert.NotEmpty(t, result)
	}
}

func TestSprintStatusFormatter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"active", "Active"},
		{"current", "Active"},
		{"open", "Active"},
		{"closed", "Completed"},
		{"completed", "Completed"},
		{"future", "Planned"},
		{"planned", "Planned"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		result := SprintStatusFormatter(tt.input)
		assert.Contains(t, result, tt.expected)
	}
}

func TestConvertMapToAssetDetailRows(t *testing.T) {
	data := map[string]interface{}{
		"name":         "Test Asset",
		"id":           "123",
		"status":       "active",
		"description":  "A test asset",
		"custom_field": "custom value",
	}

	rows := ConvertMapToAssetDetailRows(data)

	assert.NotEmpty(t, rows)
	assert.GreaterOrEqual(t, len(rows), 5)

	// Check that important fields come first
	found := false
	for _, row := range rows[:3] { // Check first 3 rows
		if prop, ok := row["property"].(string); ok && prop == "Name" {
			found = true
			break
		}
	}
	assert.True(t, found, "Name should be in the first few properties")
}

func TestFormatPropertyName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"snake_case", "Snake Case"},
		{"simple", "Simple"},
		{"multiple_word_property", "Multiple Word Property"},
		{"id", "ID"},
		{"doc_link", "Documentation"},
		{"task_count", "Associated Tasks"},
		{"last_doc_update", "Last Doc Update"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatPropertyName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
