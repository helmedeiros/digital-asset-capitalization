package model

import (
	"testing"
	"time"
)

func TestAsset(t *testing.T) {
	mother := NewAssetMother()

	t.Run("should create a valid asset", func(t *testing.T) {
		asset := mother.CreateValidAsset()

		if asset.Name != "Test Asset" {
			t.Errorf("expected name to be 'Test Asset', got %s", asset.Name)
		}

		if asset.Description != "Test Description" {
			t.Errorf("expected description to be 'Test Description', got %s", asset.Description)
		}

		if asset.ID == "" {
			t.Error("expected ID to be set")
		}

		if asset.CreatedAt.IsZero() {
			t.Error("expected CreatedAt to be set")
		}

		if asset.UpdatedAt.IsZero() {
			t.Error("expected UpdatedAt to be set")
		}

		if asset.LastDocUpdateAt.IsZero() {
			t.Error("expected LastDocUpdateAt to be set")
		}

		if len(asset.ContributionTypes) != 0 {
			t.Error("expected ContributionTypes to be empty")
		}

		if asset.AssociatedTaskCount != 0 {
			t.Error("expected AssociatedTaskCount to be 0")
		}
	})

	t.Run("should not create asset with empty name", func(t *testing.T) {
		asset, err := NewAsset("", "description")
		if err != ErrEmptyName {
			t.Errorf("expected error to be %v, got %v", ErrEmptyName, err)
		}
		if asset != nil {
			t.Error("expected asset to be nil")
		}
	})

	t.Run("should not create asset with empty description", func(t *testing.T) {
		asset, err := NewAsset("name", "")
		if err != ErrEmptyDescription {
			t.Errorf("expected error to be %v, got %v", ErrEmptyDescription, err)
		}
		if asset != nil {
			t.Error("expected asset to be nil")
		}
	})

	t.Run("should track creation and update timestamps", func(t *testing.T) {
		asset := mother.CreateValidAsset()
		createdAt := asset.CreatedAt
		updatedAt := asset.UpdatedAt

		// Wait a bit to ensure timestamps are different
		time.Sleep(time.Millisecond)

		err := asset.UpdateDescription("new description")
		if err != nil {
			t.Fatalf("unexpected error updating description: %v", err)
		}

		if asset.CreatedAt != createdAt {
			t.Error("CreatedAt should not change")
		}

		if asset.UpdatedAt == updatedAt {
			t.Error("UpdatedAt should change")
		}
	})

	t.Run("should track last documentation update", func(t *testing.T) {
		asset := mother.CreateValidAsset()
		lastDocUpdateAt := asset.LastDocUpdateAt

		// Wait a bit to ensure timestamps are different
		time.Sleep(time.Millisecond)

		asset.UpdateDocumentation()

		if asset.LastDocUpdateAt == lastDocUpdateAt {
			t.Error("LastDocUpdateAt should change")
		}
	})

	t.Run("should track contribution types", func(t *testing.T) {
		asset := mother.CreateAssetWithContributionTypes("development", "maintenance")

		if len(asset.ContributionTypes) != 2 {
			t.Errorf("expected 2 contribution types, got %d", len(asset.ContributionTypes))
		}

		if asset.ContributionTypes[0] != "development" {
			t.Errorf("expected first type to be 'development', got %s", asset.ContributionTypes[0])
		}

		if asset.ContributionTypes[1] != "maintenance" {
			t.Errorf("expected second type to be 'maintenance', got %s", asset.ContributionTypes[1])
		}
	})

	t.Run("should track task count", func(t *testing.T) {
		asset := mother.CreateAssetWithTaskCount(3)

		if asset.AssociatedTaskCount != 3 {
			t.Errorf("expected task count to be 3, got %d", asset.AssociatedTaskCount)
		}

		asset.DecrementTaskCount()
		if asset.AssociatedTaskCount != 2 {
			t.Errorf("expected task count to be 2, got %d", asset.AssociatedTaskCount)
		}

		asset.DecrementTaskCount()
		asset.DecrementTaskCount()
		asset.DecrementTaskCount() // Should not go below 0

		if asset.AssociatedTaskCount != 0 {
			t.Errorf("expected task count to be 0, got %d", asset.AssociatedTaskCount)
		}
	})
}
