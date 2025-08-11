package domain

import (
	"testing"
	"time"
)

func TestNewInvestment(t *testing.T) {
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)

	investment := NewInvestment("Test Asset", "PROJECT", []string{"Sprint1"}, startDate, endDate, EUR)

	if investment.AssetName != "Test Asset" {
		t.Errorf("Expected asset name 'Test Asset', got %s", investment.AssetName)
	}

	if investment.Project != "PROJECT" {
		t.Errorf("Expected project 'PROJECT', got %s", investment.Project)
	}

	if len(investment.Sprints) != 1 || investment.Sprints[0] != "Sprint1" {
		t.Errorf("Expected sprints ['Sprint1'], got %v", investment.Sprints)
	}

	if investment.StartDate != startDate {
		t.Errorf("Expected start date %v, got %v", startDate, investment.StartDate)
	}

	if investment.EndDate != endDate {
		t.Errorf("Expected end date %v, got %v", endDate, investment.EndDate)
	}

	if investment.TotalCost.Currency != EUR {
		t.Errorf("Expected currency EUR, got %v", investment.TotalCost.Currency)
	}
}

func TestInvestment_AddEngineerInvestment(t *testing.T) {
	investment := NewInvestment("Test", "PROJECT", []string{"Sprint1"}, time.Now(), time.Now(), EUR)

	engineerInvestment := EngineerInvestment{
		Name:       "John Doe",
		Level:      Senior,
		TotalHours: 80,
		HourlyRate: 75.0,
		TotalCost:  NewMoney(6000, EUR),
		Sprints:    []string{"Sprint1"},
	}

	investment.AddEngineerInvestment(engineerInvestment)

	if len(investment.EngineersInvolved) != 1 {
		t.Errorf("Expected 1 engineer, got %d", len(investment.EngineersInvolved))
	}

	if investment.EngineersInvolved[0].Name != "John Doe" {
		t.Errorf("Expected engineer name 'John Doe', got %s", investment.EngineersInvolved[0].Name)
	}
}

func TestInvestment_GetDurationInDays(t *testing.T) {
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC) // 14 days

	investment := NewInvestment("Test", "PROJECT", []string{"Sprint1"}, startDate, endDate, EUR)

	duration := investment.GetDurationInDays()
	if duration != 14 {
		t.Errorf("Expected 14 days, got %d", duration)
	}
}

func TestInvestment_GetEngineerCount(t *testing.T) {
	investment := NewInvestment("Test", "PROJECT", []string{"Sprint1"}, time.Now(), time.Now(), EUR)

	// Initially should be 0
	if investment.GetEngineerCount() != 0 {
		t.Errorf("Expected 0 engineers, got %d", investment.GetEngineerCount())
	}

	// Add an engineer
	investment.AddEngineerInvestment(EngineerInvestment{Name: "John"})

	if investment.GetEngineerCount() != 1 {
		t.Errorf("Expected 1 engineer, got %d", investment.GetEngineerCount())
	}
}

func TestInvestment_CalculateTotalCost(t *testing.T) {
	investment := NewInvestment("Test", "PROJECT", []string{"Sprint1"}, time.Now(), time.Now(), EUR)

	investment.EngineerCosts = NewMoney(5000, EUR)
	investment.OverheadCosts = NewMoney(3000, EUR)
	investment.InfrastructureCosts = NewMoney(2000, EUR)

	investment.CalculateTotalCost()

	expected := 10000.0
	if investment.TotalCost.Amount != expected {
		t.Errorf("Expected total cost %.2f, got %.2f", expected, investment.TotalCost.Amount)
	}
}

// Additional test cases for complete coverage

func TestInvestment_AddTaskInvestment(t *testing.T) {
	investment := NewInvestment("Test", "PROJECT", []string{"Sprint1"}, time.Now(), time.Now(), EUR)

	task := TaskInvestment{
		TaskKey:   "TASK-1",
		TaskTitle: "Test Task",
		WorkType:  "development",
		Sprint:    "Sprint1",
		Engineers: make(map[string]EngineerTaskEffort),
		TotalCost: NewMoney(1000, EUR),
	}

	investment.AddTaskInvestment(task)

	if len(investment.TaskBreakdown) != 1 {
		t.Errorf("Expected 1 task, got %d", len(investment.TaskBreakdown))
	}

	if investment.TaskBreakdown[0].TaskKey != "TASK-1" {
		t.Errorf("Expected task key 'TASK-1', got %s", investment.TaskBreakdown[0].TaskKey)
	}

	// Check work type breakdown
	if breakdown, exists := investment.WorkTypeBreakdown["development"]; !exists {
		t.Error("Expected 'development' in work type breakdown")
	} else if breakdown.Amount != 1000.0 {
		t.Errorf("Expected breakdown amount 1000.0, got %.2f", breakdown.Amount)
	}

	// Add another task of same work type
	task2 := TaskInvestment{
		TaskKey:   "TASK-2",
		TaskTitle: "Another Test Task",
		WorkType:  "development",
		TotalCost: NewMoney(500, EUR),
	}

	investment.AddTaskInvestment(task2)

	// Should accumulate in work type breakdown
	if breakdown := investment.WorkTypeBreakdown["development"]; breakdown.Amount != 1500.0 {
		t.Errorf("Expected accumulated breakdown amount 1500.0, got %.2f", breakdown.Amount)
	}
}

