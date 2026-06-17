package usecase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/config/domain"
)

func TestSyncTeamFromJira_Execute_InvalidProjectTeamDataAddsValidationError(t *testing.T) {
	t.Parallel()
	mockTeamSyncPort := &MockTeamSyncPort{}
	mockConfigRepo := &MockConfigurationRepository{}

	existingConfig, _ := domain.NewTeamConfig(make(map[string][]string))
	mockConfigRepo.On("LoadTeamConfig").Return(existingConfig, nil)

	// ProjectTeamData returned by the port is invalid: empty
	// ProjectKey makes Validate() return an error, exercising the
	// previously-untested validation-error branch.
	invalid := &domain.ProjectTeamData{
		ProjectKey: "",
		Members: []domain.TeamMember{
			{Name: "Alice", DisplayName: "Alice"},
		},
	}
	mockTeamSyncPort.On("GetProjectMembers", "TEST").Return(invalid, nil)

	useCase := NewSyncTeamFromJira(mockTeamSyncPort, mockConfigRepo)
	result, err := useCase.Execute("TEST")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.HasErrors())
	assert.Equal(t, "validation", result.Errors[0].Type)
	assert.Contains(t, result.Errors[0].Message, "Invalid team data")
}

func TestSyncTeamFromJira_Execute_TimelineMaintenanceAddsAndRemovesMembers(t *testing.T) {
	t.Parallel()
	mockTeamSyncPort := &MockTeamSyncPort{}
	mockConfigRepo := &MockConfigurationRepository{}

	// Build a config that already has a TEST team timeline with one
	// member ("Old Member"). This drives the HasTeamTimeline=true
	// branch in Execute that wasn't being exercised before.
	existingConfig, _ := domain.NewTeamConfig(map[string][]string{
		"TEST": {"Old Member"},
	})
	joined := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, existingConfig.AddMemberWithDates("TEST", "Old Member", joined))

	mockConfigRepo.On("LoadTeamConfig").Return(existingConfig, nil)
	mockConfigRepo.On("SaveTeamConfig", mock.AnythingOfType("*domain.TeamConfig")).Return(nil)

	// New roster: removes "Old Member", adds "New Member".
	projectTeamData := &domain.ProjectTeamData{
		ProjectKey: "TEST",
		Members: []domain.TeamMember{
			{Name: "New Member", DisplayName: "New Member"},
		},
	}
	mockTeamSyncPort.On("GetProjectMembers", "TEST").Return(projectTeamData, nil)

	useCase := NewSyncTeamFromJira(mockTeamSyncPort, mockConfigRepo)
	result, err := useCase.Execute("TEST")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.HasErrors())
	assert.Contains(t, result.AddedMembers, "New Member")
	assert.Contains(t, result.RemovedMembers, "Old Member")

	// Both branches inside the HasTeamTimeline block (AddMemberWithDates
	// for added, SetMemberLeft for removed) ran.
	mockTeamSyncPort.AssertExpectations(t)
	mockConfigRepo.AssertExpectations(t)
}
