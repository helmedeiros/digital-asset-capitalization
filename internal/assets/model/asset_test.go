package model

import (
	"testing"
	"time"
)

func TestAsset(t *testing.T) {
	t.Run("should create a valid asset", func(t *testing.T) {
		name := "Frontend Application"
		description := "Main web application for the platform"

		asset, err := NewAsset(name, description)
		if err != nil {
			t.Fatalf("unexpected error creating asset: %v", err)
		}

		if asset == nil {
			t.Fatal("expected asset to be created")
		}

		if asset.Name != name {
			t.Errorf("expected name to be %s, got %s", name, asset.Name)
		}

		if asset.Description != description {
			t.Errorf("expected description to be %s, got %s", description, asset.Description)
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
		asset, err := NewAsset("name", "description")
		if err != nil {
			t.Fatalf("unexpected error creating asset: %v", err)
		}

		createdAt := asset.CreatedAt
		updatedAt := asset.UpdatedAt

		// Wait a bit to ensure timestamps are different
		time.Sleep(time.Millisecond)

		err = asset.UpdateDescription("new description")
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
		asset, err := NewAsset("name", "description")
		if err != nil {
			t.Fatalf("unexpected error creating asset: %v", err)
		}

		lastDocUpdateAt := asset.LastDocUpdateAt

		// Wait a bit to ensure timestamps are different
		time.Sleep(time.Millisecond)

		asset.UpdateDocumentation()

		if asset.LastDocUpdateAt == lastDocUpdateAt {
			t.Error("LastDocUpdateAt should change")
		}
	})
}
