package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// TabGrouping decides which workbook tab a sprint's allocation lands in.
type TabGrouping string

const (
	GroupNone      TabGrouping = "none"
	GroupByMonth   TabGrouping = "month"
	GroupByQuarter TabGrouping = "quarter"
	GroupByYear    TabGrouping = "year"
)

// ParseTabGrouping reads a grouping name; empty means a single tab.
func ParseTabGrouping(s string) (TabGrouping, error) {
	switch g := TabGrouping(strings.ToLower(strings.TrimSpace(s))); g {
	case "", GroupNone:
		return GroupNone, nil
	case GroupByMonth, GroupByQuarter, GroupByYear:
		return g, nil
	default:
		return "", fmt.Errorf("unsupported grouping %q (use month, quarter, year or none)", s)
	}
}

// TabName names the tab for a sprint starting at the given time. Sprints are
// grouped by their start only, so one that ends in a later month stays in the
// month it started.
func (g TabGrouping) TabName(start time.Time) string {
	switch g {
	case GroupByMonth:
		return start.Format("2006-01")
	case GroupByQuarter:
		return fmt.Sprintf("%d-Q%d", start.Year(), (int(start.Month())-1)/3+1)
	case GroupByYear:
		return start.Format("2006")
	default:
		return "All"
	}
}

// SprintAllocation is one sprint's allocation table: the header row followed
// by one row per issue.
type SprintAllocation struct {
	Name  string
	Start time.Time
	Rows  [][]string
	// Active marks a sprint that is not finished, so its work is still running.
	Active bool
}

// SheetTab is a worksheet: its name and the rows to write.
type SheetTab struct {
	Name string
	Rows [][]string
}

// sprintGapRows is the number of blank rows between two sprints in a tab.
const sprintGapRows = 2

// BuildWorkbook groups the sprint allocations into tabs and stacks the
// sprints of a tab in start order, each under its own title row and separated
// by blank rows. Tabs come out in chronological order.
func BuildWorkbook(allocs []SprintAllocation, grouping TabGrouping) []SheetTab {
	sorted := append([]SprintAllocation(nil), allocs...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Start.Before(sorted[j].Start) })

	var tabs []SheetTab
	index := make(map[string]int)
	for _, alloc := range sorted {
		name := grouping.TabName(alloc.Start)
		i, ok := index[name]
		if !ok {
			i = len(tabs)
			index[name] = i
			tabs = append(tabs, SheetTab{Name: name})
		}
		tab := &tabs[i]
		if len(tab.Rows) > 0 {
			for range sprintGapRows {
				tab.Rows = append(tab.Rows, []string{})
			}
		}
		tab.Rows = append(tab.Rows, []string{alloc.Name})
		tab.Rows = append(tab.Rows, alloc.Rows...)
	}
	return tabs
}
