package domain

import (
	"testing"
	"time"
)

func TestNewCostModel(t *testing.T) {
	costModel, err := NewCostModel(EUR, 8.0, 2.0)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if costModel.Currency != EUR {
		t.Errorf("Expected currency EUR, got %v", costModel.Currency)
	}

	if costModel.WorkingHoursPerDay != 8.0 {
		t.Errorf("Expected working hours 8.0, got %.1f", costModel.WorkingHoursPerDay)
	}

	if costModel.OverheadMultiplier != 2.0 {
		t.Errorf("Expected overhead multiplier 2.0, got %.1f", costModel.OverheadMultiplier)
	}

	if costModel.EngineerRates == nil {
		t.Error("Expected engineer rates map to be initialized")
	}

	if costModel.DefaultRatesByLevel == nil {
		t.Error("Expected default rates map to be initialized")
	}
}

func TestNewCostModel_InvalidValues(t *testing.T) {
	// Test invalid working hours
	_, err := NewCostModel(EUR, 0, 2.0)
	if err == nil {
		t.Error("Expected error for zero working hours")
	}

	// Test invalid overhead multiplier
	_, err = NewCostModel(EUR, 8.0, 0.5)
	if err == nil {
		t.Error("Expected error for overhead multiplier below 1.0")
	}
}

func TestCostModel_AddEngineerRate(t *testing.T) {
	costModel, _ := NewCostModel(EUR, 8.0, 2.0)

	rate := EngineerRate{
		Name:       "John Doe",
		HourlyRate: 75.0,
		Level:      Senior,
		Team:       "PROJECT",
	}

	err := costModel.AddEngineerRate(rate)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(costModel.EngineerRates) != 1 {
		t.Errorf("Expected 1 engineer rate, got %d", len(costModel.EngineerRates))
	}

	storedRate, exists := costModel.EngineerRates["John Doe"]
	if !exists {
		t.Error("Expected engineer rate to be stored")
	}

	if storedRate.HourlyRate != 75.0 {
		t.Errorf("Expected hourly rate 75.0, got %.1f", storedRate.HourlyRate)
	}
}

func TestCostModel_SetDefaultRate(t *testing.T) {
	costModel, _ := NewCostModel(EUR, 8.0, 2.0)

	err := costModel.SetDefaultRate(Senior, 75.0)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	rate, exists := costModel.DefaultRatesByLevel[Senior]
	if !exists {
		t.Error("Expected senior rate to be set")
	}

	if rate != 75.0 {
		t.Errorf("Expected senior rate 75.0, got %.1f", rate)
	}
}

func TestCostModel_GetEngineerRateOrDefault(t *testing.T) {
	costModel, _ := NewCostModel(EUR, 8.0, 2.0)

	// Set a default rate for senior level
	costModel.SetDefaultRate(Senior, 75.0)

	// Add specific engineer rate
	costModel.AddEngineerRate(EngineerRate{
		Name:       "John Doe",
		HourlyRate: 80.0,
		Level:      Senior,
		Team:       "PROJECT",
	})

	// Test getting specific engineer rate
	rate := costModel.GetEngineerRateOrDefault("John Doe", Senior)
	if rate != 80.0 {
		t.Errorf("Expected specific rate 80.0, got %.1f", rate)
	}

	// Test getting default rate for unknown engineer
	rate = costModel.GetEngineerRateOrDefault("Jane Doe", Senior)
	if rate != 75.0 {
		t.Errorf("Expected default rate 75.0, got %.1f", rate)
	}
}

