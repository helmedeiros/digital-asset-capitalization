package domain

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSprintBoundedTimeCalculation tests all scenarios for sprint-bounded time calculation
func TestSprintBoundedTimeCalculation(t *testing.T) {
	tests := []struct {
		name                string
		issue               JiraIssue
		sprintBoundary      SprintBoundary
		expectedHours       float64
		expectedDescription string
	}{
		// === Single Sprint Stories (Current Working Cases) ===
		{
			name: "Scenario 1A - Story starts and completes within one sprint",
			issue: createIssueWithStatusChanges([]statusChange{
				{timestamp: "2024-03-05T10:00:00.000Z", from: "To Do", to: StatusInProgress},
				{timestamp: "2024-03-10T15:00:00.000Z", from: StatusInProgress, to: StatusDone},
			}),
			sprintBoundary: SprintBoundary{
				StartDate: parseTime("2024-03-01T00:00:00.000Z"),
				EndDate:   parseTime("2024-03-15T23:59:59.999Z"),
			},
			expectedHours:       125.0, // 5 days * 25 hours (5 days, 5 hours)
			expectedDescription: "Full time allocation within sprint boundaries",
		},
		{
			name: "Scenario 1B - Story goes directly from To Do to Done within one sprint",
			issue: createIssueWithStatusChanges([]statusChange{
				{timestamp: "2024-03-10T10:00:00.000Z", from: "To Do", to: StatusDone},
			}),
			sprintBoundary: SprintBoundary{
				StartDate: parseTime("2024-03-01T00:00:00.000Z"),
				EndDate:   parseTime("2024-03-15T23:59:59.999Z"),
			},
			expectedHours:       1.0, // Minimum 1 hour for same-day completion
			expectedDescription: "Minimum 1 hour allocation for direct completion",
		},

		// === Cross-Sprint Stories (Problem Cases) ===
		{
			name: "Scenario 2A - Story starts in Sprint A, completes in Sprint B (Sprint A perspective)",
			issue: createIssueWithStatusChanges([]statusChange{
				{timestamp: "2024-03-05T10:00:00.000Z", from: "To Do", to: StatusInProgress},
				{timestamp: "2024-03-25T15:00:00.000Z", from: StatusInProgress, to: StatusDone},
			}),
			sprintBoundary: SprintBoundary{
				StartDate: parseTime("2024-03-01T00:00:00.000Z"),
				EndDate:   parseTime("2024-03-15T23:59:59.999Z"),
			},
			expectedHours:       254.0, // 10 days, 14 hours (Mar 5 10:00 to Mar 15 23:59:59.999)
			expectedDescription: "Only count time within Sprint A boundaries",
		},
		{
			name: "Scenario 2A - Story starts in Sprint A, completes in Sprint B (Sprint B perspective)",
			issue: createIssueWithStatusChanges([]statusChange{
				{timestamp: "2024-03-05T10:00:00.000Z", from: "To Do", to: StatusInProgress},
				{timestamp: "2024-03-25T15:00:00.000Z", from: StatusInProgress, to: StatusDone},
			}),
			sprintBoundary: SprintBoundary{
				StartDate: parseTime("2024-03-16T00:00:00.000Z"),
				EndDate:   parseTime("2024-03-30T23:59:59.999Z"),
			},
			expectedHours:       231.0, // 9 days, 15 hours (Mar 16 00:00 to Mar 25 15:00)
			expectedDescription: "Only count time within Sprint B boundaries",
		},
		{
			name: "Scenario 2B - Story starts before Sprint A, completes in Sprint A",
			issue: createIssueWithStatusChanges([]statusChange{
				{timestamp: "2024-03-05T10:00:00.000Z", from: "To Do", to: StatusInProgress},
				{timestamp: "2024-03-25T15:00:00.000Z", from: StatusInProgress, to: StatusDone},
			}),
			sprintBoundary: SprintBoundary{
				StartDate: parseTime("2024-03-16T00:00:00.000Z"),
				EndDate:   parseTime("2024-03-30T23:59:59.999Z"),
			},
			expectedHours:       231.0, // 9 days, 15 hours (Mar 16 00:00 to Mar 25 15:00)
			expectedDescription: "Only count time within sprint boundaries, ignore pre-sprint work",
		},
		{
			name: "Scenario 2C - Story starts in Sprint A, completes after Sprint A",
			issue: createIssueWithStatusChanges([]statusChange{
				{timestamp: "2024-03-05T10:00:00.000Z", from: "To Do", to: StatusInProgress},
				{timestamp: "2024-03-25T15:00:00.000Z", from: StatusInProgress, to: StatusDone},
			}),
			sprintBoundary: SprintBoundary{
				StartDate: parseTime("2024-03-01T00:00:00.000Z"),
				EndDate:   parseTime("2024-03-15T23:59:59.999Z"),
			},
			expectedHours:       254.0, // 10 days, 14 hours (Mar 5 10:00 to Mar 15 23:59:59.999)
			expectedDescription: "Only count time within sprint boundaries, ignore post-sprint work",
		},

		// === Complex Multi-Sprint Stories ===
		{
			name: "Scenario 3A - Multiple status transitions across sprints (Sprint A)",
			issue: createIssueWithStatusChanges([]statusChange{
				{timestamp: "2024-03-05T10:00:00.000Z", from: "To Do", to: StatusInProgress},
				{timestamp: "2024-03-10T15:00:00.000Z", from: StatusInProgress, to: StatusBlocked},
				{timestamp: "2024-03-20T10:00:00.000Z", from: StatusBlocked, to: StatusInProgress},
				{timestamp: "2024-03-25T15:00:00.000Z", from: StatusInProgress, to: StatusDone},
			}),
			sprintBoundary: SprintBoundary{
				StartDate: parseTime("2024-03-01T00:00:00.000Z"),
				EndDate:   parseTime("2024-03-15T23:59:59.999Z"),
			},
			expectedHours:       125.0, // 5 days, 5 hours (Mar 5 10:00 to Mar 10 15:00)
			expectedDescription: "Only count active work time within sprint boundaries",
		},
		{
			name: "Scenario 3A - Multiple status transitions across sprints (Sprint B)",
			issue: createIssueWithStatusChanges([]statusChange{
				{timestamp: "2024-03-05T10:00:00.000Z", from: "To Do", to: StatusInProgress},
				{timestamp: "2024-03-10T15:00:00.000Z", from: StatusInProgress, to: StatusBlocked},
				{timestamp: "2024-03-20T10:00:00.000Z", from: StatusBlocked, to: StatusInProgress},
				{timestamp: "2024-03-25T15:00:00.000Z", from: StatusInProgress, to: StatusDone},
			}),
			sprintBoundary: SprintBoundary{
				StartDate: parseTime("2024-03-16T00:00:00.000Z"),
				EndDate:   parseTime("2024-03-30T23:59:59.999Z"),
			},
			expectedHours:       125.0, // 5 days, 5 hours (Mar 20 10:00 to Mar 25 15:00)
			expectedDescription: "Only count active work time within sprint boundaries",
		},
		{
			name: "Scenario 3B - Story spans three sprints (Sprint A)",
			issue: createIssueWithStatusChanges([]statusChange{
				{timestamp: "2024-03-05T10:00:00.000Z", from: "To Do", to: StatusInProgress},
				{timestamp: "2024-03-20T15:00:00.000Z", from: StatusInProgress, to: StatusBlocked},
				{timestamp: "2024-04-10T10:00:00.000Z", from: StatusBlocked, to: StatusInProgress},
				{timestamp: "2024-04-12T15:00:00.000Z", from: StatusInProgress, to: StatusDone},
			}),
			sprintBoundary: SprintBoundary{
				StartDate: parseTime("2024-03-01T00:00:00.000Z"),
				EndDate:   parseTime("2024-03-15T23:59:59.999Z"),
			},
			expectedHours:       254.0, // 10 days, 14 hours (Mar 5 10:00 to Mar 15 23:59:59.999)
			expectedDescription: "Count active work time within Sprint A boundaries",
		},
		{
			name: "Scenario 3B - Story spans three sprints (Sprint B - partially blocked)",
			issue: createIssueWithStatusChanges([]statusChange{
				{timestamp: "2024-03-05T10:00:00.000Z", from: "To Do", to: StatusInProgress},
				{timestamp: "2024-03-20T15:00:00.000Z", from: StatusInProgress, to: StatusBlocked},
				{timestamp: "2024-04-10T10:00:00.000Z", from: StatusBlocked, to: StatusInProgress},
				{timestamp: "2024-04-12T15:00:00.000Z", from: StatusInProgress, to: StatusDone},
			}),
			sprintBoundary: SprintBoundary{
				StartDate: parseTime("2024-03-16T00:00:00.000Z"),
				EndDate:   parseTime("2024-03-30T23:59:59.999Z"),
			},
			expectedHours:       111.0, // 4 days 15 hours (Mar 16 00:00 to Mar 20 15:00, StatusInProgress period only)
			expectedDescription: "Count only In Progress time, not blocked time",
		},
		{
			name: "Scenario 3B - Story spans three sprints (Sprint C)",
			issue: createIssueWithStatusChanges([]statusChange{
				{timestamp: "2024-03-05T10:00:00.000Z", from: "To Do", to: StatusInProgress},
				{timestamp: "2024-03-20T15:00:00.000Z", from: StatusInProgress, to: StatusBlocked},
				{timestamp: "2024-04-10T10:00:00.000Z", from: StatusBlocked, to: StatusInProgress},
				{timestamp: "2024-04-12T15:00:00.000Z", from: StatusInProgress, to: StatusDone},
			}),
			sprintBoundary: SprintBoundary{
				StartDate: parseTime("2024-04-01T00:00:00.000Z"),
				EndDate:   parseTime("2024-04-15T23:59:59.999Z"),
			},
			expectedHours:       53.0, // 2 days, 5 hours (Apr 10 10:00 to Apr 12 15:00)
			expectedDescription: "Count active work time within Sprint C boundaries",
		},
		{
			name: "Scenario 3C - Story moves backward in status across sprints (Sprint A)",
			issue: createIssueWithStatusChanges([]statusChange{
				{timestamp: "2024-03-05T10:00:00.000Z", from: "To Do", to: StatusInProgress},
				{timestamp: "2024-03-10T15:00:00.000Z", from: StatusInProgress, to: StatusUnderReview},
				{timestamp: "2024-03-20T10:00:00.000Z", from: StatusUnderReview, to: StatusInProgress},
				{timestamp: "2024-03-25T15:00:00.000Z", from: StatusInProgress, to: StatusDone},
			}),
			sprintBoundary: SprintBoundary{
				StartDate: parseTime("2024-03-01T00:00:00.000Z"),
				EndDate:   parseTime("2024-03-15T23:59:59.999Z"),
			},
			expectedHours:       254.0, // 10 days, 14 hours (Mar 5 10:00 to Mar 15 23:59:59.999, both In Progress and Under Review count as work)
			expectedDescription: "Count both In Progress and Under Review as work time",
		},

		// === Edge Cases ===
		{
			name: "Scenario 4A - Story completed same day it starts (cross-sprint Sprint A)",
			issue: createIssueWithStatusChanges([]statusChange{
				{timestamp: "2024-03-15T10:00:00.000Z", from: "To Do", to: StatusInProgress},
				{timestamp: "2024-03-16T14:00:00.000Z", from: StatusInProgress, to: StatusDone},
			}),
			sprintBoundary: SprintBoundary{
				StartDate: parseTime("2024-03-01T00:00:00.000Z"),
				EndDate:   parseTime("2024-03-15T23:59:59.999Z"),
			},
			expectedHours:       14.0, // 14 hours (Mar 15 10:00 to Mar 15 23:59)
			expectedDescription: "Count time within Sprint A boundaries only",
		},
		{
			name: "Scenario 4A - Story completed same day it starts (cross-sprint Sprint B)",
			issue: createIssueWithStatusChanges([]statusChange{
				{timestamp: "2024-03-15T10:00:00.000Z", from: "To Do", to: StatusInProgress},
				{timestamp: "2024-03-16T14:00:00.000Z", from: StatusInProgress, to: StatusDone},
			}),
			sprintBoundary: SprintBoundary{
				StartDate: parseTime("2024-03-16T00:00:00.000Z"),
				EndDate:   parseTime("2024-03-30T23:59:59.999Z"),
			},
			expectedHours:       14.0, // 14 hours (Mar 16 00:00 to Mar 16 14:00)
			expectedDescription: "Count time within Sprint B boundaries only",
		},
		{
			name:  "Scenario 4B - Story with no status changes",
			issue: createIssueWithStatusChanges([]statusChange{}),
			sprintBoundary: SprintBoundary{
				StartDate: parseTime("2024-03-01T00:00:00.000Z"),
				EndDate:   parseTime("2024-03-15T23:59:59.999Z"),
			},
			expectedHours:       0.0, // No time allocation
			expectedDescription: "No allocation for stories with no status changes",
		},
		{
			name: "Story with only non-status changes",
			issue: createIssueWithNonStatusChanges([]nonStatusChange{
				{timestamp: "2024-03-05T10:00:00.000Z", field: "description", from: "", to: "Updated description"},
			}),
			sprintBoundary: SprintBoundary{
				StartDate: parseTime("2024-03-01T00:00:00.000Z"),
				EndDate:   parseTime("2024-03-15T23:59:59.999Z"),
			},
			expectedHours:       0.0, // No time allocation
			expectedDescription: "No allocation for stories with only non-status changes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the sprint-bounded time calculator
			calculator := NewSprintBoundedTimeCalculator()

			// Calculate working hours within sprint boundaries
			hours, err := calculator.CalculateWorkingHours(context.Background(), tt.issue, tt.sprintBoundary)

			// Verify results
			require.NoError(t, err, "CalculateWorkingHours should not return an error")
			assert.Equal(t, tt.expectedHours, hours, "Expected hours should match calculated hours: %s", tt.expectedDescription)
		})
	}
}

