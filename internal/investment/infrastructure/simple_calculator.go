package infrastructure

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain/ports"
)

// SimpleInvestmentCalculator provides a straightforward investment calculation
// using the existing sprint allocation data directly
type SimpleInvestmentCalculator struct {
	costModelRepo ports.CostModelRepository
}

// NewSimpleInvestmentCalculator creates a new simple calculator
func NewSimpleInvestmentCalculator(costModelRepo ports.CostModelRepository) *SimpleInvestmentCalculator {
	return &SimpleInvestmentCalculator{
		costModelRepo: costModelRepo,
	}
}

// CalculateSprintInvestmentFromCSV calculates investment from CSV sprint allocation data
func (c *SimpleInvestmentCalculator) CalculateSprintInvestmentFromCSV(ctx context.Context, project, sprint string, csvData string) (*domain.Investment, error) {
	// Get cost model
	costModel, err := c.costModelRepo.GetCostModel(ctx, project)
	if err != nil {
		costModel, err = c.costModelRepo.GetDefaultCostModel(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get cost model: %w", err)
		}
	}

	// Parse sprint dates from the data or use defaults
	startDate := time.Date(2025, 6, 25, 0, 0, 0, 0, time.UTC) // Default start
	endDate := time.Date(2025, 7, 9, 0, 0, 0, 0, time.UTC)    // Default end

	// Create investment
	investment := domain.NewInvestment(
		fmt.Sprintf("%s-%s", project, sprint),
		project,
		[]string{sprint},
		startDate,
		endDate,
		costModel.Currency,
	)

	// Parse CSV data
	lines := strings.Split(csvData, "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("invalid CSV data")
	}

	header := strings.Split(lines[0], ",")
	engineerStartIndex := c.findEngineerColumnsStart(header)
	if engineerStartIndex == -1 {
		return nil, fmt.Errorf("could not find engineer columns")
	}

	engineerMap := make(map[string]*domain.EngineerInvestment)
	sprintDurationDays := 10.0 // 2 weeks working days

	// Process each task row
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}

		record := c.parseCSVLine(line)
		if len(record) < engineerStartIndex {
			continue
		}

		taskKey := c.getFieldValue(record, 1, "")
		taskTitle := c.getFieldValue(record, 3, "")
		workType := c.getFieldValue(record, 4, "")
		_ = c.getFieldValue(record, 5, "") // assetName - not used in this implementation

		// Process engineer allocations
		for j := engineerStartIndex; j < len(record) && j < len(header); j++ {
			engineerName := strings.TrimSpace(header[j])
			allocationStr := strings.TrimSpace(record[j])

			if allocationStr == "" || allocationStr == "0.00%" || allocationStr == "\"\"" {
				continue
			}

			// Parse allocation percentage
			allocationStr = strings.Trim(allocationStr, "\"")
			allocationStr = strings.TrimSuffix(allocationStr, "%")
			allocation, err := strconv.ParseFloat(allocationStr, 64)
			if err != nil {
				continue
			}

			if allocation <= 0 {
				continue
			}

			// Calculate hours for this allocation
			hours := sprintDurationDays * costModel.WorkingHoursPerDay * (allocation / 100.0)

			// Get engineer rate
			rate := costModel.GetEngineerRateOrDefault(engineerName, domain.Mid)

			// Calculate costs
			directCost := domain.NewMoney(hours*rate, costModel.Currency)
			overheadCost := directCost.Multiply(costModel.OverheadMultiplier - 1.0)
			totalCost := directCost.Add(overheadCost)

			// Add to engineer investment
			if engineer, exists := engineerMap[engineerName]; exists {
				engineer.TotalHours += hours
				engineer.DirectCost = engineer.DirectCost.Add(directCost)
				engineer.OverheadCost = engineer.OverheadCost.Add(overheadCost)
				engineer.TotalCost = engineer.TotalCost.Add(totalCost)
			} else {
				level := costModel.InferEngineerLevel(rate)
				engineerMap[engineerName] = &domain.EngineerInvestment{
					Name:         engineerName,
					Level:        level,
					TotalHours:   hours,
					HourlyRate:   rate,
					DirectCost:   directCost,
					OverheadCost: overheadCost,
					TotalCost:    totalCost,
					Sprints:      []string{sprint},
				}
			}

			// Add task investment
			taskInvestment := domain.TaskInvestment{
				TaskKey:   taskKey,
				TaskTitle: taskTitle,
				WorkType:  workType,
				Sprint:    sprint,
				Engineers: map[string]domain.EngineerTaskEffort{
					engineerName: {
						Allocation:   allocation,
						Hours:        hours,
						DirectCost:   directCost,
						OverheadCost: overheadCost,
						TotalCost:    totalCost,
					},
				},
				TotalCost: totalCost,
				StartDate: startDate,
				EndDate:   endDate,
			}

			investment.AddTaskInvestment(taskInvestment)
		}
	}

	// Add all engineer investments
	for _, engineer := range engineerMap {
		investment.AddEngineerInvestment(*engineer)
	}

	// Calculate infrastructure costs for the sprint period
	infraCost, err := costModel.CalculateInfrastructureCostForPeriod(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate infrastructure costs: %w", err)
	}
	investment.SetInfrastructureCosts(domain.NewMoney(infraCost, costModel.Currency))

	// Calculate total
	investment.CalculateTotalCost()

	return investment, nil
}

// Helper methods from TimeAllocationAdapter
func (c *SimpleInvestmentCalculator) findEngineerColumnsStart(header []string) int {
	standardColumns := []string{"sprint", "issueKey", "issueType", "issueTitle", "workType", "assetName", "status", "dateStarted", "dateCompleted"}

	if len(header) > len(standardColumns) {
		allMatch := true
		for i, expected := range standardColumns {
			if i >= len(header) || strings.TrimSpace(header[i]) != expected {
				allMatch = false
				break
			}
		}
		if allMatch {
			return len(standardColumns)
		}
	}

	for i, col := range header {
		if c.looksLikeEngineerName(col) {
			return i
		}
	}

	return -1
}

func (c *SimpleInvestmentCalculator) looksLikeEngineerName(name string) bool {
	parts := strings.Split(strings.TrimSpace(name), " ")
	return len(parts) >= 2 && len(name) > 3 && name[0] >= 'A' && name[0] <= 'Z'
}

func (c *SimpleInvestmentCalculator) parseCSVLine(line string) []string {
	// Simple CSV parsing - could be improved for complex cases
	parts := strings.Split(line, ",")
	for i, part := range parts {
		parts[i] = strings.Trim(strings.TrimSpace(part), "\"")
	}
	return parts
}

func (c *SimpleInvestmentCalculator) getFieldValue(record []string, index int, _ string) string {
	if index < len(record) {
		return strings.TrimSpace(record[index])
	}
	return ""
}
