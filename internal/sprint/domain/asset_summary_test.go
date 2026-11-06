package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAssetSummary(t *testing.T) {
	t.Parallel()

	header := []string{"sprint", "issueKey", "workType", "assetName", "dateStarted", "dateCompleted", "engineeringHours", "Ann", "Bob"}
	s1 := SprintAllocation{Name: "S1", Rows: [][]string{header,
		{"S1", "K-1", "cap-development", "cap-asset-a", "2026-03-02", "2026-03-05", "0", "50.00%", "50.00%"},
		{"S1", "K-2", "cap-maintenance", "cap-asset-b", "2026-03-03", "2026-03-04", "0", "", "100.00%"},
	}}
	s2 := SprintAllocation{Name: "S2", Active: true, Rows: [][]string{header,
		{"S2", "K-3", "cap-development", "cap-asset-a", "2026-03-10", "", "0", "100.00%", ""},
		{"S2", "K-4", "cap-discovery", "", "2026-03-11", "", "0", "", "100.00%"},
	}}

	rows := BuildAssetSummary([]SprintAllocation{s1, s2}, time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC))

	require.Len(t, rows, 4)
	a := rows[1]
	assert.Equal(t, "cap-asset-a", a[0])
	assert.Equal(t, "2.00", a[1], "investment")
	assert.Equal(t, "2026-03-02", a[8], "implementation started")
	assert.Equal(t, "", a[9], "still implementing")
	assert.Equal(t, "19", a[10], "running days until now")
	assert.Equal(t, "implementing", a[11])
	assert.Equal(t, "Ann", a[12])
	assert.Equal(t, "75.00%", a[14])

	assert.Equal(t, "(unassigned)", rows[2][0], "ties sort by name")

	b := rows[3]
	assert.Equal(t, "cap-asset-b", b[0])
	assert.Equal(t, "no development", b[11])
	assert.Equal(t, "2", b[10])
	assert.Equal(t, "Bob", b[12])

}

func TestBuildAssetSummary_Empty(t *testing.T) {
	t.Parallel()
	assert.Len(t, BuildAssetSummary(nil, time.Now()), 1)
}

func TestBuildAssetSummary_WithoutHoursColumn(t *testing.T) {
	t.Parallel()

	rows := BuildAssetSummary([]SprintAllocation{{Name: "S1", Rows: [][]string{
		{"issueKey", "workType", "assetName", "dateStarted", "dateCompleted", "workStream", "Ann"},
		{"K-1", "cap-development", "cap-asset-a", "2026-03-02", "2026-03-05", "", "100.00%"},
	}}}, time.Now())

	assert.Equal(t, "1.00", rows[1][1])
	assert.Equal(t, "Ann", rows[1][12])
}
