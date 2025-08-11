package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain"
)

// InvestmentJSONRepository implements InvestmentRepository using JSON files
type InvestmentJSONRepository struct {
	configDir string
}

// NewInvestmentJSONRepository creates a new JSON-based investment repository
func NewInvestmentJSONRepository(configDir string) *InvestmentJSONRepository {
	return &InvestmentJSONRepository{
		configDir: configDir,
	}
}

// SaveInvestment saves an investment calculation
func (r *InvestmentJSONRepository) SaveInvestment(_ context.Context, investment *domain.Investment) error {
	// Ensure config directory exists
	if err := os.MkdirAll(r.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create filename from asset name (sanitized)
	filename := fmt.Sprintf("investment-%s.json", r.sanitizeFilename(investment.AssetName))
	filePath := filepath.Join(r.configDir, filename)

	data, err := json.MarshalIndent(investment, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal investment: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write investment file: %w", err)
	}

	return nil
}

// GetInvestment retrieves an investment by asset name
func (r *InvestmentJSONRepository) GetInvestment(_ context.Context, assetName string) (*domain.Investment, error) {
	filename := fmt.Sprintf("investment-%s.json", r.sanitizeFilename(assetName))
	filePath := filepath.Join(r.configDir, filename)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("investment not found for asset %s", assetName)
		}
		return nil, fmt.Errorf("failed to read investment file: %w", err)
	}

	var investment domain.Investment
	if err := json.Unmarshal(data, &investment); err != nil {
		return nil, fmt.Errorf("failed to parse investment: %w", err)
	}

	return &investment, nil
}

// ListInvestments lists all investments for a project
func (r *InvestmentJSONRepository) ListInvestments(_ context.Context, project string) ([]*domain.Investment, error) {
	// Read all investment files in the directory
	files, err := os.ReadDir(r.configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*domain.Investment{}, nil
		}
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	var investments []*domain.Investment

	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "investment-") || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(r.configDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue // Skip files we can't read
		}

		var investment domain.Investment
		if err := json.Unmarshal(data, &investment); err != nil {
			continue // Skip files we can't parse
		}

		// Filter by project if specified
		if project == "" || investment.Project == project {
			investments = append(investments, &investment)
		}
	}

	return investments, nil
}

// DeleteInvestment removes an investment calculation
func (r *InvestmentJSONRepository) DeleteInvestment(_ context.Context, assetName string) error {
	filename := fmt.Sprintf("investment-%s.json", r.sanitizeFilename(assetName))
	filePath := filepath.Join(r.configDir, filename)

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("investment not found for asset %s", assetName)
		}
		return fmt.Errorf("failed to delete investment file: %w", err)
	}

	return nil
}

// sanitizeFilename removes invalid characters from asset names to create safe filenames
func (r *InvestmentJSONRepository) sanitizeFilename(assetName string) string {
	// Replace spaces and special characters with underscores
	sanitized := strings.ReplaceAll(assetName, " ", "_")
	sanitized = strings.ReplaceAll(sanitized, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, "\\", "_")
	sanitized = strings.ReplaceAll(sanitized, ":", "_")
	sanitized = strings.ReplaceAll(sanitized, "*", "_")
	sanitized = strings.ReplaceAll(sanitized, "?", "_")
	sanitized = strings.ReplaceAll(sanitized, "\"", "_")
	sanitized = strings.ReplaceAll(sanitized, "<", "_")
	sanitized = strings.ReplaceAll(sanitized, ">", "_")
	sanitized = strings.ReplaceAll(sanitized, "|", "_")

	// Convert to lowercase
	sanitized = strings.ToLower(sanitized)

	return sanitized
}