// TestSprintBoundaryValidation tests validation of sprint boundaries
func TestSprintBoundaryValidation(t *testing.T) {
	tests := []struct {
		name           string
		sprintBoundary SprintBoundary
		expectedError  string
	}{
		{
			name: "Valid sprint boundary",
			sprintBoundary: SprintBoundary{
				StartDate: parseTime("2024-03-01T00:00:00.000Z"),
				EndDate:   parseTime("2024-03-15T23:59:59.999Z"),
			},
			expectedError: "",
		},
		{
			name: "Invalid sprint boundary - end before start",
			sprintBoundary: SprintBoundary{
				StartDate: parseTime("2024-03-15T00:00:00.000Z"),
				EndDate:   parseTime("2024-03-01T23:59:59.999Z"),
			},
			expectedError: "sprint end date must be after start date",
		},
		{
			name: "Invalid sprint boundary - zero start date",
			sprintBoundary: SprintBoundary{
				StartDate: time.Time{},
				EndDate:   parseTime("2024-03-15T23:59:59.999Z"),
			},
			expectedError: "sprint start date is required",
		},
		{
			name: "Invalid sprint boundary - zero end date",
			sprintBoundary: SprintBoundary{
				StartDate: parseTime("2024-03-01T00:00:00.000Z"),
				EndDate:   time.Time{},
			},
			expectedError: "sprint end date is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.sprintBoundary.Validate()

			if tt.expectedError == "" {
				assert.NoError(t, err, "Valid sprint boundary should not return an error")
			} else {
				assert.Error(t, err, "Invalid sprint boundary should return an error")
				assert.Contains(t, err.Error(), tt.expectedError, "Error message should contain expected text")
			}
		})
	}
}

