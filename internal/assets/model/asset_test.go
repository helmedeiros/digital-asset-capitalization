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

		if len(asset.ID) != 16 {
			t.Errorf("expected ID to be 16 characters long, got %d", len(asset.ID))
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

		if asset.Version != 1 {
			t.Errorf("expected Version to be 1, got %d", asset.Version)
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
		version := asset.Version

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

		if asset.Version != version+1 {
			t.Errorf("expected Version to be %d, got %d", version+1, asset.Version)
		}
	})

	t.Run("should track last documentation update", func(t *testing.T) {
		asset := mother.CreateValidAsset()
		lastDocUpdateAt := asset.LastDocUpdateAt
		version := asset.Version

		// Wait a bit to ensure timestamps are different
		time.Sleep(time.Millisecond)

		asset.UpdateDocumentation()

		if asset.LastDocUpdateAt == lastDocUpdateAt {
			t.Error("LastDocUpdateAt should change")
		}

		if asset.Version != version+1 {
			t.Errorf("expected Version to be %d, got %d", version+1, asset.Version)
		}
	})

	t.Run("should validate contribution types", func(t *testing.T) {
		asset := mother.CreateValidAsset()

		tests := []struct {
			name             string
			contributionType string
			expectedError    error
		}{
			{
				name:             "valid contribution type",
				contributionType: "development",
				expectedError:    nil,
			},
			{
				name:             "empty contribution type",
				contributionType: "",
				expectedError:    ErrEmptyContributionType,
			},
			{
				name:             "invalid contribution type",
				contributionType: "invalid",
				expectedError:    ErrInvalidContributionType,
			},
			{
				name:             "duplicate contribution type",
				contributionType: "development",
				expectedError:    ErrDuplicateContributionType,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := asset.AddContributionType(tt.contributionType)
				if err != tt.expectedError {
					t.Errorf("expected error to be %v, got %v", tt.expectedError, err)
				}
				if err == nil && len(asset.ContributionTypes) == 0 {
					t.Error("expected contribution type to be added")
				}
			})
		}
	})

	t.Run("should track task count and version", func(t *testing.T) {
		asset := mother.CreateAssetWithTaskCount(3)
		version := asset.Version

		if asset.AssociatedTaskCount != 3 {
			t.Errorf("expected task count to be 3, got %d", asset.AssociatedTaskCount)
		}

		asset.DecrementTaskCount()
		if asset.AssociatedTaskCount != 2 {
			t.Errorf("expected task count to be 2, got %d", asset.AssociatedTaskCount)
		}
		if asset.Version != version+1 {
			t.Errorf("expected Version to be %d, got %d", version+1, asset.Version)
		}

		asset.DecrementTaskCount()
		asset.DecrementTaskCount()
		asset.DecrementTaskCount() // Should not go below 0

		if asset.AssociatedTaskCount != 0 {
			t.Errorf("expected task count to be 0, got %d", asset.AssociatedTaskCount)
		}
		if asset.Version != version+3 {
			t.Errorf("expected Version to be %d, got %d", version+3, asset.Version)
		}
	})

	t.Run("should generate unique IDs", func(t *testing.T) {
		asset1 := mother.CreateValidAsset()
		asset2 := mother.CreateValidAsset()

		if asset1.ID == asset2.ID {
			t.Error("expected different IDs for different assets")
		}
	})
}
