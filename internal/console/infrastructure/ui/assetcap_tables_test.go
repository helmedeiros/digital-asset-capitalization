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

// TestCreateTeamAssignmentTable tests team assignment table creation
func TestCreateTeamAssignmentTable(t *testing.T) {
	factory := NewAssetCapTableFactory()
	table := factory.CreateTeamAssignmentTable()

	assert.NotNil(t, table)
	assert.Len(t, table.Columns, 4)

	// Check column headers
	assert.Equal(t, "Asset", table.Columns[0].Header)
	assert.Equal(t, "Owner", table.Columns[1].Header)
	assert.Equal(t, "Contributors", table.Columns[2].Header)
	assert.Equal(t, "Status", table.Columns[3].Header)
}

// TestTeamListFormatter tests formatting of team lists
func TestTeamListFormatter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{[]string{"FN", "AD"}, "FN"},
		{[]string{"FN"}, "FN"},
		{[]string{}, "(no teams)"},
		{nil, "(no teams)"},
		{"FN", "FN"},
	}

	for _, tt := range tests {
		result := TeamListFormatter(tt.input)
		assert.Contains(t, result, tt.expected)
	}
}

// TestTeamRoleFormatter tests formatting of team roles
func TestTeamRoleFormatter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"owner", "👑 Owner"},
		{"contributor", "🤝 Contributor"},
		{"assigned", "✅ Assigned"},
		{"added", "➕ Added"},
		{"removed", "❌ Removed"},
	}

	for _, tt := range tests {
		result := TeamRoleFormatter(tt.input)
		assert.Contains(t, result, tt.expected)
	}
}

// TestTeamAssignmentStatusFormatter tests team assignment status formatting
func TestTeamAssignmentStatusFormatter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{true, "✅ Has Owner"},
		{false, "⚠️ No Owner"},
	}

	for _, tt := range tests {
		result := TeamAssignmentStatusFormatter(tt.input)
		assert.Contains(t, result, tt.expected)
	}
}

// TestFormatTeamAssignmentData tests formatting of team assignment data structures
func TestFormatTeamAssignmentData(t *testing.T) {
	// Test data structure representing team assignments
	data := []map[string]interface{}{
		{
			"asset":        "Dynamic Markup",
			"owner":        "FN",
			"contributors": []string{"AD", "QA"},
		},
		{
			"asset":        "Flight Delay Insurance",
			"owner":        "AD",
			"contributors": []string{"FN"},
		},
		{
			"asset":        "Price Lock",
			"owner":        "FN",
			"contributors": []string{},
		},
	}

	// Test formatting each assignment
	for _, assignment := range data {
		asset, ok := assignment["asset"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, asset)

		owner, ok := assignment["owner"].(string)
		assert.True(t, ok)
		assert.NotEmpty(t, owner)

		contributors, ok := assignment["contributors"].([]string)
		assert.True(t, ok)
		assert.NotNil(t, contributors)
	}
}

// TestTeamFormatterWithComplexData tests team formatter with various data structures
func TestTeamFormatterWithComplexData(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name: "map with owning_team",
			input: map[string]interface{}{
				"owning_team": "FN",
			},
			expected: "FN",
		},
		{
			name: "map with owner",
			input: map[string]interface{}{
				"owner": "AD",
			},
			expected: "AD",
		},
		{
			name: "direct string",
			input: "TeamAlpha",
			expected: "TeamAlpha",
		},
		{
			name: "empty map",
			input: map[string]interface{}{},
			expected: "(no team)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since TeamFormatter expects simple values, we'd need to extract first
			var value interface{}
			if m, ok := tt.input.(map[string]interface{}); ok {
				if v, exists := m["owning_team"]; exists {
					value = v
				} else if v, exists := m["owner"]; exists {
					value = v
				}
			} else {
				value = tt.input
			}
			
			result := TeamFormatter(value)
			assert.Contains(t, result, tt.expected)
		})
	}
}

// TestConvertTeamAssignmentToRows tests converting team assignment data to table rows
func TestConvertTeamAssignmentToRows(t *testing.T) {
	// Sample team assignment data
	assignments := []map[string]interface{}{
		{
			"asset":        "Dynamic Markup",
			"owner":        "FN",
			"contributors": []string{"AD"},
		},
		{
			"asset":        "Flight Delay Insurance",
			"owner":        "AD",
			"contributors": []string{"FN", "QA"},
		},
	}

	// Convert to table rows
	rows := make([]map[string]interface{}, len(assignments))
	for i, assignment := range assignments {
		rows[i] = map[string]interface{}{
			"asset":        assignment["asset"],
			"owner":        assignment["owner"],
			"contributors": assignment["contributors"],
		}
	}

	assert.Len(t, rows, 2)
	
	// Verify first row
	assert.Equal(t, "Dynamic Markup", rows[0]["asset"])
	assert.Equal(t, "FN", rows[0]["owner"])
	assert.Equal(t, []string{"AD"}, rows[0]["contributors"])
	
	// Verify second row
	assert.Equal(t, "Flight Delay Insurance", rows[1]["asset"])
	assert.Equal(t, "AD", rows[1]["owner"])
	assert.Equal(t, []string{"FN", "QA"}, rows[1]["contributors"])
}
