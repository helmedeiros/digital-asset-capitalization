package usecase

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
)

// MockJiraPort for testing
type MockJiraPort struct {
	sprints []ports.Sprint
	err     error
}

func (m *MockJiraPort) GetSprintsForProject(_ string, _ []string) ([]ports.Sprint, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.sprints, nil
}

func (m *MockJiraPort) GetSprintsForProjectWithBoardInfo(_ string, _ []string) ([]ports.Sprint, []ports.BoardInfo, error) {
	if m.err != nil {
		return nil, nil, m.err
	}
	// Return empty board info for tests
	return m.sprints, []ports.BoardInfo{}, nil
}

func (m *MockJiraPort) GetIssuesForSprint(_, _ string) ([]ports.JiraIssue, error) {
	return nil, nil
}

func (m *MockJiraPort) GetIssuesForTeamMember(_ string) ([]ports.JiraIssue, error) {
	return nil, nil
}

func (m *MockJiraPort) GetSprintIssues(_ *domain.Sprint) ([]ports.JiraIssue, error) {
	return nil, nil
}

func (m *MockJiraPort) GetTeamIssues(_ *domain.Team) ([]ports.JiraIssue, error) {
	return nil, nil
}

func (m *MockJiraPort) GetSprintByName(_ string, _ string) (*ports.Sprint, error) {
	return nil, nil
}

func (m *MockJiraPort) GetIssuesForSprintOnBoard(_, _ string, _ int) ([]ports.JiraIssue, error) {
	return nil, nil
}

func (m *MockJiraPort) UpdateCustomFields(_ string, _ ports.CustomFieldUpdate) error {
	return nil
}

func (m *MockJiraPort) FetchCustomFields(_ string) (*ports.CustomFieldValues, error) {
	return nil, nil
}

func TestListSprintsUseCase_Execute(t *testing.T) {
	t.Parallel()
	t.Run("should list sprints for a project in Q2 2025", func(t *testing.T) {
		// Given
		mockJira := &MockJiraPort{
			sprints: []ports.Sprint{
				{
					ID:        "8802",
					Name:      "Spiderman",
					State:     "active",
					StartDate: "2025-06-25T10:18:22.960Z",
					EndDate:   "2025-07-09T09:30:00.000Z",
					Goal:      "Go live with baggage insurance",
				},
				{
					ID:        "9230",
					Name:      "TBD 9",
					State:     "future",
					StartDate: "2025-07-09T10:27:04.000Z",
					EndDate:   "2025-07-23T09:30:00.000Z",
					Goal:      "",
				},
			},
		}

		useCase := NewListSprintsUseCase(mockJira)

		// When
		result, err := useCase.Execute("FN", "Q2 2025")

		// Then
		require.NoError(t, err)
		assert.Len(t, result.Sprints, 1)
		assert.Equal(t, "Spiderman", result.Sprints[0].Name)
		assert.Equal(t, "active", result.Sprints[0].State)
	})

	t.Run("should return error when jira port fails", func(t *testing.T) {
		// Given
		mockJira := &MockJiraPort{
			err: assert.AnError,
		}

		useCase := NewListSprintsUseCase(mockJira)

		// When
		result, err := useCase.Execute("FN", "Q2 2025")

		// Then
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to fetch sprints")
	})

	t.Run("should filter sprints by date range", func(t *testing.T) {
		// Given
		mockJira := &MockJiraPort{
			sprints: []ports.Sprint{
				{
					ID:        "8802",
					Name:      "Spiderman",
					State:     "active",
					StartDate: "2025-06-25T10:18:22.960Z",
					EndDate:   "2025-07-09T09:30:00.000Z",
					Goal:      "Go live with baggage insurance",
				},
				{
					ID:        "9230",
					Name:      "TBD 9",
					State:     "future",
					StartDate: "2025-07-09T10:27:04.000Z",
					EndDate:   "2025-07-23T09:30:00.000Z",
					Goal:      "",
				},
				{
					ID:        "9999",
					Name:      "Old Sprint",
					State:     "closed",
					StartDate: "2025-01-01T00:00:00.000Z",
					EndDate:   "2025-01-15T00:00:00.000Z",
					Goal:      "Old goal",
				},
			},
		}

		useCase := NewListSprintsUseCase(mockJira)

		// When
		result, err := useCase.Execute("FN", "Q2 2025")

		// Then
		require.NoError(t, err)
		assert.Len(t, result.Sprints, 1) // Should only include Q2 sprints
		assert.Equal(t, "Spiderman", result.Sprints[0].Name)
	})
}

func TestListSprintsUseCase_parseYear(t *testing.T) {
	t.Parallel()
	u := NewListSprintsUseCase(nil)

	start, end, err := u.parseYear("2025")
	assert.NoError(t, err)
	assert.Equal(t, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), start)
	assert.Equal(t, time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC), end)

	// Invalid year string
	_, _, err = u.parseYear("abcd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid year")
}

