package infrastructure

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/helmedeiros/digital-asset-capitalization/internal/investment/domain"
)

func TestCostModelJSONRepository_GetCostModel(t *testing.T) {
	// Create a temporary directory for test
	tempDir := t.TempDir()
	repo := NewCostModelJSONRepository(tempDir)

	ctx := context.Background()

	// Test getting non-existent cost model
	_, err := repo.GetCostModel(ctx, "NONEXISTENT")
	if err == nil {
		t.Error("Expected error for non-existent cost model")
	}

	// Create and save a cost model first
	costModel, err := domain.NewCostModel(domain.EUR, 8.0, 2.0)
	if err != nil {
		t.Fatalf("Failed to create cost model: %v", err)
	}

	err = repo.SaveCostModel(ctx, "TEST", costModel)
	if err != nil {
		t.Fatalf("Failed to save cost model: %v", err)
	}

	// Now retrieve it
	retrievedModel, err := repo.GetCostModel(ctx, "TEST")
	if err != nil {
		t.Fatalf("Failed to get cost model: %v", err)
	}

	if retrievedModel.Currency != domain.EUR {
		t.Errorf("Expected currency EUR, got %v", retrievedModel.Currency)
	}

	if retrievedModel.WorkingHoursPerDay != 8.0 {
		t.Errorf("Expected working hours 8.0, got %.1f", retrievedModel.WorkingHoursPerDay)
	}

	if retrievedModel.OverheadMultiplier != 2.0 {
		t.Errorf("Expected overhead multiplier 2.0, got %.1f", retrievedModel.OverheadMultiplier)
	}
}

func TestCostModelJSONRepository_SaveCostModel(t *testing.T) {
	tempDir := t.TempDir()
	repo := NewCostModelJSONRepository(tempDir)

	ctx := context.Background()
	costModel, _ := domain.NewCostModel(domain.USD, 7.5, 1.8)

	err := repo.SaveCostModel(ctx, "SAVE_TEST", costModel)
	if err != nil {
		t.Fatalf("Failed to save cost model: %v", err)
	}

	// Verify the file was created
	filePath := filepath.Join(tempDir, "cost-model-SAVE_TEST.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Expected cost model file to be created")
	}
}

func TestCostModelJSONRepository_GetDefaultCostModel(t *testing.T) {
	tempDir := t.TempDir()
	repo := NewCostModelJSONRepository(tempDir)

	ctx := context.Background()
	costModel, err := repo.GetDefaultCostModel(ctx)
	if err != nil {
		t.Fatalf("Failed to get default cost model: %v", err)
	}

	if costModel.Currency != domain.EUR {
		t.Errorf("Expected currency EUR, got %v", costModel.Currency)
	}

	if costModel.WorkingHoursPerDay != 8.0 {
		t.Errorf("Expected working hours 8.0, got %.1f", costModel.WorkingHoursPerDay)
	}

	if costModel.OverheadMultiplier != 2.0 {
		t.Errorf("Expected overhead multiplier 2.0, got %.1f", costModel.OverheadMultiplier)
	}

	// Check that default rates are set
	if len(costModel.DefaultRatesByLevel) == 0 {
		t.Error("Expected default rates to be set")
	}

	if costModel.GetTotalMonthlyCost() <= 0 {
		t.Error("Expected infrastructure costs to be set")
	}
}

func TestCostModelJSONRepository_CreateFNCostModel(t *testing.T) {
	tempDir := t.TempDir()
	repo := NewCostModelJSONRepository(tempDir)

	ctx := context.Background()
	costModel, err := repo.CreateFNCostModel(ctx)
	if err != nil {
		t.Fatalf("Failed to create FN cost model: %v", err)
	}

	// Should have engineer rates pre-populated
	if len(costModel.EngineerRates) == 0 {
		t.Error("Expected FN cost model to have engineer rates")
	}

	// Check specific engineers were added
	if _, exists := costModel.EngineerRates["Santhosh Balakrishnan"]; !exists {
		t.Error("Expected Santhosh Balakrishnan to be in FN cost model")
	}

	if _, exists := costModel.EngineerRates["Talita Roberti"]; !exists {
		t.Error("Expected Talita Roberti to be in FN cost model")
	}

	// Verify the file was saved
	filePath := filepath.Join(tempDir, "cost-model-FN.json")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Expected FN cost model file to be saved")
	}
}

// Additional comprehensive test cases for infrastructure layer

func TestCostModelJSONRepository_GetCostModel_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	repo := NewCostModelJSONRepository(tempDir)

	// Create invalid JSON file
	filePath := filepath.Join(tempDir, "cost-model-INVALID.json")
	err := os.WriteFile(filePath, []byte("invalid json content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid JSON file: %v", err)
	}

	ctx := context.Background()
	_, err = repo.GetCostModel(ctx, "INVALID")
	if err == nil {
		t.Error("Expected error for invalid JSON content")
	}
}