func TestCostModel_CalculateInfrastructureCostForPeriod(t *testing.T) {
	costModel, _ := NewCostModel(EUR, 8.0, 2.0)
	costModel.InfrastructureCosts.CloudCostsPerMonth = 1000.0
	costModel.InfrastructureCosts.ToolingCostsPerMonth = 500.0
	costModel.InfrastructureCosts.LicenseCostsPerMonth = 300.0

	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC) // About 30 days

	cost, err := costModel.CalculateInfrastructureCostForPeriod(startDate, endDate)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedMonthly := 1800.0 // 1000 + 500 + 300
	// For roughly 30 days, should be close to the monthly cost
	if cost < expectedMonthly*0.9 || cost > expectedMonthly*1.1 {
		t.Errorf("Expected infrastructure cost around %.2f, got %.2f", expectedMonthly, cost)
	}
}

func TestCostModel_GetTotalMonthlyCost(t *testing.T) {
	costModel, _ := NewCostModel(EUR, 8.0, 2.0)
	costModel.InfrastructureCosts.CloudCostsPerMonth = 1000.0
	costModel.InfrastructureCosts.ToolingCostsPerMonth = 500.0
	costModel.InfrastructureCosts.LicenseCostsPerMonth = 300.0

	total := costModel.GetTotalMonthlyCost()
	expected := 1800.0
	if total != expected {
		t.Errorf("Expected total monthly cost %.2f, got %.2f", expected, total)
	}
}

// Additional test cases for complete coverage

func TestCostModel_AddEngineerRate_NegativeRate(t *testing.T) {
	costModel, _ := NewCostModel(EUR, 8.0, 2.0)

	rate := EngineerRate{
		Name:       "John Doe",
		HourlyRate: -50.0, // Negative rate
		Level:      Senior,
		Team:       "PROJECT",
	}

	err := costModel.AddEngineerRate(rate)
	if err == nil {
		t.Error("Expected error for negative rate")
	}
	if err != ErrNegativeRate {
		t.Errorf("Expected ErrNegativeRate, got %v", err)
	}
}

func TestCostModel_SetDefaultRate_NegativeRate(t *testing.T) {
	costModel, _ := NewCostModel(EUR, 8.0, 2.0)

	err := costModel.SetDefaultRate(Senior, -75.0)
	if err == nil {
		t.Error("Expected error for negative rate")
	}
	if err != ErrNegativeRate {
		t.Errorf("Expected ErrNegativeRate, got %v", err)
	}
}

func TestCostModel_GetEngineerRate(t *testing.T) {
	costModel, _ := NewCostModel(EUR, 8.0, 2.0)

	// Add an engineer
	costModel.AddEngineerRate(EngineerRate{
		Name:       "John Doe",
		HourlyRate: 80.0,
		Level:      Senior,
		Team:       "PROJECT",
	})

	// Test getting existing engineer rate
	rate, err := costModel.GetEngineerRate("John Doe")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if rate != 80.0 {
		t.Errorf("Expected rate 80.0, got %.1f", rate)
	}

	// Test getting non-existent engineer rate
	_, err = costModel.GetEngineerRate("Jane Doe")
	if err == nil {
		t.Error("Expected error for non-existent engineer")
	}
	if err != ErrEngineerNotFound {
		t.Errorf("Expected ErrEngineerNotFound, got %v", err)
	}
}

func TestCostModel_GetEngineerRateOrDefault_NoDefault(t *testing.T) {
	costModel, _ := NewCostModel(EUR, 8.0, 2.0)

	// Test with no default and no specific rate
	rate := costModel.GetEngineerRateOrDefault("Unknown Engineer", Senior)
	if rate != 0 {
		t.Errorf("Expected rate 0.0 for unknown engineer with no default, got %.1f", rate)
	}
}

func TestCostModel_CalculateInfrastructureCostForPeriod_InvalidDates(t *testing.T) {
	costModel, _ := NewCostModel(EUR, 8.0, 2.0)

	startDate := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC) // End before start

	_, err := costModel.CalculateInfrastructureCostForPeriod(startDate, endDate)
	if err == nil {
		t.Error("Expected error for invalid date range")
	}
	if err != ErrInvalidDateRange {
		t.Errorf("Expected ErrInvalidDateRange, got %v", err)
	}
}

