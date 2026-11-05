package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

type stubBoardSource struct{ keys []string }

func (s stubBoardSource) FindIssueKeysForSprint(_ context.Context, _, _ string) ([]string, error) {
	return s.keys, nil
}

func TestFetchTasks_AddsBoardIssuesToTheSprint(t *testing.T) {
	t.Parallel()

	sprintTask, err := domain.NewTaskWithSprints("PRI-1", "in sprint", "PRI", []string{"S1"}, "JIRA")
	require.NoError(t, err)
	boardTask, err := domain.NewTaskWithoutSprint("PRI-2", "kanban only", "PRI", "JIRA")
	require.NoError(t, err)

	remote := new(MockTaskRepository)
	remote.On("FindByProjectAndSprint", mock.Anything, "PRI", "S1").Return([]*domain.Task{sprintTask}, nil)
	remote.On("FindByKey", mock.Anything, "PRI-2").Return(boardTask, nil)
	local := new(MockTaskRepository)
	local.On("Save", mock.Anything, mock.Anything).Return(nil)

	uc := NewFetchTasksUseCase(remote, local, nil)
	// PRI-1 is returned by both sources and must not be fetched twice
	uc.SetBoardIssueSource(stubBoardSource{keys: []string{"PRI-1", "PRI-2"}})

	require.NoError(t, uc.Execute(context.Background(), "PRI", "S1", "jira"))

	assert.True(t, boardTask.HasSprint("S1"))
	remote.AssertNotCalled(t, "FindByKey", mock.Anything, "PRI-1")
	local.AssertNumberOfCalls(t, "Save", 2)
}