func TestCostModelJSONRepository_GetCostModel_FileReadError(t *testing.T) {
	// Test with non-existent directory
	repo := NewCostModelJSONRepository("/non/existent/directory")

	ctx := context.Background()
	_, err := repo.GetCostModel(ctx, "TEST")
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
}

func TestCostModelJSONRepository_SaveCostModel_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()

	// Create cost model with circular reference (will cause JSON marshal error)
	costModel := &domain.CostModel{
		Currency:           domain.EUR,
		WorkingHoursPerDay: 8.0,
		OverheadMultiplier: 2.0,
	}

	ctx := context.Background()
	// This should work normally, but let's test with permission denied directory
	readOnlyDir := filepath.Join(tempDir, "readonly")
	err := os.Mkdir(readOnlyDir, 0444) // Read-only
	if err != nil {
		t.Fatalf("Failed to create read-only directory: %v", err)
	}

	readOnlyRepo := NewCostModelJSONRepository(readOnlyDir)
	err = readOnlyRepo.SaveCostModel(ctx, "TEST", costModel)
	if err == nil {
		t.Error("Expected error for read-only directory")
	}
}

func TestCostModelJSONRepository_CreateFNCostModel_SpecificRates(t *testing.T) {
	tempDir := t.TempDir()
	repo := NewCostModelJSONRepository(tempDir)

	ctx := context.Background()
	costModel, err := repo.CreateFNCostModel(ctx)
	if err != nil {
		t.Fatalf("Failed to create FN cost model: %v", err)
	}

	// Test specific engineer rates
	expectedRates := map[string]float64{
		"Viktor Kovarik":        75.0,
		"Ahmed Naser":           70.0,
		"Santhosh Balakrishnan": 65.0,
		"Thais Hamilton":        75.0,
		"Cesar Vortmann":        80.0,
		"Helio Medeiros":        85.0,
		"Talita Roberti":        65.0,
		"Georgii Maltsev":       70.0,
	}

	for name, expectedRate := range expectedRates {
		if engineer, exists := costModel.EngineerRates[name]; !exists {
			t.Errorf("Expected engineer %s to be in FN cost model", name)
		} else if engineer.HourlyRate != expectedRate {
			t.Errorf("Expected %s rate %.1f, got %.1f", name, expectedRate, engineer.HourlyRate)
		}
	}

	// Test engineer levels
	if costModel.EngineerRates["Helio Medeiros"].Level != domain.Staff {
		t.Errorf("Expected Helio Medeiros to be Staff level")
	}
	if costModel.EngineerRates["Santhosh Balakrishnan"].Level != domain.Mid {
		t.Errorf("Expected Santhosh Balakrishnan to be Mid level")
	}
}

func TestCostModelJSONRepository_DirectoryCreation(t *testing.T) {
	tempDir := t.TempDir()
	nonExistentSubDir := filepath.Join(tempDir, "sub", "nested", "dir")
	repo := NewCostModelJSONRepository(nonExistentSubDir)

	ctx := context.Background()
	costModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)

	// Should create the directory structure
	err := repo.SaveCostModel(ctx, "TEST", costModel)
	if err != nil {
		t.Fatalf("Failed to save cost model with directory creation: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(nonExistentSubDir); os.IsNotExist(err) {
		t.Error("Expected directory to be created")
	}
}

func TestCostModelJSONRepository_GetDefaultCostModel_AllRates(t *testing.T) {
	tempDir := t.TempDir()
	repo := NewCostModelJSONRepository(tempDir)

	ctx := context.Background()
	costModel, err := repo.GetDefaultCostModel(ctx)
	if err != nil {
		t.Fatalf("Failed to get default cost model: %v", err)
	}

	// Test all default rates are set
	expectedRates := map[domain.EngineerLevel]float64{
		domain.Junior:    45.0,
		domain.Mid:       60.0,
		domain.Senior:    75.0,
		domain.Staff:     85.0,
		domain.Principal: 95.0,
	}

	for level, expectedRate := range expectedRates {
		if rate, exists := costModel.DefaultRatesByLevel[level]; !exists {
			t.Errorf("Expected default rate for %s to be set", level)
		} else if rate != expectedRate {
			t.Errorf("Expected default rate for %s to be %.1f, got %.1f", level, expectedRate, rate)
		}
	}

	// Test infrastructure costs
	expectedCloudCost := 2000.0
	expectedToolingCost := 500.0
	expectedLicenseCost := 300.0

	if costModel.InfrastructureCosts.CloudCostsPerMonth != expectedCloudCost {
		t.Errorf("Expected cloud cost %.1f, got %.1f", expectedCloudCost, costModel.InfrastructureCosts.CloudCostsPerMonth)
	}
	if costModel.InfrastructureCosts.ToolingCostsPerMonth != expectedToolingCost {
		t.Errorf("Expected tooling cost %.1f, got %.1f", expectedToolingCost, costModel.InfrastructureCosts.ToolingCostsPerMonth)
	}
	if costModel.InfrastructureCosts.LicenseCostsPerMonth != expectedLicenseCost {
		t.Errorf("Expected license cost %.1f, got %.1f", expectedLicenseCost, costModel.InfrastructureCosts.LicenseCostsPerMonth)
	}
}

