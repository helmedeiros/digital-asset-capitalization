package service

import (
	"context"
	"testing"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain/ports"
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
	if _, exists := m.investments[assetName]; !exists {
		return domain.ErrInvestmentNotFound
	}
	delete(m.investments, assetName)
	return nil
}

func TestInvestmentService_InitializeCostModel(t *testing.T) {
	t.Parallel()
	costModelRepo := &mockCostModelRepository{}
	investmentRepo := &mockInvestmentRepository{}

	service := NewInvestmentService(costModelRepo, nil, investmentRepo)

	ctx := context.Background()
	costModel, err := service.InitializeCostModel(ctx, "TEST")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if costModel.Currency != domain.EUR {
		t.Errorf("Expected currency EUR, got %v", costModel.Currency)
	}

	// Verify it was saved
	savedModel, err := costModelRepo.GetCostModel(ctx, "TEST")
	if err != nil {
		t.Fatalf("Expected saved model to be retrievable, got error: %v", err)
	}

	if savedModel.Currency != domain.EUR {
		t.Errorf("Expected saved model currency EUR, got %v", savedModel.Currency)
	}
}

func TestInvestmentService_GetCostModel(t *testing.T) {
	t.Parallel()
	costModelRepo := &mockCostModelRepository{
		costModels: make(map[string]*domain.CostModel),
	}

	// Pre-populate with a cost model
	testModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	costModelRepo.costModels["TEST"] = testModel

	service := NewInvestmentService(costModelRepo, nil, nil)

	ctx := context.Background()
	costModel, err := service.GetCostModel(ctx, "TEST")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if costModel.Currency != domain.EUR {
		t.Errorf("Expected currency EUR, got %v", costModel.Currency)
	}
}

func TestInvestmentService_UpdateCostModel(t *testing.T) {
	t.Parallel()
	costModelRepo := &mockCostModelRepository{
		costModels: make(map[string]*domain.CostModel),
	}

	service := NewInvestmentService(costModelRepo, nil, nil)

	ctx := context.Background()
	testModel, _ := domain.NewCostModel(domain.USD, 8.0, 1.8)

	err := service.UpdateCostModel(ctx, "TEST", testModel)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify it was saved
	savedModel, err := costModelRepo.GetCostModel(ctx, "TEST")
	if err != nil {
		t.Fatalf("Expected saved model to be retrievable, got error: %v", err)
	}

	if savedModel.Currency != domain.USD {
		t.Errorf("Expected saved model currency USD, got %v", savedModel.Currency)
	}

	if savedModel.OverheadMultiplier != 1.8 {
		t.Errorf("Expected overhead multiplier 1.8, got %.1f", savedModel.OverheadMultiplier)
	}
}

func TestInvestmentService_ListInvestments(t *testing.T) {
	t.Parallel()
	investmentRepo := &mockInvestmentRepository{
		investments: make(map[string]*domain.Investment),
	}

	// Pre-populate with test investments
	investment1 := domain.NewInvestment("Asset1", "TEST", []string{"Sprint1"}, time.Now(), time.Now(), domain.EUR)
	investment2 := domain.NewInvestment("Asset2", "OTHER", []string{"Sprint1"}, time.Now(), time.Now(), domain.EUR)

	investmentRepo.investments["Asset1"] = investment1
	investmentRepo.investments["Asset2"] = investment2

	service := NewInvestmentService(nil, nil, investmentRepo)

	ctx := context.Background()
	investments, err := service.ListInvestments(ctx, "TEST")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(investments) != 1 {
		t.Errorf("Expected 1 investment for project TEST, got %d", len(investments))
	}

	if investments[0].AssetName != "Asset1" {
		t.Errorf("Expected asset name Asset1, got %s", investments[0].AssetName)
	}
}

// Additional comprehensive test cases for service layer

