package infrastructure

import (
	"context"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain"
)

const (
	testCSVHeader = "sprint,issueKey,issueType,issueTitle,workType,assetName,status,dateStarted,dateCompleted,John Doe"
	testCSVData   = testCSVHeader + "\nSprint1,TASK-1,Story,Test Task,development,Test Asset,Done,2025-01-01,2025-01-15,50.00%"
)

func TestNewSimpleInvestmentCalculator(t *testing.T) {
	costModelRepo := &mockCostModelRepo{}
	calculator := NewSimpleInvestmentCalculator(costModelRepo)

	if calculator == nil {
		t.Fatal("Expected non-nil calculator")
	}
	if calculator.costModelRepo != costModelRepo {
		t.Error("Expected cost model repo to be set correctly")
	}
}

func TestSimpleInvestmentCalculator_CalculateSprintInvestmentFromCSV(t *testing.T) {
	costModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	costModel.SetDefaultRate(domain.Senior, 75.0)
	costModel.SetDefaultRate(domain.Mid, 60.0)

	costModelRepo := &mockCostModelRepo{
		costModels: map[string]*domain.CostModel{
			"TEST": costModel,
		},
	}

	calculator := NewSimpleInvestmentCalculator(costModelRepo)

	csvData := `sprint,issueKey,issueType,issueTitle,workType,assetName,status,dateStarted,dateCompleted,John Doe,Jane Smith
Sprint1,TASK-1,Story,Test Task,development,Test Asset,Done,2025-01-01,2025-01-15,50.00%,30.00%
Sprint1,TASK-2,Bug,Bug Fix,maintenance,Test Asset,Done,2025-01-01,2025-01-10,0.00%,70.00%`

	ctx := context.Background()
	investment, err := calculator.CalculateSprintInvestmentFromCSV(ctx, "TEST", "Sprint1", csvData)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if investment.AssetName != "TEST-Sprint1" {
		t.Errorf("Expected asset name 'TEST-Sprint1', got %s", investment.AssetName)
	}

	if len(investment.EngineersInvolved) == 0 {
		t.Error("Expected engineers to be involved")
	}
}

func TestSimpleInvestmentCalculator_CalculateSprintInvestmentFromCSV_WithFallbackCostModel(t *testing.T) {
	defaultCostModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	defaultCostModel.SetDefaultRate(domain.Senior, 75.0)

	costModelRepo := &mockCostModelRepo{
		costModels:   make(map[string]*domain.CostModel), // Empty - force fallback
		defaultModel: defaultCostModel,
	}

	calculator := NewSimpleInvestmentCalculator(costModelRepo)

	csvData := testCSVData

	ctx := context.Background()
	investment, err := calculator.CalculateSprintInvestmentFromCSV(ctx, "NONEXISTENT", "Sprint1", csvData)

	if err != nil {
		t.Fatalf("Expected no error with fallback, got %v", err)
	}

	if investment.TotalCost.Currency != domain.EUR {
		t.Errorf("Expected EUR currency from default model, got %v", investment.TotalCost.Currency)
	}
}

func TestSimpleInvestmentCalculator_CalculateSprintInvestmentFromCSV_CostModelError(t *testing.T) {
	costModelRepo := &mockCostModelRepo{
		costModels:   make(map[string]*domain.CostModel),
		defaultModel: nil, // Both will fail
		forceError:   true,
	}

	calculator := NewSimpleInvestmentCalculator(costModelRepo)

	csvData := testCSVData

	ctx := context.Background()
	_, err := calculator.CalculateSprintInvestmentFromCSV(ctx, "NONEXISTENT", "Sprint1", csvData)

	if err == nil {
		t.Error("Expected error when both cost models fail")
	}
}

func TestSimpleInvestmentCalculator_FindEngineerColumnsStart(t *testing.T) {
	calculator := NewSimpleInvestmentCalculator(nil)

	// Standard header
	header := []string{"sprint", "issueKey", "issueType", "issueTitle", "workType", "assetName", "status", "dateStarted", "dateCompleted", "John Doe", "Jane Smith"}
	startIndex := calculator.findEngineerColumnsStart(header)
	expectedIndex := 9
	if startIndex != expectedIndex {
		t.Errorf("Expected engineer columns to start at index %d, got %d", expectedIndex, startIndex)
	}

	// Header with engineer names earlier
	header2 := []string{"col1", "col2", "Alice Johnson", "Bob Smith"}
	startIndex = calculator.findEngineerColumnsStart(header2)
	expectedIndex = 2
	if startIndex != expectedIndex {
		t.Errorf("Expected engineer columns to start at index %d, got %d", expectedIndex, startIndex)
	}

	// No engineer names
	header3 := []string{"col1", "col2", "data1", "data2"}
	startIndex = calculator.findEngineerColumnsStart(header3)
	if startIndex != -1 {
		t.Errorf("Expected -1 when no engineer names found, got %d", startIndex)
	}
}

