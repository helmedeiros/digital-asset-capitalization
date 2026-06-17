package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain/ports"
)

const (
	testEngineerJohnDoe   = "John Doe"
	testEngineerJaneSmith = "Jane Smith"
)

// Mock implementations for testing
type mockCostModelRepository struct {
	costModels map[string]*domain.CostModel
}

func (m *mockCostModelRepository) GetCostModel(_ context.Context, project string) (*domain.CostModel, error) {
	if model, exists := m.costModels[project]; exists {
		return model, nil
	}
	return nil, domain.ErrCostModelNotFound
}

func (m *mockCostModelRepository) SaveCostModel(_ context.Context, project string, model *domain.CostModel) error {
	if m.costModels == nil {
		m.costModels = make(map[string]*domain.CostModel)
	}
	m.costModels[project] = model
	return nil
}

func (m *mockCostModelRepository) GetDefaultCostModel(_ context.Context) (*domain.CostModel, error) {
	costModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	return costModel, nil
}

type mockTimeAllocationProvider struct {
	allocations []ports.EngineerAllocation
}

func (m *mockTimeAllocationProvider) GetSprintAllocations(_ context.Context, _, _ string) ([]ports.EngineerAllocation, error) {
	return m.allocations, nil
}

func (m *mockTimeAllocationProvider) GetAssetAllocations(_ context.Context, _ string) ([]ports.EngineerAllocation, error) {
	return m.allocations, nil
}

func (m *mockTimeAllocationProvider) GetTaskAllocations(_ context.Context, _ []string) ([]ports.EngineerAllocation, error) {
	return m.allocations, nil
}

type mockInvestmentRepository struct {
	investments map[string]*domain.Investment
}

func (m *mockInvestmentRepository) SaveInvestment(_ context.Context, investment *domain.Investment) error {
	if m.investments == nil {
		m.investments = make(map[string]*domain.Investment)
	}
	m.investments[investment.AssetName] = investment
	return nil
}

func (m *mockInvestmentRepository) GetInvestment(_ context.Context, assetName string) (*domain.Investment, error) {
	if investment, exists := m.investments[assetName]; exists {
		return investment, nil
	}
	return nil, domain.ErrInvestmentNotFound
}

func (m *mockInvestmentRepository) ListInvestments(_ context.Context, project string) ([]*domain.Investment, error) {
	var result []*domain.Investment
	for _, investment := range m.investments {
		if investment.Project == project {
			result = append(result, investment)
		}
	}
	return result, nil
}

func (m *mockInvestmentRepository) DeleteInvestment(_ context.Context, assetName string) error {
	if m.investments == nil {
		return domain.ErrInvestmentNotFound
	}
	delete(m.investments, assetName)
	return nil
}

