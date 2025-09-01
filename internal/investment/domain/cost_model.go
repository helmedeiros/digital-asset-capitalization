package domain

import (
	"errors"
	"time"
)

// Domain-specific errors
var (
	ErrInvalidCostModel    = errors.New("invalid cost model")
	ErrNegativeRate        = errors.New("rate cannot be negative")
	ErrInvalidDateRange    = errors.New("invalid date range")
	ErrEngineerNotFound    = errors.New("engineer not found in cost model")
	ErrInvalidWorkingHours = errors.New("working hours per day must be positive")
	ErrInvalidOverhead     = errors.New("overhead multiplier must be positive")
	ErrCostModelNotFound   = errors.New("cost model not found")
	ErrInvestmentNotFound  = errors.New("investment not found")
)

// Currency represents different currencies for cost calculations
type Currency string

const (
	EUR Currency = "EUR"
	USD Currency = "USD"
	GBP Currency = "GBP"
)

// String returns the string representation of Currency
func (c Currency) String() string {
	return string(c)
}

// EngineerLevel represents different engineer experience levels
type EngineerLevel string

const (
	Junior    EngineerLevel = "junior"
	Mid       EngineerLevel = "mid"
	Senior    EngineerLevel = "senior"
	Staff     EngineerLevel = "staff"
	Principal EngineerLevel = "principal"
)

// CostModel defines the cost structure for investment calculations
type CostModel struct {
	Currency            Currency                  `json:"currency"`
	WorkingHoursPerDay  float64                   `json:"working_hours_per_day"`
	OverheadMultiplier  float64                   `json:"overhead_multiplier"`
	InfrastructureCosts InfrastructureCosts       `json:"infrastructure_costs"`
	EngineerRates       map[string]EngineerRate   `json:"engineer_rates"`
	DefaultRatesByLevel map[EngineerLevel]float64 `json:"default_rates_by_level"`
}

// EngineerRate represents the cost information for a specific engineer
type EngineerRate struct {
	Name       string        `json:"name"`
	HourlyRate float64       `json:"hourly_rate"`
	Level      EngineerLevel `json:"level"`
	Team       string        `json:"team"`
}

// InfrastructureCosts represents fixed infrastructure costs
type InfrastructureCosts struct {
	CloudCostsPerMonth   float64 `json:"cloud_costs_per_month"`
	ToolingCostsPerMonth float64 `json:"tooling_costs_per_month"`
	LicenseCostsPerMonth float64 `json:"license_costs_per_month"`
}

// NewCostModel creates a new cost model with validation
func NewCostModel(currency Currency, workingHours, overhead float64) (*CostModel, error) {
	if workingHours <= 0 {
		return nil, ErrInvalidWorkingHours
	}
	if overhead < 1.0 {
		return nil, ErrInvalidOverhead
	}

	return &CostModel{
		Currency:            currency,
		WorkingHoursPerDay:  workingHours,
		OverheadMultiplier:  overhead,
		EngineerRates:       make(map[string]EngineerRate),
		DefaultRatesByLevel: make(map[EngineerLevel]float64),
		InfrastructureCosts: InfrastructureCosts{},
	}, nil
}

// AddEngineerRate adds a rate for a specific engineer
func (cm *CostModel) AddEngineerRate(engineer EngineerRate) error {
	if engineer.HourlyRate < 0 {
		return ErrNegativeRate
	}
	cm.EngineerRates[engineer.Name] = engineer
	return nil
}

// SetDefaultRate sets a default hourly rate for an engineer level
func (cm *CostModel) SetDefaultRate(level EngineerLevel, rate float64) error {
	if rate < 0 {
		return ErrNegativeRate
	}
	cm.DefaultRatesByLevel[level] = rate
	return nil
}

// GetEngineerRate retrieves the hourly rate for an engineer
func (cm *CostModel) GetEngineerRate(engineerName string) (float64, error) {
	if rate, exists := cm.EngineerRates[engineerName]; exists {
		return rate.HourlyRate, nil
	}
	return 0, ErrEngineerNotFound
}

// GetEngineerRateOrDefault gets engineer rate or falls back to level default
func (cm *CostModel) GetEngineerRateOrDefault(engineerName string, level EngineerLevel) float64 {
	if rate, exists := cm.EngineerRates[engineerName]; exists {
		return rate.HourlyRate
	}
	if defaultRate, exists := cm.DefaultRatesByLevel[level]; exists {
		return defaultRate
	}
	return 0
}

// CalculateInfrastructureCostForPeriod calculates infrastructure costs for a given period
func (cm *CostModel) CalculateInfrastructureCostForPeriod(startDate, endDate time.Time) (float64, error) {
	if endDate.Before(startDate) {
		return 0, ErrInvalidDateRange
	}

	days := endDate.Sub(startDate).Hours() / 24
	months := days / 30.44 // Average days per month

	totalMonthlyCost := cm.InfrastructureCosts.CloudCostsPerMonth +
		cm.InfrastructureCosts.ToolingCostsPerMonth +
		cm.InfrastructureCosts.LicenseCostsPerMonth

	return totalMonthlyCost * months, nil
}

// GetTotalMonthlyCost returns the total monthly infrastructure cost
func (cm *CostModel) GetTotalMonthlyCost() float64 {
	return cm.InfrastructureCosts.CloudCostsPerMonth +
		cm.InfrastructureCosts.ToolingCostsPerMonth +
		cm.InfrastructureCosts.LicenseCostsPerMonth
}

// InferEngineerLevel infers engineer level from hourly rate
func (cm *CostModel) InferEngineerLevel(rate float64) EngineerLevel {
	// Check against default rates in order of seniority (to handle overlapping tolerances)
	// When ranges overlap, prefer the lower level (more conservative)
	levels := []EngineerLevel{Junior, Mid, Senior, Staff, Principal}

	for _, level := range levels {
		if defaultRate, exists := cm.DefaultRatesByLevel[level]; exists {
			if rate >= defaultRate*0.9 && rate <= defaultRate*1.1 {
				return level
			}
		}
	}

	// Fallback based on rate ranges
	switch {
	case rate >= 80:
		return Principal
	case rate >= 70:
		return Staff
	case rate >= 60:
		return Senior
	case rate >= 45:
		return Mid
	default:
		return Junior
	}
}

// Validate ensures the cost model is valid
func (cm *CostModel) Validate() error {
	if cm.WorkingHoursPerDay <= 0 {
		return ErrInvalidWorkingHours
	}
	if cm.OverheadMultiplier < 1.0 {
		return ErrInvalidOverhead
	}
	for _, rate := range cm.EngineerRates {
		if rate.HourlyRate < 0 {
			return ErrNegativeRate
		}
	}
	return nil
}
