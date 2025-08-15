package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Simple validation tests for adapter methods
func TestAdapterMethodValidations(t *testing.T) {
	ctx := context.Background()

	t.Run("TaskServiceAdapter validations", func(t *testing.T) {
		adapter := &TaskServiceAdapter{service: nil}

		// Test FetchTasks validation
		_, err := adapter.FetchTasks(ctx, "", "Sprint1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")

		_, err = adapter.FetchTasks(ctx, "PROJ", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sprint is required")

		// Test ShowTasks validation
		_, err = adapter.ShowTasks(ctx, "", "Sprint1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")

		_, err = adapter.ShowTasks(ctx, "PROJ", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sprint is required")

		// Test ClassifyTasks validation
		_, err = adapter.ClassifyTasks(ctx, "", "Sprint1", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")

		_, err = adapter.ClassifyTasks(ctx, "PROJ", "", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sprint is required")

		// Test InspectTask validation
		_, err = adapter.InspectTask(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task key is required")
	})

	t.Run("SprintServiceAdapter validations", func(t *testing.T) {
		adapter := &SprintServiceAdapter{service: nil}

		// Test ListSprints validation - only project validation works since period gets default
		_, err := adapter.ListSprints(ctx, "", "Q1 2025")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")

		// Test AllocateSprint validation
		_, err = adapter.AllocateSprint(ctx, "", "Sprint1", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")

		_, err = adapter.AllocateSprint(ctx, "PROJ", "", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sprint is required")
	})

	t.Run("InvestmentServiceAdapter validations", func(t *testing.T) {
		adapter := &InvestmentServiceAdapter{service: nil}

		// Test CalculateInvestment validation
		_, err := adapter.CalculateInvestment(ctx, "", "PROJ", []string{"Sprint1"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "asset name is required")

		_, err = adapter.CalculateInvestment(ctx, "Asset1", "", []string{"Sprint1"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")

		_, err = adapter.CalculateInvestment(ctx, "Asset1", "PROJ", []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one sprint is required")

		// Test ListInvestments validation
		_, err = adapter.ListInvestments(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")

		// Test ShowRates validation
		_, err = adapter.ShowRates(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})
}