// TestTimeCalculationStrategy tests the strategy pattern implementation
func TestTimeCalculationStrategy(t *testing.T) {
	t.Run("Legacy strategy maintains backward compatibility", func(t *testing.T) {
		issue := createIssueWithStatusChanges([]statusChange{
			{timestamp: "2024-03-05T10:00:00.000Z", from: "To Do", to: StatusInProgress},
			{timestamp: "2024-03-25T15:00:00.000Z", from: StatusInProgress, to: StatusDone},
		})

		sprintBoundary := SprintBoundary{
			StartDate: parseTime("2024-03-01T00:00:00.000Z"),
			EndDate:   parseTime("2024-03-15T23:59:59.999Z"),
		}

		// Test legacy strategy (should return full lifecycle time)
		legacyCalculator := NewLegacyTimeCalculator()
		legacyHours, err := legacyCalculator.CalculateWorkingHours(context.Background(), issue, sprintBoundary)
		require.NoError(t, err)

		// Test sprint-bounded strategy
		sprintCalculator := NewSprintBoundedTimeCalculator()
		sprintHours, err := sprintCalculator.CalculateWorkingHours(context.Background(), issue, sprintBoundary)
		require.NoError(t, err)

		// Legacy should return full lifecycle time (20 days, 5 hours)
		assert.Equal(t, 485.0, legacyHours, "Legacy calculator should return full lifecycle time")

		// Sprint-bounded should return only time within sprint boundaries (10 days, 14 hours)
		assert.Equal(t, 254.0, sprintHours, "Sprint-bounded calculator should return only time within sprint boundaries")

		// Sprint-bounded should be less than legacy for cross-sprint stories
		assert.Less(t, sprintHours, legacyHours, "Sprint-bounded time should be less than legacy time for cross-sprint stories")
	})
}

