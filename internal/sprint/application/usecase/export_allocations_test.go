package usecase

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
)

type fakeAllocator struct {
	sprints []ports.Sprint
	csv     map[string]string
	called  []string
}

func (f *fakeAllocator) ListSprints(string, string) (*ListSprintsResult, error) {
	return &ListSprintsResult{Sprints: f.sprints}, nil
}

func (f *fakeAllocator) ProcessJiraIssuesWithOptions(_, sprint, _ string, bounded bool, _ ...SprintAllocationOption) (string, error) {
	if !bounded {
		return "", assert.AnError
	}
	f.called = append(f.called, sprint)
	return f.csv[sprint], nil
}

type captureWriter struct {
	path string
	tabs []domain.SheetTab
}

func (c *captureWriter) Write(path string, tabs []domain.SheetTab) error {
	c.path, c.tabs = path, tabs
	return nil
}

func TestExportAllocations(t *testing.T) {
	t.Parallel()

	alloc := &fakeAllocator{
		sprints: []ports.Sprint{
			// overlaps Q4 but started in September: belongs to Q3, not exported
			{Name: "S0", State: "closed", StartDate: "2026-09-29T09:00:00Z", EndDate: "2026-10-13T09:00:00Z"},
			{Name: "S1", State: "closed", StartDate: "2026-10-13T09:00:00Z", EndDate: "2026-11-03T09:00:00Z"},
			{Name: "S2", State: "active", StartDate: "2026-11-03T09:00:00Z", EndDate: "2026-11-17T09:00:00Z"},
		},
		csv: map[string]string{"S1": "issueKey,status\nA-1,Done\n", "S2": "issueKey,status\nA-2,Done\n"},
	}
	w := &captureWriter{}

	res, err := NewExportAllocationsUseCase(alloc, w).Execute(ExportAllocationsInput{
		Project: "A", Period: "Q4 2026", Grouping: domain.GroupByQuarter, Output: "out.xlsx",
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"S1", "S2"}, alloc.called)
	assert.Equal(t, []string{"2026-Q4"}, res.Tabs)
	assert.Equal(t, "out.xlsx", w.path)
	require.Len(t, w.tabs, 1)
	assert.Equal(t, [][]string{
		{"S1"}, {"issueKey", "status"}, {"A-1", "Done"},
		{}, {},
		{"S2 (active)"}, {"issueKey", "status"}, {"A-2", "Done"},
	}, w.tabs[0].Rows)
}

func TestExportAllocations_Errors(t *testing.T) {
	t.Parallel()

	uc := NewExportAllocationsUseCase(&fakeAllocator{}, &captureWriter{})

	_, err := uc.Execute(ExportAllocationsInput{Project: "A", Period: "2026"})
	assert.Error(t, err, "output required")

	_, err = uc.Execute(ExportAllocationsInput{Project: "A", Period: "2026", Output: "x.xlsx"})
	assert.ErrorContains(t, err, "no sprints", "nothing starts in the period")

	_, err = uc.Execute(ExportAllocationsInput{Project: "A", Period: "someday", Output: "x.xlsx"})
	assert.Error(t, err, "bad period")
}
