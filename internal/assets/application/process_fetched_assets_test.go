package application

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
)

func validAsset() *domain.Asset {
	return &domain.Asset{
		ID:          "asset-1",
		Name:        "Search",
		Description: "Search subsystem",
		Status:      "active",
		DocLink:     "https://example.invalid/search",
		LaunchDate:  time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestProcessFetchedAssets_BucketsByValidation(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockAssetRepository)
	// First asset is valid and new.
	good := validAsset()
	mockRepo.On("FindByID", "asset-1").Return(nil, errors.New("not found")).Once()
	mockRepo.On("Save", good).Return(nil).Once()

	// Second asset has no DocLink/Status — should land in NotSyncedAssets.
	bad := &domain.Asset{ID: "asset-2", Name: "Empty", Description: "x"}

	svc := &AssetServiceImpl{repo: mockRepo}
	result, err := svc.processFetchedAssets([]*domain.Asset{good, bad})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.SyncedAssets, 1)
	assert.Equal(t, "Search", result.SyncedAssets[0].Name)
	require.Len(t, result.NotSyncedAssets, 1)
	assert.Equal(t, "Empty", result.NotSyncedAssets[0].Name)
	assert.Contains(t, result.NotSyncedAssets[0].MissingFields, "Status")
	assert.Contains(t, result.NotSyncedAssets[0].MissingFields, "DocLink")
	mockRepo.AssertExpectations(t)
}

func TestProcessFetchedAssets_PreservesLocalMetadata(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockAssetRepository)
	fresh := validAsset() // Keywords/teams empty, AssociatedTaskCount=0.

	existing := validAsset()
	existing.Keywords = []string{"search", "indexer"}
	existing.OwningTeam = "Voyager"
	existing.ContributingTeams = []string{"Catalog"}
	existing.AssociatedTaskCount = 12

	mockRepo.On("FindByID", "asset-1").Return(existing, nil).Once()
	// The saved asset is the same pointer as `fresh`, mutated by merge.
	mockRepo.On("Save", fresh).Return(nil).Once()

	svc := &AssetServiceImpl{repo: mockRepo}
	_, err := svc.processFetchedAssets([]*domain.Asset{fresh})
	require.NoError(t, err)

	assert.Equal(t, []string{"search", "indexer"}, fresh.Keywords)
	assert.Equal(t, "Voyager", fresh.OwningTeam)
	assert.Equal(t, []string{"Catalog"}, fresh.ContributingTeams)
	assert.Equal(t, 12, fresh.AssociatedTaskCount)
	mockRepo.AssertExpectations(t)
}

func TestProcessFetchedAssets_DoesNotOverwriteFreshMetadata(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockAssetRepository)
	fresh := validAsset()
	fresh.Keywords = []string{"fresh"}
	fresh.OwningTeam = "FreshTeam"
	fresh.ContributingTeams = []string{"Other"}
	fresh.AssociatedTaskCount = 3

	existing := validAsset()
	existing.Keywords = []string{"stale"}
	existing.OwningTeam = "StaleTeam"
	existing.ContributingTeams = []string{"Stale"}
	existing.AssociatedTaskCount = 99

	mockRepo.On("FindByID", "asset-1").Return(existing, nil).Once()
	mockRepo.On("Save", fresh).Return(nil).Once()

	svc := &AssetServiceImpl{repo: mockRepo}
	_, err := svc.processFetchedAssets([]*domain.Asset{fresh})
	require.NoError(t, err)

	assert.Equal(t, []string{"fresh"}, fresh.Keywords)
	assert.Equal(t, "FreshTeam", fresh.OwningTeam)
	assert.Equal(t, []string{"Other"}, fresh.ContributingTeams)
	// AssociatedTaskCount is preserved from existing only when existing > 0;
	// for this branch the existing.AssociatedTaskCount (99) IS preserved
	// because mergeLocalMetadata unconditionally overwrites when existing > 0.
	assert.Equal(t, 99, fresh.AssociatedTaskCount)
	mockRepo.AssertExpectations(t)
}

func TestProcessFetchedAssets_SaveErrorWrapsAssetName(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockAssetRepository)
	fresh := validAsset()
	mockRepo.On("FindByID", "asset-1").Return(nil, errors.New("not found")).Once()
	mockRepo.On("Save", fresh).Return(errors.New("disk full")).Once()

	svc := &AssetServiceImpl{repo: mockRepo}
	_, err := svc.processFetchedAssets([]*domain.Asset{fresh})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save asset Search")
	mockRepo.AssertExpectations(t)
}

func TestNotSyncedFromAsset_CapturesAvailableFields(t *testing.T) {
	t.Parallel()
	asset := &domain.Asset{
		ID:          "id-1",
		Name:        "X",
		Description: "desc",
		Status:      "active",
		DocLink:     "https://e/d",
		Why:         "why",
		Benefits:    "benefits",
		How:         "how",
		Metrics:     "metrics",
		LaunchDate:  time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	}
	got := notSyncedFromAsset(asset, []string{"Foo"})
	assert.Equal(t, "X", got.Name)
	assert.Equal(t, []string{"Foo"}, got.MissingFields)
	assert.Equal(t, "id-1", got.AvailableFields["ID"])
	assert.Equal(t, "2026-01-02", got.AvailableFields["LaunchDate"])
	assert.Equal(t, "why", got.AvailableFields["Why"])
}

func TestMergeLocalMetadata_NilOrEmptyFieldsOnExistingAreIgnored(t *testing.T) {
	t.Parallel()
	fresh := validAsset()
	fresh.Keywords = []string{"kept"}
	existing := validAsset() // No keywords/teams on existing either.

	mergeLocalMetadata(fresh, existing)

	// Nothing to merge — fresh stays intact, no panic.
	assert.Equal(t, []string{"kept"}, fresh.Keywords)
	assert.Equal(t, "", fresh.OwningTeam)
	assert.Empty(t, fresh.ContributingTeams)
	assert.Equal(t, 0, fresh.AssociatedTaskCount)
}