func TestSimpleInvestmentCalculator_LooksLikeEngineerName(t *testing.T) {
	calculator := NewSimpleInvestmentCalculator(nil)

	testCases := map[string]bool{
		"John Doe":      true,
		"Alice Johnson": true,
		"Bob":           false, // Single name
		"john smith":    false, // Lowercase first letter
		"123":           false, // Numeric
		"":              false, // Empty
		"A":             false, // Too short
		"Data Column":   true,  // Could be mistaken for name
	}

	for name, expected := range testCases {
		result := calculator.looksLikeEngineerName(name)
		if result != expected {
			t.Errorf("Expected looksLikeEngineerName('%s') to be %t, got %t", name, expected, result)
		}
	}
}

func TestSimpleInvestmentCalculator_ParseCSVLine(t *testing.T) {
	calculator := NewSimpleInvestmentCalculator(nil)

	// Simple CSV line
	line := "Sprint1,TASK-1,Story,Test Task,development"
	fields := calculator.parseCSVLine(line)
	expected := []string{"Sprint1", "TASK-1", "Story", "Test Task", "development"}

	if len(fields) != len(expected) {
		t.Errorf("Expected %d fields, got %d", len(expected), len(fields))
	}

	for i, field := range fields {
		if field != expected[i] {
			t.Errorf("Expected field %d to be '%s', got '%s'", i, expected[i], field)
		}
	}

	// CSV line with quoted fields containing commas
	line2 := `Sprint1,"TASK-1","Story","Test, Task",development`
	fields2 := calculator.parseCSVLine(line2)
	// Simple CSV parser just removes quotes, doesn't handle complex CSV parsing
	if fields2[3] != "Test" { // After splitting by comma, it becomes just "Test"
		t.Errorf("Expected simple parsed field 'Test', got '%s'", fields2[3])
	}
}

func TestSimpleInvestmentCalculator_GetFieldValue(t *testing.T) {
	calculator := NewSimpleInvestmentCalculator(nil)

	record := []string{"value1", "value2", "value3"}

	// Valid index
	result := calculator.getFieldValue(record, 1, "default")
	if result != "value2" {
		t.Errorf("Expected 'value2', got '%s'", result)
	}

	// Invalid index (should return empty string)
	result = calculator.getFieldValue(record, 10, "default")
	if result != "" {
		t.Errorf("Expected empty string for out of bounds, got '%s'", result)
	}

	// Test with whitespace
	recordWithSpaces := []string{" spaced value ", "normal"}
	result = calculator.getFieldValue(recordWithSpaces, 0, "default")
	if result != "spaced value" {
		t.Errorf("Expected trimmed value 'spaced value', got '%s'", result)
	}
}

func TestSimpleInvestmentCalculator_InferEngineerLevel(t *testing.T) {
	_ = NewSimpleInvestmentCalculator(nil)

	costModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	costModel.SetDefaultRate(domain.Junior, 45.0)
	costModel.SetDefaultRate(domain.Mid, 60.0)
	costModel.SetDefaultRate(domain.Senior, 75.0)
	costModel.SetDefaultRate(domain.Staff, 85.0)
	costModel.SetDefaultRate(domain.Principal, 95.0)

	testCases := map[float64]domain.EngineerLevel{
		45.0: domain.Junior,
		60.0: domain.Mid,
		75.0: domain.Senior,
		85.0: domain.Staff,
		95.0: domain.Principal,
		// Rate-based inference (fallback ranges only, avoiding tolerance overlaps)
		105.0: domain.Principal, // >= 80 fallback (clearly outside all tolerances)
		73.0:  domain.Senior,    // Within Senior tolerance (67.5-82.5)
		68.0:  domain.Senior,    // >= 60 < 70 fallback (outside Mid 66.0 max, below Senior 67.5 min)
		50.0:  domain.Mid,       // >= 45 < 60
		40.0:  domain.Junior,    // < 45
	}

	for rate, expectedLevel := range testCases {
		level := costModel.InferEngineerLevel(rate)
		if level != expectedLevel {
			t.Errorf("Expected rate %.1f to infer level %s, got %s", rate, expectedLevel, level)
		}
	}
}

