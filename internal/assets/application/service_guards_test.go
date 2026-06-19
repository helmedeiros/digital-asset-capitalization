package application

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/assets/infrastructure/id"
)

// Required-field guards on the team-management methods. The existing
// service_test.go covers the happy path + repository errors; these
// pin the early-return guards plus the domain-level error wraps that
// fire when the team name is non-empty but trims to empty.

func TestAssetServiceImpl_GetAssetTeamInfo_EmptyAssetName(t *testing.T) {
	t.Parallel()
	svc := NewAssetServiceLegacy(new(MockAssetRepository), id.NewHashIDGenerator())
	_, err := svc.GetAssetTeamInfo("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "asset name is required")
}

func TestAssetServiceImpl_AddContributingTeam_EmptyAssetName(t *testing.T) {
	t.Parallel()
	svc := NewAssetServiceLegacy(new(MockAssetRepository), id.NewHashIDGenerator())
	err := svc.AddContributingTeam("", "team-beta")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "asset name is required")
}

func TestAssetServiceImpl_AddContributingTeam_EmptyTeamName(t *testing.T) {
	t.Parallel()
	svc := NewAssetServiceLegacy(new(MockAssetRepository), id.NewHashIDGenerator())
	err := svc.AddContributingTeam("asset-1", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "team name is required")
}

func TestAssetServiceImpl_AddContributingTeam_DomainErrorWraps(t *testing.T) {
	t.Parallel()
	// Service-level guard treats teamName == "" as a missing flag, but
	// passes whitespace through; domain.AddContributingTeam trims and
	// rejects the empty result. This pins the wrap branch around the
	// domain call.
	mockRepo := new(MockAssetRepository)
	asset := &domain.Asset{ID: "id", Name: "asset-1"}
	mockRepo.On("FindByName", "asset-1").Return(asset, nil)

	svc := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())
	err := svc.AddContributingTeam("asset-1", "   ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add contributing team")
	mockRepo.AssertExpectations(t)
}

func TestAssetServiceImpl_RemoveContributingTeam_EmptyAssetName(t *testing.T) {
	t.Parallel()
	svc := NewAssetServiceLegacy(new(MockAssetRepository), id.NewHashIDGenerator())
	err := svc.RemoveContributingTeam("", "team-beta")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "asset name is required")
}

func TestAssetServiceImpl_RemoveContributingTeam_EmptyTeamName(t *testing.T) {
	t.Parallel()
	svc := NewAssetServiceLegacy(new(MockAssetRepository), id.NewHashIDGenerator())
	err := svc.RemoveContributingTeam("asset-1", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "team name is required")
}

func TestAssetServiceImpl_RemoveContributingTeam_DomainErrorWraps(t *testing.T) {
	t.Parallel()
	// domain.RemoveContributingTeam returns an error when the team
	// isn't present in the contributing list — exercises the wrap.
	mockRepo := new(MockAssetRepository)
	asset := &domain.Asset{ID: "id", Name: "asset-1"}
	mockRepo.On("FindByName", "asset-1").Return(asset, nil)

	svc := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())
	err := svc.RemoveContributingTeam("asset-1", "team-not-present")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove contributing team")
	mockRepo.AssertExpectations(t)
}

func TestAssetServiceImpl_RemoveContributingTeam_SaveErrorWraps(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockAssetRepository)
	asset := &domain.Asset{ID: "id", Name: "asset-1"}
	require.NoError(t, asset.AddContributingTeam("team-beta"))
	mockRepo.On("FindByName", "asset-1").Return(asset, nil)
	mockRepo.On("Save", asset).Return(errors.New("disk full"))

	svc := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())
	err := svc.RemoveContributingTeam("asset-1", "team-beta")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to save asset")
	mockRepo.AssertExpectations(t)
}

// validateRequiredFields branches not currently covered: Name empty
// and Description empty. The existing tests all use assets with both
// of these populated.

