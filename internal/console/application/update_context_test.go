package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/console/domain"
)

// updateContextFromResult is a small dispatcher; the default path is
// already covered transitively but the three resource-specific arms
// (asset/task/sprint) were untested. The whole function is a method
// on ConsoleService, but since it only touches ctx + cmd + result, we
// can drive it directly off a zero-valued ConsoleService.
func TestConsoleService_UpdateContextFromResult_AssetBranch(t *testing.T) {
	t.Parallel()
	svc := &ConsoleService{}
	ctx := domain.NewContext("session-1")

	cmd := &domain.Command{
		Intent: domain.CommandIntent{
			Resource: domain.ResourceTypeAsset,
			Action:   domain.CommandTypeCreate,
		},
	}
	cmd.AddParameter("name", "Payment Gateway")

	result := &domain.CommandResult{Output: map[string]string{"status": "ok"}}

	svc.updateContextFromResult(ctx, cmd, result)

	contains := false
	for _, recent := range ctx.RecentAssets {
		if recent == "Payment Gateway" {
			contains = true
		}
	}
	require.True(t, contains, "asset branch should record the asset in RecentAssets")
}

func TestConsoleService_UpdateContextFromResult_AssetReadAlsoUpdates(t *testing.T) {
	t.Parallel()
	svc := &ConsoleService{}
	ctx := domain.NewContext("s")

	cmd := &domain.Command{
		Intent: domain.CommandIntent{
			Resource: domain.ResourceTypeAsset,
			Action:   domain.CommandTypeRead,
		},
	}
	cmd.AddParameter("name", "Search")
	svc.updateContextFromResult(ctx, cmd, &domain.CommandResult{})
	assert.Contains(t, ctx.RecentAssets, "Search")
}

func TestConsoleService_UpdateContextFromResult_AssetNoUpdateOnUnknownAction(t *testing.T) {
	t.Parallel()
	svc := &ConsoleService{}
	ctx := domain.NewContext("s")

	cmd := &domain.Command{
		Intent: domain.CommandIntent{
			Resource: domain.ResourceTypeAsset,
			Action:   domain.CommandTypeDelete,
		},
	}
	cmd.AddParameter("name", "Search")
	svc.updateContextFromResult(ctx, cmd, &domain.CommandResult{})
	assert.NotContains(t, ctx.RecentAssets, "Search", "delete action should not update asset context")
}

func TestConsoleService_UpdateContextFromResult_TaskBranch(t *testing.T) {
	t.Parallel()
	svc := &ConsoleService{}
	ctx := domain.NewContext("s")

	cmd := &domain.Command{
		Intent: domain.CommandIntent{Resource: domain.ResourceTypeTask},
	}
	cmd.AddParameter("key", "FN-42")
	svc.updateContextFromResult(ctx, cmd, &domain.CommandResult{})
	assert.Contains(t, ctx.RecentTasks, "FN-42")
}

func TestConsoleService_UpdateContextFromResult_SprintBranch(t *testing.T) {
	t.Parallel()
	svc := &ConsoleService{}
	ctx := domain.NewContext("s")

	cmd := &domain.Command{
		Intent: domain.CommandIntent{Resource: domain.ResourceTypeSprint},
	}
	cmd.AddParameter("project", "FN")
	cmd.AddParameter("sprint", "Sprint 1")
	svc.updateContextFromResult(ctx, cmd, &domain.CommandResult{})
	assert.Equal(t, "FN", ctx.CurrentProject)
	assert.Equal(t, "Sprint 1", ctx.CurrentSprint)
}

func TestConsoleService_UpdateContextFromResult_UnknownResourceIsNoop(t *testing.T) {
	t.Parallel()
	svc := &ConsoleService{}
	before := domain.NewContext("s")
	cmd := &domain.Command{
		Intent: domain.CommandIntent{Resource: domain.ResourceType("nothing")},
	}
	svc.updateContextFromResult(before, cmd, &domain.CommandResult{})
	assert.Empty(t, before.RecentAssets)
	assert.Empty(t, before.RecentTasks)
	assert.Equal(t, "", before.CurrentProject)
	assert.Equal(t, "", before.CurrentSprint)
}
