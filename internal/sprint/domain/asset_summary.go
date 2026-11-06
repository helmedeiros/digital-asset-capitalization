package domain

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// AssetSummaryTab is the name of the cross-sprint asset overview tab.
const AssetSummaryTab = "Assets"

const (
	unassignedAsset = "(unassigned)"
	dateLayout      = "2006-01-02"
	workTypeDev     = "cap-development"
	workTypeMaint   = "cap-maintenance"
	workTypeDisc    = "cap-discovery"
)

type assetStats struct {
	issues                  int
	dev, maint, disc, other float64 // person-sprints per work type
	people                  map[string]float64
	firstActivity           string
	lastActivity            string
	devStart, devEnd        string
	active                  bool
	devActive               bool
}

// BuildAssetSummary condenses all sprint allocations into one row per asset:
// the effort invested (in person-sprints, where 1.0 is one person for one
// whole sprint), when implementation started and finished, how long the asset
// has been worked on and who invested the most. Rows are ordered by effort.
func BuildAssetSummary(allocs []SprintAllocation, now time.Time) [][]string {
	stats := map[string]*assetStats{}
	for _, alloc := range allocs {
		if len(alloc.Rows) < 2 {
			continue
		}
		col := columnIndex(alloc.Rows[0])
		people := personColumns(alloc.Rows[0], col)
		for _, row := range alloc.Rows[1:] {
			asset := cell(row, col["assetName"])
			if asset == "" {
				asset = unassignedAsset
			}
			st := stats[asset]
			if st == nil {
				st = &assetStats{people: map[string]float64{}}
				stats[asset] = st
			}
			st.issues++
			var total float64
			for _, p := range people {
				v := percent(cell(row, p.index))
				st.people[p.name] += v
				total += v
			}
			wt := cell(row, col["workType"])
			switch wt {
			case workTypeDev:
				st.dev += total
			case workTypeMaint:
				st.maint += total
			case workTypeDisc:
				st.disc += total
			default:
				st.other += total
			}
			start, end := cell(row, col["dateStarted"]), cell(row, col["dateCompleted"])
			st.firstActivity = minDate(st.firstActivity, start)
			st.lastActivity = maxDate(st.lastActivity, maxDate(start, end))
			if alloc.Active {
				st.active = true
			}
			if wt == workTypeDev {
				st.devStart = minDate(st.devStart, start)
				st.devEnd = maxDate(st.devEnd, maxDate(start, end))
				if alloc.Active {
					st.devActive = true
				}
			}
		}
	}

	names := make([]string, 0, len(stats))
	for n := range stats {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		a, b := stats[names[i]].sum(), stats[names[j]].sum()
		if a != b {
			return a > b
		}
		return names[i] < names[j]
	})

	rows := make([][]string, 0, len(names)+1)
	rows = append(rows, []string{"asset", "investment (person-sprints)", "development", "maintenance", "discovery", "other", "issues",
		"first activity", "implementation started", "implementation completed", "running (days)", "status", "top contributor", "top contributor (person-sprints)", "top contributor share"})
	for _, n := range names {
		st := stats[n]
		top, topVal := st.topPerson()
		share := 0.0
		if t := st.sum(); t > 0 {
			share = topVal / t
		}
		status := "stable"
		end := st.lastActivity
		if st.devActive {
			status, end = "implementing", now.Format(dateLayout)
		} else if st.active {
			status, end = "running", now.Format(dateLayout)
		} else if st.dev == 0 {
			status = "no development"
		}
		completed := st.devEnd
		if st.devActive {
			completed = ""
		}
		rows = append(rows, []string{n, f2(st.sum()), f2(st.dev), f2(st.maint), f2(st.disc), f2(st.other), strconv.Itoa(st.issues),
			st.firstActivity, st.devStart, completed, days(st.firstActivity, end), status, top, f2(topVal), fmt.Sprintf("%.2f%%", share*100)})
	}
	return rows
}

func (s *assetStats) sum() float64 { return s.dev + s.maint + s.disc + s.other }

func (s *assetStats) topPerson() (string, float64) {
	var best string
	var bestVal float64
	for n, v := range s.people {
		if v > bestVal || (v == bestVal && v > 0 && n < best) {
			best, bestVal = n, v
		}
	}
	return best, bestVal
}

type personColumn struct {
	name  string
	index int
}

// personColumns are the columns after the fixed ones (one per team member);
// engineeringHours is only present when hours were requested.
func personColumns(header []string, col map[string]int) []personColumn {
	first, ok := col["engineeringHours"]
	if !ok {
		first, ok = col["workStream"]
	}
	if !ok {
		return nil
	}
	var out []personColumn
	for i := first + 1; i < len(header); i++ {
		out = append(out, personColumn{name: header[i], index: i})
	}
	return out
}

func columnIndex(header []string) map[string]int {
	m := make(map[string]int, len(header))
	for i, h := range header {
		m[h] = i
	}
	return m
}

func cell(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

// percent reads "6.66%" as 0.0666; blanks are zero.
func percent(s string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
	if err != nil {
		return 0
	}
	return v / 100
}

func minDate(a, b string) string {
	if a == "" || (b != "" && b < a) {
		return b
	}
	return a
}

func maxDate(a, b string) string {
	if b > a {
		return b
	}
	return a
}

func days(from, to string) string {
	f, err1 := time.Parse(dateLayout, from)
	t, err2 := time.Parse(dateLayout, to)
	if err1 != nil || err2 != nil {
		return ""
	}
	return strconv.Itoa(int(t.Sub(f).Hours()/24) + 1)
}

func f2(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }
