package service

import (
	"context"
	"fmt"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/investment/application/usecase"
	"github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain/ports"
)

// InvestmentService provides high-level investment calculation operations
type InvestmentService struct {
	calculateAssetInvestment *usecase.CalculateAssetInvestmentUseCase
	costModelRepo            ports.CostModelRepository
	investmentRepo           ports.InvestmentRepository
}

// NewInvestmentService creates a new investment service
func NewInvestmentService(
	costModelRepo ports.CostModelRepository,
	allocationProvider ports.TimeAllocationProvider,
	investmentRepo ports.InvestmentRepository,
) *InvestmentService {
	calculateAssetInvestment := usecase.NewCalculateAssetInvestmentUseCase(
		costModelRepo,
		allocationProvider,
		investmentRepo,
	)

	return &InvestmentService{
		calculateAssetInvestment: calculateAssetInvestment,
		costModelRepo:            costModelRepo,
		investmentRepo:           investmentRepo,
	}
}

// CalculateAssetInvestment calculates investment for a specific asset
func (s *InvestmentService) CalculateAssetInvestment(ctx context.Context, assetName, project string, sprints []string) (*domain.Investment, error) {
	input := ports.AssetInvestmentInput{
		AssetName: assetName,
		Project:   project,
		Sprints:   sprints,
	}

	return s.calculateAssetInvestment.Execute(ctx, input)
}

// CalculateSprintInvestment calculates investment for a specific sprint
func (s *InvestmentService) CalculateSprintInvestment(ctx context.Context, project, sprint string, startDate, endDate time.Time) (*domain.Investment, error) {
	input := ports.AssetInvestmentInput{
		AssetName: fmt.Sprintf("%s-%s", project, sprint),
		Project:   project,
		Sprints:   []string{sprint},
		StartDate: startDate,
		EndDate:   endDate,
	}

	return s.calculateAssetInvestment.Execute(ctx, input)
}

// GetInvestment retrieves a saved investment calculation
func (s *InvestmentService) GetInvestment(ctx context.Context, assetName string) (*domain.Investment, error) {
	return s.investmentRepo.GetInvestment(ctx, assetName)
}

// ListInvestments lists all investments for a project
func (s *InvestmentService) ListInvestments(ctx context.Context, project string) ([]*domain.Investment, error) {
	return s.investmentRepo.ListInvestments(ctx, project)
}

// DeleteInvestment removes an investment calculation
func (s *InvestmentService) DeleteInvestment(ctx context.Context, assetName string) error {
	return s.investmentRepo.DeleteInvestment(ctx, assetName)
}

// InitializeCostModel creates and saves a cost model for a project
func (s *InvestmentService) InitializeCostModel(ctx context.Context, project string) (*domain.CostModel, error) {
	// Check if cost model already exists
	if existing, err := s.costModelRepo.GetCostModel(ctx, project); err == nil {
		return existing, nil
	}

	// Get default cost model
	defaultModel, err := s.costModelRepo.GetDefaultCostModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get default cost model: %w", err)
	}

	// Save as project-specific model
	if err := s.costModelRepo.SaveCostModel(ctx, project, defaultModel); err != nil {
		return nil, fmt.Errorf("failed to save cost model: %w", err)
	}

	return defaultModel, nil
}

// GetCostModel retrieves the cost model for a project
func (s *InvestmentService) GetCostModel(ctx context.Context, project string) (*domain.CostModel, error) {
	return s.costModelRepo.GetCostModel(ctx, project)
}

// UpdateCostModel updates the cost model for a project
func (s *InvestmentService) UpdateCostModel(ctx context.Context, project string, model *domain.CostModel) error {
	if err := model.Validate(); err != nil {
		return fmt.Errorf("invalid cost model: %w", err)
	}
	return s.costModelRepo.SaveCostModel(ctx, project, model)
}
