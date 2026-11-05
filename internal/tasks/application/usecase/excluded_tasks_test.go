package usecase

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/helmedeiros/digital-asset-capitalization/internal/tasks/domain"
)

type stubExcludedLabels []string

func (s stubExcludedLabels) GetExcludedLabels(string) []string { return s }

func TestDropExcludedTasks(t *testing.T) {
	t.Parallel()

	tasks := []*domain.Task{
		{Key: "T-1", Labels: []string{"cap-asset-pricing"}},
		{Key: "T-2", Labels: []string{"operational", "assetcap-exclude"}},
		{Key: "T-3"},
	}

	kept, excluded := dropExcludedTasks(stubExcludedLabels{"assetcap-exclude"}, "T", tasks)
	assert.Equal(t, []*domain.Task{tasks[0], tasks[2]}, kept)
	assert.Equal(t, []string{"T-2"}, excluded)

	kept, excluded = dropExcludedTasks(stubExcludedLabels{}, "T", tasks)
	assert.Len(t, kept, 3, "no labels configured")
	assert.Empty(t, excluded)

	kept, excluded = dropExcludedTasks(nil, "T", tasks)
	assert.Len(t, kept, 3, "no provider")
	assert.Empty(t, excluded)
}

func TestFetchTasks_RemovesStoredCopiesOfExcludedTasks(t *testing.T) {
	t.Parallel()

	kept, err := domain.NewTaskWithSprints("PRI-1", "work", "PRI", []string{"S1"}, "JIRA")
	require.NoError(t, err)
	umbrella, err := domain.NewTaskWithSprints("PRI-2", "Operator", "PRI", []string{"S1"}, "JIRA")
	require.NoError(t, err)
	umbrella.Labels = []string{"assetcap-exclude"}

	remote := new(MockTaskRepository)
	remote.On("FindByProjectAndSprint", mock.Anything, "PRI", "S1").Return([]*domain.Task{kept, umbrella}, nil)
	local := new(MockTaskRepository)
	local.On("Save", mock.Anything, kept).Return(nil)
	local.On("Delete", mock.Anything, "PRI-2").Return(nil)

	uc := NewFetchTasksUseCase(remote, local, nil)
	uc.SetExcludedLabelProvider(stubExcludedLabels{"assetcap-exclude"})

	require.NoError(t, uc.Execute(context.Background(), "PRI", "S1", "jira"))

	local.AssertCalled(t, "Delete", mock.Anything, "PRI-2")
	local.AssertExpectations(t)
}
