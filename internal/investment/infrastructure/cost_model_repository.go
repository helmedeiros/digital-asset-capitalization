package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain"
)

// CostModelJSONRepository implements CostModelRepository using JSON files
type CostModelJSONRepository struct {
	configDir string
}

// NewCostModelJSONRepository creates a new JSON-based cost model repository
func NewCostModelJSONRepository(configDir string) *CostModelJSONRepository {
	return &CostModelJSONRepository{
		configDir: configDir,
	}
}

// GetCostModel retrieves the cost model for a project/team
func (r *CostModelJSONRepository) GetCostModel(_ context.Context, project string) (*domain.CostModel, error) {
	filePath := filepath.Join(r.configDir, fmt.Sprintf("cost-model-%s.json", project))

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("cost model not found for project %s", project)
		}
		return nil, fmt.Errorf("failed to read cost model file: %w", err)
	}

	var costModel domain.CostModel
	if err := json.Unmarshal(data, &costModel); err != nil {
		return nil, fmt.Errorf("failed to parse cost model: %w", err)
	}

	return &costModel, nil
}

// SaveCostModel saves a cost model
func (r *CostModelJSONRepository) SaveCostModel(_ context.Context, project string, model *domain.CostModel) error {
	// Ensure config directory exists
	if err := os.MkdirAll(r.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	filePath := filepath.Join(r.configDir, fmt.Sprintf("cost-model-%s.json", project))

	data, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cost model: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cost model file: %w", err)
	}

	return nil
}

// GetDefaultCostModel returns a default cost model with Berlin market rates
func (r *CostModelJSONRepository) GetDefaultCostModel(_ context.Context) (*domain.CostModel, error) {
	costModel, err := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	if err != nil {
		return nil, fmt.Errorf("failed to create default cost model: %w", err)
	}

	// Set default rates based on Berlin market (2025)
	if err := costModel.SetDefaultRate(domain.Junior, 45.0); err != nil {
		return nil, fmt.Errorf("failed to set junior rate: %w", err)
	}
	if err := costModel.SetDefaultRate(domain.Mid, 60.0); err != nil {
		return nil, fmt.Errorf("failed to set mid rate: %w", err)
	}
	if err := costModel.SetDefaultRate(domain.Senior, 75.0); err != nil {
		return nil, fmt.Errorf("failed to set senior rate: %w", err)
	}
	if err := costModel.SetDefaultRate(domain.Staff, 85.0); err != nil {
		return nil, fmt.Errorf("failed to set staff rate: %w", err)
	}
	if err := costModel.SetDefaultRate(domain.Principal, 95.0); err != nil {
		return nil, fmt.Errorf("failed to set principal rate: %w", err)
	}

	// Set default infrastructure costs
	costModel.InfrastructureCosts.CloudCostsPerMonth = 2000.0
	costModel.InfrastructureCosts.ToolingCostsPerMonth = 500.0
	costModel.InfrastructureCosts.LicenseCostsPerMonth = 300.0

	return costModel, nil
}

// CreateFNCostModel creates a cost model specifically for the FN team
func (r *CostModelJSONRepository) CreateFNCostModel(ctx context.Context) (*domain.CostModel, error) {
	costModel, err := r.GetDefaultCostModel(ctx)
	if err != nil {
		return nil, err
	}

	// Add specific engineer rates for FN team (example rates)
	engineers := map[string]domain.EngineerRate{
		"Viktor Kovarik":        {Name: "Viktor Kovarik", HourlyRate: 75.0, Level: domain.Senior, Team: "FN"},
		"Ahmed Naser":           {Name: "Ahmed Naser", HourlyRate: 70.0, Level: domain.Senior, Team: "FN"},
		"Santhosh Balakrishnan": {Name: "Santhosh Balakrishnan", HourlyRate: 65.0, Level: domain.Mid, Team: "FN"},
		"Thais Hamilton":        {Name: "Thais Hamilton", HourlyRate: 75.0, Level: domain.Senior, Team: "FN"},
		"Cesar Vortmann":        {Name: "Cesar Vortmann", HourlyRate: 80.0, Level: domain.Staff, Team: "FN"},
		"Helio Medeiros":        {Name: "Helio Medeiros", HourlyRate: 85.0, Level: domain.Staff, Team: "FN"},
		"Talita Roberti":        {Name: "Talita Roberti", HourlyRate: 65.0, Level: domain.Mid, Team: "FN"},
		"Georgii Maltsev":       {Name: "Georgii Maltsev", HourlyRate: 70.0, Level: domain.Senior, Team: "FN"},
	}

	for _, engineer := range engineers {
		if err := costModel.AddEngineerRate(engineer); err != nil {
			return nil, fmt.Errorf("failed to add engineer rate for %s: %w", engineer.Name, err)
		}
	}

	// Save the FN cost model
	if err := r.SaveCostModel(ctx, "FN", costModel); err != nil {
		return nil, fmt.Errorf("failed to save FN cost model: %w", err)
	}

	return costModel, nil
}
