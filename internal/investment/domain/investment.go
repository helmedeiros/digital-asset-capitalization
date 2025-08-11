package domain

import (
	"fmt"
	"time"
)

// Investment represents a calculated investment amount for a digital asset
type Investment struct {
	AssetName           string               `json:"asset_name"`
	Project             string               `json:"project"`
	Sprints             []string             `json:"sprints"`
	StartDate           time.Time            `json:"start_date"`
	EndDate             time.Time            `json:"end_date"`
	TotalCost           Money                `json:"total_cost"`
	EngineerCosts       Money                `json:"engineer_costs"`
	InfrastructureCosts Money                `json:"infrastructure_costs"`
	OverheadCosts       Money                `json:"overhead_costs"`
	EngineersInvolved   []EngineerInvestment `json:"engineers_involved"`
	TaskBreakdown       []TaskInvestment     `json:"task_breakdown"`
	WorkTypeBreakdown   map[string]Money     `json:"work_type_breakdown"`
	CalculatedAt        time.Time            `json:"calculated_at"`
}

// EngineerInvestment represents an individual engineer's investment
type EngineerInvestment struct {
	Name         string        `json:"name"`
	Level        EngineerLevel `json:"level"`
	TotalHours   float64       `json:"total_hours"`
	HourlyRate   float64       `json:"hourly_rate"`
	DirectCost   Money         `json:"direct_cost"`
	OverheadCost Money         `json:"overhead_cost"`
	TotalCost    Money         `json:"total_cost"`
	Sprints      []string      `json:"sprints"`
}

// TaskInvestment represents investment for a specific task
type TaskInvestment struct {
	TaskKey   string                        `json:"task_key"`
	TaskTitle string                        `json:"task_title"`
	WorkType  string                        `json:"work_type"`
	Sprint    string                        `json:"sprint"`
	Engineers map[string]EngineerTaskEffort `json:"engineers"`
	TotalCost Money                         `json:"total_cost"`
	StartDate time.Time                     `json:"start_date"`
	EndDate   time.Time                     `json:"end_date"`
}

// EngineerTaskEffort represents an engineer's effort on a specific task
type EngineerTaskEffort struct {
	Allocation   float64 `json:"allocation"` // Percentage (0-100)
	Hours        float64 `json:"hours"`
	DirectCost   Money   `json:"direct_cost"`
	OverheadCost Money   `json:"overhead_cost"`
	TotalCost    Money   `json:"total_cost"`
}

// Money represents a monetary value with currency
type Money struct {
	Amount   float64  `json:"amount"`
	Currency Currency `json:"currency"`
}

// NewMoney creates a new Money instance
func NewMoney(amount float64, currency Currency) Money {
	return Money{
		Amount:   amount,
		Currency: currency,
	}
}

// Add adds two money amounts (must be same currency)
func (m Money) Add(other Money) Money {
	if m.Currency != other.Currency {
		// For now, assume same currency. In production, we'd need currency conversion
		return m
	}
	return Money{
		Amount:   m.Amount + other.Amount,
		Currency: m.Currency,
	}
}

// Multiply multiplies money by a factor
func (m Money) Multiply(factor float64) Money {
	return Money{
		Amount:   m.Amount * factor,
		Currency: m.Currency,
	}
}

// IsZero checks if the money amount is zero
func (m Money) IsZero() bool {
	return m.Amount == 0
}

// String returns the string representation of Money
func (m Money) String() string {
	return fmt.Sprintf("%.2f %s", m.Amount, m.Currency)
}

// NewInvestment creates a new investment calculation result
func NewInvestment(assetName, project string, sprints []string, startDate, endDate time.Time, currency Currency) *Investment {
	return &Investment{
		AssetName:           assetName,
		Project:             project,
		Sprints:             sprints,
		StartDate:           startDate,
		EndDate:             endDate,
		TotalCost:           NewMoney(0, currency),
		EngineerCosts:       NewMoney(0, currency),
		InfrastructureCosts: NewMoney(0, currency),
		OverheadCosts:       NewMoney(0, currency),
		EngineersInvolved:   make([]EngineerInvestment, 0),
		TaskBreakdown:       make([]TaskInvestment, 0),
		WorkTypeBreakdown:   make(map[string]Money),
		CalculatedAt:        time.Now(),
	}
}

// AddEngineerInvestment adds an engineer's investment to the total
func (inv *Investment) AddEngineerInvestment(engineer EngineerInvestment) {
	inv.EngineersInvolved = append(inv.EngineersInvolved, engineer)
	inv.EngineerCosts = inv.EngineerCosts.Add(engineer.DirectCost)
	inv.OverheadCosts = inv.OverheadCosts.Add(engineer.OverheadCost)
}

// AddTaskInvestment adds a task's investment to the total
func (inv *Investment) AddTaskInvestment(task TaskInvestment) {
	inv.TaskBreakdown = append(inv.TaskBreakdown, task)

	// Add to work type breakdown
	if existing, exists := inv.WorkTypeBreakdown[task.WorkType]; exists {
		inv.WorkTypeBreakdown[task.WorkType] = existing.Add(task.TotalCost)
	} else {
		inv.WorkTypeBreakdown[task.WorkType] = task.TotalCost
	}
}

// SetInfrastructureCosts sets the infrastructure costs
func (inv *Investment) SetInfrastructureCosts(costs Money) {
	inv.InfrastructureCosts = costs
}

// CalculateTotalCost calculates and sets the total investment cost
func (inv *Investment) CalculateTotalCost() {
	inv.TotalCost = inv.EngineerCosts.Add(inv.OverheadCosts).Add(inv.InfrastructureCosts)
}

// GetDurationInDays returns the investment period duration in days
func (inv *Investment) GetDurationInDays() int {
	return int(inv.EndDate.Sub(inv.StartDate).Hours() / 24)
}

// GetEngineerCount returns the number of engineers involved
func (inv *Investment) GetEngineerCount() int {
	return len(inv.EngineersInvolved)
}

// GetTaskCount returns the number of tasks in the investment
func (inv *Investment) GetTaskCount() int {
	return len(inv.TaskBreakdown)
}
