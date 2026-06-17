package domain

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSprintBoundary(t *testing.T) {
	t.Parallel()
	start := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 29, 18, 0, 0, 0, time.UTC)

	boundary, err := NewSprintBoundary(start, end)

	require.NoError(t, err)
	assert.Equal(t, start, boundary.StartDate)
	assert.Equal(t, end, boundary.EndDate)
}

func TestNewSprintBoundary_ValidationErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		startDate time.Time
		endDate   time.Time
		expectErr bool
	}{
		{
			name:      "end before start",
			startDate: time.Date(2024, 1, 29, 18, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
			expectErr: true,
		},
		{
			name:      "zero start date",
			startDate: time.Time{},
			endDate:   time.Date(2024, 1, 29, 18, 0, 0, 0, time.UTC),
			expectErr: true,
		},
		{
			name:      "zero end date",
			startDate: time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
			endDate:   time.Time{},
			expectErr: true,
		},
		{
			name:      "valid dates",
			startDate: time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 1, 29, 18, 0, 0, 0, time.UTC),
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			boundary, err := NewSprintBoundary(tt.startDate, tt.endDate)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Equal(t, SprintBoundary{}, boundary)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.startDate, boundary.StartDate)
				assert.Equal(t, tt.endDate, boundary.EndDate)
			}
		})
	}
}

func TestNewWorkTimeCalculator(t *testing.T) {
	t.Parallel()
	strategy := NewSprintBoundedTimeCalculator()
	calculator := NewWorkTimeCalculator(strategy)

	assert.NotNil(t, calculator)
}

func TestWorkTimeCalculator_SetStrategy(t *testing.T) {
	t.Parallel()
	strategy1 := NewSprintBoundedTimeCalculator()
	strategy2 := NewLegacyTimeCalculator()

	calculator := NewWorkTimeCalculator(strategy1)
	calculator.SetStrategy(strategy2)

	// We can't directly test the internal strategy field, but we can verify the method doesn't panic
	assert.NotNil(t, calculator)
}

func TestCalendarToWorkingHours(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		start    time.Time
		end      time.Time
		expected float64
	}{
		{
			name:     "full work week Mon 9am to Fri 5pm",
			start:    time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),  // Monday
			end:      time.Date(2024, 1, 19, 17, 0, 0, 0, time.UTC), // Friday
			expected: 40.0,
		},
		{
			name:     "spans weekend Fri 9am to Mon 5pm",
			start:    time.Date(2024, 1, 19, 9, 0, 0, 0, time.UTC),  // Friday
			end:      time.Date(2024, 1, 22, 17, 0, 0, 0, time.UTC), // Monday
			expected: 16.0,                                          // 8h Fri + 8h Mon
		},
		{
			name:     "same day Mon 10am to 3pm",
			start:    time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC), // Monday
			end:      time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC),
			expected: 5.0,
		},
		{
			name:     "zero duration same time",
			start:    time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			expected: 0.0,
		},
		{
			name:     "negative range end before start",
			start:    time.Date(2024, 1, 16, 10, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			expected: 0.0,
		},
		{
			name:     "starts on Saturday ends Monday 5pm",
			start:    time.Date(2024, 1, 20, 10, 0, 0, 0, time.UTC), // Saturday
			end:      time.Date(2024, 1, 22, 17, 0, 0, 0, time.UTC), // Monday
			expected: 8.0,                                           // only Monday counts
		},
		{
			name:     "partial day Mon 2pm to 5pm",
			start:    time.Date(2024, 1, 15, 14, 0, 0, 0, time.UTC), // Monday
			end:      time.Date(2024, 1, 15, 17, 0, 0, 0, time.UTC),
			expected: 3.0,
		},
		{
			name:     "before working hours Mon 6am to 10am",
			start:    time.Date(2024, 1, 15, 6, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			expected: 1.0, // only 9am-10am counts
		},
		{
			name:     "after working hours Mon 5pm to 8pm",
			start:    time.Date(2024, 1, 15, 17, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 1, 15, 20, 0, 0, 0, time.UTC),
			expected: 0.0, // nothing after 5pm
		},
		{
			name:     "two full weeks Mon to Fri",
			start:    time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),  // Monday
			end:      time.Date(2024, 1, 26, 17, 0, 0, 0, time.UTC), // Friday next week
			expected: 80.0,
		},
		{
			name:     "entirely on weekend",
			start:    time.Date(2024, 1, 20, 9, 0, 0, 0, time.UTC),  // Saturday
			end:      time.Date(2024, 1, 21, 17, 0, 0, 0, time.UTC), // Sunday
			expected: 0.0,
		},
		{
			name:     "overnight Mon 3pm to Tue 11am",
			start:    time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC), // Monday 3pm
			end:      time.Date(2024, 1, 16, 11, 0, 0, 0, time.UTC), // Tuesday 11am
			expected: 4.0,                                           // 2h Mon (3pm-5pm) + 2h Tue (9am-11am)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalendarToWorkingHours(tt.start, tt.end)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWorkTimeCalculator_CalculateWorkingHours(t *testing.T) {
	t.Parallel()
	// Create a mock issue
	issue := JiraIssue{
		Key: "TEST-1",
	}

	// Create sprint boundary
	boundary, err := NewSprintBoundary(
		time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 29, 18, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)

	// Test with sprint-bounded strategy
	strategy := NewSprintBoundedTimeCalculator()
	calculator := NewWorkTimeCalculator(strategy)

	hours, err := calculator.CalculateWorkingHours(context.Background(), issue, boundary)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, hours, 0.0)
}