func TestInvestment_SetInfrastructureCosts(t *testing.T) {
	investment := NewInvestment("Test", "PROJECT", []string{"Sprint1"}, time.Now(), time.Now(), EUR)

	infraCosts := NewMoney(2500, EUR)
	investment.SetInfrastructureCosts(infraCosts)

	if investment.InfrastructureCosts.Amount != 2500.0 {
		t.Errorf("Expected infrastructure costs 2500.0, got %.2f", investment.InfrastructureCosts.Amount)
	}
}

func TestInvestment_GetTaskCount(t *testing.T) {
	investment := NewInvestment("Test", "PROJECT", []string{"Sprint1"}, time.Now(), time.Now(), EUR)

	// Initially should be 0
	if investment.GetTaskCount() != 0 {
		t.Errorf("Expected 0 tasks, got %d", investment.GetTaskCount())
	}

	// Add a task
	task := TaskInvestment{
		TaskKey:   "TASK-1",
		TaskTitle: "Test Task",
		TotalCost: NewMoney(1000, EUR),
	}
	investment.AddTaskInvestment(task)

	if investment.GetTaskCount() != 1 {
		t.Errorf("Expected 1 task, got %d", investment.GetTaskCount())
	}
}

func TestMoney_Add_DifferentCurrencies(t *testing.T) {
	money1 := NewMoney(100.0, EUR)
	money2 := NewMoney(50.0, USD)

	// Should return the first money when currencies differ (current implementation)
	result := money1.Add(money2)

	if result.Amount != 100.0 {
		t.Errorf("Expected amount 100.0, got %.2f", result.Amount)
	}
	if result.Currency != EUR {
		t.Errorf("Expected currency EUR, got %v", result.Currency)
	}
}

func TestMoney_IsZero(t *testing.T) {
	// Test zero money
	zeroMoney := NewMoney(0, EUR)
	if !zeroMoney.IsZero() {
		t.Error("Expected zero money to be zero")
	}

	// Test non-zero money
	nonZeroMoney := NewMoney(100.0, EUR)
	if nonZeroMoney.IsZero() {
		t.Error("Expected non-zero money to not be zero")
	}
}

func TestEngineerTaskEffort_Structure(t *testing.T) {
	effort := EngineerTaskEffort{
		Allocation:   25.5,
		Hours:        40.0,
		DirectCost:   NewMoney(3000, EUR),
		OverheadCost: NewMoney(1500, EUR),
		TotalCost:    NewMoney(4500, EUR),
	}

	if effort.Allocation != 25.5 {
		t.Errorf("Expected allocation 25.5, got %.1f", effort.Allocation)
	}
	if effort.Hours != 40.0 {
		t.Errorf("Expected hours 40.0, got %.1f", effort.Hours)
	}
	if effort.TotalCost.Amount != 4500.0 {
		t.Errorf("Expected total cost 4500.0, got %.2f", effort.TotalCost.Amount)
	}
}

func TestTaskInvestment_Structure(t *testing.T) {
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	task := TaskInvestment{
		TaskKey:   "TASK-123",
		TaskTitle: "Implement feature X",
		WorkType:  "development",
		Sprint:    "Sprint 1",
		Engineers: map[string]EngineerTaskEffort{
			"John Doe": {
				Allocation: 50.0,
				Hours:      80.0,
				TotalCost:  NewMoney(6000, EUR),
			},
		},
		TotalCost: NewMoney(6000, EUR),
		StartDate: startDate,
		EndDate:   endDate,
	}

	if task.TaskKey != "TASK-123" {
		t.Errorf("Expected task key 'TASK-123', got %s", task.TaskKey)
	}
	if task.WorkType != "development" {
		t.Errorf("Expected work type 'development', got %s", task.WorkType)
	}
	if len(task.Engineers) != 1 {
		t.Errorf("Expected 1 engineer, got %d", len(task.Engineers))
	}
	if effort, exists := task.Engineers["John Doe"]; !exists {
		t.Error("Expected 'John Doe' in engineers map")
	} else if effort.Allocation != 50.0 {
		t.Errorf("Expected allocation 50.0, got %.1f", effort.Allocation)
	}
}

func TestEngineerInvestment_Structure(t *testing.T) {
	engineer := EngineerInvestment{
		Name:         "Jane Smith",
		Level:        Staff,
		TotalHours:   160.0,
		HourlyRate:   85.0,
		DirectCost:   NewMoney(13600, EUR),
		OverheadCost: NewMoney(6800, EUR),
		TotalCost:    NewMoney(20400, EUR),
		Sprints:      []string{"Sprint1", "Sprint2"},
	}

	if engineer.Name != "Jane Smith" {
		t.Errorf("Expected name 'Jane Smith', got %s", engineer.Name)
	}
	if engineer.Level != Staff {
		t.Errorf("Expected level Staff, got %v", engineer.Level)
	}
	if engineer.TotalHours != 160.0 {
		t.Errorf("Expected total hours 160.0, got %.1f", engineer.TotalHours)
	}
	if len(engineer.Sprints) != 2 {
		t.Errorf("Expected 2 sprints, got %d", len(engineer.Sprints))
	}
}

func TestNewInvestment_CalculatedAtSet(t *testing.T) {
	before := time.Now()
	investment := NewInvestment("Test", "PROJECT", []string{"Sprint1"}, time.Now(), time.Now(), EUR)
	after := time.Now()

	if investment.CalculatedAt.Before(before) || investment.CalculatedAt.After(after) {
		t.Error("CalculatedAt should be set to current time")
	}
}
