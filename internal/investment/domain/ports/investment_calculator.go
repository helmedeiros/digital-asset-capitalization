package ports

import (
	"context"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain"
)

// InvestmentCalculator defines the interface for calculating investment costs
type InvestmentCalculator interface {
	// CalculateAssetInvestment calculates the total investment for a specific asset
	CalculateAssetInvestment(ctx context.Context, input AssetInvestmentInput) (*domain.Investment, error)

	// CalculateSprintInvestment calculates investment for a specific sprint
	CalculateSprintInvestment(ctx context.Context, input SprintInvestmentInput) (*domain.Investment, error)

	// CalculateTaskInvestment calculates investment for specific tasks
	CalculateTaskInvestment(ctx context.Context, input TaskInvestmentInput) (*domain.Investment, error)
}

// CostModelRepository defines the interface for managing cost models
type CostModelRepository interface {
	// GetCostModel retrieves the cost model for a project/team
	GetCostModel(ctx context.Context, project string) (*domain.CostModel, error)

	// SaveCostModel saves a cost model
	SaveCostModel(ctx context.Context, project string, model *domain.CostModel) error

	// GetDefaultCostModel returns a default cost model
	GetDefaultCostModel(ctx context.Context) (*domain.CostModel, error)
}

// TimeAllocationProvider defines the interface for retrieving time allocation data
type TimeAllocationProvider interface {
	// GetSprintAllocations gets engineer time allocations for a sprint
	GetSprintAllocations(ctx context.Context, project, sprint string) ([]EngineerAllocation, error)

	// GetAssetAllocations gets engineer time allocations for an asset across sprints
	GetAssetAllocations(ctx context.Context, assetName string) ([]EngineerAllocation, error)

	// GetTaskAllocations gets engineer time allocations for specific tasks
	GetTaskAllocations(ctx context.Context, taskKeys []string) ([]EngineerAllocation, error)
}

// InvestmentRepository defines the interface for storing investment calculations
type InvestmentRepository interface {
	// SaveInvestment saves an investment calculation
	SaveInvestment(ctx context.Context, investment *domain.Investment) error

	// GetInvestment retrieves an investment by asset name
	GetInvestment(ctx context.Context, assetName string) (*domain.Investment, error)

	// ListInvestments lists all investments for a project
	ListInvestments(ctx context.Context, project string) ([]*domain.Investment, error)

	// DeleteInvestment removes an investment calculation
	DeleteInvestment(ctx context.Context, assetName string) error
}

// Input types for different investment calculations

// AssetInvestmentInput represents input for calculating asset investment
type AssetInvestmentInput struct {
	AssetName string            `json:"asset_name"`
	Project   string            `json:"project"`
	Sprints   []string          `json:"sprints"`
	StartDate time.Time         `json:"start_date"`
	EndDate   time.Time         `json:"end_date"`
	CostModel *domain.CostModel `json:"cost_model,omitempty"`
}

// SprintInvestmentInput represents input for calculating sprint investment
type SprintInvestmentInput struct {
	Project   string            `json:"project"`
	Sprint    string            `json:"sprint"`
	StartDate time.Time         `json:"start_date"`
	EndDate   time.Time         `json:"end_date"`
	CostModel *domain.CostModel `json:"cost_model,omitempty"`
}

// TaskInvestmentInput represents input for calculating task-specific investment
type TaskInvestmentInput struct {
	TaskKeys  []string          `json:"task_keys"`
	Project   string            `json:"project"`
	Sprint    string            `json:"sprint"`
	StartDate time.Time         `json:"start_date"`
	EndDate   time.Time         `json:"end_date"`
	CostModel *domain.CostModel `json:"cost_model,omitempty"`
}

// EngineerAllocation represents time allocation data for an engineer
type EngineerAllocation struct {
	EngineerName string    `json:"engineer_name"`
	TaskKey      string    `json:"task_key"`
	TaskTitle    string    `json:"task_title"`
	WorkType     string    `json:"work_type"`
	AssetName    string    `json:"asset_name"`
	Sprint       string    `json:"sprint"`
	Allocation   float64   `json:"allocation"` // Percentage (0-100)
	StartDate    time.Time `json:"start_date"`
	EndDate      time.Time `json:"end_date"`
}
