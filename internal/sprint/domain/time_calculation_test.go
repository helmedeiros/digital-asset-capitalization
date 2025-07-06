package domain

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSprintBoundary(t *testing.T) {
	start := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
	end := time.Date(2024, 1, 29, 18, 0, 0, 0, time.UTC)

	boundary, err := NewSprintBoundary(start, end)

	require.NoError(t, err)
	assert.Equal(t, start, boundary.StartDate)
	assert.Equal(t, end, boundary.EndDate)
}

func TestNewSprintBoundary_ValidationErrors(t *testing.T) {
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
	strategy := NewSprintBoundedTimeCalculator()
	calculator := NewWorkTimeCalculator(strategy)

	assert.NotNil(t, calculator)
}

func TestWorkTimeCalculator_SetStrategy(t *testing.T) {
	strategy1 := NewSprintBoundedTimeCalculator()
	strategy2 := NewLegacyTimeCalculator()

	calculator := NewWorkTimeCalculator(strategy1)
	calculator.SetStrategy(strategy2)

	// We can't directly test the internal strategy field, but we can verify the method doesn't panic
	assert.NotNil(t, calculator)
}

func TestWorkTimeCalculator_CalculateWorkingHours(t *testing.T) {
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
