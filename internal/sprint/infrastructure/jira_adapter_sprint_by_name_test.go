package infrastructure

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// boardsThenSprintsHandler builds an httptest handler that fakes the
// minimum Jira surface needed to exercise GetSprintByName /
// GetIssuesForSprintOnBoard /
// getIssuesForBoardByDateRange via the existing createTestJiraAdapter
// constructor. boardsBody is returned for /rest/agile/1.0/board (the
// project-boards lookup); the per-board sprint endpoint maps boardID
// → response body; the /rest/api/3/search/jql and
// /rest/agile/1.0/board/{id}/issue endpoints get fed by jqlIssuesBody
// and boardIssuesBody respectively (any non-empty body counts as a
// successful response).
type fakeJiraHandler struct {
	boardsBody       string
	boardsStatus     int
	sprintBodyByID   map[string]string // boardID → response body
	sprintStatusByID map[string]int    // boardID → status (defaults 200)
	jqlIssuesBody    string
	jqlIssuesStatus  int
	boardIssuesBody  string

	jqlCalled      int
	boardIssCalled int
}

func (h *fakeJiraHandler) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch {
		case path == fieldAPIPath:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))

		case strings.HasPrefix(path, "/rest/api/3/search/jql"):
			h.jqlCalled++
			status := h.jqlIssuesStatus
			if status == 0 {
				status = http.StatusOK
			}
			w.WriteHeader(status)
			if h.jqlIssuesBody != "" {
				_, _ = w.Write([]byte(h.jqlIssuesBody))
			} else {
				_, _ = w.Write([]byte(`{"issues":[]}`))
			}

		case strings.Contains(path, "/rest/agile/1.0/board/") && strings.HasSuffix(path, "/issue"):
			h.boardIssCalled++
			w.WriteHeader(http.StatusOK)
			body := h.boardIssuesBody
			if body == "" {
				body = `{"issues":[]}`
			}
			_, _ = w.Write([]byte(body))

		case strings.Contains(path, "/rest/agile/1.0/board/") && strings.HasSuffix(path, "/sprint"):
			parts := strings.Split(path, "/")
			var boardID string
			for i, p := range parts {
				if p == "board" && i+1 < len(parts) {
					boardID = parts[i+1]
					break
				}
			}
			status := h.sprintStatusByID[boardID]
			if status == 0 {
				status = http.StatusOK
			}
			w.WriteHeader(status)
			if body, ok := h.sprintBodyByID[boardID]; ok {
				_, _ = w.Write([]byte(body))
			} else {
				_, _ = w.Write([]byte(`{"values":[],"isLast":true,"startAt":0,"maxResults":50}`))
			}

		case path == "/rest/agile/1.0/board":
			status := h.boardsStatus
			if status == 0 {
				status = http.StatusOK
			}
			w.WriteHeader(status)
			if h.boardsBody != "" {
				_, _ = w.Write([]byte(h.boardsBody))
			} else {
				_, _ = w.Write([]byte(`{"values":[]}`))
			}

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func newFakeJiraAdapter(t *testing.T, h *fakeJiraHandler) *JiraAdapter {
	t.Helper()
	cleanupFiles := setupTestFiles(t)
	t.Cleanup(cleanupFiles)
	server := httptest.NewServer(h.handler())
	t.Cleanup(server.Close)
	return createTestJiraAdapter(t, server)
}

// GetSprintByName

func TestJiraAdapter_GetSprintByName_FoundOnFirstBoard(t *testing.T) {
	h := &fakeJiraHandler{
		boardsBody: `{"values":[{"id":1,"name":"B1","type":"scrum"}]}`,
		sprintBodyByID: map[string]string{
			"1": `{"values":[{"id":"42","name":"Alpha","state":"active","startDate":"2026-01-01T00:00:00Z","endDate":"2026-01-14T00:00:00Z"}],"isLast":true,"startAt":0,"maxResults":50}`,
		},
	}
	adapter := newFakeJiraAdapter(t, h)
	sprint, err := adapter.GetSprintByName("TEST", "Alpha")
	require.NoError(t, err)
	require.NotNil(t, sprint)
	assert.Equal(t, "42", sprint.ID)
	assert.Equal(t, "Alpha", sprint.Name)
}

func TestJiraAdapter_GetSprintByName_NotFound(t *testing.T) {
	h := &fakeJiraHandler{
		boardsBody: `{"values":[{"id":1,"name":"B1","type":"scrum"}]}`,
		sprintBodyByID: map[string]string{
			"1": `{"values":[{"id":"99","name":"Other","state":"closed","startDate":"2026-01-01T00:00:00Z","endDate":"2026-01-14T00:00:00Z"}],"isLast":true,"startAt":0,"maxResults":50}`,
		},
	}
	adapter := newFakeJiraAdapter(t, h)
	_, err := adapter.GetSprintByName("TEST", "Missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sprint 'Missing' not found")
}

func TestJiraAdapter_GetSprintByName_BoardsLookupFails(t *testing.T) {
	h := &fakeJiraHandler{boardsStatus: http.StatusInternalServerError, boardsBody: `boom`}
	adapter := newFakeJiraAdapter(t, h)
	_, err := adapter.GetSprintByName("TEST", "Alpha")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get sprints for project")
}

// GetIssuesForSprintOnBoard

