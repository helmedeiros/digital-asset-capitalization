package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewTable(t *testing.T) {
	columns := []TableColumn{
		{Header: "Name", Key: "name", Align: "left"},
		{Header: "Age", Key: "age", Align: "right"},
	}

	table := NewTable(columns)

	if len(table.Columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(table.Columns))
	}

	if table.Columns[0].Header != "Name" {
		t.Errorf("Expected first column header to be 'Name', got '%s'", table.Columns[0].Header)
	}
}

func TestTableAddRow(t *testing.T) {
	columns := []TableColumn{
		{Header: "Name", Key: "name"},
		{Header: "Age", Key: "age"},
	}

	table := NewTable(columns)
	row := map[string]interface{}{
		"name": "John",
		"age":  30,
	}

	table.AddRow(row)

	if len(table.Rows) != 1 {
		t.Errorf("Expected 1 row, got %d", len(table.Rows))
	}

	if table.Rows[0]["name"] != "John" {
		t.Errorf("Expected name to be 'John', got '%v'", table.Rows[0]["name"])
	}
}

func TestTableRender(t *testing.T) {
	columns := []TableColumn{
		{Header: "Name", Key: "name", Width: 10},
		{Header: "Status", Key: "status", Width: 10},
	}

	table := NewTable(columns)
	table.AddRow(map[string]interface{}{
		"name":   "Test",
		"status": "active",
	})

	result := table.Render()

	// Should contain headers
	if !strings.Contains(result, "Name") || !strings.Contains(result, "Status") {
		t.Error("Expected table to contain headers")
	}

	// Should contain data
	if !strings.Contains(result, "Test") || !strings.Contains(result, "active") {
		t.Error("Expected table to contain row data")
	}

	// Should contain box drawing characters
	if !strings.Contains(result, "┌") || !strings.Contains(result, "┐") {
		t.Error("Expected table to contain top border")
	}
}

func TestDefaultFormatter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"hello", "hello"},
		{42, "42"},
		{3.14, "3.14"},
		{true, "true"},
		{false, "false"},
		{nil, ""},
	}

	for _, test := range tests {
		result := DefaultFormatter(test.input)
		if !strings.Contains(result, test.expected) {
			t.Errorf("DefaultFormatter(%v) expected to contain '%s', got '%s'", test.input, test.expected, result)
		}
	}
}

func TestMoneyFormatter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{100.50, "$100.50"},
		{"$50.00", "$50.00"},
		{"EUR 75.25", "EUR 75.25"},
	}

	for _, test := range tests {
		result := MoneyFormatter(test.input)
		if !strings.Contains(result, test.expected) {
			t.Errorf("MoneyFormatter(%v) expected to contain '%s', got '%s'", test.input, test.expected, result)
		}
	}
}

func TestDateFormatter(t *testing.T) {
	testTime := time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC)
	result := DateFormatter(testTime)

	if !strings.Contains(result, "2023-12-25") {
		t.Errorf("Expected date formatter to contain '2023-12-25', got '%s'", result)
	}
}

func TestStatusFormatter(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
	}{
		{"active", "active"},
		{"completed", "completed"},
		{"failed", "failed"},
		{"pending", "pending"},
	}

	for _, test := range tests {
		result := StatusFormatter(test.input)
		if !strings.Contains(result, test.expected) {
			t.Errorf("StatusFormatter(%v) expected to contain '%s', got '%s'", test.input, test.expected, result)
		}

		// Should contain the original value
		if !strings.Contains(result, fmt.Sprintf("%v", test.input)) {
			t.Errorf("StatusFormatter(%v) expected to contain original value", test.input)
		}
	}
}

func TestRenderKeyValueList(t *testing.T) {
	pairs := []KeyValuePair{
		{Key: "Name", Value: "Test Asset"},
		{Key: "Status", Value: "active"},
		{Key: "Count", Value: 42},
	}

	result := RenderKeyValueList(pairs, DefaultTableStyle())

	// Should contain all keys and values
	expectedStrings := []string{"Name:", "Test Asset", "Status:", "active", "Count:", "42"}
	for _, expected := range expectedStrings {
		if !strings.Contains(result, expected) {
			t.Errorf("Expected key-value list to contain '%s'", expected)
		}
	}
}

func TestCreateAssetTable(t *testing.T) {
	table := CreateAssetTable()

	if len(table.Columns) == 0 {
		t.Error("Expected asset table to have columns")
	}

	// Should have typical asset columns
	hasNameColumn := false
	hasStatusColumn := false

	for _, col := range table.Columns {
		if col.Key == "name" {
			hasNameColumn = true
		}
		if col.Key == "status" {
			hasStatusColumn = true
		}
	}

	if !hasNameColumn {
		t.Error("Expected asset table to have name column")
	}

	if !hasStatusColumn {
		t.Error("Expected asset table to have status column")
	}
}

func TestCreateTaskTable(t *testing.T) {
	table := CreateTaskTable()

	if len(table.Columns) == 0 {
		t.Error("Expected task table to have columns")
	}

	// Should have typical task columns
	hasKeyColumn := false
	hasSummaryColumn := false

	for _, col := range table.Columns {
		if col.Key == "key" {
			hasKeyColumn = true
		}
		if col.Key == "summary" {
			hasSummaryColumn = true
		}
	}

	if !hasKeyColumn {
		t.Error("Expected task table to have key column")
	}

	if !hasSummaryColumn {
		t.Error("Expected task table to have summary column")
	}
}

func TestSimpleList(t *testing.T) {
	items := []string{"Item 1", "Item 2", "Item 3"}
	result := SimpleList(items, "•")

	for _, item := range items {
		if !strings.Contains(result, item) {
			t.Errorf("Expected list to contain '%s'", item)
		}
	}

	if !strings.Contains(result, "•") {
		t.Error("Expected list to contain bullet character")
	}
}

func TestNumberedList(t *testing.T) {
	items := []string{"First", "Second", "Third"}
	result := NumberedList(items)

	expectedNumbers := []string{"1.", "2.", "3."}
	for i, num := range expectedNumbers {
		if !strings.Contains(result, num) {
			t.Errorf("Expected numbered list to contain '%s'", num)
		}

		if !strings.Contains(result, items[i]) {
			t.Errorf("Expected numbered list to contain '%s'", items[i])
		}
	}
}

func TestSortMapKeys(t *testing.T) {
	testMap := map[string]interface{}{
		"zebra": 1,
		"alpha": 2,
		"beta":  3,
	}

	keys := SortMapKeys(testMap)

	expectedOrder := []string{"alpha", "beta", "zebra"}

	if len(keys) != len(expectedOrder) {
		t.Errorf("Expected %d keys, got %d", len(expectedOrder), len(keys))
	}

	for i, expected := range expectedOrder {
		if keys[i] != expected {
			t.Errorf("Expected key at position %d to be '%s', got '%s'", i, expected, keys[i])
		}
	}
}