func TestCostModel_Validate(t *testing.T) {
	costModel, _ := NewCostModel(EUR, 8.0, 2.0)

	// Valid model should pass validation
	err := costModel.Validate()
	if err != nil {
		t.Errorf("Expected no error for valid model, got %v", err)
	}

	// Test with invalid working hours
	costModel.WorkingHoursPerDay = 0
	err = costModel.Validate()
	if err != ErrInvalidWorkingHours {
		t.Errorf("Expected ErrInvalidWorkingHours, got %v", err)
	}

	// Reset and test with invalid overhead
	costModel.WorkingHoursPerDay = 8.0
	costModel.OverheadMultiplier = 0.5
	err = costModel.Validate()
	if err != ErrInvalidOverhead {
		t.Errorf("Expected ErrInvalidOverhead, got %v", err)
	}

	// Reset and test with negative engineer rate
	costModel.OverheadMultiplier = 2.0
	costModel.EngineerRates["Test"] = EngineerRate{HourlyRate: -50.0}
	err = costModel.Validate()
	if err != ErrNegativeRate {
		t.Errorf("Expected ErrNegativeRate, got %v", err)
	}
}

func TestEngineerLevel_Constants(t *testing.T) {
	// Test that all constants are defined correctly
	levels := []EngineerLevel{Junior, Mid, Senior, Staff, Principal}
	expected := []string{"junior", "mid", "senior", "staff", "principal"}

	for i, level := range levels {
		if string(level) != expected[i] {
			t.Errorf("Expected level %s, got %s", expected[i], string(level))
		}
	}
}

func TestCurrency_Constants(t *testing.T) {
	// Test that all currency constants are defined correctly
	currencies := []Currency{EUR, USD, GBP}
	expected := []string{"EUR", "USD", "GBP"}

	for i, currency := range currencies {
		if string(currency) != expected[i] {
			t.Errorf("Expected currency %s, got %s", expected[i], string(currency))
		}
	}
}

func TestInfrastructureCosts_ZeroCosts(t *testing.T) {
	costModel, _ := NewCostModel(EUR, 8.0, 2.0)

	// Test with zero infrastructure costs
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)

	cost, err := costModel.CalculateInfrastructureCostForPeriod(startDate, endDate)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if cost != 0.0 {
		t.Errorf("Expected zero cost for zero infrastructure costs, got %.2f", cost)
	}
}

func TestCostModel_InferEngineerLevel(t *testing.T) {
	costModel, _ := NewCostModel(EUR, 8.0, 2.0)

	// Set up default rates for testing
	costModel.SetDefaultRate(Junior, 30.0)
	costModel.SetDefaultRate(Mid, 50.0)
	costModel.SetDefaultRate(Senior, 65.0)
	costModel.SetDefaultRate(Staff, 75.0)
	costModel.SetDefaultRate(Principal, 85.0)

	tests := []struct {
		rate     float64
		expected EngineerLevel
		name     string
	}{
		{30.0, Junior, "exact junior rate"},
		{29.0, Junior, "junior rate within tolerance"},
		{31.0, Junior, "junior rate within tolerance upper"},
		{50.0, Mid, "exact mid rate"},
		{48.0, Mid, "mid rate within tolerance"},
		{52.0, Mid, "mid rate within tolerance upper"},
		{65.0, Senior, "exact senior rate"},
		{75.0, Staff, "exact staff rate"},
		{85.0, Principal, "exact principal rate"},
		{90.0, Principal, "high rate fallback to principal"},
		{72.0, Staff, "staff rate fallback"},
		{62.0, Senior, "senior rate fallback"},
		{47.0, Mid, "mid rate fallback"},
		{25.0, Junior, "low rate fallback to junior"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := costModel.InferEngineerLevel(tt.rate)
			if result != tt.expected {
				t.Errorf("Expected level %v for rate %.1f, got %v", tt.expected, tt.rate, result)
			}
		})
	}
}
