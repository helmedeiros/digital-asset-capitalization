package usecase

import (
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
)

// sprintAllocator is what the export needs from the sprint service.
type sprintAllocator interface {
	ListSprints(project, period string) (*ListSprintsResult, error)
	ProcessJiraIssuesWithOptions(project, sprint, override string, useSprintBoundedCalculation bool, opts ...SprintAllocationOption) (string, error)
}

// ExportAllocationsInput selects the sprints and the workbook layout.
type ExportAllocationsInput struct {
	Project  string
	Period   string // e.g. "Q4 2026" or "2026"
	Grouping domain.TabGrouping
	Output   string
	// Options are passed to every sprint's allocation (work streams, hours, exclusions)
	Options []SprintAllocationOption
}

// ExportAllocationsResult summarises what was written.
type ExportAllocationsResult struct {
	Sprints []string
	Tabs    []string
}

// ExportAllocationsUseCase writes the allocation of every sprint of a period
// into one workbook.
type ExportAllocationsUseCase struct {
	sprints sprintAllocator
	writer  ports.WorkbookWriter
}

// NewExportAllocationsUseCase creates the use case.
func NewExportAllocationsUseCase(sprints sprintAllocator, writer ports.WorkbookWriter) *ExportAllocationsUseCase {
	return &ExportAllocationsUseCase{sprints: sprints, writer: writer}
}

// Execute allocates each sprint that starts inside the period (sprint-bounded,
// like `sprint allocate --sprint-bounded`), groups them into tabs by start date
// and writes the workbook.
func (u *ExportAllocationsUseCase) Execute(in ExportAllocationsInput) (*ExportAllocationsResult, error) {
	if in.Output == "" {
		return nil, fmt.Errorf("output file is required")
	}
	listed, err := u.sprints.ListSprints(in.Project, in.Period)
	if err != nil {
		return nil, err
	}
	periodStart, periodEnd, err := NewListSprintsUseCase(nil).parsePeriod(in.Period)
	if err != nil {
		return nil, fmt.Errorf("invalid period format: %w", err)
	}

	var allocs []domain.SprintAllocation
	var names []string
	for _, sprint := range listed.Sprints {
		start, err := time.Parse(time.RFC3339, sprint.StartDate)
		if err != nil || !startsWithin(start, periodStart, periodEnd) {
			continue
		}
		out, err := u.sprints.ProcessJiraIssuesWithOptions(in.Project, sprint.Name, "", true, in.Options...)
		if err != nil {
			return nil, fmt.Errorf("failed to allocate %s: %w", sprint.Name, err)
		}
		rows, err := csv.NewReader(strings.NewReader(out)).ReadAll()
		if err != nil {
			return nil, fmt.Errorf("failed to read allocation of %s: %w", sprint.Name, err)
		}
		allocs = append(allocs, domain.SprintAllocation{Name: sprintTitle(sprint), Start: start, Rows: rows, Active: !strings.EqualFold(sprint.State, "closed")})
		names = append(names, sprint.Name)
	}
	if len(allocs) == 0 {
		return nil, fmt.Errorf("no sprints of project %s start in %s", in.Project, in.Period)
	}

	tabs := domain.BuildWorkbook(allocs, in.Grouping)
	tabs = append([]domain.SheetTab{{Name: domain.AssetSummaryTab, Rows: domain.BuildAssetSummary(allocs, time.Now())}}, tabs...)
	if err := u.writer.Write(in.Output, tabs); err != nil {
		return nil, err
	}
	result := &ExportAllocationsResult{Sprints: names}
	for _, tab := range tabs {
		result.Tabs = append(result.Tabs, tab.Name)
	}
	return result, nil
}

// startsWithin compares calendar days, so a sprint starting late on the last
// day of the period still belongs to it.
func startsWithin(start, from, to time.Time) bool {
	day := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	return !day.Before(from) && !day.After(to)
}

// sprintTitle is the title row above a sprint's table; unfinished sprints are
// flagged because their allocation is not final.
func sprintTitle(sprint ports.Sprint) string {
	if state := strings.ToLower(sprint.State); state != "" && state != "closed" {
		return fmt.Sprintf("%s (%s)", sprint.Name, state)
	}
	return sprint.Name
}
