package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAsset(t *testing.T) {
	mother := NewAssetMother()

	t.Run("should create a valid asset", func(t *testing.T) {
		asset, err := NewAssetWithDetails("Test Asset", "Test Description", "Test Why", "Test Benefits", "Test How", "Test Metrics")
		require.NoError(t, err)

		assert.Equal(t, "Test Asset", asset.Name, "expected name to be 'Test Asset'")
		assert.Equal(t, "Test Description", asset.Description, "expected description to be 'Test Description'")
		assert.Equal(t, "Test Why", asset.Why, "expected why to be 'Test Why'")
		assert.Equal(t, "Test Benefits", asset.Benefits, "expected benefits to be 'Test Benefits'")
		assert.Equal(t, "Test How", asset.How, "expected how to be 'Test How'")
		assert.Equal(t, "Test Metrics", asset.Metrics, "expected metrics to be 'Test Metrics'")
		assert.NotEmpty(t, asset.ID, "expected ID to be set")
		assert.Len(t, asset.ID, 16, "expected ID to be 16 characters long")
		assert.False(t, asset.CreatedAt.IsZero(), "expected CreatedAt to be set")
		assert.False(t, asset.UpdatedAt.IsZero(), "expected UpdatedAt to be set")
		assert.False(t, asset.LastDocUpdateAt.IsZero(), "expected LastDocUpdateAt to be set")
		assert.Empty(t, asset.ContributionTypes, "expected ContributionTypes to be empty")
		assert.Equal(t, 0, asset.AssociatedTaskCount, "expected AssociatedTaskCount to be 0")
		assert.Equal(t, 1, asset.Version, "expected Version to be 1")
	})

	t.Run("should not create asset with empty name", func(t *testing.T) {
		asset, err := NewAsset("", "description")
		assert.ErrorIs(t, err, ErrEmptyName, "expected error to be ErrEmptyName")
		assert.Nil(t, asset, "expected asset to be nil")
	})

	t.Run("should not create asset with empty description", func(t *testing.T) {
		asset, err := NewAsset("name", "")
		assert.ErrorIs(t, err, ErrEmptyDescription, "expected error to be ErrEmptyDescription")
		assert.Nil(t, asset, "expected asset to be nil")
	})

	t.Run("should track creation and update timestamps", func(t *testing.T) {
		asset := mother.CreateValidAsset()
		createdAt := asset.CreatedAt
		updatedAt := asset.UpdatedAt
		version := asset.Version

		// Wait a bit to ensure timestamps are different
		time.Sleep(time.Millisecond)

		err := asset.UpdateDescription("new description")
		require.NoError(t, err, "unexpected error updating description")

		assert.Equal(t, createdAt, asset.CreatedAt, "CreatedAt should not change")
		assert.NotEqual(t, updatedAt, asset.UpdatedAt, "UpdatedAt should change")
		assert.Equal(t, version+1, asset.Version, "Version should increment")
	})

	t.Run("should track last documentation update", func(t *testing.T) {
		asset := mother.CreateValidAsset()
		lastDocUpdateAt := asset.LastDocUpdateAt
		version := asset.Version

		// Wait a bit to ensure timestamps are different
		time.Sleep(time.Millisecond)

		asset.UpdateDocumentation()

		assert.NotEqual(t, lastDocUpdateAt, asset.LastDocUpdateAt, "LastDocUpdateAt should change")
		assert.Equal(t, version+1, asset.Version, "Version should increment")
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
				assert.ErrorIs(t, err, tt.expectedError, "unexpected error")
				if err == nil {
					assert.NotEmpty(t, asset.ContributionTypes, "expected contribution type to be added")
				}
			})
		}
	})

	t.Run("should track task count and version", func(t *testing.T) {
		asset := mother.CreateAssetWithTaskCount(3)
		version := asset.Version

		assert.Equal(t, 3, asset.AssociatedTaskCount, "expected task count to be 3")

		asset.DecrementTaskCount()
		assert.Equal(t, 2, asset.AssociatedTaskCount, "expected task count to be 2")
		assert.Equal(t, version+1, asset.Version, "Version should increment")

		asset.DecrementTaskCount()
		asset.DecrementTaskCount()
		asset.DecrementTaskCount() // Should not go below 0

		assert.Equal(t, 0, asset.AssociatedTaskCount, "expected task count to be 0")
		assert.Equal(t, version+3, asset.Version, "Version should increment")
	})

	t.Run("should generate unique IDs", func(t *testing.T) {
		asset1 := mother.CreateValidAsset()
		asset2 := mother.CreateValidAsset()

		assert.NotEqual(t, asset1.ID, asset2.ID, "expected different IDs for different assets")
	})
}
