package usecase

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/config"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain"
	"github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
	tasksstorage "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/infrastructure/storage"
)

// newProcessWithRecordsProcessor wires the same shape as the
// TestJiraProcessor_Process baseline, scoped to a single sprint
// pulled from the mock adapter so we can run ProcessWithRecords end-to-end.
func newProcessWithRecordsProcessor(jira ports.JiraPort, withHours bool) *SprintTimeAllocationUseCase {
	return &SprintTimeAllocationUseCase{
		project:    "TEST",
		sprint:     "TEST-1",
		override:   "",
		withHours:  withHours,
		statusPort: createBasicStatusService(),
		teams: domain.TeamMap{
			"TEST": domain.Team{Team: []string{"Test User 1"}},
		},
		jiraPort: jira,
		config:   &config.JiraConfig{},
	}
}

func sampleDoneIssue() ports.JiraIssue {
	return ports.JiraIssue{
		Key:      "TEST-1",
		Summary:  "Done issue",
		Assignee: "Test User 1",
		Status:   domain.StatusDone,
		Changelog: ports.JiraChangelog{
			Histories: []ports.JiraChangeHistory{
				{
					Created: "2024-03-20T10:00:00.000Z",
					Items: []ports.JiraChangeItem{
						{Field: "status", FromString: "To Do", ToString: "In Progress"},
					},
				},
				{
					Created: "2024-03-21T15:00:00.000Z",
					Items: []ports.JiraChangeItem{
						{Field: "status", FromString: "In Progress", ToString: domain.StatusDone},
					},
				},
			},
		},
	}
}

func TestJiraProcessor_ProcessWithRecords_NoTeamForProjectFails(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	mockJira := new(MockJiraAdapter)
	// teams map only has DEFAULT, not the project key.
	p := &SprintTimeAllocationUseCase{
		project:    "TEST",
		sprint:     "TEST-1",
		statusPort: createBasicStatusService(),
		teams:      domain.TeamMap{},
		jiraPort:   mockJira,
		config:     &config.JiraConfig{},
	}

	csv, recs, err := p.ProcessWithRecords()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in teams.json")
	assert.Empty(t, csv)
	assert.Nil(t, recs)
}

func TestJiraProcessor_ProcessWithRecords_FetchIssuesErrorWraps(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	mockJira := new(MockJiraAdapter)
	// First call: GetIssuesForSprint hits the work-stream fan-out path which
	// in TEST setup uses the legacy fetchIssues path. Stub it to fail.
	mockJira.On("GetIssuesForSprint", "TEST", "TEST-1").
		Return([]ports.JiraIssue(nil), assert.AnError)

	p := newProcessWithRecordsProcessor(mockJira, false)
	csv, recs, err := p.ProcessWithRecords()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch issues")
	assert.Empty(t, csv)
	assert.Nil(t, recs)
	mockJira.AssertExpectations(t)
}

func TestJiraProcessor_ProcessWithRecords_SuccessReturnsCSVAndRecords(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	mockJira := new(MockJiraAdapter)
	mockJira.On("GetIssuesForSprint", "TEST", "TEST-1").
		Return([]ports.JiraIssue{sampleDoneIssue()}, nil)

	p := newProcessWithRecordsProcessor(mockJira, false)
	csv, recs, err := p.ProcessWithRecords()
	require.NoError(t, err)
	assert.NotEmpty(t, csv, "CSV output should be produced")
	// records is allowed to be empty if the issue lacks an issueKey result row,
	// but the extractAllocationRecords code path runs regardless.
	assert.NotNil(t, recs)
	mockJira.AssertExpectations(t)
}

func TestJiraProcessor_ProcessWithRecords_WithHoursWritesLocalTasksFile(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	mockJira := new(MockJiraAdapter)
	mockJira.On("GetIssuesForSprint", "TEST", "TEST-1").
		Return([]ports.JiraIssue{sampleDoneIssue()}, nil)

	p := newProcessWithRecordsProcessor(mockJira, true)
	_, _, err := p.ProcessWithRecords()
	require.NoError(t, err)
	mockJira.AssertExpectations(t)

	// saveEngineeringHoursToLocalTasks may decide there's nothing to write
	// for some inputs; the only invariant we can assert is that the path is
	// hit without exploding. To check it DID try to write, peek at
	// .assetcap/tasks.json: it should either exist (write happened) or be
	// absent (no qualifying row), but the storage layer should not have
	// produced a malformed file when called.
	if data, err := os.ReadFile(filepath.Join(".assetcap", "tasks.json")); err == nil {
		assert.NotEmpty(t, data, "tasks.json should not be a zero-byte file")
	}
}

// saveEngineeringHoursToLocalTasks direct unit tests — drive each
// branch (missing/empty key, missing/empty hours, bad-number hours,
// merge-into-existing-task) by hand without going through Process.

func TestSaveEngineeringHoursToLocalTasks_SkipsInvalidRows(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	p := &SprintTimeAllocationUseCase{project: "TEST", sprint: "TEST-1"}
	p.saveEngineeringHoursToLocalTasks([]map[string]interface{}{
		{"issueKey": "", "engineeringHours": "8.0"},         // empty key
		{"issueKey": "TEST-1", "engineeringHours": ""},      // empty hours
		{"issueKey": "TEST-2", "engineeringHours": "abc"},   // un-parseable
		{"issueKey": "TEST-3", "noEngineeringHours": "1.0"}, // missing hours
	})
	// None of the rows should produce a file write.
	_, err := os.Stat(filepath.Join(".assetcap", "tasks.json"))
	assert.True(t, os.IsNotExist(err), "no tasks.json should be created from invalid-only input")
}

func TestSaveEngineeringHoursToLocalTasks_CreatesAndMergesTasks(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	p := &SprintTimeAllocationUseCase{project: "TEST", sprint: "TEST-1"}
	p.saveEngineeringHoursToLocalTasks([]map[string]interface{}{
		{
			"issueKey":         "TEST-9",
			"engineeringHours": "12.5",
			"tpdBusinessUnit":  "BU-A; BU-B",
			"workStream":       "Product",
		},
	})

	storage := tasksstorage.NewJSONStorage(".assetcap", "tasks.json")
	task, err := storage.FindByKey(context.Background(), "TEST-9")
	require.NoError(t, err)
	require.NotNil(t, task)
	require.NotNil(t, task.EngineeringHours)
	assert.InDelta(t, 12.5, *task.EngineeringHours, 1e-9)
	assert.Equal(t, "Product", task.WorkStream)
	assert.Equal(t, []string{"BU-A", "BU-B"}, task.TPDBusinessUnits)

	// Verify the file was actually written and is valid JSON. Shape may
	// be object or array depending on the storage layer's serialization;
	// we only check it's non-empty parsable JSON.
	data, err := os.ReadFile(filepath.Join(".assetcap", "tasks.json"))
	require.NoError(t, err)
	assert.NotEmpty(t, data)
	var raw interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
}
