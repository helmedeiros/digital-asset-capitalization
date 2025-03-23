package domain

import (
	"testing"
	"time"
)

func TestSprint_IsActive(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")

	tests := []struct {
		name     string
		sprint   Sprint
		expected bool
	}{
		{
			name: "active sprint",
			sprint: Sprint{
				StartDate: yesterday,
				EndDate:   tomorrow,
			},
			expected: true,
		},
		{
			name: "future sprint",
			sprint: Sprint{
				StartDate: tomorrow,
				EndDate:   now.AddDate(0, 0, 15).Format("2006-01-02"),
			},
			expected: false,
		},
		{
			name: "past sprint",
			sprint: Sprint{
				StartDate: now.AddDate(0, 0, -15).Format("2006-01-02"),
				EndDate:   yesterday,
			},
			expected: false,
		},
		{
			name: "sprint starting today",
			sprint: Sprint{
				StartDate: now.Format("2006-01-02"),
				EndDate:   now.AddDate(0, 0, 14).Format("2006-01-02"),
			},
			expected: true,
		},
		{
			name: "sprint ending today",
			sprint: Sprint{
				StartDate: now.AddDate(0, 0, -14).Format("2006-01-02"),
				EndDate:   now.Format("2006-01-02"),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.sprint.IsActive()
			if result != tt.expected {
				t.Errorf("IsActive() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSprint_IsCompleted(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")

	tests := []struct {
		name     string
		sprint   Sprint
		expected bool
	}{
		{
			name: "completed sprint",
			sprint: Sprint{
				StartDate: now.AddDate(0, 0, -15).Format("2006-01-02"),
				EndDate:   yesterday,
				Status:    SprintStatusComplete,
			},
			expected: true,
		},
		{
			name: "active sprint",
			sprint: Sprint{
				StartDate: yesterday,
				EndDate:   tomorrow,
				Status:    SprintStatusActive,
			},
			expected: false,
		},
		{
			name: "future sprint",
			sprint: Sprint{
				StartDate: tomorrow,
				EndDate:   now.AddDate(0, 0, 15).Format("2006-01-02"),
				Status:    SprintStatusFuture,
			},
			expected: false,
		},
		{
			name: "ended but not closed sprint",
			sprint: Sprint{
				StartDate: now.AddDate(0, 0, -15).Format("2006-01-02"),
				EndDate:   yesterday,
				Status:    SprintStatusActive,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.sprint.IsCompleted()
			if result != tt.expected {
				t.Errorf("IsCompleted() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSprint_GetDuration(t *testing.T) {
	tests := []struct {
		name     string
		sprint   Sprint
		expected time.Duration
	}{
		{
			name: "two week sprint",
			sprint: Sprint{
				StartDate: "2024-03-01",
				EndDate:   "2024-03-15",
			},
			expected: 14 * 24 * time.Hour,
		},
		{
			name: "one week sprint",
			sprint: Sprint{
				StartDate: "2024-03-01",
				EndDate:   "2024-03-08",
			},
			expected: 7 * 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.sprint.GetDuration()
			if result != tt.expected {
				t.Errorf("GetDuration() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSprint_GetRemainingTime(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")

	tests := []struct {
		name     string
		sprint   Sprint
		expected time.Duration
	}{
		{
			name: "active sprint with 7 days remaining",
			sprint: Sprint{
				StartDate: yesterday,
				EndDate:   now.AddDate(0, 0, 8).Format("2006-01-02"),
			},
			expected: 7 * 24 * time.Hour,
		},
		{
			name: "completed sprint",
			sprint: Sprint{
				StartDate: now.AddDate(0, 0, -15).Format("2006-01-02"),
				EndDate:   yesterday,
			},
			expected: 0,
		},
		{
			name: "future sprint",
			sprint: Sprint{
				StartDate: tomorrow,
				EndDate:   now.AddDate(0, 0, 15).Format("2006-01-02"),
			},
			expected: 14 * 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.sprint.GetRemainingTime()
			if result != tt.expected {
				t.Errorf("GetRemainingTime() = %v, want %v", result, tt.expected)
			}
		})
	}
}
