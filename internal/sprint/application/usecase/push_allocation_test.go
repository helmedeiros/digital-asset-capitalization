package usecase

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
)

type mockJiraPortForPush struct {
	fetchResults map[string]*ports.CustomFieldValues
	fetchErrors  map[string]error
	updateCalls  []updateCall
	updateError  error
}

type updateCall struct {
	IssueKey string
	Update   ports.CustomFieldUpdate
}

func (m *mockJiraPortForPush) UpdateCustomFields(issueKey string, update ports.CustomFieldUpdate) error {
	m.updateCalls = append(m.updateCalls, updateCall{IssueKey: issueKey, Update: update})
	return m.updateError
}

func (m *mockJiraPortForPush) FetchCustomFields(issueKey string) (*ports.CustomFieldValues, error) {
	if e, ok := m.fetchErrors[issueKey]; ok && e != nil {
		return nil, e
	}
	if v, ok := m.fetchResults[issueKey]; ok {
		return v, nil
	}
	return &ports.CustomFieldValues{}, nil
}

func (m *mockJiraPortForPush) GetSprintsForProject(_ string, _ []string) ([]ports.Sprint, error) {
	return nil, nil
}
func (m *mockJiraPortForPush) GetSprintsForProjectWithBoardInfo(_ string, _ []string) ([]ports.Sprint, []ports.BoardInfo, error) {
	return nil, nil, nil
}
func (m *mockJiraPortForPush) GetIssuesForSprint(_, _ string) ([]ports.JiraIssue, error) {
	return nil, nil
}
func (m *mockJiraPortForPush) GetIssuesForTeamMember(_ string) ([]ports.JiraIssue, error) {
	return nil, nil
}
func (m *mockJiraPortForPush) GetSprintIssues(_ *domain.Sprint) ([]ports.JiraIssue, error) {
	return nil, nil
}
func (m *mockJiraPortForPush) GetTeamIssues(_ *domain.Team) ([]ports.JiraIssue, error) {
	return nil, nil
}
func (m *mockJiraPortForPush) GetSprintByName(_, _ string) (*ports.Sprint, error) {
	return nil, nil
}
func (m *mockJiraPortForPush) GetIssuesForSprintOnBoard(_, _ string, _ int) ([]ports.JiraIssue, error) {
	return nil, nil
}

func floatPtr(v float64) *float64 {
	return &v
}

func TestPushAllocationUseCase_EmptyFieldsGetUpdated(t *testing.T) {
	mock := &mockJiraPortForPush{
		fetchResults: map[string]*ports.CustomFieldValues{
			"COP-1": {},
		},
	}

	uc := NewPushAllocationUseCase(mock, false)

	hours := 12.5
	records := []AllocationRecord{
		{
			IssueKey:         "COP-1",
			EngineeringHours: &hours,
			WorkStream:       "Product",
			TPDBusinessUnit:  "B2C",
		},
	}

	result, err := uc.Execute(records)
	require.NoError(t, err)
	assert.Equal(t, 3, result.UpdatedCount)
	assert.Equal(t, 0, result.SkippedCount)
	assert.Len(t, mock.updateCalls, 1)
	assert.Equal(t, "COP-1", mock.updateCalls[0].IssueKey)
	assert.NotNil(t, mock.updateCalls[0].Update.EngineeringHours)
	assert.Equal(t, 12.5, *mock.updateCalls[0].Update.EngineeringHours)
	assert.NotNil(t, mock.updateCalls[0].Update.WorkStream)
	assert.Equal(t, "Product", *mock.updateCalls[0].Update.WorkStream)
	assert.Equal(t, []string{"B2C"}, mock.updateCalls[0].Update.TPDBusinessUnits)
}