func TestInvestmentService_CalculateAssetInvestment(t *testing.T) {
	t.Parallel()
	costModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	costModel.SetDefaultRate(domain.Senior, 75.0)

	costModelRepo := &mockCostModelRepository{
		costModels: map[string]*domain.CostModel{
			"TEST": costModel,
		},
	}

	allocationProvider := &mockTimeAllocationProvider{
		allocations: []ports.EngineerAllocation{
			{
				EngineerName: "John Doe",
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

	service := NewInvestmentService(costModelRepo, allocationProvider, investmentRepo)

	ctx := context.Background()
	investment, err := service.CalculateAssetInvestment(ctx, "Test Asset", "TEST", []string{"Sprint1"})

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if investment.AssetName != "Test Asset" {
		t.Errorf("Expected asset name 'Test Asset', got %s", investment.AssetName)
	}
}

func TestInvestmentService_CalculateSprintInvestment(t *testing.T) {
	t.Parallel()
	costModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	costModel.SetDefaultRate(domain.Senior, 75.0)

	costModelRepo := &mockCostModelRepository{
		costModels: map[string]*domain.CostModel{
			"TEST": costModel,
		},
	}

	allocationProvider := &mockTimeAllocationProvider{
		allocations: []ports.EngineerAllocation{
			{
				EngineerName: "John Doe",
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

	service := NewInvestmentService(costModelRepo, allocationProvider, investmentRepo)

	ctx := context.Background()
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	investment, err := service.CalculateSprintInvestment(ctx, "TEST", "Sprint1", startDate, endDate)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Asset name should be formatted as project-sprint
	if investment.AssetName != "TEST-Sprint1" {
		t.Errorf("Expected asset name 'TEST-Sprint1', got %s", investment.AssetName)
	}

	if investment.StartDate != startDate {
		t.Errorf("Expected start date %v, got %v", startDate, investment.StartDate)
	}
}

func TestInvestmentService_GetInvestment(t *testing.T) {
	t.Parallel()
	testInvestment := domain.NewInvestment("Test Asset", "TEST", []string{"Sprint1"}, time.Now(), time.Now(), domain.EUR)

	investmentRepo := &mockInvestmentRepository{
		investments: map[string]*domain.Investment{
			"Test Asset": testInvestment,
		},
	}

	service := NewInvestmentService(nil, nil, investmentRepo)

	ctx := context.Background()
	investment, err := service.GetInvestment(ctx, "Test Asset")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if investment.AssetName != "Test Asset" {
		t.Errorf("Expected asset name 'Test Asset', got %s", investment.AssetName)
	}
}

func TestInvestmentService_GetInvestment_NotFound(t *testing.T) {
	t.Parallel()
	investmentRepo := &mockInvestmentRepository{
		investments: make(map[string]*domain.Investment),
	}

	service := NewInvestmentService(nil, nil, investmentRepo)

	ctx := context.Background()
	_, err := service.GetInvestment(ctx, "Non-existent Asset")

	if err != domain.ErrInvestmentNotFound {
		t.Errorf("Expected ErrInvestmentNotFound, got %v", err)
	}
}

func TestInvestmentService_DeleteInvestment(t *testing.T) {
	t.Parallel()
	testInvestment := domain.NewInvestment("Test Asset", "TEST", []string{"Sprint1"}, time.Now(), time.Now(), domain.EUR)

	investmentRepo := &mockInvestmentRepository{
		investments: map[string]*domain.Investment{
			"Test Asset": testInvestment,
		},
	}

	service := NewInvestmentService(nil, nil, investmentRepo)

	ctx := context.Background()
	err := service.DeleteInvestment(ctx, "Test Asset")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify it was deleted
	_, err = service.GetInvestment(ctx, "Test Asset")
	if err != domain.ErrInvestmentNotFound {
		t.Error("Expected investment to be deleted")
	}
}

func TestInvestmentService_DeleteInvestment_NotFound(t *testing.T) {
	t.Parallel()
	investmentRepo := &mockInvestmentRepository{
		investments: make(map[string]*domain.Investment),
	}

	service := NewInvestmentService(nil, nil, investmentRepo)

	ctx := context.Background()
	err := service.DeleteInvestment(ctx, "Non-existent Asset")

	if err != domain.ErrInvestmentNotFound {
		t.Errorf("Expected ErrInvestmentNotFound, got %v", err)
	}
}

func TestInvestmentService_InitializeCostModel_AlreadyExists(t *testing.T) {
	t.Parallel()
	existingModel, _ := domain.NewCostModel(domain.USD, 8.0, 2.5)

	costModelRepo := &mockCostModelRepository{
		costModels: map[string]*domain.CostModel{
			"TEST": existingModel,
		},
	}

	service := NewInvestmentService(costModelRepo, nil, nil)

	ctx := context.Background()
	costModel, err := service.InitializeCostModel(ctx, "TEST")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should return existing model
	if costModel.Currency != domain.USD {
		t.Errorf("Expected currency USD (existing), got %v", costModel.Currency)
	}
	if costModel.OverheadMultiplier != 2.5 {
		t.Errorf("Expected overhead multiplier 2.5 (existing), got %.1f", costModel.OverheadMultiplier)
	}
}

func TestInvestmentService_GetCostModel_NotFound(t *testing.T) {
	t.Parallel()
	costModelRepo := &mockCostModelRepository{
		costModels: make(map[string]*domain.CostModel),
	}

	service := NewInvestmentService(costModelRepo, nil, nil)

	ctx := context.Background()
	_, err := service.GetCostModel(ctx, "NONEXISTENT")

	if err != domain.ErrCostModelNotFound {
		t.Errorf("Expected ErrCostModelNotFound, got %v", err)
	}
}

func TestInvestmentService_UpdateCostModel_InvalidModel(t *testing.T) {
	t.Parallel()
	costModelRepo := &mockCostModelRepository{
		costModels: make(map[string]*domain.CostModel),
	}

	service := NewInvestmentService(costModelRepo, nil, nil)

	// Create invalid model
	invalidModel := &domain.CostModel{
		Currency:           domain.EUR,
		WorkingHoursPerDay: -1, // Invalid
		OverheadMultiplier: 2.0,
	}

	ctx := context.Background()
	err := service.UpdateCostModel(ctx, "TEST", invalidModel)

	if err == nil {
		t.Error("Expected error for invalid cost model")
	}
}

func TestInvestmentService_ListInvestments_EmptyProject(t *testing.T) {
	t.Parallel()
	investmentRepo := &mockInvestmentRepository{
		investments: make(map[string]*domain.Investment),
	}

	service := NewInvestmentService(nil, nil, investmentRepo)

	ctx := context.Background()
	investments, err := service.ListInvestments(ctx, "EMPTY_PROJECT")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(investments) != 0 {
		t.Errorf("Expected 0 investments for empty project, got %d", len(investments))
	}
}

// Mock time allocation provider for service tests
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

// Mock implementations with error scenarios for better test coverage

type mockCostModelRepositoryWithErrors struct {
	*mockCostModelRepository
	shouldFailGetDefault bool
	shouldFailSave       bool
}

func (m *mockCostModelRepositoryWithErrors) GetDefaultCostModel(_ context.Context) (*domain.CostModel, error) {
	if m.shouldFailGetDefault {
		return nil, domain.ErrCostModelNotFound
	}
	return m.mockCostModelRepository.GetDefaultCostModel(context.Background())
}

func (m *mockCostModelRepositoryWithErrors) SaveCostModel(ctx context.Context, project string, model *domain.CostModel) error {
	if m.shouldFailSave {
		return domain.ErrInvalidCostModel
	}
	return m.mockCostModelRepository.SaveCostModel(ctx, project, model)
}

func TestInvestmentService_InitializeCostModel_GetDefaultError(t *testing.T) {
	t.Parallel()
	costModelRepo := &mockCostModelRepositoryWithErrors{
		mockCostModelRepository: &mockCostModelRepository{},
		shouldFailGetDefault:    true,
	}

	service := NewInvestmentService(costModelRepo, nil, nil)

	ctx := context.Background()
	_, err := service.InitializeCostModel(ctx, "TEST")

	if err == nil {
		t.Fatal("Expected error when GetDefaultCostModel fails")
	}

	expectedError := "failed to get default cost model"
	if !contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedError, err)
	}
}

func TestInvestmentService_InitializeCostModel_SaveError(t *testing.T) {
	t.Parallel()
	costModelRepo := &mockCostModelRepositoryWithErrors{
		mockCostModelRepository: &mockCostModelRepository{},
		shouldFailSave:          true,
	}

	service := NewInvestmentService(costModelRepo, nil, nil)

	ctx := context.Background()
	_, err := service.InitializeCostModel(ctx, "TEST")

	if err == nil {
		t.Fatal("Expected error when SaveCostModel fails")
	}

	expectedError := "failed to save cost model"
	if !contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain '%s', got: %v", expectedError, err)
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || s[len(s)-len(substr):] == substr || s[:len(substr)] == substr || containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