func TestJiraAdapter_GetIssuesForSprintOnBoard_ScrumBoardSprintMatched(t *testing.T) {
	// Board 1 has the sprint, so the JQL path runs (sprintID populated).
	h := &fakeJiraHandler{
		boardsBody: `{"values":[{"id":1,"name":"B1","type":"scrum"}]}`,
		sprintBodyByID: map[string]string{
			"1": `{"values":[{"id":"42","name":"Alpha","state":"active","startDate":"2026-01-01T00:00:00Z","endDate":"2026-01-14T00:00:00Z"}],"isLast":true,"startAt":0,"maxResults":50}`,
		},
		jqlIssuesBody: `{"issues":[{"key":"FN-1","fields":{"summary":"S"}}]}`,
	}
	adapter := newFakeJiraAdapter(t, h)
	issues, err := adapter.GetIssuesForSprintOnBoard("TEST", "Alpha", 1)
	require.NoError(t, err)
	assert.NotNil(t, issues)
	assert.Equal(t, 1, h.jqlCalled, "JQL search endpoint should be hit once")
}

func TestJiraAdapter_GetIssuesForSprintOnBoard_KanbanFallsBackToDateRange(t *testing.T) {
	// Kanban board: getSprintsForBoard for board 99 returns the
	// magic "Board does not support sprints" body, triggering the
	// kanban-fallback branch. Board 1 (scrum) carries the sprint
	// dates used by GetSprintByName.
	h := &fakeJiraHandler{
		boardsBody: `{"values":[{"id":1,"name":"B1","type":"scrum"}]}`,
		sprintBodyByID: map[string]string{
			"1":  `{"values":[{"id":"42","name":"Alpha","state":"active","startDate":"2026-01-01T00:00:00Z","endDate":"2026-01-14T00:00:00Z"}],"isLast":true,"startAt":0,"maxResults":50}`,
			"99": `{"error":"Board does not support sprints"}`,
		},
		sprintStatusByID: map[string]int{"99": http.StatusBadRequest},
	}
	adapter := newFakeJiraAdapter(t, h)
	issues, err := adapter.GetIssuesForSprintOnBoard("TEST", "Alpha", 99)
	require.NoError(t, err)
	assert.NotNil(t, issues)
	assert.Equal(t, 1, h.boardIssCalled, "kanban path should hit /board/{id}/issue once")
}

func TestJiraAdapter_GetIssuesForSprintOnBoard_KanbanFallbackButSprintNotFound(t *testing.T) {
	// Kanban board returns no sprints AND GetSprintByName fails →
	// the date-range fallback fails with a wrapped error.
	h := &fakeJiraHandler{
		boardsBody: `{"values":[{"id":1,"name":"B1","type":"scrum"}]}`,
		sprintBodyByID: map[string]string{
			"1":  `{"values":[],"isLast":true,"startAt":0,"maxResults":50}`,
			"99": `{"error":"Board does not support sprints"}`,
		},
		sprintStatusByID: map[string]int{"99": http.StatusBadRequest},
	}
	adapter := newFakeJiraAdapter(t, h)
	_, err := adapter.GetIssuesForSprintOnBoard("TEST", "Alpha", 99)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get sprint details for date range")
}

func TestJiraAdapter_GetIssuesForSprintOnBoard_SprintNotOnBoardFallsBackToDateRange(t *testing.T) {
	// Board 1 returns the sprint (so GetSprintByName succeeds);
	// board 99 returns a DIFFERENT sprint list (sprint endpoint
	// works but doesn't contain "Alpha"). That trips the
	// "sprint not on this board" branch, which goes through the
	// date-range path.
	h := &fakeJiraHandler{
		boardsBody: `{"values":[{"id":1,"name":"B1","type":"scrum"},{"id":99,"name":"B99","type":"scrum"}]}`,
		sprintBodyByID: map[string]string{
			"1":  `{"values":[{"id":"42","name":"Alpha","state":"active","startDate":"2026-01-01T00:00:00Z","endDate":"2026-01-14T00:00:00Z"}],"isLast":true,"startAt":0,"maxResults":50}`,
			"99": `{"values":[{"id":"77","name":"Beta","state":"active","startDate":"2026-02-01T00:00:00Z","endDate":"2026-02-14T00:00:00Z"}],"isLast":true,"startAt":0,"maxResults":50}`,
		},
	}
	adapter := newFakeJiraAdapter(t, h)
	issues, err := adapter.GetIssuesForSprintOnBoard("TEST", "Alpha", 99)
	require.NoError(t, err)
	assert.NotNil(t, issues)
	assert.Equal(t, 1, h.boardIssCalled, "date-range path should hit /board/{id}/issue once")
}

// getIssuesForBoardByDateRange (covered transitively above; pin the
// two parse-error branches here).

func TestJiraAdapter_getIssuesForBoardByDateRange_BadStartDate(t *testing.T) {
	adapter := newFakeJiraAdapter(t, &fakeJiraHandler{})
	_, err := adapter.getIssuesForBoardByDateRange("TEST", 5, "not-a-date", "2026-01-14T00:00:00Z")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse start date")
}

func TestJiraAdapter_getIssuesForBoardByDateRange_BadEndDate(t *testing.T) {
	adapter := newFakeJiraAdapter(t, &fakeJiraHandler{})
	_, err := adapter.getIssuesForBoardByDateRange("TEST", 5, "2026-01-01T00:00:00Z", "not-a-date")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse end date")
}