func TestCalculateAssetInvestmentUseCase_Execute(t *testing.T) {
	t.Parallel()
	// Set up mocks
	costModelRepo := &mockCostModelRepository{
		costModels: make(map[string]*domain.CostModel),
	}

	// Create and save a test cost model
	costModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	costModel.AddEngineerRate(domain.EngineerRate{
		Name:       testEngineerJohnDoe,
		HourlyRate: 75.0,
		Level:      domain.Senior,
		Team:       "TEST",
	})
	costModelRepo.costModels["TEST"] = costModel

	// Set up allocation provider with test data
	allocationProvider := &mockTimeAllocationProvider{
		allocations: []ports.EngineerAllocation{
			{
				EngineerName: testEngineerJohnDoe,
				TaskKey:      "TEST-123",
				TaskTitle:    "Test Task",
				WorkType:     "cap-development",
				AssetName:    "Test Asset",
				Sprint:       "Sprint1",
				Allocation:   100,
				StartDate:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:      time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	investmentRepo := &mockInvestmentRepository{
		investments: make(map[string]*domain.Investment),
	}

	// Create the use case
	useCase := NewCalculateAssetInvestmentUseCase(
		costModelRepo,
		allocationProvider,
		investmentRepo,
	)

	// Execute the use case
	ctx := context.Background()
	input := ports.AssetInvestmentInput{
		AssetName: "Test Asset",
		Project:   "TEST",
		Sprints:   []string{"Sprint1"},
		StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC),
	}
	investment, err := useCase.Execute(ctx, input)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if investment.AssetName != "Test Asset" {
		t.Errorf("Expected asset name 'Test Asset', got %s", investment.AssetName)
	}

	if investment.Project != "TEST" {
		t.Errorf("Expected project 'TEST', got %s", investment.Project)
	}

	if len(investment.EngineersInvolved) != 1 {
		t.Errorf("Expected 1 engineer involved, got %d", len(investment.EngineersInvolved))
	}

	if investment.EngineersInvolved[0].Name != testEngineerJohnDoe {
		t.Errorf("Expected engineer 'John Doe', got %s", investment.EngineersInvolved[0].Name)
	}

	// Should have calculated costs
	if investment.TotalCost.Amount == 0 {
		t.Error("Expected non-zero total cost")
	}

	// Should have saved the investment
	savedInvestment, err := investmentRepo.GetInvestment(ctx, "Test Asset")
	if err != nil {
		t.Fatalf("Expected investment to be saved, got error: %v", err)
	}

	if savedInvestment.AssetName != "Test Asset" {
		t.Errorf("Expected saved investment asset name 'Test Asset', got %s", savedInvestment.AssetName)
	}
}

func TestCalculateAssetInvestmentUseCase_Execute_CostModelNotFound(t *testing.T) {
	t.Parallel()
	// Set up mocks with empty cost model repo - it should fallback to default
	costModelRepo := &mockCostModelRepository{
		costModels: make(map[string]*domain.CostModel),
	}

	allocationProvider := &mockTimeAllocationProvider{}
	investmentRepo := &mockInvestmentRepository{}

	useCase := NewCalculateAssetInvestmentUseCase(
		costModelRepo,
		allocationProvider,
		investmentRepo,
	)

	ctx := context.Background()
	input := ports.AssetInvestmentInput{
		AssetName: "Test Asset",
		Project:   "NONEXISTENT",
		Sprints:   []string{"Sprint1"},
		StartDate: time.Now(),
		EndDate:   time.Now(),
	}
	_, err := useCase.Execute(ctx, input)

	// Should succeed with fallback to default cost model
	if err != nil {
		t.Fatalf("Expected success with default cost model fallback, got error: %v", err)
	}
}

func TestCalculateAssetInvestmentUseCase_Execute_EmptySprints(t *testing.T) {
	t.Parallel()
	costModelRepo := &mockCostModelRepository{
		costModels: make(map[string]*domain.CostModel),
	}

	costModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	costModelRepo.costModels["TEST"] = costModel

	allocationProvider := &mockTimeAllocationProvider{
		allocations: []ports.EngineerAllocation{}, // Empty allocations
	}

	investmentRepo := &mockInvestmentRepository{
		investments: make(map[string]*domain.Investment),
	}

	useCase := NewCalculateAssetInvestmentUseCase(
		costModelRepo,
		allocationProvider,
		investmentRepo,
	)

	ctx := context.Background()
	input := ports.AssetInvestmentInput{
		AssetName: "Empty Asset",
		Project:   "TEST",
		Sprints:   []string{"Sprint1"},
		StartDate: time.Now(),
		EndDate:   time.Now(),
	}
	investment, err := useCase.Execute(ctx, input)

	if err != nil {
		t.Fatalf("Expected no error for empty allocations, got %v", err)
	}

	// Should still create investment with zero costs
	if len(investment.EngineersInvolved) != 0 {
		t.Errorf("Expected 0 engineers for empty allocations, got %d", len(investment.EngineersInvolved))
	}

	if investment.TotalCost.Amount != 0 {
		t.Errorf("Expected zero total cost for empty allocations, got %.2f", investment.TotalCost.Amount)
	}
}

// Additional comprehensive test cases

func TestCalculateAssetInvestmentUseCase_Execute_WithInfrastructureCosts(t *testing.T) {
	t.Parallel()
	costModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	costModel.InfrastructureCosts.CloudCostsPerMonth = 1000.0
	costModel.InfrastructureCosts.ToolingCostsPerMonth = 500.0
	costModel.SetDefaultRate(domain.Senior, 75.0)

	costModelRepo := &mockCostModelRepository{
		costModels: map[string]*domain.CostModel{
			"TEST": costModel,
		},
	}

	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)

	allocationProvider := &mockTimeAllocationProvider{
		allocations: []ports.EngineerAllocation{
			{
				EngineerName: testEngineerJohnDoe,
				TaskKey:      "TASK-1",
				TaskTitle:    "Test Task",
				WorkType:     "development",
				AssetName:    "Test Asset",
				Sprint:       "Sprint1",
				Allocation:   50.0,
				StartDate:    startDate,
				EndDate:      endDate,
			},
		},
	}

	investmentRepo := &mockInvestmentRepository{
		investments: make(map[string]*domain.Investment),
	}

	useCase := NewCalculateAssetInvestmentUseCase(
		costModelRepo,
		allocationProvider,
		investmentRepo,
	)

	ctx := context.Background()
	input := ports.AssetInvestmentInput{
		AssetName: "Test Asset",
		Project:   "TEST",
		Sprints:   []string{"Sprint1"},
		StartDate: startDate,
		EndDate:   endDate,
	}
	investment, err := useCase.Execute(ctx, input)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should have infrastructure costs
	if investment.InfrastructureCosts.Amount <= 0 {
		t.Error("Expected positive infrastructure costs")
	}

	// Total cost should include infrastructure costs
	if investment.TotalCost.Amount <= investment.EngineerCosts.Amount+investment.OverheadCosts.Amount {
		t.Error("Total cost should include infrastructure costs")
	}
}

func TestCalculateAssetInvestmentUseCase_Execute_WithDateRangeFiltering(t *testing.T) {
	t.Parallel()
	costModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	costModel.SetDefaultRate(domain.Senior, 75.0)

	costModelRepo := &mockCostModelRepository{
		costModels: map[string]*domain.CostModel{
			"TEST": costModel,
		},
	}

	// Create allocations with different date ranges
	filterStartDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	filterEndDate := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)

	allocationProvider := &mockTimeAllocationProvider{
		allocations: []ports.EngineerAllocation{
			{
				EngineerName: testEngineerJohnDoe,
				TaskKey:      "TASK-1",
				TaskTitle:    "Task in range",
				WorkType:     "development",
				AssetName:    "Test Asset",
				Sprint:       "Sprint1",
				Allocation:   50.0,
				StartDate:    time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC), // Inside range
				EndDate:      time.Date(2025, 1, 25, 0, 0, 0, 0, time.UTC),
			},
			{
				EngineerName: "Jane Doe",
				TaskKey:      "TASK-2",
				TaskTitle:    "Task outside range",
				WorkType:     "development",
				AssetName:    "Test Asset",
				Sprint:       "Sprint1",
				Allocation:   50.0,
				StartDate:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), // Outside range
				EndDate:      time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	investmentRepo := &mockInvestmentRepository{
		investments: make(map[string]*domain.Investment),
	}

	useCase := NewCalculateAssetInvestmentUseCase(
		costModelRepo,
		allocationProvider,
		investmentRepo,
	)

	ctx := context.Background()
	input := ports.AssetInvestmentInput{
		AssetName: "Test Asset",
		Project:   "TEST",
		Sprints:   []string{"Sprint1"},
		StartDate: filterStartDate,
		EndDate:   filterEndDate,
	}
	investment, err := useCase.Execute(ctx, input)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should only include tasks that overlap with date range
	if len(investment.TaskBreakdown) != 1 {
		t.Errorf("Expected 1 task after date filtering, got %d", len(investment.TaskBreakdown))
	}

	// Should only have one engineer involved
	if len(investment.EngineersInvolved) != 1 {
		t.Errorf("Expected 1 engineer after date filtering, got %d", len(investment.EngineersInvolved))
	}

	if investment.EngineersInvolved[0].Name != testEngineerJohnDoe {
		t.Errorf("Expected engineer 'John Doe', got %s", investment.EngineersInvolved[0].Name)
	}
}

func TestCalculateAssetInvestmentUseCase_Execute_WithProvidedCostModel(t *testing.T) {
	t.Parallel()
	// Provide cost model in input instead of using repository
	costModel, _ := domain.NewCostModel(domain.USD, 7.5, 1.8)
	costModel.SetDefaultRate(domain.Staff, 90.0)

	costModelRepo := &mockCostModelRepository{
		costModels: make(map[string]*domain.CostModel), // Empty repo
	}

	allocationProvider := &mockTimeAllocationProvider{
		allocations: []ports.EngineerAllocation{
			{
				EngineerName: testEngineerJohnDoe,
				TaskKey:      "TASK-1",
				TaskTitle:    "Test Task",
				WorkType:     "development",
				AssetName:    "Test Asset",
				Sprint:       "Sprint1",
				Allocation:   100.0,
				StartDate:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:      time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	investmentRepo := &mockInvestmentRepository{
		investments: make(map[string]*domain.Investment),
	}

	useCase := NewCalculateAssetInvestmentUseCase(
		costModelRepo,
		allocationProvider,
		investmentRepo,
	)

	ctx := context.Background()
	input := ports.AssetInvestmentInput{
		AssetName: "Test Asset",
		Project:   "TEST",
		Sprints:   []string{"Sprint1"},
		CostModel: costModel, // Provide cost model directly
	}
	investment, err := useCase.Execute(ctx, input)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should use provided cost model currency
	if investment.TotalCost.Currency != domain.USD {
		t.Errorf("Expected currency USD, got %v", investment.TotalCost.Currency)
	}
}

func TestCalculateAssetInvestmentUseCase_Execute_InvalidCostModel(t *testing.T) {
	t.Parallel()
	// Create invalid cost model
	invalidCostModel := &domain.CostModel{
		Currency:           domain.EUR,
		WorkingHoursPerDay: -1, // Invalid
		OverheadMultiplier: 2.0,
	}

	costModelRepo := &mockCostModelRepository{
		costModels: make(map[string]*domain.CostModel),
	}

	allocationProvider := &mockTimeAllocationProvider{
		allocations: []ports.EngineerAllocation{},
	}

	investmentRepo := &mockInvestmentRepository{
		investments: make(map[string]*domain.Investment),
	}

	useCase := NewCalculateAssetInvestmentUseCase(
		costModelRepo,
		allocationProvider,
		investmentRepo,
	)

	ctx := context.Background()
	input := ports.AssetInvestmentInput{
		AssetName: "Test Asset",
		Project:   "TEST",
		Sprints:   []string{"Sprint1"},
		CostModel: invalidCostModel,
	}
	_, err := useCase.Execute(ctx, input)

	if err == nil {
		t.Error("Expected error for invalid cost model")
	}
}

func TestCalculateAssetInvestmentUseCase_Execute_MultipleEngineersAndTasks(t *testing.T) {
	t.Parallel()
	costModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	costModel.SetDefaultRate(domain.Senior, 75.0)
	costModel.SetDefaultRate(domain.Mid, 60.0)

	// Add specific engineer rates
	costModel.AddEngineerRate(domain.EngineerRate{
		Name:       testEngineerJohnDoe,
		HourlyRate: 80.0,
		Level:      domain.Senior,
		Team:       "TEST",
	})

	costModelRepo := &mockCostModelRepository{
		costModels: map[string]*domain.CostModel{
			"TEST": costModel,
		},
	}

	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	allocationProvider := &mockTimeAllocationProvider{
		allocations: []ports.EngineerAllocation{
			{
				EngineerName: testEngineerJohnDoe, // Has specific rate
				TaskKey:      "TASK-1",
				TaskTitle:    "Backend Development",
				WorkType:     "development",
				AssetName:    "Test Asset",
				Sprint:       "Sprint1",
				Allocation:   50.0,
				StartDate:    startDate,
				EndDate:      endDate,
			},
			{
				EngineerName: testEngineerJaneSmith, // Uses default rate
				TaskKey:      "TASK-1",
				TaskTitle:    "Backend Development",
				WorkType:     "development",
				AssetName:    "Test Asset",
				Sprint:       "Sprint1",
				Allocation:   30.0,
				StartDate:    startDate,
				EndDate:      endDate,
			},
			{
				EngineerName: testEngineerJaneSmith,
				TaskKey:      "TASK-2",
				TaskTitle:    "Testing",
				WorkType:     "maintenance",
				AssetName:    "Test Asset",
				Sprint:       "Sprint2",
				Allocation:   20.0,
				StartDate:    startDate,
				EndDate:      endDate,
			},
		},
	}

	investmentRepo := &mockInvestmentRepository{
		investments: make(map[string]*domain.Investment),
	}

	useCase := NewCalculateAssetInvestmentUseCase(
		costModelRepo,
		allocationProvider,
		investmentRepo,
	)

	ctx := context.Background()
	input := ports.AssetInvestmentInput{
		AssetName: "Test Asset",
		Project:   "TEST",
		Sprints:   []string{"Sprint1", "Sprint2"},
		StartDate: startDate,
		EndDate:   endDate,
	}
	investment, err := useCase.Execute(ctx, input)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should have 2 engineers involved
	if len(investment.EngineersInvolved) != 2 {
		t.Errorf("Expected 2 engineers, got %d", len(investment.EngineersInvolved))
	}

	// Should have 3 tasks (2 distinct tasks, but one task has 2 engineers so creates separate task entries)
	if len(investment.TaskBreakdown) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(investment.TaskBreakdown))
	}

	// Should have work type breakdown
	if len(investment.WorkTypeBreakdown) != 2 {
		t.Errorf("Expected 2 work types, got %d", len(investment.WorkTypeBreakdown))
	}

	// Check that John Doe's rate is higher than Jane Smith's (since John has specific rate)
	var johnInvestment, janeInvestment *domain.EngineerInvestment
	for i, engineer := range investment.EngineersInvolved {
		if engineer.Name == testEngineerJohnDoe {
			johnInvestment = &investment.EngineersInvolved[i]
		} else if engineer.Name == testEngineerJaneSmith {
			janeInvestment = &investment.EngineersInvolved[i]
		}
	}

	if johnInvestment == nil || janeInvestment == nil {
		t.Fatal("Expected to find both John Doe and Jane Smith in engineers")
		return
	}

	if johnInvestment.HourlyRate != 80.0 {
		t.Errorf("Expected John's rate 80.0, got %.1f", johnInvestment.HourlyRate)
	}
	if janeInvestment.HourlyRate != 60.0 { // Default mid rate
		t.Errorf("Expected Jane's rate 60.0 (default mid), got %.1f", janeInvestment.HourlyRate)
	}

	// Jane should have worked on both sprints
	if len(janeInvestment.Sprints) != 2 {
		t.Errorf("Expected Jane to work on 2 sprints, got %d", len(janeInvestment.Sprints))
	}
}

func TestCalculateAssetInvestmentUseCase_Execute_AllocationProviderError(t *testing.T) {
	t.Parallel()
	costModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	costModelRepo := &mockCostModelRepository{
		costModels: map[string]*domain.CostModel{
			"TEST": costModel,
		},
	}

	// Create error-returning allocation provider
	allocationProvider := &mockTimeAllocationProviderWithError{}

	investmentRepo := &mockInvestmentRepository{
		investments: make(map[string]*domain.Investment),
	}

	useCase := NewCalculateAssetInvestmentUseCase(
		costModelRepo,
		allocationProvider,
		investmentRepo,
	)

	ctx := context.Background()
	input := ports.AssetInvestmentInput{
		AssetName: "Test Asset",
		Project:   "TEST",
		Sprints:   []string{"Sprint1"},
	}
	_, err := useCase.Execute(ctx, input)

	if err == nil {
		t.Error("Expected error from allocation provider")
	}
}

// Mock that returns errors
type mockTimeAllocationProviderWithError struct{}

func (m *mockTimeAllocationProviderWithError) GetSprintAllocations(_ context.Context, _, _ string) ([]ports.EngineerAllocation, error) {
	return nil, domain.ErrEngineerNotFound
}

func (m *mockTimeAllocationProviderWithError) GetAssetAllocations(_ context.Context, _ string) ([]ports.EngineerAllocation, error) {
	return nil, domain.ErrEngineerNotFound
}

func (m *mockTimeAllocationProviderWithError) GetTaskAllocations(_ context.Context, _ []string) ([]ports.EngineerAllocation, error) {
	return nil, domain.ErrEngineerNotFound
}

// Additional tests to improve coverage to 90%+

func TestCalculateAssetInvestmentUseCase_InferEngineerLevel_AllCases(t *testing.T) {
	t.Parallel()
	costModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)

	// Set up all default rates
	costModel.SetDefaultRate(domain.Junior, 45.0)
	costModel.SetDefaultRate(domain.Mid, 60.0)
	costModel.SetDefaultRate(domain.Senior, 75.0)
	costModel.SetDefaultRate(domain.Staff, 85.0)
	costModel.SetDefaultRate(domain.Principal, 95.0)

	costModelRepo := &mockCostModelRepository{
		costModels: map[string]*domain.CostModel{"TEST": costModel},
	}

	_ = NewCalculateAssetInvestmentUseCase(costModelRepo, &mockTimeAllocationProvider{}, &mockInvestmentRepository{})

	// Test all inference levels based on rate ranges
	testCases := map[float64]domain.EngineerLevel{
		// Exact matches (within 10% tolerance)
		45.0: domain.Junior,    // Junior: 45.0 ± 10% = 40.5-49.5
		60.0: domain.Mid,       // Mid: 60.0 ± 10% = 54.0-66.0
		75.0: domain.Senior,    // Senior: 75.0 ± 10% = 67.5-82.5
		85.0: domain.Staff,     // Staff: 85.0 ± 10% = 76.5-93.5
		95.0: domain.Principal, // Principal: 95.0 ± 10% = 85.5-104.5
		// Range-based fallbacks (rates that don't match tolerance ranges)
		100.0: domain.Principal, // >= 80 (outside all tolerances)
		83.0:  domain.Staff,     // Within Staff tolerance (76.5-93.5)
		70.0:  domain.Senior,    // Within Senior tolerance (67.5-82.5)
		67.0:  domain.Senior,    // >= 60 < 70 (outside Mid tolerance but in range)
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

func TestCalculateAssetInvestmentUseCase_ContainsString_AllCases(t *testing.T) {
	t.Parallel()
	useCase := NewCalculateAssetInvestmentUseCase(&mockCostModelRepository{}, &mockTimeAllocationProvider{}, &mockInvestmentRepository{})

	// Test string found
	slice := []string{"Sprint1", "Sprint2", "Sprint3"}
	if !useCase.containsString(slice, "Sprint2") {
		t.Error("Expected to find 'Sprint2' in slice")
	}

	// Test string not found
	if useCase.containsString(slice, "Sprint4") {
		t.Error("Expected not to find 'Sprint4' in slice")
	}

	// Test empty slice
	emptySlice := []string{}
	if useCase.containsString(emptySlice, "Sprint1") {
		t.Error("Expected not to find 'Sprint1' in empty slice")
	}

	// Test empty string search
	if useCase.containsString(slice, "") {
		t.Error("Expected not to find empty string in slice")
	}
}

func TestCalculateAssetInvestmentUseCase_Execute_InfrastructureCostCalculationError(t *testing.T) {
	t.Parallel()
	// Create cost model that will cause infrastructure cost calculation error
	costModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	costModel.SetDefaultRate(domain.Senior, 75.0)

	costModelRepo := &mockCostModelRepository{
		costModels: map[string]*domain.CostModel{"TEST": costModel},
	}

	allocationProvider := &mockTimeAllocationProvider{
		allocations: []ports.EngineerAllocation{
			{
				EngineerName: testEngineerJohnDoe,
				TaskKey:      "TASK-1",
				TaskTitle:    "Test Task",
				WorkType:     "development",
				AssetName:    "Test Asset",
				Sprint:       "Sprint1",
				Allocation:   50.0,
				StartDate:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:      time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	investmentRepo := &mockInvestmentRepository{
		investments: make(map[string]*domain.Investment),
	}

	useCase := NewCalculateAssetInvestmentUseCase(costModelRepo, allocationProvider, investmentRepo)

	ctx := context.Background()

	// Use invalid date range (end before start) to trigger infrastructure cost calculation error
	input := ports.AssetInvestmentInput{
		AssetName: "Test Asset",
		Project:   "TEST",
		Sprints:   []string{"Sprint1"},
		StartDate: time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC), // After end date
		EndDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),  // Before start date
	}

	_, err := useCase.Execute(ctx, input)
	if err == nil {
		t.Error("Expected error for invalid date range in infrastructure cost calculation")
	}
}

func TestCalculateAssetInvestmentUseCase_Execute_SaveInvestmentError(t *testing.T) {
	t.Parallel()
	costModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	costModel.SetDefaultRate(domain.Senior, 75.0)

	costModelRepo := &mockCostModelRepository{
		costModels: map[string]*domain.CostModel{"TEST": costModel},
	}

	allocationProvider := &mockTimeAllocationProvider{
		allocations: []ports.EngineerAllocation{
			{
				EngineerName: testEngineerJohnDoe,
				TaskKey:      "TASK-1",
				TaskTitle:    "Test Task",
				WorkType:     "development",
				AssetName:    "Test Asset",
				Sprint:       "Sprint1",
				Allocation:   50.0,
				StartDate:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:      time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	// Create mock that fails on save
	investmentRepo := &mockInvestmentRepositoryWithSaveError{}

	useCase := NewCalculateAssetInvestmentUseCase(costModelRepo, allocationProvider, investmentRepo)

	ctx := context.Background()
	input := ports.AssetInvestmentInput{
		AssetName: "Test Asset",
		Project:   "TEST",
		Sprints:   []string{"Sprint1"},
	}

	_, err := useCase.Execute(ctx, input)
	if err == nil {
		t.Error("Expected error from save investment failure")
	}
}

// Mock that fails on save investment
type mockInvestmentRepositoryWithSaveError struct{}

func (m *mockInvestmentRepositoryWithSaveError) SaveInvestment(_ context.Context, _ *domain.Investment) error {
	return domain.ErrInvestmentNotFound // Return any error
}

func (m *mockInvestmentRepositoryWithSaveError) GetInvestment(_ context.Context, _ string) (*domain.Investment, error) {
	return nil, domain.ErrInvestmentNotFound
}

func (m *mockInvestmentRepositoryWithSaveError) ListInvestments(_ context.Context, _ string) ([]*domain.Investment, error) {
	return nil, nil
}

func (m *mockInvestmentRepositoryWithSaveError) DeleteInvestment(_ context.Context, _ string) error {
	return nil
}
