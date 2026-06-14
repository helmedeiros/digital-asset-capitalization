package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sprintusecase "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/application/usecase"
	sprintports "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/domain/ports"
)

func TestSprintServiceAdapter_ListSprints(t *testing.T) {
	ctx := context.Background()

	t.Run("empty project is rejected", func(t *testing.T) {
		adapter := &SprintServiceAdapter{service: &MockSprintService{}}
		got, err := adapter.ListSprints(ctx, "", "")
		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "project is required")
	})

	t.Run("empty period falls back to 'current' on the service call", func(t *testing.T) {
		mock := &MockSprintService{}
		mock.On("ListSprints", "FN", "current").Return(&sprintusecase.ListSprintsResult{}, nil)
		adapter := &SprintServiceAdapter{service: mock}
		_, err := adapter.ListSprints(ctx, "FN", "")
		require.NoError(t, err)
		mock.AssertExpectations(t)
	})

	t.Run("service error is wrapped", func(t *testing.T) {
		mock := &MockSprintService{}
		mock.On("ListSprints", "FN", "Q2 2026").Return(nil, errors.New("repo down"))
		adapter := &SprintServiceAdapter{service: mock}
		got, err := adapter.ListSprints(ctx, "FN", "Q2 2026")
		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "failed to list sprints")
	})

	t.Run("empty result returns a no-sprints-found message", func(t *testing.T) {
		mock := &MockSprintService{}
		mock.On("ListSprints", "FN", "Q2 2026").Return(&sprintusecase.ListSprintsResult{}, nil)
		adapter := &SprintServiceAdapter{service: mock}
		got, err := adapter.ListSprints(ctx, "FN", "Q2 2026")
		require.NoError(t, err)
		m, ok := got.(map[string]string)
		require.True(t, ok, "no-sprints branch should return map[string]string")
		assert.Contains(t, m["message"], "No sprints found")
	})

	t.Run("populated result transforms sprints (with and without goal)", func(t *testing.T) {
		mock := &MockSprintService{}
		mock.On("ListSprints", "FN", "Q2 2026").Return(&sprintusecase.ListSprintsResult{
			Sprints: []sprintports.Sprint{
				{Name: "Sprint 1", StartDate: "2026-04-01", EndDate: "2026-04-14", State: "closed", Goal: "Ship feature X"},
				{Name: "Sprint 2", StartDate: "2026-04-15", EndDate: "2026-04-28", State: "active"},
			},
		}, nil)
		adapter := &SprintServiceAdapter{service: mock}
		got, err := adapter.ListSprints(ctx, "FN", "Q2 2026")
		require.NoError(t, err)

		m, ok := got.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "FN", m["project"])
		assert.Equal(t, "Q2 2026", m["period"])

		sprints, ok := m["sprints"].([]map[string]interface{})
		require.True(t, ok)
		require.Len(t, sprints, 2)
		assert.Equal(t, "Sprint 1", sprints[0]["name"])
		assert.Equal(t, "Ship feature X", sprints[0]["goal"], "goal should be threaded when present")
		_, hasGoal := sprints[1]["goal"]
		assert.False(t, hasGoal, "sprint without a goal should not carry a goal key")
	})
}

func TestSprintServiceAdapter_AllocateSprint(t *testing.T) {
	ctx := context.Background()

	t.Run("empty project is rejected", func(t *testing.T) {
		adapter := &SprintServiceAdapter{service: &MockSprintService{}}
		_, err := adapter.AllocateSprint(ctx, "", "Sprint 1", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project is required")
	})

	t.Run("empty sprint is rejected", func(t *testing.T) {
		adapter := &SprintServiceAdapter{service: &MockSprintService{}}
		_, err := adapter.AllocateSprint(ctx, "FN", "", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "sprint is required")
	})

	t.Run("service error is wrapped", func(t *testing.T) {
		mock := &MockSprintService{}
		mock.On("ProcessJiraIssuesWithStrategy", "FN", "Sprint 1", "", false).
			Return("", errors.New("boom"))
		adapter := &SprintServiceAdapter{service: mock}
		_, err := adapter.AllocateSprint(ctx, "FN", "Sprint 1", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to allocate sprint")
	})

	t.Run("happy path returns task count and total hours derived from the CSV", func(t *testing.T) {
		csv := strings.Join([]string{
			"col1,col2,col3,col4,col5,col6", // header
			"a,b,c,d,e,f",                   // 1 task -> +8h
			"a,b,c,d,e,f",                   // 1 task -> +8h
			"",                              // blank line ignored
		}, "\n")
		mock := &MockSprintService{}
		mock.On("ProcessJiraIssuesWithStrategy", "FN", "Sprint 1", "", true).
			Return(csv, nil)
		adapter := &SprintServiceAdapter{service: mock}
		got, err := adapter.AllocateSprint(ctx, "FN", "Sprint 1", true)
		require.NoError(t, err)

		m, ok := got.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "FN", m["project"])
		assert.Equal(t, "Sprint 1", m["sprint"])
		assert.Equal(t, true, m["bounded"])
		// Two non-empty data rows past the header -> 2 tasks * 8h each.
		assert.Equal(t, 2, m["total_tasks"])
		assert.InDelta(t, 16.0, m["estimated_hours"], 1e-9)
		assert.Equal(t, "sprint-bounded", m["calculation_type"])
	})
}

