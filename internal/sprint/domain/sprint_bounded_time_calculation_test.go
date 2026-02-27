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
			expectedHours:       31.0, // 3 full workdays (Wed-Fri) + partial Wed (10am-5pm=7h) + partial Fri (9am-3pm=not applicable, it's Sun Mar 10) → recalculated as working hours
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
			expectedHours:       71.0, // Working hours: Mar 5 (Tue) 10am-5pm=7h, Mar 6-8 (Wed-Fri) 24h, Mar 11-15 (Mon-Fri) 40h → 71h
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
			expectedHours:       46.0, // Working hours: Mar 16 (Sat) skip, Mar 17 (Sun) skip, Mar 18-22 (Mon-Fri) 40h, Mar 25 (Mon) 9am-3pm=6h → 46h
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
			expectedHours:       46.0, // Working hours: same as Sprint B perspective above
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
			expectedHours:       71.0, // Working hours: same as Sprint A perspective above
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
			expectedHours:       31.0, // Working hours: Mar 5-8 (Tue-Fri) In Progress, Mar 10 (Sun) transition → 31h
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
			expectedHours:       29.0, // Working hours: Mar 20 (Wed) 10am-5pm=7h, Mar 21-22 (Thu-Fri) 16h, Mar 25 (Mon) 9am-3pm=6h → 29h
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
			expectedHours:       71.0, // Working hours: same calculation as Scenario 2A Sprint A
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
			expectedHours:       22.0, // Working hours: Mar 18-19 (Mon-Tue) 16h, Mar 20 (Wed) 9am-3pm=6h → 22h (weekends skipped)
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
			expectedHours:       21.0, // Working hours: Apr 10 (Wed) 10am-5pm=7h, Apr 11 (Thu) 8h, Apr 12 (Fri) 9am-3pm=6h → 21h
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
			expectedHours:       71.0, // Working hours: same as Scenario 2A Sprint A (entire period is work time)
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
			expectedHours:       7.0, // Working hours: Mar 15 (Fri) 10am-5pm=7h
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
			expectedHours:       1.0, // Working hours: Mar 16 (Sat) skip → fallback: issue completed within sprint → minimum 1h
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

		// Sprint-bounded should return only working hours within sprint boundaries
		assert.Equal(t, 71.0, sprintHours, "Sprint-bounded calculator should return only working hours within sprint boundaries")

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
