package infrastructure

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestTimeAllocationAdapter_GetAssetAllocations_NotImplemented(t *testing.T) {
	adapter := NewTimeAllocationAdapter()
	ctx := context.Background()

	_, err := adapter.GetAssetAllocations(ctx, "Test Asset")
	if err == nil {
		t.Error("Expected error for not implemented GetAssetAllocations")
	}

	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("Expected 'not yet implemented' error, got: %v", err)
	}
}

func TestTimeAllocationAdapter_GetTaskAllocations_NotImplemented(t *testing.T) {
	adapter := NewTimeAllocationAdapter()
	ctx := context.Background()

	_, err := adapter.GetTaskAllocations(ctx, []string{"TASK-1", "TASK-2"})
	if err == nil {
		t.Error("Expected error for not implemented GetTaskAllocations")
	}

	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("Expected 'not yet implemented' error, got: %v", err)
	}
}

func TestTimeAllocationAdapter_ParseSprintAllocationOutput_ValidCSV(t *testing.T) {
	adapter := NewTimeAllocationAdapter()

	// Create valid CSV output
	csvOutput := `sprint,issueKey,issueType,issueTitle,workType,assetName,status,dateStarted,dateCompleted,John Doe,Jane Smith
Sprint1,TASK-1,Story,Test Task,development,Test Asset,Done,2025-01-01,2025-01-15,50.00%,30.00%
Sprint1,TASK-2,Bug,Bug Fix,maintenance,Test Asset,Done,2025-01-01,2025-01-10,0.00%,70.00%`

	allocations, err := adapter.parseSprintAllocationOutput(csvOutput, "Sprint1")
	if err != nil {
		t.Fatalf("Failed to parse valid CSV: %v", err)
	}

	expectedAllocations := 3 // John Doe on TASK-1, Jane Smith on TASK-1, Jane Smith on TASK-2
	if len(allocations) != expectedAllocations {
		t.Errorf("Expected %d allocations, got %d", expectedAllocations, len(allocations))
	}

	// Check first allocation (John Doe on TASK-1)
	johnTask1 := allocations[0]
	if johnTask1.EngineerName != "John Doe" {
		t.Errorf("Expected engineer 'John Doe', got %s", johnTask1.EngineerName)
	}
	if johnTask1.TaskKey != "TASK-1" {
		t.Errorf("Expected task key 'TASK-1', got %s", johnTask1.TaskKey)
	}
	if johnTask1.Allocation != 50.0 {
		t.Errorf("Expected allocation 50.0, got %.1f", johnTask1.Allocation)
	}
	if johnTask1.WorkType != "development" {
		t.Errorf("Expected work type 'development', got %s", johnTask1.WorkType)
	}
}

func TestTimeAllocationAdapter_ParseSprintAllocationOutput_InvalidCSV(t *testing.T) {
	adapter := NewTimeAllocationAdapter()

	// Invalid CSV content
	invalidCSV := "invalid,csv\nwith,missing,columns"

	_, err := adapter.parseSprintAllocationOutput(invalidCSV, "Sprint1")
	if err == nil {
		t.Error("Expected error for invalid CSV")
	}
}

func TestTimeAllocationAdapter_ParseSprintAllocationOutput_EmptyCSV(t *testing.T) {
	adapter := NewTimeAllocationAdapter()

	// Empty CSV (just header)
	emptyCSV := "sprint,issueKey,issueType,issueTitle,workType,assetName,status,dateStarted,dateCompleted"

	_, err := adapter.parseSprintAllocationOutput(emptyCSV, "Sprint1")
	if err == nil {
		t.Error("Expected error for CSV without data rows")
	}
}

func TestTimeAllocationAdapter_ParseSprintAllocationOutput_MalformedData(t *testing.T) {
	adapter := NewTimeAllocationAdapter()

	// CSV with malformed data
	malformedCSV := `sprint,issueKey,issueType,issueTitle,workType,assetName,status,dateStarted,dateCompleted,John Doe
Sprint1,TASK-1,Story` // Missing columns

	_, err := adapter.parseSprintAllocationOutput(malformedCSV, "Sprint1")
	if err == nil {
		t.Error("Expected error for malformed CSV data")
	}
}

