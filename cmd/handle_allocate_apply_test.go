package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sprintusecase "github.com/helmedeiros/digital-asset-capitalization/internal/sprint/application/usecase"
	tasksdomain "github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

// stubLockRepo is a hand-rolled SprintLockRepository for the
// handleAllocateApply branches that read a lock. The interactive y/N
// reprompt path lives behind os.Stdin and is out of scope for this
// test file -- the reachable branches without stdin injection are:
//
//   - dryRun=true skips the lock check entirely
//   - lock check error surfaces wrapped
//   - lock exists + force=false returns the documented refusal
//   - lock exists + force=true falls through (the stdin reprompt
//     happens after, so we can't drive past it without rewiring)
//   - PushAllocationToJira error wraps
//   - happy path (no lock + no push error) returns nil
type stubLockRepo struct {
	lock    *tasksdomain.SprintLock
	findErr error
}

func (r *stubLockRepo) FindLock(context.Context, string, string) (*tasksdomain.SprintLock, error) {
	return r.lock, r.findErr
}
func (r *stubLockRepo) SaveLock(context.Context, *tasksdomain.SprintLock) error {
	return nil
}

func TestApp_HandleAllocateApply(t *testing.T) {
	t.Run("dry-run skips lock check entirely", func(t *testing.T) {
		sprintSvc := &MockSprintService{}
		// dry-run still calls PushAllocationToJira with dryRun=true.
		sprintSvc.On("PushAllocationToJira", "FN", "S1", "", false, true).
			Return("csv,output\n", (*sprintusecase.PushResult)(nil), nil)

		// allocationLockRepo deliberately left nil; lock branch is skipped on dryRun=true.
		app := &App{sprintService: sprintSvc}
		err := app.handleAllocateApply("FN", "S1", "", false, true, false, nil)
		require.NoError(t, err)
		sprintSvc.AssertExpectations(t)
	})

	t.Run("lock check error surfaces wrapped", func(t *testing.T) {
		app := &App{
			sprintService:      &MockSprintService{},
			allocationLockRepo: &stubLockRepo{findErr: errors.New("repo down")},
		}
		err := app.handleAllocateApply("FN", "S1", "", false, false, false, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check allocation lock")
	})

	t.Run("existing lock without force is refused with the documented error", func(t *testing.T) {
		app := &App{
			sprintService: &MockSprintService{},
			allocationLockRepo: &stubLockRepo{lock: &tasksdomain.SprintLock{
				LockedAt:  time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
				TaskCount: 12,
			}},
		}
		err := app.handleAllocateApply("FN", "S1", "", false, false, false, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "was already pushed")
		assert.Contains(t, err.Error(), "Use --force to override")
	})

	t.Run("push error surfaces unchanged on the happy path through the lock branch", func(t *testing.T) {
		sprintSvc := &MockSprintService{}
		sprintSvc.On("PushAllocationToJira", "FN", "S1", "", true, false).
			Return("", (*sprintusecase.PushResult)(nil), errors.New("jira gone"))
		app := &App{
			sprintService:      sprintSvc,
			allocationLockRepo: &stubLockRepo{},
		}
		err := app.handleAllocateApply("FN", "S1", "", true, false, false, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "jira gone")
	})

	t.Run("happy path with no lock and a successful push returns nil", func(t *testing.T) {
		sprintSvc := &MockSprintService{}
		sprintSvc.On("PushAllocationToJira", "FN", "S1", "", false, false).
			Return("csv,output\n", &sprintusecase.PushResult{}, nil)
		app := &App{
			sprintService:      sprintSvc,
			allocationLockRepo: &stubLockRepo{},
		}
		err := app.handleAllocateApply("FN", "S1", "", false, false, false, nil)
		require.NoError(t, err)
		sprintSvc.AssertExpectations(t)
	})
}
