package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain/ports"
)

// CalculateAssetInvestmentUseCase handles the calculation of investment for a specific asset
type CalculateAssetInvestmentUseCase struct {
	costModelRepo      ports.CostModelRepository
	allocationProvider ports.TimeAllocationProvider
	investmentRepo     ports.InvestmentRepository
}

// NewCalculateAssetInvestmentUseCase creates a new use case instance
func NewCalculateAssetInvestmentUseCase(
	costModelRepo ports.CostModelRepository,
	allocationProvider ports.TimeAllocationProvider,
	investmentRepo ports.InvestmentRepository,
) *CalculateAssetInvestmentUseCase {
	return &CalculateAssetInvestmentUseCase{
		costModelRepo:      costModelRepo,
		allocationProvider: allocationProvider,
		investmentRepo:     investmentRepo,
	}
}

// Execute calculates the investment for an asset
func (uc *CalculateAssetInvestmentUseCase) Execute(ctx context.Context, input ports.AssetInvestmentInput) (*domain.Investment, error) {
	// Get cost model
	costModel := input.CostModel
	if costModel == nil {
		var err error
		costModel, err = uc.costModelRepo.GetCostModel(ctx, input.Project)
		if err != nil {
			// Fallback to default cost model
			costModel, err = uc.costModelRepo.GetDefaultCostModel(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get cost model: %w", err)
			}
		}
	}

	// Validate cost model
	if err := costModel.Validate(); err != nil {
		return nil, fmt.Errorf("invalid cost model: %w", err)
	}

	// Get time allocations for the asset
	allocations, err := uc.allocationProvider.GetAssetAllocations(ctx, input.AssetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get time allocations: %w", err)
	}

	// Filter allocations by date range if provided
	if !input.StartDate.IsZero() && !input.EndDate.IsZero() {
		allocations = uc.filterAllocationsByDateRange(allocations, input.StartDate, input.EndDate)
	}

	// Create investment calculation
	investment := domain.NewInvestment(
		input.AssetName,
		input.Project,
		input.Sprints,
		input.StartDate,
		input.EndDate,
		costModel.Currency,
	)

	// Calculate engineer investments
	engineerMap := make(map[string]*domain.EngineerInvestment)

	for _, allocation := range allocations {
		// Calculate hours for this allocation
		days := uc.calculateWorkingDays(allocation.StartDate, allocation.EndDate)
		hours := days * costModel.WorkingHoursPerDay * (allocation.Allocation / 100.0)

		// Get engineer rate
		rate := costModel.GetEngineerRateOrDefault(allocation.EngineerName, domain.Mid) // Default to mid-level

		// Calculate costs
		directCost := domain.NewMoney(hours*rate, costModel.Currency)
		overheadCost := directCost.Multiply(costModel.OverheadMultiplier - 1.0) // Subtract 1 to get only overhead portion
		totalCost := directCost.Add(overheadCost)

		// Add to engineer investment
		if engineer, exists := engineerMap[allocation.EngineerName]; exists {
			engineer.TotalHours += hours
			engineer.DirectCost = engineer.DirectCost.Add(directCost)
			engineer.OverheadCost = engineer.OverheadCost.Add(overheadCost)
			engineer.TotalCost = engineer.TotalCost.Add(totalCost)

			// Add sprint if not already present
			if !uc.containsString(engineer.Sprints, allocation.Sprint) {
				engineer.Sprints = append(engineer.Sprints, allocation.Sprint)
			}
		} else {
			level := costModel.InferEngineerLevel(rate)
			engineerMap[allocation.EngineerName] = &domain.EngineerInvestment{
				Name:         allocation.EngineerName,
				Level:        level,
				TotalHours:   hours,
				HourlyRate:   rate,
				DirectCost:   directCost,
				OverheadCost: overheadCost,
				TotalCost:    totalCost,
				Sprints:      []string{allocation.Sprint},
			}
		}

		// Add task investment
		taskInvestment := domain.TaskInvestment{
			TaskKey:   allocation.TaskKey,
			TaskTitle: allocation.TaskTitle,
			WorkType:  allocation.WorkType,
			Sprint:    allocation.Sprint,
			Engineers: map[string]domain.EngineerTaskEffort{
				allocation.EngineerName: {
					Allocation:   allocation.Allocation,
					Hours:        hours,
					DirectCost:   directCost,
					OverheadCost: overheadCost,
					TotalCost:    totalCost,
				},
			},
			TotalCost: totalCost,
			StartDate: allocation.StartDate,
			EndDate:   allocation.EndDate,
		}

		investment.AddTaskInvestment(taskInvestment)
	}

	// Add all engineer investments
	for _, engineer := range engineerMap {
		investment.AddEngineerInvestment(*engineer)
	}

	// Calculate infrastructure costs
	if !input.StartDate.IsZero() && !input.EndDate.IsZero() {
		infraCost, err := costModel.CalculateInfrastructureCostForPeriod(input.StartDate, input.EndDate)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate infrastructure costs: %w", err)
		}
		investment.SetInfrastructureCosts(domain.NewMoney(infraCost, costModel.Currency))
	}

	// Calculate total
	investment.CalculateTotalCost()

	// Save investment
	if err := uc.investmentRepo.SaveInvestment(ctx, investment); err != nil {
		return nil, fmt.Errorf("failed to save investment: %w", err)
	}

	return investment, nil
}

// filterAllocationsByDateRange filters allocations to only include those within the date range
func (uc *CalculateAssetInvestmentUseCase) filterAllocationsByDateRange(allocations []ports.EngineerAllocation, start, end time.Time) []ports.EngineerAllocation {
	var filtered []ports.EngineerAllocation
	for _, allocation := range allocations {
		// Check if allocation overlaps with the date range
		if allocation.EndDate.After(start) && allocation.StartDate.Before(end) {
			filtered = append(filtered, allocation)
		}
	}
	return filtered
}

// calculateWorkingDays calculates the number of working days between two dates
func (uc *CalculateAssetInvestmentUseCase) calculateWorkingDays(start, end time.Time) float64 {
	totalDays := end.Sub(start).Hours() / 24

	// Simple approximation: 5/7 of days are working days
	// In production, we'd want to account for holidays and actual calendar
	workingDays := totalDays * (5.0 / 7.0)

	return workingDays
}

// containsString checks if a slice contains a string
func (uc *CalculateAssetInvestmentUseCase) containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