func TestTimeAllocationAdapter_ParseSprintAllocationOutput_ZeroAllocations(t *testing.T) {
	adapter := NewTimeAllocationAdapter()

	// CSV with zero allocations (should be skipped)
	csvOutput := `sprint,issueKey,issueType,issueTitle,workType,assetName,status,dateStarted,dateCompleted,John Doe,Jane Smith
Sprint1,TASK-1,Story,Test Task,development,Test Asset,Done,2025-01-01,2025-01-15,0.00%,0.00%`

	allocations, err := adapter.parseSprintAllocationOutput(csvOutput, "Sprint1")
	if err != nil {
		t.Fatalf("Failed to parse CSV with zero allocations: %v", err)
	}

	// Should skip zero allocations
	if len(allocations) != 0 {
		t.Errorf("Expected 0 allocations (zeros should be skipped), got %d", len(allocations))
	}
}

func TestTimeAllocationAdapter_ParseSprintAllocationOutput_InvalidAllocationPercentage(t *testing.T) {
	adapter := NewTimeAllocationAdapter()

	// CSV with invalid percentage format
	csvOutput := `sprint,issueKey,issueType,issueTitle,workType,assetName,status,dateStarted,dateCompleted,John Doe
Sprint1,TASK-1,Story,Test Task,development,Test Asset,Done,2025-01-01,2025-01-15,invalid%`

	allocations, err := adapter.parseSprintAllocationOutput(csvOutput, "Sprint1")
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Should skip invalid allocations and continue processing
	if len(allocations) != 0 {
		t.Errorf("Expected 0 allocations (invalid should be skipped), got %d", len(allocations))
	}
}

func TestTimeAllocationAdapter_FindEngineerColumnsStart_StandardColumns(t *testing.T) {
	adapter := NewTimeAllocationAdapter()

	// Standard header
	header := []string{"sprint", "issueKey", "issueType", "issueTitle", "workType", "assetName", "status", "dateStarted", "dateCompleted", "John Doe", "Jane Smith"}

	startIndex := adapter.findEngineerColumnsStart(header)
	expectedIndex := 9 // After 9 standard columns
	if startIndex != expectedIndex {
		t.Errorf("Expected engineer columns to start at index %d, got %d", expectedIndex, startIndex)
	}
}

func TestTimeAllocationAdapter_FindEngineerColumnsStart_NonStandardColumns(t *testing.T) {
	adapter := NewTimeAllocationAdapter()

	// Non-standard header (should fall back to name detection)
	header := []string{"col1", "col2", "col3", "John Doe", "Jane Smith"}

	startIndex := adapter.findEngineerColumnsStart(header)
	expectedIndex := 3 // Should find "John Doe" at index 3
	if startIndex != expectedIndex {
		t.Errorf("Expected engineer columns to start at index %d, got %d", expectedIndex, startIndex)
	}
}

func TestTimeAllocationAdapter_FindEngineerColumnsStart_NoEngineerNames(t *testing.T) {
	adapter := NewTimeAllocationAdapter()

	// Header without engineer names
	header := []string{"col1", "col2", "col3", "data1", "data2"}

	startIndex := adapter.findEngineerColumnsStart(header)
	if startIndex != -1 {
		t.Errorf("Expected -1 for no engineer columns found, got %d", startIndex)
	}
}

func TestTimeAllocationAdapter_LooksLikeEngineerName(t *testing.T) {
	adapter := NewTimeAllocationAdapter()

	testCases := map[string]bool{
		"John Doe":           true,
		"Jane Smith Johnson": true,
		"Bob":                false, // Single name
		"john doe":           false, // Lowercase first letter
		"123":                false, // Numeric
		"":                   false, // Empty
		"A B":                false, // Too short - less than 4 chars total
	}

	for name, expected := range testCases {
		result := adapter.looksLikeEngineerName(name)
		if result != expected {
			t.Errorf("Expected looksLikeEngineerName('%s') to be %t, got %t", name, expected, result)
		}
	}
}

