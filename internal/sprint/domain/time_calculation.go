package domain

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
)

// TimeCalculationStrategy defines the interface for different time calculation strategies
// This follows the Strategy pattern to allow different calculation approaches
type TimeCalculationStrategy interface {
	CalculateWorkingHours(ctx context.Context, issue JiraIssue, sprintBoundary SprintBoundary) (float64, error)
}

// SprintBoundary represents the time boundaries of a sprint
// This is a value object that encapsulates sprint date range logic
type SprintBoundary struct {
	StartDate time.Time
	EndDate   time.Time
}

// NewSprintBoundary creates a new sprint boundary with validation
func NewSprintBoundary(startDate, endDate time.Time) (SprintBoundary, error) {
	boundary := SprintBoundary{
		StartDate: startDate,
		EndDate:   endDate,
	}

	if err := boundary.Validate(); err != nil {
		return SprintBoundary{}, err
	}

	return boundary, nil
}

// Validate ensures the sprint boundary is valid
func (sb SprintBoundary) Validate() error {
	if sb.StartDate.IsZero() {
		return errors.New("sprint start date is required")
	}

	if sb.EndDate.IsZero() {
		return errors.New("sprint end date is required")
	}

	if sb.EndDate.Before(sb.StartDate) {
		return errors.New("sprint end date must be after start date")
	}

	return nil
}

// Contains checks if a given time falls within the sprint boundary
func (sb SprintBoundary) Contains(t time.Time) bool {
	return !t.Before(sb.StartDate) && !t.After(sb.EndDate)
}

// ClampTime constrains a time to fall within the sprint boundary
func (sb SprintBoundary) ClampTime(t time.Time) time.Time {
	if t.Before(sb.StartDate) {
		return sb.StartDate
	}
	if t.After(sb.EndDate) {
		return sb.EndDate
	}
	return t
}

// StatusChangePeriod represents a period of time with a specific status
// This helps in calculating work time for different status transitions
type StatusChangePeriod struct {
	StartTime time.Time
	EndTime   time.Time
	Status    string
}

// Duration returns the duration of the status change period
func (scp StatusChangePeriod) Duration() time.Duration {
	if scp.EndTime.IsZero() {
		return 0
	}
	return scp.EndTime.Sub(scp.StartTime)
}

// IsWorkTime determines if a status represents active work time
func (scp StatusChangePeriod) IsWorkTime() bool {
	// First check for exact matches to handle custom statuses
	normalizedStatus := strings.ToLower(strings.TrimSpace(scp.Status))

	// Explicitly exclude non-work statuses (including custom ones)
	nonWorkPatterns := []string{
		"done",
		"deployed",
		"closed",
		"resolved",
		"won't",
		"cancelled",
		"duplicate",
		"to do",
		"todo",
		"open",
		"backlog",
		"blocked",
		"on hold",
	}

	// Check if it's a non-work status
	for _, pattern := range nonWorkPatterns {
		if strings.Contains(normalizedStatus, pattern) {
			return false
		}
	}

	// Work status patterns
	workPatterns := []string{
		"in progress",
		"progress",
		"development",
		"review",
		"testing",
		"qa",
	}

	// Check if it's explicitly a work status
	for _, pattern := range workPatterns {
		if strings.Contains(normalizedStatus, pattern) {
			return true
		}
	}

	// Default to false for unknown statuses
	return false
}

// SprintBoundedTimeCalculator implements TimeCalculationStrategy
// This is the main implementation that respects sprint boundaries
type SprintBoundedTimeCalculator struct {
	// Could add configuration options here if needed
}

// NewSprintBoundedTimeCalculator creates a new sprint-bounded time calculator
func NewSprintBoundedTimeCalculator() *SprintBoundedTimeCalculator {
	return &SprintBoundedTimeCalculator{}
}

// CalculateWorkingHours calculates working hours within sprint boundaries
func (calc *SprintBoundedTimeCalculator) CalculateWorkingHours(_ context.Context, issue JiraIssue, sprintBoundary SprintBoundary) (float64, error) {
	// Validate sprint boundary
	if err := sprintBoundary.Validate(); err != nil {
		return 0, err
	}

	// Get status change periods
	periods := calc.extractStatusChangePeriods(issue)

	// Calculate work time within sprint boundaries
	var totalHours float64
	for _, period := range periods {
		if period.IsWorkTime() {
			hours := calc.calculatePeriodHours(period, sprintBoundary)
			totalHours += hours
		}
	}

	// Apply fallback logic for tasks without proper history
	if totalHours == 0 {
		// Check if task was completed within sprint
		if calc.hasCompletionInSprint(issue, sprintBoundary) {
			totalHours = 1.0
		} else if calc.isCurrentlyDone(issue) && len(issue.GetStatusChanges()) > 0 {
			// Task is done and has some changelog activity - give default hours
			// This handles tasks that were bulk-updated or have incomplete changelog
			// Only apply if there's some activity history, not for completely unchanged tasks
			totalHours = 8.0
		}
	}

	// Round to avoid floating-point precision issues
	return roundHours(totalHours), nil
}