func TestValidateRequiredFields_ReportsEveryMissingField(t *testing.T) {
	t.Parallel()
	// Asset with EVERYTHING missing (Name + Description + ID + Status +
	// DocLink). Confirms every branch is reached and order matches the
	// function's declared order.
	asset := &domain.Asset{}
	missing := validateRequiredFields(asset)
	assert.Equal(t, []string{"Name", "Description", "ID", "Status", "DocLink"}, missing)
}

func TestValidateRequiredFields_NoMissingWhenAllSet(t *testing.T) {
	t.Parallel()
	asset := &domain.Asset{
		ID:          "id-1",
		Name:        "Asset One",
		Description: "Desc",
		Status:      "Operational",
		DocLink:     "https://confluence.example.com/pages/123",
	}
	assert.Empty(t, validateRequiredFields(asset))
}

// extractPageIDFromDocLink. The function returns "" on URL parse
// failure OR when the URL has no /pages/{id}/ segment; the existing
// tests cover only the happy path.

func TestExtractPageIDFromDocLink_NoPagesSegmentReturnsEmpty(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", extractPageIDFromDocLink("https://example.com/no-pages-here"))
	assert.Equal(t, "", extractPageIDFromDocLink("https://example.com/"))
	assert.Equal(t, "", extractPageIDFromDocLink(""))
}

func TestExtractPageIDFromDocLink_PagesSegmentWithoutID(t *testing.T) {
	t.Parallel()
	// /pages with NO following segment falls into the "no id" path.
	assert.Equal(t, "", extractPageIDFromDocLink("https://example.com/pages"))
	assert.Equal(t, "", extractPageIDFromDocLink("https://example.com/space/pages"))
}

func TestExtractPageIDFromDocLink_HappyPath(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "12345", extractPageIDFromDocLink("https://example.com/spaces/X/pages/12345/Title"))
	assert.Equal(t, "67890", extractPageIDFromDocLink("https://wiki.example.com/pages/67890"))
}

// AssignTeam — the existing happy + asset-not-found + save tests
// cover most of the function but not the contributing-teams-empty
// path (which skips the SetContributingTeams call). Also pin the
// "owningTeam empty" + "contributingTeams empty" combined no-op.

func TestAssetServiceImpl_AssignTeam_OnlyOwningTeamSet(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockAssetRepository)
	asset := &domain.Asset{ID: "id", Name: "asset-1"}
	mockRepo.On("FindByName", "asset-1").Return(asset, nil)
	mockRepo.On("Save", asset).Return(nil)

	svc := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())
	err := svc.AssignTeam("asset-1", "team-alpha", nil)
	require.NoError(t, err)
	assert.Equal(t, "team-alpha", asset.GetOwningTeam())
	assert.Empty(t, asset.GetContributingTeams())
	mockRepo.AssertExpectations(t)
}

func TestAssetServiceImpl_AssignTeam_OnlyContributingTeamsSet(t *testing.T) {
	t.Parallel()
	mockRepo := new(MockAssetRepository)
	asset := &domain.Asset{ID: "id", Name: "asset-1"}
	mockRepo.On("FindByName", "asset-1").Return(asset, nil)
	mockRepo.On("Save", asset).Return(nil)

	svc := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())
	err := svc.AssignTeam("asset-1", "", []string{"team-beta", "team-gamma"})
	require.NoError(t, err)
	assert.Equal(t, "", asset.GetOwningTeam())
	assert.Equal(t, []string{"team-beta", "team-gamma"}, asset.GetContributingTeams())
	mockRepo.AssertExpectations(t)
}

func TestAssetServiceImpl_AssignTeam_NothingToAssignStillSaves(t *testing.T) {
	t.Parallel()
	// Both inputs empty — function still calls Save (the early-skip
	// only applies to the individual setter calls, not the save).
	mockRepo := new(MockAssetRepository)
	asset := &domain.Asset{ID: "id", Name: "asset-1"}
	mockRepo.On("FindByName", "asset-1").Return(asset, nil)
	mockRepo.On("Save", asset).Return(nil)

	svc := NewAssetServiceLegacy(mockRepo, id.NewHashIDGenerator())
	err := svc.AssignTeam("asset-1", "", nil)
	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