// Helper types and functions for creating test data
type statusChange struct {
	timestamp string
	from      string
	to        string
}

type nonStatusChange struct {
	timestamp string
	field     string
	from      string
	to        string
}

func createIssueWithStatusChanges(changes []statusChange) JiraIssue {
	histories := make([]JiraChangeHistory, len(changes))
	for i, change := range changes {
		histories[i] = JiraChangeHistory{
			Created: change.timestamp,
			Items: []JiraChangeItem{
				{
					Field:      "status",
					FromString: change.from,
					ToString:   change.to,
				},
			},
		}
	}

	return JiraIssue{
		Key: "TEST-123",
		Fields: JiraFields{
			Summary: "Test Issue",
			Assignee: JiraAssignee{
				DisplayName: "Test User",
			},
			Status: JiraStatus{
				Name: StatusDone,
			},
			IssueType: IssueType{
				Name: "Task",
			},
		},
		Changelog: JiraChangelog{
			Histories: histories,
		},
	}
}

func createIssueWithNonStatusChanges(changes []nonStatusChange) JiraIssue {
	histories := make([]JiraChangeHistory, len(changes))
	for i, change := range changes {
		histories[i] = JiraChangeHistory{
			Created: change.timestamp,
			Items: []JiraChangeItem{
				{
					Field:      change.field,
					FromString: change.from,
					ToString:   change.to,
				},
			},
		}
	}

	return JiraIssue{
		Key: "TEST-124",
		Fields: JiraFields{
			Summary: "Test Issue",
			Assignee: JiraAssignee{
				DisplayName: "Test User",
			},
			Status: JiraStatus{
				Name: "To Do",
			},
			IssueType: IssueType{
				Name: "Task",
			},
		},
		Changelog: JiraChangelog{
			Histories: histories,
		},
	}
}

func parseTime(timestamp string) time.Time {
	t, err := time.Parse("2006-01-02T15:04:05.000Z", timestamp)
	if err != nil {
		panic(err)
	}
	return t.UTC()
}
