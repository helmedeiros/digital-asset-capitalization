package usecase

import (
	"fmt"
	"strings"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
)

// ListSprintsResult represents the result of listing sprints
type ListSprintsResult struct {
	Project   string
	Period    string
	Sprints   []ports.Sprint
	BoardInfo []ports.BoardInfo
}

// ListSprintsUseCase handles listing sprints for a project and time period
type ListSprintsUseCase struct {
	jiraPort ports.JiraPort
}

// NewListSprintsUseCase creates a new list sprints use case
func NewListSprintsUseCase(jiraPort ports.JiraPort) *ListSprintsUseCase {
	return &ListSprintsUseCase{
		jiraPort: jiraPort,
	}
}

// Execute lists sprints for a project and time period
func (u *ListSprintsUseCase) Execute(project, period string) (*ListSprintsResult, error) {
	if project == "" {
		return nil, fmt.Errorf("project is required")
	}

	if period == "" {
		return nil, fmt.Errorf("period is required")
	}

	// Parse the period to get date range
	startDate, endDate, err := u.parsePeriod(period)
	if err != nil {
		return nil, fmt.Errorf("invalid period format: %w", err)
	}

	// Get all sprints for the project (active, future, closed) with board info
	states := []string{"active", "future", "closed"}
	sprints, boardInfo, err := u.jiraPort.GetSprintsForProjectWithBoardInfo(project, states)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch sprints: %w", err)
	}

	// Filter sprints by date range
	filteredSprints := u.filterSprintsByDateRange(sprints, startDate, endDate)

	return &ListSprintsResult{
		Project:   project,
		Period:    period,
		Sprints:   filteredSprints,
		BoardInfo: boardInfo,
	}, nil
}

// parsePeriod parses a period string (e.g., "Q2 2025") and returns start and end dates
func (u *ListSprintsUseCase) parsePeriod(period string) (time.Time, time.Time, error) {
	period = strings.TrimSpace(period)

	// Handle quarter format: "Q1 2025", "Q2 2025", etc.
	if strings.HasPrefix(strings.ToUpper(period), "Q") {
		return u.parseQuarter(period)
	}

	// Handle year format: "2025"
	if len(period) == 4 {
		return u.parseYear(period)
	}

	return time.Time{}, time.Time{}, fmt.Errorf("unsupported period format: %s", period)
}

// parseQuarter parses quarter format like "Q2 2025"
func (u *ListSprintsUseCase) parseQuarter(period string) (time.Time, time.Time, error) {
	parts := strings.Fields(period)
	if len(parts) != 2 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid quarter format: %s", period)
	}

	quarterStr := strings.ToUpper(parts[0])
	yearStr := parts[1]

	if !strings.HasPrefix(quarterStr, "Q") {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid quarter format: %s", period)
	}

	quarter := quarterStr[1:]

	switch quarter {
	case "1":
		startDate := time.Date(parseYear(yearStr), 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(parseYear(yearStr), 3, 31, 23, 59, 59, 0, time.UTC)
		return startDate, endDate, nil
	case "2":
		startDate := time.Date(parseYear(yearStr), 4, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(parseYear(yearStr), 6, 30, 23, 59, 59, 0, time.UTC)
		return startDate, endDate, nil
	case "3":
		startDate := time.Date(parseYear(yearStr), 7, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(parseYear(yearStr), 9, 30, 23, 59, 59, 0, time.UTC)
		return startDate, endDate, nil
	case "4":
		startDate := time.Date(parseYear(yearStr), 10, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(parseYear(yearStr), 12, 31, 23, 59, 59, 0, time.UTC)
		return startDate, endDate, nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("invalid quarter number: %s", quarter)
	}
}

// parseYear parses year format like "2025"
func (u *ListSprintsUseCase) parseYear(period string) (time.Time, time.Time, error) {
	year := parseYear(period)
	if year == 0 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid year: %s", period)
	}
	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)
	return startDate, endDate, nil
}

// parseYear is a helper function to parse year string to int
func parseYear(yearStr string) int {
	// This is a simplified implementation - in production you'd want proper error handling
	year := 0
	_, err := fmt.Sscanf(yearStr, "%d", &year)
	if err != nil {
		return 0
	}
	return year
}

// filterSprintsByDateRange filters sprints that overlap with the given date range
func (u *ListSprintsUseCase) filterSprintsByDateRange(sprints []ports.Sprint, startDate, endDate time.Time) []ports.Sprint {
	var filtered []ports.Sprint

	for _, sprint := range sprints {
		sprintStart, err := time.Parse(time.RFC3339, sprint.StartDate)
		if err != nil {
			continue // Skip sprints with invalid start dates
		}

		sprintEnd, err := time.Parse(time.RFC3339, sprint.EndDate)
		if err != nil {
			continue // Skip sprints with invalid end dates
		}

		// Check if sprint overlaps with the date range
		// Sprint overlaps if:
		// 1. Sprint starts before or on the end date AND
		// 2. Sprint ends after or on the start date
		if sprintStart.Before(endDate.AddDate(0, 0, 1)) && sprintEnd.After(startDate.AddDate(0, 0, -1)) {
			filtered = append(filtered, sprint)
		}
	}

	return filtered
}