// extractStatusChangePeriods extracts all status change periods from the issue
func (calc *SprintBoundedTimeCalculator) extractStatusChangePeriods(issue JiraIssue) []StatusChangePeriod {
	var periods []StatusChangePeriod

	// Sort status changes chronologically
	statusChanges := issue.GetStatusChanges()
	if len(statusChanges) == 0 {
		return periods
	}

	// Create periods between status changes
	for i := 0; i < len(statusChanges); i++ {
		// Parse timestamp
		startTime, err := time.Parse("2006-01-02T15:04:05.000Z", statusChanges[i].Created)
		if err != nil {
			// Try RFC3339 format
			startTime, err = time.Parse(time.RFC3339, statusChanges[i].Created)
			if err != nil {
				continue
			}
		}
		startTime = startTime.UTC()

		// Get the status that this change transitions TO
		var status string
		for _, item := range statusChanges[i].Items {
			if item.IsStatusChange() {
				status = item.ToString
				break
			}
		}

		// Determine end time (next status change or zero if it's the last one)
		var endTime time.Time
		if i+1 < len(statusChanges) {
			endTime, err = time.Parse("2006-01-02T15:04:05.000Z", statusChanges[i+1].Created)
			if err != nil {
				endTime, err = time.Parse(time.RFC3339, statusChanges[i+1].Created)
				if err != nil {
					continue
				}
			}
			endTime = endTime.UTC()
		}

		periods = append(periods, StatusChangePeriod{
			StartTime: startTime,
			EndTime:   endTime,
			Status:    status,
		})
	}

	return periods
}

// calculatePeriodHours calculates hours for a specific period within sprint boundaries
func (calc *SprintBoundedTimeCalculator) calculatePeriodHours(period StatusChangePeriod, sprintBoundary SprintBoundary) float64 {
	// Clamp period to sprint boundaries
	startTime := sprintBoundary.ClampTime(period.StartTime)
	endTime := period.EndTime

	// If end time is zero (ongoing), use sprint end time
	if endTime.IsZero() {
		endTime = sprintBoundary.EndDate
	} else {
		endTime = sprintBoundary.ClampTime(endTime)
	}

	// If start time is after end time, no overlap with sprint
	if startTime.After(endTime) {
		return 0
	}

	// Calculate duration in hours
	duration := endTime.Sub(startTime)
	hours := duration.Hours()

	// Ensure non-negative hours
	if hours < 0 {
		hours = 0
	}

	return hours
}

// isCurrentlyDone checks if the issue's current status is a done state
func (calc *SprintBoundedTimeCalculator) isCurrentlyDone(issue JiraIssue) bool {
	currentStatus := strings.ToLower(strings.TrimSpace(issue.Fields.Status.Name))
	return strings.Contains(currentStatus, "done") ||
		strings.Contains(currentStatus, "deployed") ||
		strings.Contains(currentStatus, "closed") ||
		strings.Contains(currentStatus, "resolved")
}

// hasCompletionInSprint checks if the issue was completed within the sprint
func (calc *SprintBoundedTimeCalculator) hasCompletionInSprint(issue JiraIssue, sprintBoundary SprintBoundary) bool {
	statusChanges := issue.GetStatusChanges()

	for _, change := range statusChanges {
		timestamp, err := time.Parse("2006-01-02T15:04:05.000Z", change.Created)
		if err != nil {
			timestamp, err = time.Parse(time.RFC3339, change.Created)
			if err != nil {
				continue
			}
		}
		timestamp = timestamp.UTC()

		if sprintBoundary.Contains(timestamp) {
			for _, item := range change.Items {
				if item.IsStatusChange() {
					// Check if status indicates completion using normalized checks
					normalizedStatus := strings.ToLower(strings.TrimSpace(item.ToString))
					if strings.Contains(normalizedStatus, "done") ||
						strings.Contains(normalizedStatus, "deployed") ||
						strings.Contains(normalizedStatus, "closed") ||
						strings.Contains(normalizedStatus, "resolved") ||
						strings.Contains(normalizedStatus, "won't") ||
						strings.Contains(normalizedStatus, "cancelled") {
						return true
					}
				}
			}
		}
	}

	return false
}