func TestTimeAllocationAdapter_GetFieldValue_SafeAccess(t *testing.T) {
	adapter := NewTimeAllocationAdapter()

	record := []string{"value1", "value2", "value3"}

	// Valid index
	result := adapter.getFieldValue(record, 1, "default")
	if result != "value2" {
		t.Errorf("Expected 'value2', got '%s'", result)
	}

	// Invalid index (should return empty string - implementation ignores default parameter)
	result = adapter.getFieldValue(record, 10, "default")
	if result != "" {
		t.Errorf("Expected empty string for out of bounds index, got '%s'", result)
	}

	// Test with trimming
	recordWithSpaces := []string{" spaced value ", "normal"}
	result = adapter.getFieldValue(recordWithSpaces, 0, "default")
	if result != "spaced value" {
		t.Errorf("Expected 'spaced value' (trimmed), got '%s'", result)
	}
}

func TestTimeAllocationAdapter_ParseSprintAllocationOutput_DateParsing(t *testing.T) {
	adapter := NewTimeAllocationAdapter()

	// CSV with valid dates
	csvOutput := `sprint,issueKey,issueType,issueTitle,workType,assetName,status,dateStarted,dateCompleted,John Doe
Sprint1,TASK-1,Story,Test Task,development,Test Asset,Done,2025-01-01,2025-01-15,50.00%`

	allocations, err := adapter.parseSprintAllocationOutput(csvOutput, "Sprint1")
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(allocations) != 1 {
		t.Fatalf("Expected 1 allocation, got %d", len(allocations))
	}

	allocation := allocations[0]
	expectedStartDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	expectedEndDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	if !allocation.StartDate.Equal(expectedStartDate) {
		t.Errorf("Expected start date %v, got %v", expectedStartDate, allocation.StartDate)
	}
	if !allocation.EndDate.Equal(expectedEndDate) {
		t.Errorf("Expected end date %v, got %v", expectedEndDate, allocation.EndDate)
	}
}

func TestTimeAllocationAdapter_ParseSprintAllocationOutput_InvalidDates(t *testing.T) {
	adapter := NewTimeAllocationAdapter()

	// CSV with invalid dates (should not crash, dates should be zero)
	csvOutput := `sprint,issueKey,issueType,issueTitle,workType,assetName,status,dateStarted,dateCompleted,John Doe
Sprint1,TASK-1,Story,Test Task,development,Test Asset,Done,invalid-date,also-invalid,50.00%`

	allocations, err := adapter.parseSprintAllocationOutput(csvOutput, "Sprint1")
	if err != nil {
		t.Fatalf("Failed to parse CSV with invalid dates: %v", err)
	}

	if len(allocations) != 1 {
		t.Fatalf("Expected 1 allocation, got %d", len(allocations))
	}

	allocation := allocations[0]
	// Invalid dates should result in zero time
	if !allocation.StartDate.IsZero() {
		t.Errorf("Expected zero start date for invalid date, got %v", allocation.StartDate)
	}
	if !allocation.EndDate.IsZero() {
		t.Errorf("Expected zero end date for invalid date, got %v", allocation.EndDate)
	}
}

func TestNewTimeAllocationAdapter(t *testing.T) {
	adapter := NewTimeAllocationAdapter()
	if adapter == nil {
		t.Error("Expected non-nil adapter")
	}
}

func TestTimeAllocationAdapter_GetSprintAllocations_ExecutionError(t *testing.T) {
	adapter := NewTimeAllocationAdapter()
	ctx := context.Background()

	// This will fail because "./assetcap" doesn't exist in test environment
	_, err := adapter.GetSprintAllocations(ctx, "TEST", "Sprint1")
	if err == nil {
		t.Error("Expected error when executing non-existent command")
	}

	if !strings.Contains(err.Error(), "failed to execute") {
		t.Errorf("Expected execution error message, got: %v", err)
	}
}