func TestSimpleInvestmentCalculator_InferEngineerLevel_EdgeCases(t *testing.T) {
	_ = NewSimpleInvestmentCalculator(nil)

	costModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	costModel.SetDefaultRate(domain.Junior, 45.0)
	costModel.SetDefaultRate(domain.Mid, 60.0)
	costModel.SetDefaultRate(domain.Senior, 75.0)
	costModel.SetDefaultRate(domain.Staff, 85.0)
	costModel.SetDefaultRate(domain.Principal, 95.0)

	// Test edge cases and boundary values
	edgeCases := map[float64]domain.EngineerLevel{
		0.0:   domain.Junior,    // Very low rate
		44.9:  domain.Junior,    // Just below 45 fallback
		45.1:  domain.Junior,    // Within Junior tolerance (40.5-49.5)
		76.4:  domain.Senior,    // Within Senior tolerance (67.5-82.5)
		77.0:  domain.Senior,    // Within Senior tolerance (67.5-82.5) - first match depending on map iteration
		93.6:  domain.Principal, // Within Principal tolerance (85.5-104.5)
		104.5: domain.Principal, // Above all tolerances, fallback >= 80
		200.0: domain.Principal, // Very high rate, fallback >= 80
	}

	for rate, expectedLevel := range edgeCases {
		level := costModel.InferEngineerLevel(rate)
		if level != expectedLevel {
			t.Errorf("Expected edge case rate %.1f to infer level %s, got %s", rate, expectedLevel, level)
		}
	}
}

// Mock cost model repository for testing
type mockCostModelRepo struct {
	costModels   map[string]*domain.CostModel
	defaultModel *domain.CostModel
	forceError   bool
}

func (m *mockCostModelRepo) GetCostModel(_ context.Context, project string) (*domain.CostModel, error) {
	if m.forceError {
		return nil, domain.ErrCostModelNotFound
	}
	if model, exists := m.costModels[project]; exists {
		return model, nil
	}
	return nil, domain.ErrCostModelNotFound
}

func (m *mockCostModelRepo) SaveCostModel(_ context.Context, project string, model *domain.CostModel) error {
	if m.forceError {
		return domain.ErrCostModelNotFound
	}
	if m.costModels == nil {
		m.costModels = make(map[string]*domain.CostModel)
	}
	m.costModels[project] = model
	return nil
}

func (m *mockCostModelRepo) GetDefaultCostModel(_ context.Context) (*domain.CostModel, error) {
	if m.forceError {
		return nil, domain.ErrCostModelNotFound
	}
	if m.defaultModel != nil {
		return m.defaultModel, nil
	}
	costModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	return costModel, nil
}

func TestSimpleInvestmentCalculator_CalculateSprintInvestmentFromCSV_EmptyInput(t *testing.T) {
	calculator := NewSimpleInvestmentCalculator(&mockCostModelRepo{})

	ctx := context.Background()

	// Test with empty CSV data
	_, err := calculator.CalculateSprintInvestmentFromCSV(ctx, "TEST", "Sprint1", "")
	if err == nil {
		t.Error("Expected error for empty CSV data")
	}

	// Test with only header - should still fail as invalid CSV (< 2 lines)
	csvDataHeaderOnly := "sprint,issueKey,issueType,issueTitle,workType,assetName,status,dateStarted,dateCompleted"
	_, err = calculator.CalculateSprintInvestmentFromCSV(ctx, "TEST", "Sprint1", csvDataHeaderOnly)
	if err == nil {
		t.Error("Expected error for header-only CSV (< 2 lines)")
	}
}

func TestSimpleInvestmentCalculator_CalculateSprintInvestmentFromCSV_NoEngineerColumns(t *testing.T) {
	calculator := NewSimpleInvestmentCalculator(&mockCostModelRepo{})

	ctx := context.Background()

	// CSV with no engineer columns
	csvData := `col1,col2,col3
data1,data2,data3`

	_, err := calculator.CalculateSprintInvestmentFromCSV(ctx, "TEST", "Sprint1", csvData)
	if err == nil {
		t.Error("Expected error when no engineer columns found")
	}
}

func TestSimpleInvestmentCalculator_CalculateSprintInvestmentFromCSV_MalformedCSV(t *testing.T) {
	costModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	costModel.SetDefaultRate(domain.Senior, 75.0)

	costModelRepo := &mockCostModelRepo{
		costModels: map[string]*domain.CostModel{
			"TEST": costModel,
		},
	}

	calculator := NewSimpleInvestmentCalculator(costModelRepo)

	ctx := context.Background()

	// CSV with malformed rows (too few columns)
	csvData := `sprint,issueKey,issueType,issueTitle,workType,assetName,status,dateStarted,dateCompleted,John Doe
Sprint1,TASK-1,Story
Sprint1,TASK-2,Bug,Bug Fix,maintenance,Test Asset,Done,2025-01-01,2025-01-10,50.00%`

	investment, err := calculator.CalculateSprintInvestmentFromCSV(ctx, "TEST", "Sprint1", csvData)
	if err != nil {
		t.Fatalf("Expected no error for malformed CSV (should skip bad rows), got: %v", err)
	}

	// Should only process the valid row
	if len(investment.EngineersInvolved) == 0 {
		t.Error("Expected at least one engineer from the valid row")
	}
}