// Additional tests to reach 90% coverage for all methods

func TestCostModelJSONRepository_SaveCostModel_MarshalError(t *testing.T) {
	tempDir := t.TempDir()
	repo := NewCostModelJSONRepository(tempDir)

	ctx := context.Background()

	// Create a cost model with a structure that would cause JSON marshal issues
	// Since Go's JSON marshaler is robust, we'll test write permission errors instead
	costModel, _ := domain.NewCostModel(domain.EUR, 8.0, 2.0)

	// First save normally to ensure directory exists
	err := repo.SaveCostModel(ctx, "TEST", costModel)
	if err != nil {
		t.Fatalf("Failed to save cost model initially: %v", err)
	}

	// Change directory permissions to read-only to cause write error
	filePath := filepath.Join(tempDir, "cost-model-READONLY.json")

	// Create the file first, then make it read-only
	if err := repo.SaveCostModel(ctx, "READONLY", costModel); err != nil {
		t.Fatalf("Failed to create readonly test file: %v", err)
	}

	// Make file read-only
	if err := os.Chmod(filePath, 0444); err == nil {
		// Try to overwrite - should fail
		err = repo.SaveCostModel(ctx, "READONLY", costModel)
		if err == nil {
			t.Error("Expected error when writing to read-only file")
		}

		// Restore permissions for cleanup
		os.Chmod(filePath, 0644)
	}
}

func TestCostModelJSONRepository_GetDefaultCostModel_ErrorHandling(t *testing.T) {
	tempDir := t.TempDir()
	repo := NewCostModelJSONRepository(tempDir)

	ctx := context.Background()

	// Test with context that might affect operations
	costModel, err := repo.GetDefaultCostModel(ctx)
	if err != nil {
		t.Fatalf("Expected default cost model to always succeed, got: %v", err)
	}

	// Verify all required default values are set correctly
	if costModel.WorkingHoursPerDay != 8.0 {
		t.Errorf("Expected default working hours 8.0, got %.1f", costModel.WorkingHoursPerDay)
	}

	if costModel.OverheadMultiplier != 2.0 {
		t.Errorf("Expected default overhead multiplier 2.0, got %.1f", costModel.OverheadMultiplier)
	}

	// Test that setting rates actually works and doesn't cause errors
	for level, rate := range costModel.DefaultRatesByLevel {
		if rate <= 0 {
			t.Errorf("Expected positive rate for level %s, got %.1f", level, rate)
		}
	}
}

func TestCostModelJSONRepository_CreateFNCostModel_ErrorHandling(t *testing.T) {
	tempDir := t.TempDir()
	repo := NewCostModelJSONRepository(tempDir)

	ctx := context.Background()

	// Test successful creation first
	costModel, err := repo.CreateFNCostModel(ctx)
	if err != nil {
		t.Fatalf("Expected FN cost model creation to succeed, got: %v", err)
	}

	// Verify all FN team engineers are added correctly
	expectedEngineers := []string{
		"Viktor Kovarik", "Ahmed Naser", "Santhosh Balakrishnan", "Thais Hamilton",
		"Cesar Vortmann", "Helio Medeiros", "Talita Roberti", "Georgii Maltsev",
	}

	for _, engineer := range expectedEngineers {
		if rate, exists := costModel.EngineerRates[engineer]; !exists {
			t.Errorf("Expected engineer %s to be in FN cost model", engineer)
		} else {
			if rate.HourlyRate <= 0 {
				t.Errorf("Expected positive rate for %s, got %.1f", engineer, rate.HourlyRate)
			}
			if rate.Team != "FN" {
				t.Errorf("Expected team 'FN' for %s, got %s", engineer, rate.Team)
			}
		}
	}

	// Test with a read-only directory to trigger save error
	readOnlyDir := filepath.Join(tempDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0444); err == nil { // Read-only
		readOnlyRepo := NewCostModelJSONRepository(readOnlyDir)
		_, err = readOnlyRepo.CreateFNCostModel(ctx)
		if err == nil {
			t.Error("Expected error when saving to read-only directory")
		}

		// Restore permissions for cleanup
		os.Chmod(readOnlyDir, 0755)
	}
}