func TestPushAllocationUseCase_NonEmptyFieldsSkipped(t *testing.T) {
	mock := &mockJiraPortForPush{
		fetchResults: map[string]*ports.CustomFieldValues{
			"COP-2": {
				EngineeringHours: floatPtr(8.0),
				WorkStream:       "Operational",
				TPDBusinessUnits: []string{"B2B"},
			},
		},
	}

	uc := NewPushAllocationUseCase(mock, false)

	hours := 12.5
	records := []AllocationRecord{
		{
			IssueKey:         "COP-2",
			EngineeringHours: &hours,
			WorkStream:       "Product",
			TPDBusinessUnit:  "B2C",
		},
	}

	result, err := uc.Execute(records)
	require.NoError(t, err)
	assert.Equal(t, 0, result.UpdatedCount)
	assert.Equal(t, 3, result.SkippedCount)
	assert.Len(t, mock.updateCalls, 0)
}

func TestPushAllocationUseCase_DryRunDoesNotCallUpdate(t *testing.T) {
	mock := &mockJiraPortForPush{
		fetchResults: map[string]*ports.CustomFieldValues{
			"COP-3": {},
		},
	}

	uc := NewPushAllocationUseCase(mock, true)

	hours := 5.0
	records := []AllocationRecord{
		{
			IssueKey:         "COP-3",
			EngineeringHours: &hours,
			WorkStream:       "Product",
		},
	}

	result, err := uc.Execute(records)
	require.NoError(t, err)
	assert.Equal(t, 2, result.UpdatedCount)
	assert.Len(t, mock.updateCalls, 0)

	for _, d := range result.Details {
		if d.Status != "skipped" {
			assert.Equal(t, "will update", d.Status)
		}
	}
}

func TestPushAllocationUseCase_FetchErrorDoesNotStopProcessing(t *testing.T) {
	mock := &mockJiraPortForPush{
		fetchResults: map[string]*ports.CustomFieldValues{
			"COP-5": {},
		},
		fetchErrors: map[string]error{
			"COP-4": fmt.Errorf("network error"),
		},
	}

	uc := NewPushAllocationUseCase(mock, false)

	hours := 5.0
	records := []AllocationRecord{
		{IssueKey: "COP-4", EngineeringHours: &hours},
		{IssueKey: "COP-5", EngineeringHours: &hours},
	}

	result, err := uc.Execute(records)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ErrorCount)
	assert.Equal(t, 1, result.UpdatedCount)
	assert.Len(t, mock.updateCalls, 1)
	assert.Equal(t, "COP-5", mock.updateCalls[0].IssueKey)
}

func TestPushAllocationUseCase_UpdateErrorMarksFieldsAsError(t *testing.T) {
	mock := &mockJiraPortForPush{
		fetchResults: map[string]*ports.CustomFieldValues{
			"COP-6": {},
		},
		updateError: fmt.Errorf("permission denied"),
	}

	uc := NewPushAllocationUseCase(mock, false)

	hours := 10.0
	records := []AllocationRecord{
		{IssueKey: "COP-6", EngineeringHours: &hours, WorkStream: "Product"},
	}

	result, err := uc.Execute(records)
	require.NoError(t, err)
	assert.Equal(t, 0, result.UpdatedCount)
	assert.Equal(t, 2, result.ErrorCount)

	for _, d := range result.Details {
		if d.IssueKey == "COP-6" && d.Field != "all" {
			assert.Equal(t, "error", d.Status)
			assert.Contains(t, d.Reason, "permission denied")
		}
	}
}

func TestPushAllocationUseCase_PartialUpdate(t *testing.T) {
	mock := &mockJiraPortForPush{
		fetchResults: map[string]*ports.CustomFieldValues{
			"COP-7": {
				EngineeringHours: floatPtr(5.0),
				WorkStream:       "",
			},
		},
	}

	uc := NewPushAllocationUseCase(mock, false)

	hours := 10.0
	records := []AllocationRecord{
		{IssueKey: "COP-7", EngineeringHours: &hours, WorkStream: "Product"},
	}

	result, err := uc.Execute(records)
	require.NoError(t, err)
	assert.Equal(t, 1, result.UpdatedCount)
	assert.Equal(t, 1, result.SkippedCount)
	assert.Len(t, mock.updateCalls, 1)
	assert.Nil(t, mock.updateCalls[0].Update.EngineeringHours)
	assert.NotNil(t, mock.updateCalls[0].Update.WorkStream)
	assert.Equal(t, "Product", *mock.updateCalls[0].Update.WorkStream)
}