func TestListSprintsUseCase_parseYear_ErrorBranch(t *testing.T) {
	t.Parallel()
	u := NewListSprintsUseCase(nil)

	_, _, err := u.parseYear("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid year")
}

func TestListSprintsUseCase_parseQuarter_InvalidQuarter(t *testing.T) {
	t.Parallel()
	u := NewListSprintsUseCase(nil)

	// Test invalid quarter number
	_, _, err := u.parseQuarter("Q5 2025")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid quarter number")

	// Test invalid format
	_, _, err = u.parseQuarter("Q2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid quarter format")

	// Test invalid quarter prefix
	_, _, err = u.parseQuarter("X2 2025")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid quarter format")
}

func TestListSprintsUseCase_parseQuarter_InvalidQuarterNumber(t *testing.T) {
	t.Parallel()
	u := NewListSprintsUseCase(nil)
	_, _, err := u.parseQuarter("QX 2025")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid quarter number")
}

func TestListSprintsUseCase_parseQuarter_TooFewParts(t *testing.T) {
	t.Parallel()
	u := NewListSprintsUseCase(nil)
	_, _, err := u.parseQuarter("Q2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid quarter format")
}

func TestListSprintsUseCase_parseQuarter_TooManyParts(t *testing.T) {
	t.Parallel()
	u := NewListSprintsUseCase(nil)
	_, _, err := u.parseQuarter("Q2 2025 extra")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid quarter format")
}

func TestListSprintsUseCase_parsePeriod_UnsupportedFormat(t *testing.T) {
	t.Parallel()
	u := NewListSprintsUseCase(nil)

	// Test unsupported format
	_, _, err := u.parsePeriod("2025-Q2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported period format")
}

func TestListSprintsUseCase_parsePeriod_FourCharNonYear(t *testing.T) {
	t.Parallel()
	u := NewListSprintsUseCase(nil)
	_, _, err := u.parsePeriod("abcd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid year")
}

func TestListSprintsUseCase_filterSprintsByDateRange_InvalidDates(t *testing.T) {
	t.Parallel()
	u := NewListSprintsUseCase(nil)

	sprints := []ports.Sprint{
		{
			ID:        "1",
			Name:      "Valid Sprint",
			StartDate: "2025-04-01T00:00:00Z",
			EndDate:   "2025-04-15T00:00:00Z",
		},
		{
			ID:        "2",
			Name:      "Invalid Start Date",
			StartDate: "invalid-date",
			EndDate:   "2025-04-15T00:00:00Z",
		},
		{
			ID:        "3",
			Name:      "Invalid End Date",
			StartDate: "2025-04-01T00:00:00Z",
			EndDate:   "invalid-date",
		},
	}

	startDate := time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)

	filtered := u.filterSprintsByDateRange(sprints, startDate, endDate)

	// Should only include the valid sprint
	assert.Len(t, filtered, 1)
	assert.Equal(t, "Valid Sprint", filtered[0].Name)
}

func TestListSprintsUseCase_Execute_EmptyProject(t *testing.T) {
	t.Parallel()
	mockJira := &MockJiraPort{}
	u := NewListSprintsUseCase(mockJira)

	_, err := u.Execute("", "Q2 2025")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project is required")
}

func TestListSprintsUseCase_Execute_EmptyPeriod(t *testing.T) {
	t.Parallel()
	mockJira := &MockJiraPort{}
	u := NewListSprintsUseCase(mockJira)

	_, err := u.Execute("FN", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "period is required")
}

func TestListSprintsUseCase_Execute_JiraError(t *testing.T) {
	t.Parallel()
	mockJira := &MockJiraPort{
		err: fmt.Errorf("jira api error"),
	}
	u := NewListSprintsUseCase(mockJira)

	_, err := u.Execute("FN", "Q2 2025")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch sprints")
}

func TestListSprintsUseCase_parseQuarter_EmptyQuarter(t *testing.T) {
	t.Parallel()
	u := NewListSprintsUseCase(nil)
	_, _, err := u.parseQuarter("Q 2025")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid quarter number")
}

func TestListSprintsUseCase_parseQuarter_ValidQuarters(t *testing.T) {
	t.Parallel()
	u := NewListSprintsUseCase(nil)

	tests := []struct {
		input string
		start time.Time
		end   time.Time
	}{
		{"Q1 2025", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 3, 31, 23, 59, 59, 0, time.UTC)},
		{"Q2 2025", time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 6, 30, 23, 59, 59, 0, time.UTC)},
		{"Q3 2025", time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 9, 30, 23, 59, 59, 0, time.UTC)},
		{"Q4 2025", time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC), time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)},
	}

	for _, tt := range tests {
		start, end, err := u.parseQuarter(tt.input)
		assert.NoError(t, err, tt.input)
		assert.Equal(t, tt.start, start, tt.input)
		assert.Equal(t, tt.end, end, tt.input)
	}
}
