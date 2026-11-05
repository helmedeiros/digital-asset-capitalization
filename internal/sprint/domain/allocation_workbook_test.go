package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestParseTabGrouping(t *testing.T) {
	t.Parallel()

	for in, want := range map[string]TabGrouping{
		"month": GroupByMonth, "Quarter": GroupByQuarter, " year ": GroupByYear, "none": GroupNone, "": GroupNone,
	} {
		got, err := ParseTabGrouping(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}
	_, err := ParseTabGrouping("week")
	assert.Error(t, err)
}

func TestTabGrouping_TabName(t *testing.T) {
	t.Parallel()

	start := day("2026-10-29")
	assert.Equal(t, "2026-10", GroupByMonth.TabName(start))
	assert.Equal(t, "2026-Q4", GroupByQuarter.TabName(start))
	assert.Equal(t, "2026", GroupByYear.TabName(start))
	assert.Equal(t, "All", GroupNone.TabName(start))
	assert.Equal(t, "2026-Q1", GroupByQuarter.TabName(day("2026-03-31")))
}

func TestBuildWorkbook(t *testing.T) {
	t.Parallel()

	header := []string{"issueKey", "status"}
	allocs := []SprintAllocation{
		{Name: "S2", Start: day("2026-11-02"), Rows: [][]string{header, {"A-3", "Done"}}},
		{Name: "S0", Start: day("2026-10-06"), Rows: [][]string{header, {"A-1", "Done"}}},
		// starts in October, ends in November: belongs to October only
		{Name: "S1", Start: day("2026-10-20"), Rows: [][]string{header, {"A-2", "Done"}, {"A-4", "Done"}}},
	}

	tabs := BuildWorkbook(allocs, GroupByMonth)

	require.Len(t, tabs, 2)
	assert.Equal(t, "2026-10", tabs[0].Name)
	assert.Equal(t, "2026-11", tabs[1].Name)
	assert.Equal(t, [][]string{
		{"S0"}, header, {"A-1", "Done"},
		{}, {},
		{"S1"}, header, {"A-2", "Done"}, {"A-4", "Done"},
	}, tabs[0].Rows, "sprints by start date, two blank rows between, none after the last")
	assert.Equal(t, [][]string{{"S2"}, header, {"A-3", "Done"}}, tabs[1].Rows)

	single := BuildWorkbook(allocs, GroupNone)
	require.Len(t, single, 1)
	assert.Equal(t, "All", single[0].Name)
}

func TestBuildWorkbook_NoSprints(t *testing.T) {
	t.Parallel()

	assert.Empty(t, BuildWorkbook(nil, GroupByMonth))
}