// LegacyTimeCalculator implements the original time calculation logic
// This maintains backward compatibility
type LegacyTimeCalculator struct{}

// NewLegacyTimeCalculator creates a new legacy time calculator
func NewLegacyTimeCalculator() *LegacyTimeCalculator {
	return &LegacyTimeCalculator{}
}

// CalculateWorkingHours calculates working hours using the legacy algorithm
func (calc *LegacyTimeCalculator) CalculateWorkingHours(_ context.Context, issue JiraIssue, _ SprintBoundary) (float64, error) {
	// This replicates the original logic from sprint_time_allocation.go
	startTime, endTime := calc.getIssueTimeRange(issue)

	if startTime.IsZero() {
		return 0, nil
	}

	// Calculate hours between start and end time (ignoring sprint boundaries)
	duration := endTime.Sub(startTime)
	hours := duration.Hours()

	// Ensure hours is not negative
	if hours < 0 {
		hours = 0
	}

	return roundHours(hours), nil
}

// getIssueTimeRange replicates the original logic from the use case
func (calc *LegacyTimeCalculator) getIssueTimeRange(issue JiraIssue) (time.Time, time.Time) {
	var startTime, endTime time.Time
	var inProgress bool
	var firstInProgressTime time.Time

	// Process histories in chronological order
	for i := 0; i < len(issue.Changelog.Histories); i++ {
		history := issue.Changelog.Histories[i]

		for _, item := range history.Items {
			if !item.IsStatusChange() {
				continue
			}

			// Parse the history timestamp and ensure UTC timezone
			historyTime, err := time.Parse("2006-01-02T15:04:05.000-0700", history.Created)
			if err != nil {
				// If parsing fails, try RFC3339 format
				historyTime, err = time.Parse(time.RFC3339, history.Created)
				if err != nil {
					continue
				}
			}
			historyTime = historyTime.UTC()

			// Normalize the status for checking
			normalizedToStatus := strings.ToLower(strings.TrimSpace(item.ToString))

			// Look for transition into in-progress state
			if strings.Contains(normalizedToStatus, "progress") ||
				strings.Contains(normalizedToStatus, "development") ||
				strings.Contains(normalizedToStatus, "review") ||
				strings.Contains(normalizedToStatus, "testing") {
				if firstInProgressTime.IsZero() {
					firstInProgressTime = historyTime
				}
				startTime = firstInProgressTime // Always use the first In Progress time
				inProgress = true
			}

			// Look for transition to completion state
			if strings.Contains(normalizedToStatus, "done") ||
				strings.Contains(normalizedToStatus, "deployed") ||
				strings.Contains(normalizedToStatus, "closed") ||
				strings.Contains(normalizedToStatus, "resolved") ||
				strings.Contains(normalizedToStatus, "won't") ||
				strings.Contains(normalizedToStatus, "cancelled") {
				endTime = historyTime
				// If we weren't in progress, use the completion time as start time
				if !inProgress && startTime.IsZero() {
					startTime = historyTime
				}
			}
		}
	}

	// Ensure endTime is not before startTime
	if !endTime.IsZero() && !startTime.IsZero() && endTime.Before(startTime) {
		// If endTime is before startTime, swap them
		startTime, endTime = endTime, startTime
	}

	return startTime, endTime
}

// WorkTimeCalculator is the main calculator that uses a strategy
// This follows the Strategy pattern and Dependency Injection principles
type WorkTimeCalculator struct {
	strategy TimeCalculationStrategy
}

// NewWorkTimeCalculator creates a new work time calculator with the specified strategy
func NewWorkTimeCalculator(strategy TimeCalculationStrategy) *WorkTimeCalculator {
	return &WorkTimeCalculator{
		strategy: strategy,
	}
}

// CalculateWorkingHours delegates to the configured strategy
func (calc *WorkTimeCalculator) CalculateWorkingHours(ctx context.Context, issue JiraIssue, sprintBoundary SprintBoundary) (float64, error) {
	return calc.strategy.CalculateWorkingHours(ctx, issue, sprintBoundary)
}

// SetStrategy allows changing the calculation strategy at runtime
func (calc *WorkTimeCalculator) SetStrategy(strategy TimeCalculationStrategy) {
	calc.strategy = strategy
}

// roundHours rounds hours to 2 decimal places to avoid floating-point precision issues
func roundHours(hours float64) float64 {
	return math.Round(hours*100) / 100
}
